package recipe

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DasDarki/nwm/internal/docker"
	"github.com/DasDarki/nwm/internal/fsutil"
	"github.com/pelletier/go-toml/v2"
)

const FileName = "recipe.toml"

var (
	recipeNamePattern   = regexp.MustCompile(`^(@[a-z0-9][a-z0-9-]{0,38}/)?[a-z0-9][a-z0-9_-]{0,63}$`)
	resourceNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	variableNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

type Meta struct {
	Name        string   `toml:"name"`
	Version     string   `toml:"version,omitempty"`
	Description string   `toml:"description,omitempty"`
	Authors     []string `toml:"authors,omitempty"`
	Keywords    []string `toml:"keywords,omitempty"`
}

type ServerSection struct {
	Branch     string   `toml:"branch,omitempty"`
	NanosSharp bool     `toml:"nanossharp,omitempty"`
	Args       []string `toml:"args,omitempty"`
}

type Resource struct {
	Name   string `toml:"name"`
	Git    string `toml:"git,omitempty"`
	Ref    string `toml:"ref,omitempty"`
	Subdir string `toml:"subdir,omitempty"`
	Path   string `toml:"path,omitempty"`
	Enable bool   `toml:"enable,omitempty"`
}

func (r Resource) FromVault() bool {
	return r.Git == "" && r.Path == ""
}

type Module struct {
	Name    string `toml:"name"`
	Linux   string `toml:"linux,omitempty"`
	Windows string `toml:"windows,omitempty"`
}

type File struct {
	Source   string `toml:"source"`
	Target   string `toml:"target"`
	Template bool   `toml:"template,omitempty"`
}

type Hook struct {
	Run []string `toml:"run"`
	Dir string   `toml:"dir,omitempty"`
}

type Manifest struct {
	Recipe    Meta             `toml:"recipe"`
	Server    ServerSection    `toml:"server"`
	Variables map[string]any   `toml:"variables,omitempty"`
	Config    map[string]any   `toml:"config,omitempty"`
	Env       map[string]any   `toml:"env,omitempty"`
	Packages  []Resource       `toml:"package,omitempty"`
	Assets    []Resource       `toml:"assets,omitempty"`
	Modules   []Module         `toml:"module,omitempty"`
	Files     []File           `toml:"file,omitempty"`
	Services  []docker.Service `toml:"service,omitempty"`
	Hooks     []Hook           `toml:"hook,omitempty"`
}

func Load(directory string) (*Manifest, error) {
	path := filepath.Join(directory, FileName)
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("no %s in %s", FileName, directory)
	}
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	decoder := toml.NewDecoder(strings.NewReader(string(content))).DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		var strict *toml.StrictMissingError
		if errors.As(err, &strict) {
			return nil, fmt.Errorf("%s: %s", path, strict.String())
		}
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &manifest, nil
}

func (m *Manifest) Validate(directory string) error {
	var problems []string
	add := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf(format, args...))
	}
	if !recipeNamePattern.MatchString(m.Recipe.Name) {
		add("recipe.name %q must be lowercase, optionally scoped like @you/name", m.Recipe.Name)
	}
	if branch := m.Server.Branch; branch != "" && branch != "public" && branch != "bleeding-edge" {
		add("server.branch must be public or bleeding-edge, got %q", branch)
	}
	for name := range m.Variables {
		if !variableNamePattern.MatchString(name) {
			add("variable name %q is invalid", name)
		}
	}
	validateResources := func(kind string, resources []Resource) {
		seen := map[string]bool{}
		for _, resource := range resources {
			if !resourceNamePattern.MatchString(resource.Name) {
				add("%s name %q is invalid", kind, resource.Name)
			}
			if seen[resource.Name] {
				add("%s %q is listed twice", kind, resource.Name)
			}
			seen[resource.Name] = true
			if resource.Git != "" && resource.Path != "" {
				add("%s %q cannot have both git and path", kind, resource.Name)
			}
			if resource.Path != "" {
				checkSource(directory, resource.Path, add)
			}
		}
	}
	validateResources("package", m.Packages)
	validateResources("assets", m.Assets)
	for _, module := range m.Modules {
		if module.Name == "" || (module.Linux == "" && module.Windows == "") {
			add("module %q needs a name and a linux and/or windows source", module.Name)
		}
		for _, source := range []string{module.Linux, module.Windows} {
			if source != "" && !isURL(source) {
				checkSource(directory, source, add)
			}
		}
	}
	for _, file := range m.Files {
		if file.Source == "" || file.Target == "" {
			add("file entries need source and target")
			continue
		}
		checkSource(directory, file.Source, add)
		if _, err := fsutil.SafeJoin("/server", file.Target); err != nil {
			add("file target %q must stay inside the server directory", file.Target)
		}
	}
	serviceNames := map[string]bool{}
	for _, service := range m.Services {
		if err := service.Validate(); err != nil {
			add("%v", err)
		}
		if serviceNames[service.Name] {
			add("service %q is listed twice", service.Name)
		}
		serviceNames[service.Name] = true
	}
	for _, hook := range m.Hooks {
		if len(hook.Run) == 0 {
			add("hooks need a run command")
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("invalid recipe:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}

func checkSource(directory string, source string, add func(string, ...any)) {
	path, err := fsutil.SafeJoin(directory, source)
	if err != nil {
		add("%q must stay inside the recipe", source)
		return
	}
	if !fsutil.Exists(path) {
		add("%q does not exist in the recipe", source)
	}
}

func isURL(source string) bool {
	return strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://")
}

func Scaffold(name string) string {
	return fmt.Sprintf(`[recipe]
name = %q
version = "0.1.0"
description = "Local development server"
authors = []
keywords = []

[server]
branch = "public"
nanossharp = false
args = []

[variables]
port = 7777

[config.discover]
name = "${server.name}"
port = "${port}"

[config.game]
game_mode = "sandbox"

[[package]]
name = "sandbox"
enable = true

[[service]]
name = "redis"
image = "redis:7-alpine"
ports = ["6379"]

[env]
REDIS_URL = "redis://127.0.0.1:6379"
`, name)
}
