package dev

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/DasDarki/nwm/internal/process"
	"github.com/DasDarki/nwm/internal/recipe"
	"github.com/DasDarki/nwm/internal/server"
	"github.com/DasDarki/nwm/internal/ui"
	"github.com/fsnotify/fsnotify"
)

const (
	debounceDelay = 300 * time.Millisecond
	quietAfter    = 800 * time.Millisecond
	stopTimeout   = 20 * time.Second
)

type Session struct {
	ServerName string
	ServerDir  string
	Arguments  []string
	Env        []string
	Mounts     []*recipe.Mount
	Input      io.Reader
	Output     io.Writer
}

type pendingChange struct {
	rebuild  bool
	manifest bool
	remount  bool
	resync   map[string]bool
	files    int
}

type runner struct {
	session  *Session
	mutex    sync.Mutex
	console  *server.Console
	builders []*process.Child
	pending  map[*recipe.Mount]*pendingChange
	quiet    map[*recipe.Mount]time.Time
	timer    *time.Timer
	flush    chan struct{}
	quit     chan struct{}
	quitOnce sync.Once
	stopped  chan struct{}
}

func Run(session *Session) error {
	if session.Input == nil {
		session.Input = os.Stdin
	}
	if session.Output == nil {
		session.Output = os.Stdout
	}
	r := &runner{
		session: session,
		pending: map[*recipe.Mount]*pendingChange{},
		quiet:   map[*recipe.Mount]time.Time{},
		flush:   make(chan struct{}, 1),
		quit:    make(chan struct{}),
		stopped: make(chan struct{}, 1),
	}
	defer r.stopBuilders()

	control, err := server.ServeControl(session.ServerName, r.currentConsole)
	if err != nil {
		return err
	}
	defer control.Close()

	r.startBuilders()
	if err := r.startServer(control.Socket()); err != nil {
		return err
	}
	defer server.ClearRunning(session.ServerName)

	var roots []string
	for _, mount := range session.Mounts {
		roots = append(roots, mount.WatchRoots()...)
	}
	files, err := newWatcher(roots)
	if err != nil {
		r.stopServer()
		return err
	}
	defer files.Close()

	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	go r.readInput(control.Socket())
	r.printHelpLine()

	for {
		select {
		case event, ok := <-files.inner.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Create) {
				files.handleCreate(event.Name)
			}
			r.collect(event.Name)
		case err, ok := <-files.inner.Errors:
			if ok {
				r.say("⚠️  watcher: %v", err)
			}
		case <-r.flush:
			r.apply()
		case <-r.stopped:
			r.say("server stopped. %s starts it again, %s leaves", ui.Command("/start"), ui.Command("/quit"))
		case <-signals:
			r.say("stopping...")
			go func() {
				<-signals
				os.Exit(130)
			}()
			r.stopServer()
			return nil
		case <-r.quit:
			r.stopServer()
			return nil
		}
	}
}

func (r *runner) currentConsole() *server.Console {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.console
}

func (r *runner) startServer(socket string) error {
	console, err := server.StartConsole(r.session.ServerDir, r.session.Arguments, r.session.Env, func(line string) {
		r.write(line)
	})
	if err != nil {
		return err
	}
	r.mutex.Lock()
	r.console = console
	r.mutex.Unlock()
	server.RegisterRunning(r.session.ServerName, &server.RunInfo{PID: console.PID(), StartedAt: time.Now(), Detached: false, Socket: socket})
	go func() {
		<-console.Done()
		r.mutex.Lock()
		current := r.console == console
		if current {
			r.console = nil
		}
		r.mutex.Unlock()
		if current {
			server.ClearRunning(r.session.ServerName)
			select {
			case r.stopped <- struct{}{}:
			default:
			}
		}
	}()
	return nil
}

func (r *runner) stopServer() {
	console := r.currentConsole()
	if console == nil {
		return
	}
	r.mutex.Lock()
	r.console = nil
	r.mutex.Unlock()
	console.Stop(stopTimeout)
	server.ClearRunning(r.session.ServerName)
}

