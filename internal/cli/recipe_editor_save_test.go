package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DasDarki/nwm/internal/recipe"
	tea "github.com/charmbracelet/bubbletea"
)

func TestEditorSaveCommandsWriteTheFile(t *testing.T) {
	cases := []struct {
		name    string
		format  recipe.Format
		command string
		file    string
	}{
		{"save on Recipefile", recipe.FormatDSL, "save", recipe.DSLFileName},
		{"save dsl on Recipefile", recipe.FormatDSL, "save dsl", recipe.DSLFileName},
		{"w on Recipefile", recipe.FormatDSL, "w", recipe.DSLFileName},
		{"save toml on recipe.toml", recipe.FormatTOML, "save toml", recipe.FileName},
		{"save dsl converts recipe.toml", recipe.FormatTOML, "save dsl", recipe.DSLFileName},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			document := recipe.NewDocument(directory, testCase.format, "@me/test")
			if err := document.Save(); err != nil {
				t.Fatal(err)
			}
			opened, err := recipe.OpenDocument(directory)
			if err != nil {
				t.Fatal(err)
			}
			model := newEditorModel(opened, false)
			model = typeText(model, "package sandbox enable")
			model, _ = press(model, tea.KeyEnter)
			model = typeText(model, testCase.command)
			model, _ = press(model, tea.KeyEnter)
			if model.dirty {
				t.Errorf("editor still reports unsaved changes: %q", model.message)
			}
			reopened, err := recipe.OpenDocument(directory)
			if err != nil {
				t.Fatal(err)
			}
			if len(reopened.Manifest.Packages) != 1 || reopened.Manifest.Packages[0].Name != "sandbox" {
				t.Errorf("%s did not persist the change, file has %+v", testCase.command, reopened.Manifest.Packages)
			}
			if _, err := os.Stat(filepath.Join(directory, testCase.file)); err != nil {
				t.Errorf("expected %s: %v", testCase.file, err)
			}
		})
	}
}

func TestEditorSaveKeepsCustomFileName(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "staging.Recipefile")
	document := recipe.NewDocumentFile(path, "@me/staging")
	if err := document.Save(); err != nil {
		t.Fatal(err)
	}
	opened, err := recipe.OpenDocumentFile(path)
	if err != nil {
		t.Fatal(err)
	}
	model := newEditorModel(opened, false)
	model = typeText(model, "use redis")
	model, _ = press(model, tea.KeyEnter)
	model = typeText(model, "save dsl")
	model, _ = press(model, tea.KeyEnter)
	content, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(content), "service redis") {
		t.Errorf("staging.Recipefile was not updated: %q %v", content, err)
	}
	if _, err := os.Stat(filepath.Join(directory, recipe.DSLFileName)); err == nil {
		t.Error("saving a named recipe must not create a plain Recipefile")
	}
}
