//go:build integration

package integration

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/DasDarki/nwm/internal/fsutil"
)

type outputWatcher struct {
	lines chan string
	log   strings.Builder
}

func watchOutput(t *testing.T, reader io.Reader) *outputWatcher {
	watcher := &outputWatcher{lines: make(chan string, 1024)}
	go func() {
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			t.Log(line)
			watcher.lines <- line
		}
		close(watcher.lines)
	}()
	return watcher
}

func (w *outputWatcher) waitFor(t *testing.T, text string, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case line, ok := <-w.lines:
			if !ok {
				t.Fatalf("nwm exited before printing %q", text)
			}
			if strings.Contains(line, text) {
				return
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %q", text)
		}
	}
}

func buildNwm(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "nwm")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "../..")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("building nwm: %v", err)
	}
	return binary
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDevHotReloadWithRealServer(t *testing.T) {
	nwm := buildNwm(t)
	nwmHome := t.TempDir()
	project := t.TempDir()
	writeFile(t, filepath.Join(project, "Package.toml"), "[meta]\ntitle = \"Integration\"\nauthor = \"nwm\"\nversion = \"0.1.0\"\n\n[script]\n")
	writeFile(t, filepath.Join(project, "Server", "Index.lua"), "Console.Log(\"itest v1\")\n")
	writeFile(t, filepath.Join(project, "Recipefile"), "recipe @nwm/itest 0.1.0\n\npackage itest path . enable\n")

	command := exec.Command(nwm, "dev", "--local", "--no-services")
	command.Dir = project
	command.Env = append(os.Environ(), "NWM_HOME="+nwmHome, "NWM_NO_UPDATE_CHECK=1")
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	reader, writer := io.Pipe()
	command.Stdout = writer
	command.Stderr = writer
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan error, 1)
	go func() {
		exited <- command.Wait()
		writer.Close()
	}()
	output := watchOutput(t, reader)

	output.waitFor(t, "itest v1", 15*time.Minute)

	mounted := filepath.Join(project, ".nwm", "server", "Packages", "itest", "Server")
	if !fsutil.IsLink(mounted) {
		t.Errorf("%s should be a link (symlink or junction)", mounted)
	}

	writeFile(t, filepath.Join(project, "Server", "Index.lua"), "Console.Log(\"itest v2\")\n")
	output.waitFor(t, "hot reloaded", time.Minute)
	output.waitFor(t, "itest v2", time.Minute)

	if _, err := io.WriteString(stdin, "/quit\n"); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-exited:
		if err != nil {
			t.Fatalf("nwm dev exited with %v", err)
		}
	case <-time.After(2 * time.Minute):
		command.Process.Kill()
		t.Fatal("nwm dev did not stop after /quit")
	}

	remove := exec.Command(nwm, "server", "remove", "itest-dev", "--delete-files")
	remove.Env = command.Env
	if out, err := remove.CombinedOutput(); err != nil {
		t.Fatalf("removing the server: %v\n%s", err, out)
	}
	for _, source := range []string{"Package.toml", filepath.Join("Server", "Index.lua"), "Recipefile"} {
		if _, err := os.Stat(filepath.Join(project, source)); err != nil {
			t.Errorf("removing the server deleted the source file %s", source)
		}
	}
}
