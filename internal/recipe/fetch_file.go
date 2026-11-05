package recipe

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/DasDarki/nwm/internal/buildinfo"
	"github.com/DasDarki/nwm/internal/home"
	"github.com/DasDarki/nwm/internal/ui"
)

const maxRecipeFileSize = 1 << 20

var gistAPI = "https://api.github.com/gists/"

type gistResponse struct {
	Files map[string]struct {
		Filename  string `json:"filename"`
		RawURL    string `json:"raw_url"`
		Truncated bool   `json:"truncated"`
		Content   string `json:"content"`
	} `json:"files"`
	History []struct {
		Version string `json:"version"`
	} `json:"history"`
}

func fetchSingleFile(ctx context.Context, fetched *Fetched, report ui.Reporter) error {
	source := fetched.Source
	var content []byte
	var err error
	switch source.Kind {
	case SourceURL:
		report.Status("downloading " + source.URL)
		content, err = httpGet(ctx, source.URL, "")
	case SourceGist:
		report.Status("loading gist " + source.Gist)
		content, err = fetchGist(ctx, fetched)
	}
	if err != nil {
		return err
	}
	sum := sha256.Sum256(content)
	fetched.Commit = hex.EncodeToString(sum[:])
	if fetched.Version == "" {
		fetched.Version = "sha256:" + fetched.Commit[:12]
	}
	key := sha256.Sum256([]byte(source.Raw))
	directory := filepath.Join(home.CacheDir(), "recipes", "file-"+hex.EncodeToString(key[:8]))
	os.RemoveAll(directory)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(directory, fetched.Source.File), content, 0o644); err != nil {
		return err
	}
	fetched.Dir = directory
	return nil
}

func fetchGist(ctx context.Context, fetched *Fetched) ([]byte, error) {
	source := &fetched.Source
	address := gistAPI + source.Gist
	if source.Version != "" {
		address += "/" + source.Version
	}
	body, err := httpGet(ctx, address, "application/vnd.github+json")
	if err != nil {
		return nil, err
	}
	var gist gistResponse
	if err := json.Unmarshal(body, &gist); err != nil {
		return nil, fmt.Errorf("parsing gist %s: %w", source.Gist, err)
	}
	if source.File == "" {
		var candidates []string
		for name := range gist.Files {
			if IsRecipeFileName(name) {
				candidates = append(candidates, name)
			}
		}
		slices.Sort(candidates)
		switch len(candidates) {
		case 0:
			return nil, fmt.Errorf("gist %s contains no Recipefile, *.Recipefile, *.recipe or *.toml", source.Gist)
		case 1:
			source.File = candidates[0]
		default:
			return nil, fmt.Errorf("gist %s contains several recipes (%s), pick one with gist:%s/<file>", source.Gist, strings.Join(candidates, ", "), source.Gist)
		}
	}
	file, found := gist.Files[source.File]
	if !found {
		return nil, fmt.Errorf("gist %s has no file %q", source.Gist, source.File)
	}
	if len(gist.History) > 0 && source.Version == "" {
		fetched.Version = "rev:" + shortCommit(gist.History[0].Version)
	} else if source.Version != "" {
		fetched.Version = "rev:" + shortCommit(source.Version)
	}
	if !file.Truncated {
		return []byte(file.Content), nil
	}
	return httpGet(ctx, file.RawURL, "")
}

func httpGet(ctx context.Context, address string, accept string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "nwm/"+buildinfo.Version)
	if accept != "" {
		request.Header.Set("Accept", accept)
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" && strings.HasPrefix(address, "https://api.github.com/") {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", address, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: %s", address, response.Status)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, maxRecipeFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(content) > maxRecipeFileSize {
		return nil, fmt.Errorf("%s is larger than 1 MiB", address)
	}
	return content, nil
}
