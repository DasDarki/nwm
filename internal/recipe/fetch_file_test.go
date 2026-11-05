package recipe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type silentReporter struct{}

func (silentReporter) Status(string)    {}
func (silentReporter) Progress(float64) {}

const remoteRecipe = "recipe @me/remote 1.0.0\npackage sandbox enable\nuse redis\n"

func TestParseFileSources(t *testing.T) {
	directory := t.TempDir()
	file := filepath.Join(directory, "staging.Recipefile")
	os.WriteFile(file, []byte(remoteRecipe), 0o644)

	local, err := ParseSource(file)
	if err != nil || local.Kind != SourceLocal || local.Path != directory || local.File != "staging.Recipefile" {
		t.Errorf("local file source: %+v %v", local, err)
	}
	url, _ := ParseSource("https://example.com/recipes/rp.Recipefile?token=1")
	if url.Kind != SourceURL || url.File != "rp.Recipefile" {
		t.Errorf("url source: %+v", url)
	}
	git, _ := ParseSource("https://example.com/recipes.git#v1")
	if git.Kind != SourceGit {
		t.Errorf("git urls must stay git: %+v", git)
	}
	gist, _ := ParseSource("gist:DasDarki/0123456789abcdef0123/dev.Recipefile@abcdef1")
	if gist.Kind != SourceGist || gist.Gist != "0123456789abcdef0123" || gist.File != "dev.Recipefile" || gist.Version != "abcdef1" {
		t.Errorf("gist source: %+v", gist)
	}
	short, _ := ParseSource("gist:0123456789abcdef0123")
	if short.Gist != "0123456789abcdef0123" || short.File != "" {
		t.Errorf("short gist source: %+v", short)
	}
	if _, err := ParseSource("gist:nothex"); err == nil {
		t.Error("invalid gist ids should fail")
	}
}

func TestFetchURLAndGist(t *testing.T) {
	t.Setenv("NWM_HOME", t.TempDir())
	handler := http.NewServeMux()
	handler.HandleFunc("/rp.Recipefile", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Write([]byte(remoteRecipe))
	})
	handler.HandleFunc("/bad.Recipefile", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Write([]byte("recipe @me/bad\nfile files/a.txt a.txt\n"))
	})
	handler.HandleFunc("/gists/0123456789abcdef0123", func(writer http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(writer).Encode(map[string]any{
			"files": map[string]any{
				"notes.md":       map[string]any{"filename": "notes.md", "content": "hi"},
				"dev.Recipefile": map[string]any{"filename": "dev.Recipefile", "content": remoteRecipe},
			},
			"history": []any{map[string]any{"version": "fedcba9876543210"}},
		})
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	gistAPI = server.URL + "/gists/"
	defer func() { gistAPI = "https://api.github.com/gists/" }()

	source, _ := ParseSource(server.URL + "/rp.Recipefile")
	fetched, err := Fetch(context.Background(), source, false, silentReporter{})
	if err != nil {
		t.Fatal(err)
	}
	if fetched.Manifest.Recipe.Name != "@me/remote" || !fetched.Remote || !strings.HasPrefix(fetched.Version, "sha256:") {
		t.Errorf("url fetch: %+v", fetched)
	}

	source, _ = ParseSource("gist:0123456789abcdef0123")
	fetched, err = Fetch(context.Background(), source, false, silentReporter{})
	if err != nil {
		t.Fatal(err)
	}
	if fetched.Source.File != "dev.Recipefile" || fetched.Version != "rev:fedcba9" || len(fetched.Manifest.Services) != 1 {
		t.Errorf("gist fetch: %+v", fetched)
	}

	source, _ = ParseSource(server.URL + "/bad.Recipefile")
	if _, err := Fetch(context.Background(), source, false, silentReporter{}); err == nil || !strings.Contains(err.Error(), "cannot reference local files") {
		t.Errorf("local references should be rejected, got %v", err)
	}
}

func TestConvertedFileName(t *testing.T) {
	cases := map[string]string{
		"staging.Recipefile": "staging.toml",
		"prod.recipe":        "prod.toml",
		"Recipefile":         "",
	}
	for input, expected := range cases {
		if actual := convertedFileName(input, FormatTOML); actual != expected {
			t.Errorf("convertedFileName(%q) = %q, want %q", input, actual, expected)
		}
	}
	if actual := convertedFileName("dev.toml", FormatDSL); actual != "dev.Recipefile" {
		t.Errorf("toml to dsl gave %q", actual)
	}
}
