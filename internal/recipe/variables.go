package recipe

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/DasDarki/nwm/internal/docker"
)

var placeholderPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_.]*)\}`)

const (
	secretPrefix = "secret."
	envPrefix    = "env."
)

type Expander struct {
	values      map[string]any
	missing     map[string]bool
	keepSecrets bool
	env         map[string]any
	resolving   []string
	cycle       string
}

func NewExpander(defaults map[string]any, overrides map[string]any, builtins map[string]string, secrets map[string]string) *Expander {
	values := map[string]any{}
	maps.Copy(values, defaults)
	maps.Copy(values, overrides)
	for key, value := range builtins {
		values[key] = value
	}
	for key, value := range secrets {
		values[secretPrefix+key] = value
	}
	return &Expander{values: values, missing: map[string]bool{}}
}

func ExpandSecrets(text string, secrets map[string]string) (string, error) {
	expander := NewExpander(nil, nil, nil, secrets)
	result := expander.String(text)
	return result, expander.Err()
}

func (e *Expander) String(text string) string {
	return placeholderPattern.ReplaceAllStringFunc(text, func(match string) string {
		name := placeholderPattern.FindStringSubmatch(match)[1]
		if strings.HasPrefix(name, envPrefix) {
			return e.expandEnv(strings.TrimPrefix(name, envPrefix), match)
		}
		if e.keepSecrets && strings.HasPrefix(name, secretPrefix) {
			if _, found := e.values[name]; !found {
				e.missing[name] = true
			}
			return match
		}
		value, found := e.values[name]
		if !found {
			e.missing[name] = true
			return match
		}
		return fmt.Sprint(value)
	})
}

func (e *Expander) expandEnv(key string, placeholder string) string {
	raw, found := e.env[key]
	if !found {
		e.missing[envPrefix+key] = true
		return placeholder
	}
	if slices.Contains(e.resolving, key) {
		e.cycle = strings.Join(append(slices.Clone(e.resolving), key), " -> ")
		return placeholder
	}
	e.resolving = append(e.resolving, key)
	defer func() { e.resolving = e.resolving[:len(e.resolving)-1] }()
	return e.String(fmt.Sprint(raw))
}

func (e *Expander) Value(value any) any {
	switch typed := value.(type) {
	case string:
		if match := placeholderPattern.FindStringSubmatch(typed); match != nil && match[0] == typed && !strings.HasPrefix(match[1], envPrefix) {
			if resolved, found := e.values[match[1]]; found {
				return resolved
			}
		}
		return e.String(typed)
	case []any:
		result := make([]any, len(typed))
		for index, item := range typed {
			result[index] = e.Value(item)
		}
		return result
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			result[key] = e.Value(item)
		}
		return result
	default:
		return value
	}
}

func (e *Expander) Strings(items []string) []string {
	result := make([]string, len(items))
	for index, item := range items {
		result[index] = e.String(item)
	}
	return result
}

func (e *Expander) Err() error {
	if e.cycle != "" {
		return fmt.Errorf("env values reference each other in a circle: %s", e.cycle)
	}
	if len(e.missing) == 0 {
		return nil
	}
	names := slices.Sorted(maps.Keys(e.missing))
	return fmt.Errorf("unknown variables: %s (define them with var, env or secret, or pass --var name=value)", strings.Join(names, ", "))
}

func (m *Manifest) Expand(expander *Expander) *Manifest {
	expander.env = m.Env
	expanded := *m
	expanded.Server.Args = expander.Strings(m.Server.Args)
	if m.Config != nil {
		expanded.Config = expander.Value(m.Config).(map[string]any)
	}
	expanded.Env = map[string]any{}
	for key, value := range m.Env {
		expanded.Env[key] = expander.String(fmt.Sprint(value))
	}
	expanded.Packages = expandResources(expander, m.Packages)
	expanded.Assets = expandResources(expander, m.Assets)
	expanded.Services = make([]docker.Service, len(m.Services))
	expander.keepSecrets = true
	defer func() { expander.keepSecrets = false }()
	for index, service := range m.Services {
		copied := service
		copied.Image = expander.String(service.Image)
		copied.Ports = expander.Strings(service.Ports)
		copied.Volumes = expander.Strings(service.Volumes)
		copied.Command = expander.Strings(service.Command)
		copied.Ready = expander.Strings(service.Ready)
		copied.Env = map[string]string{}
		for key, value := range service.Env {
			copied.Env[key] = expander.String(value)
		}
		expanded.Services[index] = copied
	}
	expanded.Hooks = make([]Hook, len(m.Hooks))
	for index, hook := range m.Hooks {
		expanded.Hooks[index] = Hook{Run: expander.Strings(hook.Run), Dir: expander.String(hook.Dir)}
	}
	return &expanded
}

func expandResources(expander *Expander, resources []Resource) []Resource {
	result := make([]Resource, len(resources))
	for index, resource := range resources {
		resource.Git = expander.String(resource.Git)
		resource.Ref = expander.String(resource.Ref)
		resource.Build = expander.String(resource.Build)
		resource.Watch = expander.String(resource.Watch)
		result[index] = resource
	}
	return result
}
