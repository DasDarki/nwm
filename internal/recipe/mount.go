package recipe

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/DasDarki/nwm/internal/fsutil"
	"github.com/DasDarki/nwm/internal/server"
)

type MountEntry struct {
	Name    string
	Sources []string
	Linked  bool
}

type Mount struct {
	Name    string
	Kind    server.Kind
	Target  string
	Plan    *BuildPlan
	Entries []MountEntry
}

func (m *Mount) IsPackage() bool {
	return m.Kind == server.PackageKind
}

func Assemble(name string, kind server.Kind, plan *BuildPlan, target string, link bool) (*Mount, error) {
	manifest := filepath.Join(plan.Root, kind.Manifest)
	if _, err := os.Stat(manifest); err != nil {
		return nil, fmt.Errorf("%s %s has no %s in %s", kind.Singular, name, kind.Manifest, plan.Root)
	}
	if err := fsutil.RemoveAll(target); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return nil, err
	}
	mount := &Mount{Name: name, Kind: kind, Target: target, Plan: plan}
	if err := mount.SyncManifest(); err != nil {
		return nil, err
	}
	sources := map[string][]string{}
	var order []string
	collect := func(directory string, skip func(string) bool) error {
		entries, err := os.ReadDir(directory)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".") || entry.Name() == kind.Manifest || skip(entry.Name()) {
				continue
			}
			if _, seen := sources[entry.Name()]; !seen {
				order = append(order, entry.Name())
			}
			sources[entry.Name()] = append(sources[entry.Name()], filepath.Join(directory, entry.Name()))
		}
		return nil
	}
	if err := collect(plan.Root, func(entry string) bool { return slices.Contains(plan.Exclude, entry) }); err != nil {
		return nil, err
	}
	if plan.Output != plan.Root {
		if err := collect(plan.Output, func(string) bool { return false }); err != nil {
			return nil, err
		}
	}
	for _, entryName := range order {
		entry := MountEntry{Name: entryName, Sources: sources[entryName]}
		if link && len(entry.Sources) == 1 && isDirectory(entry.Sources[0]) {
			err := fsutil.LinkDir(entry.Sources[0], filepath.Join(target, entryName))
			if err == nil {
				entry.Linked = true
				mount.Entries = append(mount.Entries, entry)
				continue
			}
			if !errors.Is(err, fsutil.ErrLinksUnsupported) {
				return nil, err
			}
		}
		mount.Entries = append(mount.Entries, entry)
		if err := mount.copyEntry(entry); err != nil {
			return nil, err
		}
	}
	return mount, nil
}

func (m *Mount) SyncManifest() error {
	return fsutil.CopyFile(filepath.Join(m.Plan.Root, m.Kind.Manifest), filepath.Join(m.Target, m.Kind.Manifest))
}

func (m *Mount) Resync(entryName string) error {
	for _, entry := range m.Entries {
		if entry.Name == entryName && !entry.Linked {
			return m.copyEntry(entry)
		}
	}
	return nil
}

func (m *Mount) copyEntry(entry MountEntry) error {
	destination := filepath.Join(m.Target, entry.Name)
	if err := fsutil.RemoveAll(destination); err != nil {
		return err
	}
	for _, source := range entry.Sources {
		if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err := fsutil.Copy(source, destination); err != nil {
			return err
		}
	}
	return nil
}

func (m *Mount) Linked() bool {
	for _, entry := range m.Entries {
		if entry.Linked {
			return true
		}
	}
	return false
}

type Change int

const (
	ChangeNone Change = iota
	ChangeManifest
	ChangeSource
	ChangeContent
)

func (m *Mount) Classify(path string) (Change, string) {
	plan := m.Plan
	if path == filepath.Join(plan.Root, m.Kind.Manifest) {
		return ChangeManifest, ""
	}
	separateOutput := plan.Output != plan.Root
	if !separateOutput || !within(plan.Output, path) {
		for _, source := range plan.Sources {
			if !within(source, path) {
				continue
			}
			if len(plan.Watch) > 0 {
				return ChangeNone, ""
			}
			return ChangeSource, ""
		}
	}
	for _, entry := range m.Entries {
		for _, source := range entry.Sources {
			if within(source, path) {
				return ChangeContent, entry.Name
			}
		}
	}
	if m.isMountable(path) {
		return ChangeContent, ""
	}
	return ChangeNone, ""
}

func (m *Mount) isMountable(path string) bool {
	plan := m.Plan
	for _, directory := range []string{plan.Output, plan.Root} {
		relative, err := filepath.Rel(directory, path)
		if err != nil || relative == "." || strings.HasPrefix(relative, "..") {
			continue
		}
		first := strings.Split(filepath.ToSlash(relative), "/")[0]
		if strings.HasPrefix(first, ".") || first == m.Kind.Manifest {
			return false
		}
		if directory == plan.Root && slices.Contains(plan.Exclude, first) {
			continue
		}
		return true
	}
	return false
}

func (m *Mount) WatchRoots() []string {
	roots := []string{m.Plan.Root}
	if m.Plan.Output != m.Plan.Root && !within(m.Plan.Root, m.Plan.Output) {
		roots = append(roots, m.Plan.Output)
	}
	return roots
}

func within(root string, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
