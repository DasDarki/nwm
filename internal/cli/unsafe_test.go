package cli

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/DasDarki/nwm/internal/server"
	"github.com/DasDarki/nwm/internal/state"
)

func unsafeFixture(t *testing.T, modules ...string) (*state.State, *state.Server) {
	t.Helper()
	t.Setenv("NWM_HOME", t.TempDir())
	directory := t.TempDir()
	config, _ := server.LoadConfig(directory)
	for _, module := range modules {
		packageDir := filepath.Join(directory, "Packages", module)
		os.MkdirAll(packageDir, 0o755)
		os.WriteFile(filepath.Join(packageDir, "Package.toml"), []byte("[meta]\ntitle = \"m\"\n[c_module]\nmodules = [\"m\"]\n"), 0o644)
		config.AddToList("game.packages", module)
	}
	config.Save()
	current := &state.State{Servers: []state.Server{{Name: "dev", Path: directory}}}
	return current, &current.Servers[0]
}

func hasFlag(arguments []string) bool {
	return slices.Contains(arguments, server.UnsafeLibsFlag)
}

func TestUnsafeLibsGuard(t *testing.T) {
	recipeArguments := []string{"--log_level", "2", server.UnsafeLibsFlag}

	current, target := unsafeFixture(t, "mysql")
	arguments, _ := guardUnsafeLibs(current, target, recipeArguments, nil, false)
	if hasFlag(arguments) || !slices.Contains(arguments, "--log_level") {
		t.Errorf("without a terminal the flag must be dropped, got %v", arguments)
	}

	arguments, _ = guardUnsafeLibs(current, target, recipeArguments, []string{server.UnsafeLibsFlag}, false)
	if !hasFlag(arguments) {
		t.Error("an explicit flag on the command line counts as consent")
	}

	arguments, _ = guardUnsafeLibs(current, target, recipeArguments, nil, true)
	if !hasFlag(arguments) || target.UnsafeLibs == nil || !slices.Equal(target.UnsafeLibs.Modules, []string{"mysql"}) {
		t.Fatalf("--allow-unsafe-libs should approve and record the modules, got %v %+v", arguments, target.UnsafeLibs)
	}
	reloaded, _ := state.Load()
	if reloaded.Servers[0].UnsafeLibs == nil {
		t.Error("approval was not saved")
	}

	arguments, _ = guardUnsafeLibs(current, target, recipeArguments, nil, false)
	if !hasFlag(arguments) {
		t.Error("an approved server should keep the flag without asking")
	}

	newPackage := filepath.Join(target.Path, "Packages", "redis-native")
	os.MkdirAll(newPackage, 0o755)
	os.WriteFile(filepath.Join(newPackage, "Package.toml"), []byte("[meta]\ntitle = \"r\"\n[c_module]\nmodules = [\"r\"]\n"), 0o644)
	config, _ := server.LoadConfig(target.Path)
	config.AddToList("game.packages", "redis-native")
	config.Save()
	arguments, _ = guardUnsafeLibs(current, target, recipeArguments, nil, false)
	if hasFlag(arguments) {
		t.Error("a new C module must require a new approval")
	}

	current.AllowUnsafeLibs = true
	arguments, _ = guardUnsafeLibs(current, target, recipeArguments, nil, false)
	if !hasFlag(arguments) {
		t.Error("the global setting should allow the flag")
	}
}

func TestUnsafeLibsGuardIgnoresRecipesWithoutFlag(t *testing.T) {
	current, target := unsafeFixture(t, "mysql")
	arguments, _ := guardUnsafeLibs(current, target, []string{"--log_level", "2"}, nil, false)
	if len(arguments) != 2 || target.UnsafeLibs != nil {
		t.Errorf("recipes without the flag must not be touched, got %v", arguments)
	}
}
