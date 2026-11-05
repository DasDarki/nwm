package server

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/DasDarki/nwm/internal/docker"
	"github.com/pelletier/go-toml/v2"
)

const LocalFileName = "nwm.toml"

type AppliedRecipe struct {
	Source    string            `toml:"source"`
	Name      string            `toml:"name"`
	Version   string            `toml:"version,omitempty"`
	Commit    string            `toml:"commit,omitempty"`
	AppliedAt time.Time         `toml:"applied_at"`
	Variables map[string]string `toml:"variables,omitempty"`
}

type Local struct {
	Recipe   *AppliedRecipe   `toml:"recipe,omitempty"`
	Args     []string         `toml:"args,omitempty"`
	Services []docker.Service `toml:"service,omitempty"`
	path     string
}

func LoadLocal(directory string) (*Local, error) {
	local := &Local{path: filepath.Join(directory, LocalFileName)}
	content, err := os.ReadFile(local.path)
	if errors.Is(err, os.ErrNotExist) {
		return local, nil
	}
	if err != nil {
		return nil, err
	}
	if err := toml.Unmarshal(content, local); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", local.path, err)
	}
	return local, nil
}

func (l *Local) Save() error {
	content, err := toml.Marshal(l)
	if err != nil {
		return err
	}
	return os.WriteFile(l.path, content, 0o644)
}

func (l *Local) Service(name string) (*docker.Service, bool) {
	for index := range l.Services {
		if l.Services[index].Name == name {
			return &l.Services[index], true
		}
	}
	return nil, false
}

func (l *Local) PutService(service docker.Service) {
	if existing, found := l.Service(service.Name); found {
		*existing = service
		return
	}
	l.Services = append(l.Services, service)
}

func (l *Local) RemoveService(name string) bool {
	before := len(l.Services)
	l.Services = slices.DeleteFunc(l.Services, func(service docker.Service) bool { return service.Name == name })
	return len(l.Services) != before
}

func (l *Local) SelectServices(names []string) ([]docker.Service, error) {
	if len(names) == 0 {
		return l.Services, nil
	}
	selected := make([]docker.Service, 0, len(names))
	for _, name := range names {
		service, found := l.Service(name)
		if !found {
			return nil, fmt.Errorf("unknown service %q", name)
		}
		selected = append(selected, *service)
	}
	return selected, nil
}
