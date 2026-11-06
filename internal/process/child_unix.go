//go:build !windows

package process

import "syscall"

type processTree struct {
	pid int
}

func attachTree(pid int) (*processTree, error) {
	return &processTree{pid: pid}, nil
}

func (t *processTree) terminate() {
	syscall.Kill(-t.pid, syscall.SIGTERM)
}

func (t *processTree) kill() {
	syscall.Kill(-t.pid, syscall.SIGKILL)
}

func (t *processTree) close() {}
