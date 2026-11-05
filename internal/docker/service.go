package docker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var serviceNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

type Service struct {
	Name    string            `toml:"name" json:"name"`
	Image   string            `toml:"image" json:"image"`
	Ports   []string          `toml:"ports,omitempty" json:"ports,omitempty"`
	Env     map[string]string `toml:"env,omitempty" json:"env,omitempty"`
	Volumes []string          `toml:"volumes,omitempty" json:"volumes,omitempty"`
	Command []string          `toml:"command,omitempty" json:"command,omitempty"`
	Ready   []string          `toml:"ready,omitempty" json:"ready,omitempty"`
	Wait    *bool             `toml:"wait,omitempty" json:"wait,omitempty"`
}

func (s Service) Validate() error {
	if !serviceNamePattern.MatchString(s.Name) {
		return fmt.Errorf("invalid service name %q, use lowercase letters, digits, '-' and '_'", s.Name)
	}
	if strings.TrimSpace(s.Image) == "" {
		return fmt.Errorf("service %q needs an image", s.Name)
	}
	for _, port := range s.Ports {
		if _, err := HostPort(port); err != nil {
			return fmt.Errorf("service %q: %w", s.Name, err)
		}
	}
	for _, volume := range s.Volumes {
		if !strings.Contains(volume, ":") {
			return fmt.Errorf("service %q: volume %q must look like source:/container/path", s.Name, volume)
		}
	}
	return nil
}

func (s Service) ShouldWait() bool {
	return (s.Wait == nil || *s.Wait) && (len(s.Ports) > 0 || len(s.Ready) > 0)
}

func (s Service) hash() string {
	content, _ := json.Marshal(s)
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:8])
}

func ContainerName(server string, service string) string {
	return "nwm-" + strings.ToLower(server) + "-" + service
}

func VolumeName(server string, volume string) string {
	return "nwm-" + strings.ToLower(server) + "-" + volume
}

func HostPort(spec string) (string, error) {
	spec, _, _ = strings.Cut(spec, "/")
	parts := strings.Split(spec, ":")
	var host string
	switch len(parts) {
	case 1, 2:
		host = parts[0]
	case 3:
		host = parts[1]
	default:
		return "", fmt.Errorf("invalid port mapping %q", spec)
	}
	if host == "" {
		return "", fmt.Errorf("invalid port mapping %q", spec)
	}
	return host, nil
}

func publishArgument(spec string) string {
	protocol := ""
	if base, suffix, found := strings.Cut(spec, "/"); found {
		spec, protocol = base, "/"+suffix
	}
	parts := strings.Split(spec, ":")
	switch len(parts) {
	case 1:
		return "127.0.0.1:" + parts[0] + ":" + parts[0] + protocol
	case 2:
		return "127.0.0.1:" + spec + protocol
	default:
		return spec + protocol
	}
}

func isHostPath(source string) bool {
	return strings.ContainsAny(source, `/\`) || strings.HasPrefix(source, ".") || filepath.IsAbs(source) || filepath.VolumeName(source) != ""
}

func volumeArgument(server string, directory string, spec string) (string, string) {
	separator := strings.Index(spec, ":")
	if len(spec) > 2 && spec[1] == ':' && (spec[2] == '\\' || spec[2] == '/') {
		separator = 2 + strings.Index(spec[2:], ":")
	}
	source, rest := spec[:separator], spec[separator:]
	if isHostPath(source) {
		if !filepath.IsAbs(source) {
			source = filepath.Join(directory, source)
		}
		return source + rest, ""
	}
	named := VolumeName(server, source)
	return named + rest, named
}

func NamedVolumes(server string, service Service) []string {
	var names []string
	for _, volume := range service.Volumes {
		if _, named := volumeArgument(server, "", volume); named != "" {
			names = append(names, named)
		}
	}
	return names
}

func runArguments(server string, directory string, service Service) []string {
	arguments := []string{
		"run", "--detach",
		"--name", ContainerName(server, service.Name),
		"--label", "nwm.server=" + strings.ToLower(server),
		"--label", "nwm.service=" + service.Name,
		"--label", "nwm.hash=" + service.hash(),
		"--restart", "unless-stopped",
	}
	for _, port := range service.Ports {
		arguments = append(arguments, "--publish", publishArgument(port))
	}
	keys := make([]string, 0, len(service.Env))
	for key := range service.Env {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		arguments = append(arguments, "--env", key+"="+service.Env[key])
	}
	for _, volume := range service.Volumes {
		mapped, _ := volumeArgument(server, directory, volume)
		arguments = append(arguments, "--volume", mapped)
	}
	arguments = append(arguments, service.Image)
	return append(arguments, service.Command...)
}
