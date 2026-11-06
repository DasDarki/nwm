//go:build windows

package fsutil

import (
	"encoding/binary"
	"os"
	"syscall"
	"unicode/utf16"

	"golang.org/x/sys/windows"
)

const fsctlSetReparsePoint = 0x000900A4

func linkDir(target string, link string) error {
	if err := os.Mkdir(link, 0o755); err != nil {
		return err
	}
	if err := setJunction(target, link); err != nil {
		os.Remove(link)
		return ErrLinksUnsupported
	}
	return nil
}

func setJunction(target string, link string) error {
	path, err := windows.UTF16PtrFromString(link)
	if err != nil {
		return err
	}
	handle, err := windows.CreateFile(path, windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)

	substitute := utf16.Encode([]rune(`\??\` + target))
	printable := utf16.Encode([]rune(target))
	names := append(append(append([]uint16{}, substitute...), 0), append(printable, 0)...)
	dataLength := 8 + len(names)*2
	buffer := make([]byte, 8+dataLength)
	binary.LittleEndian.PutUint32(buffer[0:], windows.IO_REPARSE_TAG_MOUNT_POINT)
	binary.LittleEndian.PutUint16(buffer[4:], uint16(dataLength))
	binary.LittleEndian.PutUint16(buffer[8:], 0)
	binary.LittleEndian.PutUint16(buffer[10:], uint16(len(substitute)*2))
	binary.LittleEndian.PutUint16(buffer[12:], uint16(len(substitute)*2+2))
	binary.LittleEndian.PutUint16(buffer[14:], uint16(len(printable)*2))
	for index, character := range names {
		binary.LittleEndian.PutUint16(buffer[16+index*2:], character)
	}
	var returned uint32
	return windows.DeviceIoControl(handle, fsctlSetReparsePoint, &buffer[0], uint32(len(buffer)), nil, 0, &returned, nil)
}

func isLink(_ string, info os.FileInfo) bool {
	if info.Mode()&os.ModeSymlink != 0 {
		return true
	}
	if data, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return data.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
	}
	return false
}

func removeEntry(path string) error {
	if err := os.Remove(path); err == nil {
		return nil
	}
	pointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return windows.RemoveDirectory(pointer)
}