func (r *runner) startBuilders() {
	for _, mount := range r.session.Mounts {
		if len(mount.Plan.Watch) > 0 {
			command := exec.Command(mount.Plan.Watch[0], mount.Plan.Watch[1:]...)
			command.Dir = mount.Plan.Root
			if err := r.startChild(mount.Name, command); err != nil {
				r.say("⚠️  %s: cannot start %s: %v", mount.Name, strings.Join(mount.Plan.Watch, " "), err)
			}
		}
		if mount.Plan.Web != "" {
			command := exec.Command("npm", "run", "dev", "--", "--port", strconv.Itoa(mount.Plan.WebPort), "--strictPort")
			command.Dir = mount.Plan.Web
			if err := r.startChild(mount.Name, command); err != nil {
				r.say("⚠️  %s: cannot start the web dev server: %v", mount.Name, err)
			}
		}
	}
}

func (r *runner) startChild(name string, command *exec.Cmd) error {
	reader, writer := io.Pipe()
	command.Stdout = writer
	command.Stderr = writer
	child, err := process.StartChild(command)
	if err != nil {
		writer.Close()
		return err
	}
	prefix := ui.Accent.Render("[" + filepath.Base(command.Args[0]) + " " + name + "]")
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			r.write(prefix + " " + strings.TrimRight(scanner.Text(), "\r"))
		}
	}()
	go func() {
		<-child.Done()
		writer.Close()
	}()
	r.mutex.Lock()
	r.builders = append(r.builders, child)
	r.mutex.Unlock()
	r.say("running %s for %s", strings.Join(command.Args, " "), name)
	return nil
}

func (r *runner) stopBuilders() {
	r.mutex.Lock()
	builders := r.builders
	r.builders = nil
	r.mutex.Unlock()
	for _, builder := range builders {
		builder.Stop(3 * time.Second)
	}
}

func (r *runner) collect(path string) {
	if ignoredFile(path) {
		return
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for _, mount := range r.session.Mounts {
		change, entry := mount.Classify(path)
		if change == recipe.ChangeNone {
			continue
		}
		if change != recipe.ChangeManifest && time.Now().Before(r.quiet[mount]) {
			return
		}
		pending := r.pending[mount]
		if pending == nil {
			pending = &pendingChange{resync: map[string]bool{}}
			r.pending[mount] = pending
		}
		pending.files++
		switch change {
		case recipe.ChangeManifest:
			pending.manifest = true
		case recipe.ChangeSource:
			pending.rebuild = true
		case recipe.ChangeContent:
			if entry == "" {
				pending.remount = true
			} else {
				pending.resync[entry] = true
			}
		}
		if r.timer == nil {
			r.timer = time.AfterFunc(debounceDelay, r.requestFlush)
		} else {
			r.timer.Reset(debounceDelay)
		}
		return
	}
}

func (r *runner) requestFlush() {
	select {
	case r.flush <- struct{}{}:
	default:
	}
}

func (r *runner) apply() {
	r.mutex.Lock()
	pending := r.pending
	r.pending = map[*recipe.Mount]*pendingChange{}
	r.mutex.Unlock()

	var hot, full []string
	files := 0
	for mount, change := range pending {
		files += change.files
		if change.rebuild {
			if !r.rebuild(mount) {
				continue
			}
		}
		if change.remount {
			if remounted, err := recipe.Assemble(mount.Name, mount.Kind, mount.Plan, mount.Target, true); err == nil {
				*mount = *remounted
			} else {
				r.say("❌ %s: %v", mount.Name, err)
				continue
			}
		} else {
			for entry := range change.resync {
				if err := mount.Resync(entry); err != nil {
					r.say("❌ %s: %v", mount.Name, err)
				}
			}
		}
		if change.manifest {
			if err := mount.SyncManifest(); err != nil {
				r.say("❌ %s: %v", mount.Name, err)
				continue
			}
		}
		switch {
		case !mount.IsPackage():
			r.say("asset pack %s changed, %s loads it", mount.Name, ui.Command("/restart"))
		case change.manifest || mount.Plan.Reload == recipe.ReloadFull:
			full = append(full, mount.Name)
		default:
			hot = append(hot, mount.Name)
		}
	}
	sort.Strings(hot)
	sort.Strings(full)
	r.reload(hot, full, files)
}

func (r *runner) rebuild(mount *recipe.Mount) bool {
	r.say("🔨 building %s", mount.Name)
	command := exec.Command(mount.Plan.Build[0], mount.Plan.Build[1:]...)
	command.Dir = mount.Plan.Root
	prefix := ui.Accent.Render("[build " + mount.Name + "]")
	err := process.Stream(command, func(line string) { r.write(prefix + " " + line) })
	r.mutex.Lock()
	r.quiet[mount] = time.Now().Add(quietAfter)
	r.mutex.Unlock()
	if err != nil {
		r.say("❌ build of %s failed, keeping the running version", mount.Name)
		return false
	}
	return true
}

func (r *runner) reload(hot []string, full []string, files int) {
	console := r.currentConsole()
	if console == nil {
		if len(hot)+len(full) > 0 {
			r.say("changes in %s are picked up on the next start", strings.Join(append(hot, full...), ", "))
		}
		return
	}
	if len(hot) > 0 {
		r.say("♻️  hot reloading %s %s", strings.Join(hot, ", "), ui.Muted.Render(fmt.Sprintf("(%d changes)", files)))
		console.Send("package hotreload " + strings.Join(hot, " "))
	}
	if len(full) > 0 {
		r.say("🔄 reloading %s", strings.Join(full, ", "))
		console.Send("package reload " + strings.Join(full, " "))
	}
}

func (r *runner) readInput(socket string) {
	scanner := bufio.NewScanner(r.session.Input)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "/") {
			r.command(line, socket)
			continue
		}
		console := r.currentConsole()
		if console == nil {
			r.say("the server is not running, %s starts it", ui.Command("/start"))
			continue
		}
		console.Send(line)
	}
}

