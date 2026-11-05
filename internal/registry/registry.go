package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/DasDarki/nwm/internal/buildinfo"
	"github.com/DasDarki/nwm/internal/home"
)

const (
	Repository     = "DasDarki/nwm-registry"
	defaultIndex   = "https://raw.githubusercontent.com/" + Repository + "/main/index.json"
	cacheLifetime  = 24 * time.Hour
	PublishPrefix  = "[publish] "
	UnpublishTitle = "[unpublish] "
)

var NamePattern = regexp.MustCompile(`^@([a-z0-9][a-z0-9-]{0,38})/([a-z0-9][a-z0-9_-]{0,63})$`)

type Entry struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Repository  string    `json:"repository"`
	Subdir      string    `json:"subdir,omitempty"`
	Keywords    []string  `json:"keywords,omitempty"`
	Owners      []string  `json:"owners"`
	PublishedAt time.Time `json:"published_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Index struct {
	Version int     `json:"version"`
	Recipes []Entry `json:"recipes"`
}

func indexURL() string {
	if custom := os.Getenv("NWM_REGISTRY_URL"); custom != "" {
		return custom
	}
	return defaultIndex
}

func cachePath() string {
	return filepath.Join(home.CacheDir(), "registry", "index.json")
}

func Load(ctx context.Context, refresh bool) (*Index, error) {
	if !refresh {
		if info, err := os.Stat(cachePath()); err == nil && time.Since(info.ModTime()) < cacheLifetime {
			if index, err := readIndex(cachePath()); err == nil {
				return index, nil
			}
		}
	}
	index, err := fetch(ctx)
	if err != nil {
		if cached, cacheErr := readIndex(cachePath()); cacheErr == nil {
			return cached, nil
		}
		return nil, err
	}
	return index, nil
}

func fetch(ctx context.Context) (*Index, error) {
	source := indexURL()
	var content []byte
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("User-Agent", "nwm/"+buildinfo.Version)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			return nil, fmt.Errorf("fetching recipe registry: %w", err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fetching recipe registry: %s", response.Status)
		}
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		content = body
	} else {
		local, err := os.ReadFile(source)
		if err != nil {
			return nil, err
		}
		content = local
	}
	var index Index
	if err := json.Unmarshal(content, &index); err != nil {
		return nil, fmt.Errorf("parsing recipe registry: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cachePath()), 0o755); err == nil {
		os.WriteFile(cachePath(), content, 0o644)
	}
	return &index, nil
}

func readIndex(path string) (*Index, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var index Index
	return &index, json.Unmarshal(content, &index)
}

func (i *Index) Lookup(name string) (*Entry, error) {
	lowered := strings.ToLower(name)
	var matches []*Entry
	for index := range i.Recipes {
		entry := &i.Recipes[index]
		if entry.Name == lowered {
			return entry, nil
		}
		if !strings.HasPrefix(lowered, "@") && strings.HasSuffix(entry.Name, "/"+lowered) {
			matches = append(matches, entry)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("recipe %q is not in the registry, try %s", name, "nwm recipe search")
	case 1:
		return matches[0], nil
	default:
		names := make([]string, len(matches))
		for index, match := range matches {
			names[index] = match.Name
		}
		return nil, fmt.Errorf("%q is ambiguous: %s", name, strings.Join(names, ", "))
	}
}

func (i *Index) Search(query string) []Entry {
	terms := strings.Fields(strings.ToLower(query))
	var results []Entry
	for _, entry := range i.Recipes {
		haystack := strings.ToLower(entry.Name + " " + entry.Description + " " + strings.Join(entry.Keywords, " "))
		if !slices.ContainsFunc(terms, func(term string) bool { return !strings.Contains(haystack, term) }) {
			results = append(results, entry)
		}
	}
	slices.SortFunc(results, func(a, b Entry) int { return strings.Compare(a.Name, b.Name) })
	return results
}

type PublishRequest struct {
	Name        string
	Repository  string
	Subdir      string
	Description string
	Keywords    []string
}

func (r PublishRequest) Title() string {
	return PublishPrefix + r.Name
}

func (r PublishRequest) Body() string {
	return issueBody([][2]string{
		{"Recipe name", r.Name},
		{"Repository", r.Repository},
		{"Subdirectory", r.Subdir},
		{"Description", r.Description},
		{"Keywords", strings.Join(r.Keywords, ", ")},
	})
}

func (r PublishRequest) IssueURL() string {
	query := url.Values{}
	query.Set("template", "publish.yml")
	query.Set("title", r.Title())
	query.Set("name", r.Name)
	query.Set("repository", r.Repository)
	query.Set("subdir", r.Subdir)
	query.Set("description", r.Description)
	query.Set("keywords", strings.Join(r.Keywords, ", "))
	return "https://github.com/" + Repository + "/issues/new?" + query.Encode()
}

func UnpublishBody(name string) string {
	return issueBody([][2]string{{"Recipe name", name}})
}

func UnpublishURL(name string) string {
	query := url.Values{}
	query.Set("template", "unpublish.yml")
	query.Set("title", UnpublishTitle+name)
	query.Set("name", name)
	return "https://github.com/" + Repository + "/issues/new?" + query.Encode()
}

func issueBody(fields [][2]string) string {
	var builder strings.Builder
	for _, field := range fields {
		value := field[1]
		if value == "" {
			value = "_No response_"
		}
		builder.WriteString("### " + field[0] + "\n\n" + value + "\n\n")
	}
	return builder.String()
}
