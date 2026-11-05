package recipe

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/DasDarki/nwm/internal/docker"
	"github.com/DasDarki/nwm/internal/download"
	"github.com/DasDarki/nwm/internal/fsutil"
	"github.com/DasDarki/nwm/internal/home"
	"github.com/DasDarki/nwm/internal/nanossharp"
	"github.com/DasDarki/nwm/internal/server"
	"github.com/DasDarki/nwm/internal/ui"
)

type Target struct {
	Name string
	Dir  string
}

type ApplyOptions struct {
	Variables    map[string]string
	Secrets      map[string]string
	SkipServices bool
	SkipHooks    bool
}

func Builtins(target Target) map[string]string {
	return map[string]string{
		"server.name": target.Name,
		"server.path": target.Dir,
		"os":          runtime.GOOS,
	}
}

func parseOverrides(overrides map[string]string) map[string]any {
	parsed := map[string]any{}
	for key, raw := range overrides {
		parsed[key] = server.ParseValue(raw)
	}
	return parsed
}

func Resolve(fetched *Fetched, target Target, overrides map[string]string, secretValues map[string]string) (*Manifest, error) {
	expander := NewExpander(fetched.Manifest.Variables, parseOverrides(overrides), Builtins(target), secretValues)
	expanded := fetched.Manifest.Expand(expander)
	return expanded, expander.Err()
}

func Apply(fetched *Fetched, manifest *Manifest, target Target, options ApplyOptions) error {
	steps := []struct {
		title string
		skip  bool
		run   ui.Task
	}{
		{"Installing vault packages", !hasVault(manifest.Packages), func(ctx context.Context, report ui.Reporter) error {
			return server.Vault(ctx, target.Dir, "install", server.PackageKind, vaultNames(manifest.Packages), report)
		}},
		{"Installing vault asset packs", !hasVault(manifest.Assets), func(ctx context.Context, report ui.Reporter) error {
			return server.Vault(ctx, target.Dir, "install", server.AssetKind, vaultNames(manifest.Assets), report)
		}},
		{"Installing recipe packages", !hasLocal(manifest.Packages), func(ctx context.Context, report ui.Reporter) error {
			return placeResources(ctx, fetched.Dir, target.Dir, server.PackageKind, manifest.Packages, report)
		}},
		{"Installing recipe asset packs", !hasLocal(manifest.Assets), func(ctx context.Context, report ui.Reporter) error {
			return placeResources(ctx, fetched.Dir, target.Dir, server.AssetKind, manifest.Assets, report)
		}},
		{"Installing C modules", len(manifest.Modules) == 0, func(ctx context.Context, report ui.Reporter) error {
			return installModules(ctx, fetched.Dir, target.Dir, manifest.Modules, report)
		}},
		{"Copying files", len(manifest.Files) == 0, func(_ context.Context, report ui.Reporter) error {
			return copyFiles(fetched.Dir, target, manifest.Files, options, fetched.Manifest.Variables, report)
		}},
		{"Writing configuration", false, func(context.Context, ui.Reporter) error {
			return writeConfiguration(target.Dir, manifest)
		}},
		{"Installing NanosSharp", !manifest.Server.NanosSharp || nanossharp.Installed(target.Dir), func(ctx context.Context, report ui.Reporter) error {
			return nanossharp.Install(ctx, target.Dir, report)
		}},
		{"Recording recipe", false, func(context.Context, ui.Reporter) error {
			return recordRecipe(fetched, manifest, target, options.Variables)
		}},
		{"Starting services", options.SkipServices || len(manifest.Services) == 0, func(ctx context.Context, report ui.Reporter) error {
			if err := docker.Available(ctx); err != nil {
				return err
			}
			for _, service := range manifest.Services {
				resolved, err := ResolveServiceSecrets(service, options.Secrets)
				if err != nil {
					return err
				}
				if err := docker.Up(ctx, target.Name, target.Dir, resolved, report); err != nil {
					return err
				}
			}
			return nil
		}},
	}
	for _, step := range steps {
		if step.skip {
			continue
		}
		if err := ui.Run(step.title, step.run); err != nil {
			return err
		}
	}
	if options.SkipHooks {
		return nil
	}
	return runHooks(target, fetched.Dir, manifest.Hooks)
}

func hasVault(resources []Resource) bool {
	return len(vaultNames(resources)) > 0
}

func hasLocal(resources []Resource) bool {
	return len(resources) > len(vaultNames(resources))
}

func vaultNames(resources []Resource) []string {
	var names []string
	for _, resource := range resources {
		if resource.FromVault() {
			names = append(names, resource.Name)
		}
	}
	return names
}

