package server

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const ConfigFileName = "Config.toml"

//go:embed default_config.toml
var defaultConfig []byte

type Config struct {
	path   string
	exists bool
	data   map[string]any
}

func LoadConfig(directory string) (*Config, error) {
	config := &Config{path: filepath.Join(directory, ConfigFileName), data: map[string]any{}}
	content, err := os.ReadFile(config.path)
	if errors.Is(err, os.ErrNotExist) {
		return config, nil
	}
	if err != nil {
		return nil, err
	}
	if err := toml.Unmarshal(content, &config.data); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", config.path, err)
	}
	config.exists = true
	return config, nil
}

func (c *Config) Path() string {
	return c.path
}

func (c *Config) Exists() bool {
	return c.exists
}

func (c *Config) Get(key string) (any, bool) {
	var current any = c.data
	for _, part := range strings.Split(key, ".") {
		table, isTable := current.(map[string]any)
		if !isTable {
			return nil, false
		}
		value, found := table[part]
		if !found {
			return nil, false
		}
		current = value
	}
	return current, true
}

func (c *Config) String(key string) string {
	value, found := c.Get(key)
	if !found {
		return ""
	}
	return fmt.Sprint(value)
}

func (c *Config) Set(key string, value any) error {
	parts := strings.Split(key, ".")
	table := c.data
	for _, part := range parts[:len(parts)-1] {
		next, found := table[part]
		if !found {
			created := map[string]any{}
			table[part] = created
			table = created
			continue
		}
		nested, isTable := next.(map[string]any)
		if !isTable {
			return fmt.Errorf("%s is not a table", part)
		}
		table = nested
	}
	table[parts[len(parts)-1]] = value
	return nil
}

func (c *Config) AddToList(key string, item string) (bool, error) {
	list, err := c.list(key)
	if err != nil {
		return false, err
	}
	if slices.Contains(list, item) {
		return false, nil
	}
	return true, c.Set(key, toAnyList(append(list, item)))
}

func (c *Config) RemoveFromList(key string, item string) (bool, error) {
	list, err := c.list(key)
	if err != nil {
		return false, err
	}
	filtered := slices.DeleteFunc(slices.Clone(list), func(entry string) bool { return entry == item })
	if len(filtered) == len(list) {
		return false, nil
	}
	return true, c.Set(key, toAnyList(filtered))
}

func (c *Config) Save() error {
	var defaults map[string]any
	if err := toml.Unmarshal(defaultConfig, &defaults); err != nil {
		return err
	}
	fillMissing(c.data, defaults)
	content, err := toml.Marshal(c.data)
	if err != nil {
		return err
	}
	if err := os.WriteFile(c.path, content, 0o644); err != nil {
		return err
	}
	c.exists = true
	return nil
}

func (c *Config) Render() (string, error) {
	content, err := toml.Marshal(c.data)
	return string(content), err
}

func (c *Config) list(key string) ([]string, error) {
	value, found := c.Get(key)
	if !found {
		return nil, nil
	}
	entries, isList := value.([]any)
	if !isList {
		return nil, fmt.Errorf("%s is not a list", key)
	}
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		result = append(result, fmt.Sprint(entry))
	}
	return result, nil
}

func (c *Config) Merge(values map[string]any) {
	mergeTables(c.data, values)
}

func mergeTables(target map[string]any, values map[string]any) {
	for key, value := range values {
		incoming, incomingIsTable := value.(map[string]any)
		existing, existingIsTable := target[key].(map[string]any)
		if incomingIsTable && existingIsTable {
			mergeTables(existing, incoming)
			continue
		}
		target[key] = value
	}
}

func DefaultConfigKeys() map[string]any {
	var defaults map[string]any
	if err := toml.Unmarshal(defaultConfig, &defaults); err != nil {
		return nil
	}
	flat := map[string]any{}
	var walk func(prefix string, table map[string]any)
	walk = func(prefix string, table map[string]any) {
		for key, value := range table {
			path := prefix + key
			if nested, isTable := value.(map[string]any); isTable {
				if len(nested) == 0 {
					flat[path+"."] = nil
				}
				walk(path+".", nested)
				continue
			}
			flat[path] = value
		}
	}
	walk("", defaults)
	return flat
}

func fillMissing(target map[string]any, defaults map[string]any) {
	for key, fallback := range defaults {
		existing, found := target[key]
		if !found {
			target[key] = fallback
			continue
		}
		nestedTarget, targetIsTable := existing.(map[string]any)
		nestedDefaults, defaultsAreTable := fallback.(map[string]any)
		if targetIsTable && defaultsAreTable {
			fillMissing(nestedTarget, nestedDefaults)
		}
	}
}

func toAnyList(items []string) []any {
	result := make([]any, len(items))
	for index, item := range items {
		result[index] = item
	}
	return result
}

func ParseValue(raw string) any {
	var parsed map[string]any
	if err := toml.Unmarshal([]byte("value = "+raw), &parsed); err == nil {
		return parsed["value"]
	}
	return raw
}

func FormatValue(value any) string {
	content, err := toml.Marshal(map[string]any{"value": value})
	if err != nil {
		return fmt.Sprint(value)
	}
	return strings.TrimSpace(strings.TrimPrefix(string(content), "value = "))
}
