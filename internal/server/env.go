package server

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const EnvFileName = ".env"

func LoadEnv(directory string) (map[string]string, []string, error) {
	values := map[string]string{}
	var order []string
	file, err := os.Open(filepath.Join(directory, EnvFileName))
	if errors.Is(err, os.ErrNotExist) {
		return values, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(strings.TrimPrefix(line, "export "), "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if unquoted, err := strconv.Unquote(value); err == nil && strings.HasPrefix(value, `"`) {
			value = unquoted
		} else if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
			value = value[1 : len(value)-1]
		}
		if _, exists := values[key]; !exists {
			order = append(order, key)
		}
		values[key] = value
	}
	return values, order, scanner.Err()
}

func MergeEnv(directory string, updates map[string]string) error {
	values, order, err := LoadEnv(directory)
	if err != nil {
		return err
	}
	var added []string
	for key, value := range updates {
		if _, exists := values[key]; !exists {
			added = append(added, key)
		}
		values[key] = value
	}
	slices.Sort(added)
	order = append(order, added...)
	var builder strings.Builder
	for _, key := range order {
		builder.WriteString(key + "=" + quoteEnv(values[key]) + "\n")
	}
	return os.WriteFile(filepath.Join(directory, EnvFileName), []byte(builder.String()), 0o600)
}

func EnvList(directory string) ([]string, error) {
	values, order, err := LoadEnv(directory)
	if err != nil {
		return nil, err
	}
	list := make([]string, 0, len(order))
	for _, key := range order {
		list = append(list, key+"="+values[key])
	}
	return list, nil
}

func quoteEnv(value string) string {
	if value == "" || strings.ContainsAny(value, " \t#\"'$\\") {
		return strconv.Quote(value)
	}
	return value
}
