package process

import (
	"os/exec"
	"time"
)

func Detach(command *exec.Cmd) {
	command.SysProcAttr = detachAttributes()
}

func Terminate(pid int, timeout time.Duration) error {
	if !Alive(pid) {
		return nil
	}
	if err := requestStop(pid); err != nil {
		return forceKill(pid)
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !Alive(pid) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return forceKill(pid)
}
