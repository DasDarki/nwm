//go:build windows

package process

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

const stillActive = 259

func detachAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS,
		HideWindow:    true,
	}
}

func Alive(pid int) bool {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)
	var exitCode uint32
	if err := windows.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false
	}
	return exitCode == stillActive
}

func requestStop(pid int) error {
	return forceKill(pid)
}

func forceKill(pid int) error {
	running, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return running.Kill()
}
