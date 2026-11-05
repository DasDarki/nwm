package secrets

import (
	"crypto/rand"
	"errors"
	"fmt"
	"maps"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/DasDarki/nwm/internal/home"
	"github.com/pelletier/go-toml/v2"
)

const (
	DefaultLength = 24
	alphabet      = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
)

type Store struct {
	server string
	values map[string]string
}

func Directory() string {
	return filepath.Join(home.Dir(), "secrets")
}

func path(server string) string {
	return filepath.Join(Directory(), strings.ToLower(server)+".toml")
}

func Open(server string) (*Store, error) {
	store := &Store{server: server, values: map[string]string{}}
	content, err := os.ReadFile(path(server))
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, err
	}
	if err := toml.Unmarshal(content, &store.values); err != nil {
		return nil, fmt.Errorf("parsing secrets of %s: %w", server, err)
	}
	return store, nil
}

func (s *Store) Get(name string) (string, bool) {
	value, found := s.values[name]
	return value, found
}

func (s *Store) Set(name string, value string) {
	s.values[name] = value
}

func (s *Store) Delete(name string) bool {
	_, found := s.values[name]
	delete(s.values, name)
	return found
}

func (s *Store) Names() []string {
	return slices.Sorted(maps.Keys(s.values))
}

func (s *Store) Values() map[string]string {
	return maps.Clone(s.values)
}

func (s *Store) Save() error {
	if err := os.MkdirAll(Directory(), 0o700); err != nil {
		return err
	}
	content, err := toml.Marshal(s.values)
	if err != nil {
		return err
	}
	temporary := path(s.server) + ".tmp"
	if err := os.WriteFile(temporary, content, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, path(s.server))
}

func Remove(server string) error {
	err := os.Remove(path(server))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func Random(length int) (string, error) {
	if length <= 0 {
		length = DefaultLength
	}
	limit := big.NewInt(int64(len(alphabet)))
	result := make([]byte, length)
	for index := range result {
		position, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		result[index] = alphabet[position.Int64()]
	}
	return string(result), nil
}
