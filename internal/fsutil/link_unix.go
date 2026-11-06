//go:build !windows

package fsutil

import (
	"os"
)

func linkDir(target string, link string) error {
	return os.Symlink(target, link)
}

func isLink(_ string, info os.FileInfo) bool {
	return info.Mode()&os.ModeSymlink != 0
}

func removeEntry(path string) error {
	return os.Remove(path)
}
