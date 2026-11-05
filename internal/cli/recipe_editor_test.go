package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DasDarki/nwm/internal/recipe"
	"github.com/DasDarki/nwm/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func typeText(model editorModel, text string) editorModel {
	for _, character := range text {
		next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{character}})
		model = next.(editorModel)
	}
	return model
}

func press(model editorModel, key tea.KeyType) (editorModel, tea.Cmd) {
	next, command := model.Update(tea.KeyMsg{Type: key})
	return next.(editorModel), command
}

func TestEditorSession(t *testing.T) {
	directory := t.TempDir()
	document := recipe.NewDocument(directory, recipe.FormatDSL, "@me/test")
	model := newEditorModel(document, true)
	next, _ := model.Update(storeLoadedMessage{index: &store.Index{Packages: []store.Item{{Name: "sandbox", Title: "Sandbox"}}}})
	model = next.(editorModel)

	model = typeText(model, "pkg sand")
	model, _ = press(model, tea.KeyTab)
	if model.input.Value() != "pkg sandbox " {
		t.Fatalf("tab completion produced %q", model.input.Value())
	}
	model = typeText(model, "enable")
	model, _ = press(model, tea.KeyEnter)
	if len(document.Manifest.Packages) != 1 || !document.Manifest.Packages[0].Enable {
		t.Fatalf("package not added: %+v (message %q)", document.Manifest.Packages, model.message)
	}

	model = typeText(model, "use postgres")
	model, _ = press(model, tea.KeyEnter)
	if len(document.Manifest.Services) != 1 || len(document.Manifest.Secrets) != 1 {
		t.Fatalf("use postgres failed: %q", model.message)
	}

	model = typeText(model, "nonsense here")
	model, _ = press(model, tea.KeyEnter)
	if !model.messageIsError {
		t.Error("unknown commands should report an error")
	}

	model = typeText(model, "undo")
	model, _ = press(model, tea.KeyEnter)
	if len(document.Manifest.Services) != 0 {
		t.Errorf("undo did not revert use postgres: %+v", document.Manifest.Services)
	}

	model, _ = press(model, tea.KeyUp)
	if model.input.Value() != "undo" {
		t.Errorf("history up gave %q", model.input.Value())
	}
	model, _ = press(model, tea.KeyEsc)

	model = typeText(model, "quit")
	model, command := press(model, tea.KeyEnter)
	if command != nil || !strings.Contains(model.message, "unsaved") {
		t.Error("quit with unsaved changes should be refused")
	}

	model = typeText(model, "save")
	model, _ = press(model, tea.KeyEnter)
	content, err := os.ReadFile(filepath.Join(directory, recipe.DSLFileName))
	if err != nil || !strings.Contains(string(content), "package sandbox enable") {
		t.Fatalf("saved file: %q %v", content, err)
	}

	model = typeText(model, "save toml")
	model, _ = press(model, tea.KeyEnter)
	if _, err := os.Stat(filepath.Join(directory, recipe.FileName)); err != nil {
		t.Errorf("save toml did not write recipe.toml: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, recipe.DSLFileName)); err == nil {
		t.Error("converting should remove the Recipefile")
	}

	model = typeText(model, "quit")
	_, command = press(model, tea.KeyEnter)
	if command == nil {
		t.Error("quit after saving should exit")
	}
	if view := model.View(); !strings.Contains(view, "package sandbox enable") {
		t.Error("preview should show the recipe")
	}
}