func (r *runner) command(line string, socket string) {
	fields := strings.Fields(line)
	names := fields[1:]
	if len(names) == 0 || slices.Contains(names, "all") {
		names = r.mountNames()
	}
	switch fields[0] {
	case "/reload":
		r.reload(nil, names, 0)
	case "/hotreload":
		r.reload(names, nil, 0)
	case "/restart":
		r.say("restarting the server")
		r.stopServer()
		if err := r.startServer(socket); err != nil {
			r.say("❌ %v", err)
		}
	case "/start":
		if r.currentConsole() != nil {
			r.say("the server is already running")
			return
		}
		if err := r.startServer(socket); err != nil {
			r.say("❌ %v", err)
		}
	case "/stop":
		r.stopServer()
	case "/status":
		for _, mount := range r.session.Mounts {
			mode := "copied"
			if mount.Linked() {
				mode = "linked"
			}
			r.say("%s  %s  %s  reload %s", ui.Bold.Render(mount.Name), mode, mount.Plan.Root, mount.Plan.Reload)
		}
	case "/quit", "/exit":
		r.quitOnce.Do(func() { close(r.quit) })
	case "/help":
		r.say("/hotreload [package...]  /reload [package...]  /restart  /start  /stop  /status  /quit")
		r.say("anything else is sent to the server console")
	default:
		r.say("unknown command %s, try %s", fields[0], ui.Command("/help"))
	}
}

func (r *runner) mountNames() []string {
	var names []string
	for _, mount := range r.session.Mounts {
		if mount.IsPackage() {
			names = append(names, mount.Name)
		}
	}
	return names
}

func (r *runner) printHelpLine() {
	r.say("ready: editing a package reloads it. Type console commands, %s for nwm commands, Ctrl+C to stop", ui.Command("/help"))
}

func (r *runner) say(format string, args ...any) {
	r.write(ui.Accent.Render("[nwm]") + " " + fmt.Sprintf(format, args...))
}

func (r *runner) write(line string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	fmt.Fprintln(r.session.Output, line)
}

var ErrNoMounts = errors.New("the recipe has no local packages to develop, add one with package <name> path <dir>")
