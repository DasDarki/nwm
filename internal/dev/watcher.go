package dev

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

var ignoredDirectories = map[string]bool{
	".git":          true,
	".idea":         true,
	".vscode":       true,
	".nwm":          true,
	"node_modules":  true,
	"nebra_modules": true,
}

type watcher struct {
	inner *fsnotify.Watcher
}

func newWatcher(roots []string) (*watcher, error) {
	inner, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &watcher{inner: inner}
	for _, root := range roots {
		if err := w.addTree(root); err != nil {
			inner.Close()
			return nil, err
		}
	}
	return w, nil
}

func (w *watcher) addTree(root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		if path != root && ignoredDirectories[entry.Name()] {
			return filepath.SkipDir
		}
		return w.inner.Add(path)
	})
}

func (w *watcher) handleCreate(path string) {
	info, err := os.Stat(path)
	if err == nil && info.IsDir() && !ignoredDirectories[filepath.Base(path)] {
		w.addTree(path)
	}
}

func (w *watcher) Close() {
	w.inner.Close()
}

func ignoredFile(path string) bool {
	name := filepath.Base(path)
	if name == "4913" || strings.HasSuffix(name, "~") || strings.HasPrefix(name, ".#") {
		return true
	}
	for _, suffix := range []string{".tmp", ".swp", ".swx", ".swo", ".bak", ".crdownload"} {
		if strings.HasSuffix(strings.ToLower(name), suffix) {
			return true
		}
	}
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if ignoredDirectories[part] {
			return true
		}
	}
	return false
}
