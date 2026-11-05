package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/DasDarki/nwm/internal/home"
	"github.com/pelletier/go-toml/v2"
)

type UnsafeLibsApproval struct {
	ApprovedAt time.Time `toml:"approved_at"`
	Modules    []string  `toml:"modules"`
}

func (a *UnsafeLibsApproval) Covers(modules []string) bool {
	if a == nil {
		return false
	}
	for _, module := range modules {
		if !slices.Contains(a.Modules, module) {
			return false
		}
	}
	return true
}

type Server struct {
	Name       string              `toml:"name"`
	Path       string              `toml:"path"`
	Branch     string              `toml:"branch"`
	CreatedAt  time.Time           `toml:"created_at"`
	UpdatedAt  time.Time           `toml:"updated_at"`
	UnsafeLibs *UnsafeLibsApproval `toml:"unsafe_libs,omitempty"`
}

type State struct {
	AllowUnsafeLibs bool     `toml:"allow_unsafe_libs,omitempty"`
	Servers         []Server `toml:"servers"`
}

func Load() (*State, error) {
	content, err := os.ReadFile(home.StateFile())
	if errors.Is(err, os.ErrNotExist) {
		return &State{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading state: %w", err)
	}
	var loaded State
	if err := toml.Unmarshal(content, &loaded); err != nil {
		return nil, fmt.Errorf("parsing state %s: %w", home.StateFile(), err)
	}
	return &loaded, nil
}

func (s *State) Save() error {
	if _, err := home.Ensure(home.Dir()); err != nil {
		return err
	}
	content, err := toml.Marshal(s)
	if err != nil {
		return err
	}
	temporary := home.StateFile() + ".tmp"
	if err := os.WriteFile(temporary, content, 0o644); err != nil {
		return err
	}
	return os.Rename(temporary, home.StateFile())
}

func (s *State) Find(name string) (*Server, bool) {
	for index := range s.Servers {
		if strings.EqualFold(s.Servers[index].Name, name) {
			return &s.Servers[index], true
		}
	}
	return nil, false
}

func (s *State) FindByPath(path string) (*Server, bool) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, false
	}
	for index := range s.Servers {
		relative, err := filepath.Rel(s.Servers[index].Path, absolute)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return &s.Servers[index], true
		}
	}
	return nil, false
}

func (s *State) Add(server Server) error {
	if _, exists := s.Find(server.Name); exists {
		return fmt.Errorf("a server named %q already exists", server.Name)
	}
	s.Servers = append(s.Servers, server)
	slices.SortFunc(s.Servers, func(a, b Server) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return nil
}

func (s *State) Remove(name string) bool {
	before := len(s.Servers)
	s.Servers = slices.DeleteFunc(s.Servers, func(server Server) bool {
		return strings.EqualFold(server.Name, name)
	})
	return len(s.Servers) != before
}
