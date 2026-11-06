package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/DasDarki/nwm/internal/home"
	"github.com/DasDarki/nwm/internal/process"
)

type RunInfo struct {
	PID        int       `json:"pid"`
	StartedAt  time.Time `json:"started_at"`
	Detached   bool      `json:"detached"`
	Executable string    `json:"executable"`
	Socket     string    `json:"socket,omitempty"`
}

func LogFile(name string) string {
	return filepath.Join(home.LogsDir(), name+".log")
}

func runFile(name string) string {
	return filepath.Join(home.RunDir(), name+".json")
}

func Status(name string) (*RunInfo, bool) {
	content, err := os.ReadFile(runFile(name))
	if err != nil {
		return nil, false
	}
	var info RunInfo
	if err := json.Unmarshal(content, &info); err != nil || info.PID <= 0 {
		os.Remove(runFile(name))
		return nil, false
	}
	if !process.Alive(info.PID) {
		os.Remove(runFile(name))
		return nil, false
	}
	return &info, true
}

func StartDetached(name string, directory string, arguments []string, environment []string) (*RunInfo, error) {
	if info, running := Status(name); running {
		return info, fmt.Errorf("server %q is already running (pid %d)", name, info.PID)
	}
	executable, err := LaunchExecutable(directory)
	if err != nil {
		return nil, err
	}
	nwm, err := os.Executable()
	if err != nil {
		return nil, err
	}
	if _, err := home.Ensure(home.LogsDir()); err != nil {
		return nil, err
	}
	logFile, err := os.OpenFile(LogFile(name), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	defer logFile.Close()
	supervisorArguments := append([]string{supervisorCommand, name, directory, "--"}, arguments...)
	command := exec.Command(nwm, supervisorArguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), environment...)
	command.Stdout = logFile
	command.Stderr = logFile
	process.Detach(command)
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("starting server: %w", err)
	}
	info := &RunInfo{PID: command.Process.Pid, StartedAt: time.Now(), Detached: true, Executable: executable, Socket: SocketPath(name)}
	if err := writeRunInfo(name, info); err != nil {
		return info, err
	}
	exited := make(chan error, 1)
	go func() { exited <- command.Wait() }()
	select {
	case err := <-exited:
		os.Remove(runFile(name))
		if err == nil {
			err = errors.New("process exited immediately")
		}
		return nil, fmt.Errorf("server stopped right after starting (%v), see %s", err, LogFile(name))
	case <-time.After(2 * time.Second):
		return info, nil
	}
}

func RunForeground(name string, directory string, arguments []string, environment []string) error {
	if info, running := Status(name); running {
		return fmt.Errorf("server %q is already running (pid %d)", name, info.PID)
	}
	executable, err := LaunchExecutable(directory)
	if err != nil {
		return err
	}
	command := exec.Command(executable, arguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), environment...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	if err := command.Start(); err != nil {
		return fmt.Errorf("starting server: %w", err)
	}
	if err := writeRunInfo(name, &RunInfo{PID: command.Process.Pid, StartedAt: time.Now(), Executable: executable}); err != nil {
		command.Process.Kill()
		return err
	}
	defer os.Remove(runFile(name))
	go func() {
		for received := range signals {
			if received == syscall.SIGTERM {
				command.Process.Signal(syscall.SIGTERM)
			}
		}
	}()
	return command.Wait()
}

func Stop(name string, timeout time.Duration) error {
	info, running := Status(name)
	if !running {
		return fmt.Errorf("server %q is not running", name)
	}
	if info.Socket != "" && control(name, "stop") == nil {
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			if !process.Alive(info.PID) {
				os.Remove(runFile(name))
				return nil
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
	if err := process.Terminate(info.PID, 5*time.Second); err != nil {
		return fmt.Errorf("stopping server: %w", err)
	}
	os.Remove(runFile(name))
	return nil
}

func RegisterRunning(name string, info *RunInfo) error {
	return writeRunInfo(name, info)
}

func ClearRunning(name string) {
	os.Remove(runFile(name))
}

func writeRunInfo(name string, info *RunInfo) error {
	if _, err := home.Ensure(home.RunDir()); err != nil {
		return err
	}
	content, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return os.WriteFile(runFile(name), content, 0o644)
}
