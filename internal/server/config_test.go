package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveFillsRequiredTables(t *testing.T) {
	directory := t.TempDir()
	config, err := LoadConfig(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := config.Set("discover.port", int64(7790)); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(); err != nil {
		t.Fatal(err)
	}
	reloaded, err := LoadConfig(directory)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.String("discover.port") != "7790" {
		t.Errorf("port was overwritten by defaults: %s", reloaded.String("discover.port"))
	}
	for _, key := range []string{"general.max_players", "game.map", "debug.log_level", "optimization.max_tick_rate", "discover.query_port"} {
		if _, found := reloaded.Get(key); !found {
			t.Errorf("%s missing after save", key)
		}
	}
	if _, err := os.Stat(filepath.Join(directory, ConfigFileName)); err != nil {
		t.Fatal(err)
	}
}

func TestListOperations(t *testing.T) {
	config, err := LoadConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if added, _ := config.AddToList("game.packages", "sandbox"); !added {
		t.Error("expected sandbox to be added")
	}
	if added, _ := config.AddToList("game.packages", "sandbox"); added {
		t.Error("expected duplicate to be ignored")
	}
	if removed, _ := config.RemoveFromList("game.packages", "sandbox"); !removed {
		t.Error("expected sandbox to be removed")
	}
}

func TestParseValue(t *testing.T) {
	if value := ParseValue("7777"); value != int64(7777) {
		t.Errorf("expected integer, got %T", value)
	}
	if value := ParseValue("true"); value != true {
		t.Errorf("expected boolean, got %T", value)
	}
	if value := ParseValue("hello world"); value != "hello world" {
		t.Errorf("expected plain string, got %v", value)
	}
	if list, ok := ParseValue(`["a", "b"]`).([]any); !ok || len(list) != 2 {
		t.Errorf("expected list, got %v", list)
	}
	loose, ok := ParseValue(`[nanos-world, cyberpunk, 3, true]`).([]any)
	if !ok || len(loose) != 4 || loose[0] != "nanos-world" || loose[2] != int64(3) || loose[3] != true {
		t.Errorf("expected loose list, got %#v", loose)
	}
	if value := ParseValue(`[unbalanced`); value != "[unbalanced" {
		t.Errorf("broken lists should stay strings, got %#v", value)
	}
}

func TestEnableCModuleUsesPackageList(t *testing.T) {
	config, _ := LoadConfig(t.TempDir())
	key, err := Enable(config, PackageKind, &Resource{Name: "mysql", Type: "c_module"})
	if err != nil || key != "game.packages" {
		t.Fatalf("Enable returned %q, %v", key, err)
	}
	gameModeKey, _ := Enable(config, PackageKind, &Resource{Name: "sandbox", Type: "game_mode"})
	if gameModeKey != "game.game_mode" || config.String("game.game_mode") != "sandbox" {
		t.Errorf("game modes should stay single values, got %q", gameModeKey)
	}
}
