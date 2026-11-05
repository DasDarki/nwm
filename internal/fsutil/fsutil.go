package fsutil

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func SafeJoin(root string, name string) (string, error) {
	target := filepath.Join(root, name)
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes %s", name, root)
	}
	return target, nil
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func CopyFile(source string, target string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}

func CopyDir(source string, target string, skip func(relative string) bool) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative != "." && skip != nil && skip(relative) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		return CopyFile(path, destination)
	})
}

func Copy(source string, target string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return CopyDir(source, target, nil)
	}
	return CopyFile(source, target)
}

func ReplaceDir(staged string, target string) error {
	backup := target + ".nwm-old"
	os.RemoveAll(backup)
	if Exists(target) {
		if err := os.Rename(target, backup); err != nil {
			return err
		}
	}
	if err := os.Rename(staged, target); err != nil {
		os.Rename(backup, target)
		return err
	}
	return os.RemoveAll(backup)
}

func SkipGit(relative string) bool {
	return relative == ".git" || strings.HasPrefix(relative, ".git"+string(filepath.Separator))
}
