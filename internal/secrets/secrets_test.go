package secrets

import (
	"os"
	"testing"
)

func TestRandomUsesSafeAlphabet(t *testing.T) {
	value, err := Random(64)
	if err != nil {
		t.Fatal(err)
	}
	if len(value) != 64 {
		t.Fatalf("expected 64 characters, got %d", len(value))
	}
	for _, character := range value {
		if !containsRune(alphabet, character) {
			t.Fatalf("unexpected character %q", character)
		}
	}
}

func TestStoreRoundTrip(t *testing.T) {
	t.Setenv("NWM_HOME", t.TempDir())
	store, err := Open("Dev")
	if err != nil {
		t.Fatal(err)
	}
	store.Set("db_password", "secret")
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path("dev"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("secret file has mode %v", info.Mode().Perm())
	}
	reopened, err := Open("dev")
	if err != nil {
		t.Fatal(err)
	}
	if value, _ := reopened.Get("db_password"); value != "secret" {
		t.Errorf("got %q", value)
	}
}

func containsRune(text string, wanted rune) bool {
	for _, character := range text {
		if character == wanted {
			return true
		}
	}
	return false
}
