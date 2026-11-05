package recipe

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type SourceKind int

const (
	SourceRegistry SourceKind = iota
	SourceGit
	SourceLocal
	SourceURL
	SourceGist
)

type Source struct {
	Kind    SourceKind
	Raw     string
	Name    string
	URL     string
	Subdir  string
	Version string
	Path    string
	File    string
	Gist    string
}

func (s Source) SingleFile() bool {
	return s.Kind == SourceURL || s.Kind == SourceGist
}

func ParseSource(spec string) (Source, error) {
	spec = strings.TrimSpace(spec)
	source := Source{Raw: spec}
	switch {
	case spec == "":
		return source, fmt.Errorf("empty recipe source")
	case strings.HasPrefix(spec, "file:"):
		return localSource(source, strings.TrimPrefix(spec, "file:"))
	case looksLikePath(spec):
		return localSource(source, spec)
	case strings.HasPrefix(spec, "gist:"):
		return gistSource(source, strings.TrimPrefix(spec, "gist:"))
	case (strings.HasPrefix(spec, "https://") || strings.HasPrefix(spec, "http://")) && IsRecipeFileName(urlFileName(spec)):
		source.Kind = SourceURL
		source.URL = spec
		source.File = urlFileName(spec)
		return source, nil
	case strings.HasPrefix(spec, "github:"):
		rest, version := splitVersion(strings.TrimPrefix(spec, "github:"))
		parts := strings.Split(strings.Trim(rest, "/"), "/")
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			return source, fmt.Errorf("expected github:owner/repo[/subdir][@version], got %q", spec)
		}
		source.Kind = SourceGit
		source.URL = "https://github.com/" + parts[0] + "/" + parts[1] + ".git"
		source.Subdir = strings.Join(parts[2:], "/")
		source.Version = version
		return source, nil
	case strings.HasPrefix(spec, "git+"), strings.HasPrefix(spec, "https://"), strings.HasPrefix(spec, "http://"), strings.HasPrefix(spec, "ssh://"), strings.HasPrefix(spec, "git@"):
		url, version, _ := strings.Cut(strings.TrimPrefix(spec, "git+"), "#")
		source.Kind = SourceGit
		source.URL, source.Subdir = splitSubdir(url)
		source.Version = version
		return source, nil
	default:
		name, version := splitVersion(spec)
		source.Kind = SourceRegistry
		source.Name = strings.ToLower(name)
		source.Version = version
		return source, nil
	}
}

func (s Source) String() string {
	return s.Raw
}

func (s Source) cacheKey() string {
	sum := sha256.Sum256([]byte(s.URL))
	return hex.EncodeToString(sum[:8])
}

func localSource(source Source, path string) (Source, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return source, err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return source, fmt.Errorf("recipe %s not found", absolute)
	}
	source.Kind = SourceLocal
	source.Path = absolute
	if !info.IsDir() {
		source.Path = filepath.Dir(absolute)
		source.File = filepath.Base(absolute)
	}
	return source, nil
}

func gistSource(source Source, spec string) (Source, error) {
	rest, revision := splitVersion(spec)
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	switch {
	case len(parts) == 1:
		source.Gist = parts[0]
	case len(parts) == 2 && isGistID(parts[0]):
		source.Gist, source.File = parts[0], parts[1]
	case len(parts) == 2:
		source.Gist = parts[1]
	case len(parts) == 3:
		source.Gist, source.File = parts[1], parts[2]
	}
	if !isGistID(source.Gist) {
		return source, fmt.Errorf("expected gist:[user/]id[/file][@revision], got %q", "gist:"+spec)
	}
	source.Kind = SourceGist
	source.Version = revision
	return source, nil
}

func isGistID(text string) bool {
	if len(text) < 20 {
		return false
	}
	for _, character := range strings.ToLower(text) {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return true
}

func urlFileName(url string) string {
	trimmed, _, _ := strings.Cut(url, "?")
	trimmed, _, _ = strings.Cut(trimmed, "#")
	return path.Base(strings.TrimSuffix(trimmed, "/"))
}

func looksLikePath(spec string) bool {
	if strings.HasPrefix(spec, ".") || strings.HasPrefix(spec, "/") || strings.HasPrefix(spec, "~") || filepath.IsAbs(spec) || filepath.VolumeName(spec) != "" {
		return true
	}
	_, err := os.Stat(spec)
	return err == nil
}

func splitSubdir(url string) (string, string) {
	offset := 0
	if index := strings.Index(url, "://"); index >= 0 {
		offset = index + 3
	}
	index := strings.Index(url[offset:], "//")
	if index < 0 {
		return url, ""
	}
	return url[:offset+index], strings.Trim(url[offset+index+2:], "/")
}

func splitVersion(spec string) (string, string) {
	index := strings.LastIndex(spec, "@")
	if index <= 0 {
		return spec, ""
	}
	return spec[:index], spec[index+1:]
}