func placeResources(ctx context.Context, recipeDir string, serverDir string, kind server.Kind, resources []Resource, report ui.Reporter) error {
	parent := filepath.Join(serverDir, kind.Directory)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	for _, resource := range resources {
		if resource.FromVault() {
			continue
		}
		target := filepath.Join(parent, resource.Name)
		if resource.Git != "" {
			report.Status("cloning " + resource.Name)
			if err := CloneInto(ctx, resource.Git, resource.Ref, resource.Subdir, target, report); err != nil {
				return fmt.Errorf("%s %s: %w", kind.Singular, resource.Name, err)
			}
		} else {
			report.Status("copying " + resource.Name)
			source, err := fsutil.SafeJoin(recipeDir, resource.Path)
			if err != nil {
				return err
			}
			staged, err := os.MkdirTemp(parent, ".nwm-copy-")
			if err != nil {
				return err
			}
			if err := fsutil.CopyDir(source, staged, fsutil.SkipGit); err != nil {
				os.RemoveAll(staged)
				return err
			}
			if err := fsutil.ReplaceDir(staged, target); err != nil {
				os.RemoveAll(staged)
				return err
			}
		}
		if _, err := server.FindResource(serverDir, kind, resource.Name); err != nil {
			return fmt.Errorf("%s %s has no %s", kind.Singular, resource.Name, kind.Manifest)
		}
	}
	return nil
}

func installModules(ctx context.Context, recipeDir string, serverDir string, modules []Module, report ui.Reporter) error {
	parent := filepath.Join(serverDir, server.PackageKind.Directory)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	for _, module := range modules {
		source := module.Linux
		if runtime.GOOS == "windows" {
			source = module.Windows
		}
		if source == "" {
			report.Status(fmt.Sprintf("module %s has no %s build, skipping", module.Name, runtime.GOOS))
			continue
		}
		if err := installModule(ctx, recipeDir, parent, module, source, report); err != nil {
			return fmt.Errorf("module %s: %w", module.Name, err)
		}
	}
	return nil
}

func installModule(ctx context.Context, recipeDir string, parent string, module Module, source string, report ui.Reporter) error {
	local := source
	if isURL(source) {
		local = filepath.Join(home.CacheDir(), "modules", module.Name, path.Base(strings.SplitN(source, "?", 2)[0]))
		if err := download.File(ctx, source, local, ui.DownloadProgress(report, "downloading "+module.Name)); err != nil {
			return err
		}
		report.Progress(-1)
	} else {
		joined, err := fsutil.SafeJoin(recipeDir, source)
		if err != nil {
			return err
		}
		local = joined
	}
	staged, err := os.MkdirTemp(parent, ".nwm-module-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staged)
	switch {
	case strings.HasSuffix(local, ".zip"):
		err = download.ExtractZip(local, staged)
	case strings.HasSuffix(local, ".tar.gz"), strings.HasSuffix(local, ".tgz"):
		err = download.ExtractTarGz(local, staged)
	default:
		err = fsutil.Copy(local, filepath.Join(staged, filepath.Base(local)))
	}
	if err != nil {
		return err
	}
	root := packageRoot(staged)
	if !fsutil.Exists(filepath.Join(root, server.PackageKind.Manifest)) {
		names := moduleNames(root)
		if len(names) == 0 {
			return fmt.Errorf("no .so or .dll found in %s", source)
		}
		if err := os.WriteFile(filepath.Join(root, server.PackageKind.Manifest), []byte(modulePackageManifest(module.Name, names)), 0o644); err != nil {
			return err
		}
	}
	return fsutil.ReplaceDir(root, filepath.Join(parent, module.Name))
}

func packageRoot(directory string) string {
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 || !entries[0].IsDir() {
		return directory
	}
	return filepath.Join(directory, entries[0].Name())
}

func moduleNames(directory string) []string {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil
	}
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		var module string
		switch {
		case strings.HasSuffix(name, ".dll"):
			module = strings.TrimSuffix(name, ".dll")
		case strings.HasSuffix(name, ".so"):
			module = strings.TrimPrefix(strings.TrimSuffix(name, ".so"), "lib")
		default:
			continue
		}
		if module != "" && !slices.Contains(names, module) {
			names = append(names, module)
		}
	}
	slices.Sort(names)
	return names
}

func modulePackageManifest(name string, modules []string) string {
	quoted := make([]string, len(modules))
	for index, module := range modules {
		quoted[index] = strconv.Quote(module)
	}
	return fmt.Sprintf("[meta]\ntitle = %s\nauthor = \"nwm\"\nversion = \"1.0.0\"\n\n[c_module]\nmodules = [%s]\n", strconv.Quote(name), strings.Join(quoted, ", "))
}

