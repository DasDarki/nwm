package fsutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrLinksUnsupported = errors.New("directory links are not supported here")

func LinkDir(target string, link string) error {
	absolute, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", absolute)
	}
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return err
	}
	return linkDir(absolute, link)
}

func IsLink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return isLink(path, info)
}

func RemoveAll(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if isLink(path, info) || !info.IsDir() {
		return removeEntry(path)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := RemoveAll(filepath.Join(path, entry.Name())); err != nil {
			return err
		}
	}
	return os.Remove(path)
}
