package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

type Reporter interface {
	Status(text string)
	Progress(fraction float64)
}

type Task func(ctx context.Context, report Reporter) error

type statusMessage string

type progressMessage float64

type doneMessage struct{ err error }

type taskModel struct {
	title      string
	status     string
	fraction   float64
	width      int
	cancelling bool
	cancel     context.CancelFunc
	spinner    spinner.Model
	bar        progress.Model
	done       bool
}

func newTaskModel(title string, cancel context.CancelFunc) taskModel {
	return taskModel{
		title:    title,
		fraction: -1,
		width:    80,
		cancel:   cancel,
		spinner:  spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(Accent)),
		bar:      progress.New(progress.WithGradient("#8B5CF6", "#22C55E"), progress.WithWidth(40)),
	}
}

func (m taskModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m taskModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := message.(type) {
	case tea.KeyMsg:
		if typed.Type == tea.KeyCtrlC {
			m.cancelling = true
			m.cancel()
		}
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.bar.Width = min(50, max(10, typed.Width-12))
	case statusMessage:
		m.status = string(typed)
	case progressMessage:
		m.fraction = float64(typed)
	case doneMessage:
		m.done = true
		return m, tea.Quit
	case spinner.TickMsg:
		var command tea.Cmd
		m.spinner, command = m.spinner.Update(typed)
		return m, command
	}
	return m, nil
}

func (m taskModel) View() string {
	if m.done {
		return ""
	}
	header := m.spinner.View() + " " + Bold.Render(m.title)
	if m.cancelling {
		header += Muted.Render("  cancelling...")
	}
	lines := header + "\n"
	if m.status != "" {
		lines += "  " + Muted.Render(ansi.Truncate(m.status, max(10, m.width-4), "…")) + "\n"
	}
	if m.fraction >= 0 {
		lines += "  " + m.bar.ViewAs(min(1, m.fraction)) + "\n"
	}
	return lines
}

type programReporter struct {
	program      *tea.Program
	mutex        sync.Mutex
	lastProgress time.Time
}

func (r *programReporter) Status(text string) {
	r.program.Send(statusMessage(text))
}

func (r *programReporter) Progress(fraction float64) {
	r.mutex.Lock()
	now := time.Now()
	throttled := fraction < 1 && now.Sub(r.lastProgress) < 50*time.Millisecond
	if !throttled {
		r.lastProgress = now
	}
	r.mutex.Unlock()
	if !throttled {
		r.program.Send(progressMessage(fraction))
	}
}

type plainReporter struct{}

func (plainReporter) Status(string) {}

func (plainReporter) Progress(float64) {}

func Run(title string, task Task) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	started := time.Now()
	if !Interactive() {
		fmt.Fprintf(os.Stderr, "⏳ %s\n", title)
		err := task(ctx, plainReporter{})
		report(title, started, err)
		return err
	}
	program := tea.NewProgram(newTaskModel(title, cancel), tea.WithOutput(os.Stderr))
	reporter := &programReporter{program: program}
	finished := make(chan error, 1)
	go func() {
		err := task(ctx, reporter)
		finished <- err
		program.Send(doneMessage{err: err})
	}()
	if _, err := program.Run(); err != nil && !errors.Is(err, tea.ErrProgramKilled) {
		cancel()
		<-finished
		return err
	}
	taskErr := <-finished
	report(title, started, taskErr)
	return taskErr
}

func report(title string, started time.Time, err error) {
	if err != nil {
		if errors.Is(err, context.Canceled) {
			Warn("%s cancelled", title)
			return
		}
		Error("%s failed", title)
		return
	}
	elapsed := time.Since(started).Round(100 * time.Millisecond)
	fmt.Fprintf(os.Stderr, "✅ %s %s\n", successStyle.Render(title), Muted.Render("("+elapsed.String()+")"))
}

func DownloadProgress(report Reporter, label string) func(done int64, total int64) {
	return func(done int64, total int64) {
		if total > 0 {
			report.Progress(float64(done) / float64(total))
			report.Status(fmt.Sprintf("%s  %s / %s", label, FormatBytes(done), FormatBytes(total)))
			return
		}
		report.Status(fmt.Sprintf("%s  %s", label, FormatBytes(done)))
	}
}
