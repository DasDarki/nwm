package recipe

import (
	"testing"

	"github.com/DasDarki/nwm/internal/store"
)

func texts(suggestions []Suggestion) []string {
	result := make([]string, len(suggestions))
	for index, suggestion := range suggestions {
		result[index] = suggestion.Text
	}
	return result
}

func TestCompleter(t *testing.T) {
	manifest, err := ParseDSL("recipe test\npackage sandbox\nuse redis\n")
	if err != nil {
		t.Fatal(err)
	}
	completer := &Completer{Manifest: manifest, Editor: true, Store: &store.Index{
		Packages: []store.Item{{Name: "sandbox", Title: "Sandbox", Type: "game_mode"}, {Name: "battlefield-kill-ui", Title: "Battlefield Kill UI"}},
	}}
	cases := []struct {
		line     string
		start    int
		contains string
	}{
		{"pa", 0, "package"},
		{"package san", 8, "sandbox"},
		{"package kill", 8, "battlefield-kill-ui"},
		{"package sandbox ", 16, "enable"},
		{"use po", 4, "postgres"},
		{"config discover.p", 7, "discover.port"},
		{"remove package ", 15, "sandbox"},
		{"remove ser", 7, "service"},
		{"branch ", 7, "bleeding-edge"},
		{"secret token ", 13, "prompt"},
		{"svc db ", 7, "postgres:17"},
		{"dev pa", 4, "package"},
		{"dev package san", 12, "sandbox"},
		{"package x path . reload ", 24, "full"},
		{"package x ", 10, "watch"},
		{"tool neb", 5, "nebra"},
	}
	for _, testCase := range cases {
		start, suggestions := completer.Complete(testCase.line)
		found := false
		for _, text := range texts(suggestions) {
			if text == testCase.contains {
				found = true
			}
		}
		if start != testCase.start || !found {
			t.Errorf("Complete(%q) = %d %v, want %d containing %q", testCase.line, start, texts(suggestions), testCase.start, testCase.contains)
		}
	}
	if _, suggestions := completer.Complete("package sandbox enable "); contains(texts(suggestions), "enable") {
		t.Error("used options should not be suggested again")
	}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
