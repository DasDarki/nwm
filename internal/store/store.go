package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/DasDarki/nwm/internal/buildinfo"
	"github.com/DasDarki/nwm/internal/home"
)

const (
	indexURL      = "https://api.nanos-world.com/store/index"
	cacheLifetime = 24 * time.Hour
)

type Item struct {
	Name             string `json:"name"`
	Title            string `json:"title"`
	ShortDescription string `json:"short_description"`
	Type             string `json:"type"`
}

type Index struct {
	Packages []Item `json:"packages"`
	Assets   []Item `json:"assets"`
}

type response struct {
	Payload Index `json:"payload"`
}

func cachePath() string {
	return filepath.Join(home.CacheDir(), "store-index.json")
}

func Load(ctx context.Context, refresh bool) (*Index, error) {
	if !refresh {
		if info, err := os.Stat(cachePath()); err == nil && time.Since(info.ModTime()) < cacheLifetime {
			if index, err := readCache(); err == nil {
				return index, nil
			}
		}
	}
	index, err := fetch(ctx)
	if err != nil {
		if cached, cacheErr := readCache(); cacheErr == nil {
			return cached, nil
		}
		return nil, err
	}
	return index, nil
}

func fetch(ctx context.Context) (*Index, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, indexURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "nwm/"+buildinfo.Version)
	result, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetching store index: %w", err)
	}
	defer result.Body.Close()
	if result.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching store index: %s", result.Status)
	}
	content, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}
	var decoded response
	if err := json.Unmarshal(content, &decoded); err != nil {
		return nil, fmt.Errorf("parsing store index: %w", err)
	}
	if cached, err := json.Marshal(decoded.Payload); err == nil {
		if err := os.MkdirAll(filepath.Dir(cachePath()), 0o755); err == nil {
			os.WriteFile(cachePath(), cached, 0o644)
		}
	}
	return &decoded.Payload, nil
}

func readCache() (*Index, error) {
	content, err := os.ReadFile(cachePath())
	if err != nil {
		return nil, err
	}
	var index Index
	return &index, json.Unmarshal(content, &index)
}
