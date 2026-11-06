package recipe

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DasDarki/nwm/internal/fsutil"
	"github.com/pelletier/go-toml/v2"
)

const nebraManifest = "nebra.toml"

type BuildPlan struct {
	Root    string
	Output  string
	Build   []string
	Watch   []string
	Tool    string
	Sources []string
	Exclude []string
	Reload  string
}

func (p *BuildPlan) NeedsBuild() bool {
	return len(p.Build) > 0
}

func PlanBuild(resource Resource, root string) (*BuildPlan, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	plan := &BuildPlan{Root: root, Output: root, Reload: resource.Reload}
	if plan.Reload == "" {
		plan.Reload = ReloadHot
	}
	if err := detectNebra(plan); err != nil {
		return nil, err
	}
	if resource.Build != "" {
		if plan.Build, err = splitCommand(resource.Build); err != nil {
			return nil, err
		}
		plan.Tool = ""
	}
	if resource.Watch != "" {
		if plan.Watch, err = splitCommand(resource.Watch); err != nil {
			return nil, err
		}
	}
	if resource.Output != "" {
		output, err := fsutil.SafeJoin(root, resource.Output)
		if err != nil {
			return nil, fmt.Errorf("package %s: %w", resource.Name, err)
		}
		plan.Output = output
		plan.Exclude = appendTopLevel(plan.Exclude, root, output)
	}
	if plan.NeedsBuild() && len(plan.Sources) == 0 && len(plan.Watch) == 0 {
		plan.Sources = []string{root}
	}
	return plan, nil
}

func detectNebra(plan *BuildPlan) error {
	content, err := os.ReadFile(filepath.Join(plan.Root, nebraManifest))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var settings map[string]any
	if err := toml.Unmarshal(content, &settings); err != nil {
		return fmt.Errorf("parsing %s: %w", filepath.Join(plan.Root, nebraManifest), err)
	}
	source := stringSetting(settings, "source", "src")
	output := stringSetting(settings, "output", "out")
	sourcePath, err := fsutil.SafeJoin(plan.Root, source)
	if err != nil {
		return err
	}
	outputPath, err := fsutil.SafeJoin(plan.Root, output)
	if err != nil {
		return err
	}
	plan.Tool = "nebra"
	plan.Build = []string{"nebra", "build"}
	plan.Watch = []string{"nebra", "watch"}
	plan.Output = outputPath
	plan.Sources = []string{sourcePath}
	plan.Exclude = appendTopLevel(plan.Exclude, plan.Root, sourcePath)
	plan.Exclude = appendTopLevel(plan.Exclude, plan.Root, outputPath)
	plan.Exclude = append(plan.Exclude, nebraManifest, "nebra_modules", "nebra.lock")
	return nil
}

func stringSetting(settings map[string]any, key string, fallback string) string {
	if value, ok := settings[key].(string); ok && value != "" {
		return value
	}
	return fallback
}

func appendTopLevel(exclude []string, root string, path string) []string {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." {
		return exclude
	}
	first := strings.Split(filepath.ToSlash(relative), "/")[0]
	return append(exclude, first)
}

func splitCommand(command string) ([]string, error) {
	parsed, err := tokenize(command)
	if err != nil {
		return nil, fmt.Errorf("command %q: %w", command, err)
	}
	words := make([]string, len(parsed.tokens))
	for index, word := range parsed.tokens {
		words[index] = word.Text
	}
	if len(words) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	return words, nil
}
