package server

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func TestApplyModeKeepsSettingsAndPermissions(t *testing.T) {
	directory := t.TempDir()
	config, _ := LoadConfig(directory)
	config.Set("custom_settings.database", "secret-value")
	config.Set("discover.port", int64(7801))
	if err := config.Save(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, ConfigFileName)
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}

	for _, mode := range []string{ModeDev, ModeProd} {
		environment, err := ApplyMode(directory, mode)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(environment, []string{"NWM_MODE=" + mode}) {
			t.Errorf("environment = %v", environment)
		}
		reloaded, err := LoadConfig(directory)
		if err != nil {
			t.Fatal(err)
		}
		if reloaded.String("custom_settings.NWM_MODE") != mode {
			t.Errorf("custom setting = %q, want %q", reloaded.String("custom_settings.NWM_MODE"), mode)
		}
		if reloaded.String("custom_settings.database") != "secret-value" || reloaded.String("discover.port") != "7801" {
			t.Error("other settings were changed")
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("private config became %v", info.Mode().Perm())
	}
}