func copyFiles(recipeDir string, target Target, files []File, options ApplyOptions, defaults map[string]any, report ui.Reporter) error {
	parsed := parseOverrides(options.Variables)
	for _, file := range files {
		source, err := fsutil.SafeJoin(recipeDir, file.Source)
		if err != nil {
			return err
		}
		destination, err := fsutil.SafeJoin(target.Dir, file.Target)
		if err != nil {
			return err
		}
		report.Status(file.Source + " -> " + file.Target)
		if !file.Template {
			if err := fsutil.Copy(source, destination); err != nil {
				return err
			}
			continue
		}
		content, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		expander := NewExpander(defaults, parsed, Builtins(target), options.Secrets)
		rendered := expander.String(string(content))
		if err := expander.Err(); err != nil {
			return fmt.Errorf("%s: %w", file.Source, err)
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(destination, []byte(rendered), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func writeConfiguration(serverDir string, manifest *Manifest) error {
	config, err := server.LoadConfig(serverDir)
	if err != nil {
		return err
	}
	config.Merge(manifest.Config)
	modulePackages := make([]Resource, 0, len(manifest.Modules))
	for _, module := range manifest.Modules {
		source := module.Linux
		if runtime.GOOS == "windows" {
			source = module.Windows
		}
		if source != "" {
			modulePackages = append(modulePackages, Resource{Name: module.Name, Enable: module.Enable})
		}
	}
	for _, group := range []struct {
		kind      server.Kind
		resources []Resource
	}{{server.PackageKind, manifest.Packages}, {server.PackageKind, modulePackages}, {server.AssetKind, manifest.Assets}} {
		for _, resource := range group.resources {
			if !resource.Enable {
				continue
			}
			found, err := server.FindResource(serverDir, group.kind, resource.Name)
			if err != nil {
				return fmt.Errorf("cannot enable %s: %w", resource.Name, err)
			}
			if _, err := server.Enable(config, group.kind, found); err != nil {
				return err
			}
		}
	}
	if err := config.Save(); err != nil {
		return err
	}
	if len(manifest.Env) == 0 {
		return nil
	}
	values := map[string]string{}
	for key, value := range manifest.Env {
		values[key] = fmt.Sprint(value)
	}
	return server.MergeEnv(serverDir, values)
}

func recordRecipe(fetched *Fetched, manifest *Manifest, target Target, variables map[string]string) error {
	local, err := server.LoadLocal(target.Dir)
	if err != nil {
		return err
	}
	local.Recipe = &server.AppliedRecipe{
		Source:    fetched.Source.Raw,
		Name:      manifest.Recipe.Name,
		Version:   fetched.Version,
		Commit:    fetched.Commit,
		AppliedAt: time.Now(),
		Variables: variables,
	}
	if len(manifest.Server.Args) > 0 {
		local.Args = manifest.Server.Args
	}
	for _, service := range manifest.Services {
		local.PutService(service)
	}
	return local.Save()
}

func runHooks(target Target, recipeDir string, hooks []Hook) error {
	for _, hook := range hooks {
		directory := target.Dir
		if hook.Dir != "" {
			joined, err := fsutil.SafeJoin(target.Dir, hook.Dir)
			if err != nil {
				return err
			}
			directory = joined
		}
		ui.Step("🪝", "%s", strings.Join(hook.Run, " "))
		command := exec.Command(hook.Run[0], hook.Run[1:]...)
		command.Dir = directory
		command.Env = append(os.Environ(), "NWM_SERVER="+target.Name, "NWM_SERVER_DIR="+target.Dir, "NWM_RECIPE_DIR="+recipeDir)
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			return fmt.Errorf("hook %q failed: %w", strings.Join(hook.Run, " "), err)
		}
	}
	return nil
}

func ResolveServiceSecrets(service docker.Service, secretValues map[string]string) (docker.Service, error) {
	resolved := service
	resolved.Ready = make([]string, len(service.Ready))
	for index, argument := range service.Ready {
		expanded, err := ExpandSecrets(argument, secretValues)
		if err != nil {
			return service, fmt.Errorf("service %s: %w", service.Name, err)
		}
		resolved.Ready[index] = expanded
	}
	if len(service.Ready) == 0 {
		resolved.Ready = nil
	}
	if len(service.Env) == 0 {
		return resolved, nil
	}
	resolved.Env = make(map[string]string, len(service.Env))
	for key, value := range service.Env {
		expanded, err := ExpandSecrets(value, secretValues)
		if err != nil {
			return service, fmt.Errorf("service %s: %w", service.Name, err)
		}
		resolved.Env[key] = expanded
	}
	return resolved, nil
}

func (m *Manifest) Summary() []string {
	var lines []string
	add := func(label string, count int) {
		if count > 0 {
			lines = append(lines, fmt.Sprintf("%d %s", count, label))
		}
	}
	add("packages", len(m.Packages))
	add("asset packs", len(m.Assets))
	add("C modules", len(m.Modules))
	add("files", len(m.Files))
	add("services", len(m.Services))
	add("secrets", len(m.Secrets))
	add("env variables", len(m.Env))
	add("hooks", len(m.Hooks))
	return lines
}
