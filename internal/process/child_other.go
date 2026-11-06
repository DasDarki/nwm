//go:build !windows && !linux

package process

import "syscall"

func childAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
