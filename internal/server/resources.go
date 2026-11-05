package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/DasDarki/nwm/internal/process"
	"github.com/DasDarki/nwm/internal/ui"
	"github.com/pelletier/go-toml/v2"
)

type Kind struct {
	Singular  string
	Plural    string
	CliWord   string
	Directory string
	Manifest  string
	ConfigKey string
}

var (
	PackageKind = Kind{Singular: "package", Plural: "packages", CliWord: "package", Directory: "Packages", Manifest: "Package.toml", ConfigKey: "game.packages"}
	AssetKind   = Kind{Singular: "asset pack", Plural: "asset packs", CliWord: "assets", Directory: "Assets", Manifest: "Assets.toml", ConfigKey: "game.assets"}
)

var (
	packageTypes    = []string{"game_mode", "map", "loading_screen", "c_module", "script"}
	singleSlotTypes = []string{"game_mode", "map", "loading_screen"}
)

func SingleSlotType(packageType string) bool {
	return slices.Contains(singleSlotTypes, packageType)
}

type Resource struct {
	Name    string
	Title   string
	Author  string
	Version string
	Type    string
	Path    string
}

func ListResources(directory string, kind Kind) ([]Resource, error) {
	entries, err := os.ReadDir(filepath.Join(directory, kind.Directory))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var resources []Resource
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		resource, err := readResource(filepath.Join(directory, kind.Directory, entry.Name()), kind)
		if err != nil {
			continue
		}
		resources = append(resources, *resource)
	}
	return resources, nil
}

func FindResource(directory string, kind Kind, name string) (*Resource, error) {
	return readResource(filepath.Join(directory, kind.Directory, name), kind)
}

func readResource(path string, kind Kind) (*Resource, error) {
	content, err := os.ReadFile(filepath.Join(path, kind.Manifest))
	if err != nil {
		return nil, err
	}
	var manifest map[string]any
	if err := toml.Unmarshal(content, &manifest); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filepath.Join(path, kind.Manifest), err)
	}
	resource := &Resource{Name: filepath.Base(path), Path: path}
	if meta, ok := manifest["meta"].(map[string]any); ok {
		resource.Title = stringValue(meta["title"])
		resource.Author = stringValue(meta["author"])
		resource.Version = stringValue(meta["version"])
	}
	for _, candidate := range packageTypes {
		if _, ok := manifest[candidate]; ok {
			resource.Type = candidate
			break
		}
	}
	return resource, nil
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func Vault(ctx context.Context, directory string, action string, kind Kind, names []string, report ui.Reporter) error {
	executable, err := CliExecutable(directory)
	if err != nil {
		return err
	}
	arguments := append([]string{"--cli", action, kind.CliWord}, names...)
	command := exec.CommandContext(ctx, executable, arguments...)
	command.Dir = directory
	if err := process.Stream(command, report.Status); err != nil {
		return err
	}
	var missing []string
	for _, name := range names {
		if _, err := FindResource(directory, kind, name); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("the vault did not provide %s: %s", kind.Plural, strings.Join(missing, ", "))
	}
	return nil
}

const UnsafeLibsFlag = "--enable_unsafe_libs"

func BlockedCModules(directory string, arguments []string) []string {
	if slices.Contains(arguments, UnsafeLibsFlag) {
		return nil
	}
	return EnabledCModules(directory)
}

func EnabledCModules(directory string) []string {
	config, err := LoadConfig(directory)
	if err != nil {
		return nil
	}
	value, _ := config.Get(PackageKind.ConfigKey)
	entries, _ := value.([]any)
	var blocked []string
	for _, entry := range entries {
		name := fmt.Sprint(entry)
		if resource, err := FindResource(directory, PackageKind, name); err == nil && resource.Type == "c_module" {
			blocked = append(blocked, name)
		}
	}
	return blocked
}

func Enable(config *Config, kind Kind, resource *Resource) (string, error) {
	if kind == AssetKind || !SingleSlotType(resource.Type) {
		added, err := config.AddToList(kind.ConfigKey, resource.Name)
		if err != nil || !added {
			return "", err
		}
		return kind.ConfigKey, nil
	}
	key := "game." + resource.Type
	return key, config.Set(key, resource.Name)
}

func Disable(config *Config, kind Kind, name string) ([]string, error) {
	var changed []string
	removed, err := config.RemoveFromList(kind.ConfigKey, name)
	if err != nil {
		return nil, err
	}
	if removed {
		changed = append(changed, kind.ConfigKey)
	}
	if kind == PackageKind {
		for _, candidate := range singleSlotTypes {
			key := "game." + candidate
			if config.String(key) == name {
				if err := config.Set(key, ""); err != nil {
					return nil, err
				}
				changed = append(changed, key)
			}
		}
	}
	return slices.Compact(changed), nil
}
