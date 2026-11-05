//go:build !windows

package process

import (
	"errors"
	"syscall"
)

func detachAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

func Alive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func requestStop(pid int) error {
	return signalGroup(pid, syscall.SIGTERM)
}

func forceKill(pid int) error {
	return signalGroup(pid, syscall.SIGKILL)
}

func signalGroup(pid int, signal syscall.Signal) error {
	if err := syscall.Kill(-pid, signal); err == nil {
		return nil
	}
	return syscall.Kill(pid, signal)
}
