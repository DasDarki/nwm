package recipe

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/DasDarki/nwm/internal/server"
)

func TestModuleNames(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"libexample.so", "example.dll", "libother.so", "readme.md"} {
		os.WriteFile(filepath.Join(directory, name), []byte("x"), 0o644)
	}
	names := moduleNames(directory)
	if !slices.Equal(names, []string{"example", "other"}) {
		t.Errorf("moduleNames = %v", names)
	}
}

func TestInstallModuleCreatesPackage(t *testing.T) {
	recipeDir := t.TempDir()
	serverDir := t.TempDir()
	os.MkdirAll(filepath.Join(recipeDir, "bin"), 0o755)
	os.WriteFile(filepath.Join(recipeDir, "bin", "libexample.so"), []byte("elf"), 0o644)
	os.WriteFile(filepath.Join(recipeDir, "bin", "example.dll"), []byte("pe"), 0o644)

	module := Module{Name: "example-module", Linux: "bin/libexample.so", Windows: "bin/example.dll", Enable: true}
	if err := installModules(context.Background(), recipeDir, serverDir, []Module{module}, silentReporter{}); err != nil {
		t.Fatal(err)
	}
	resource, err := server.FindResource(serverDir, server.PackageKind, "example-module")
	if err != nil {
		t.Fatal(err)
	}
	if resource.Type != "c_module" {
		t.Errorf("generated package has type %q", resource.Type)
	}
	manifest, _ := os.ReadFile(filepath.Join(serverDir, "Packages", "example-module", "Package.toml"))
	if !strings.Contains(string(manifest), `modules = ["example"]`) {
		t.Errorf("generated Package.toml:\n%s", manifest)
	}
	expected := "libexample.so"
	if runtime.GOOS == "windows" {
		expected = "example.dll"
	}
	if _, err := os.Stat(filepath.Join(serverDir, "Packages", "example-module", expected)); err != nil {
		t.Errorf("binary missing: %v", err)
	}

	if err := writeConfiguration(serverDir, &Manifest{Modules: []Module{module}}); err != nil {
		t.Fatal(err)
	}
	config, _ := server.LoadConfig(serverDir)
	packages, _ := config.Get("game.packages")
	if list, ok := packages.([]any); !ok || len(list) != 1 || list[0] != "example-module" {
		t.Errorf("c module should be enabled in game.packages, got %v (game_mode %q)", packages, config.String("game.c_module"))
	}
}

func TestInstallModuleKeepsShippedPackageToml(t *testing.T) {
	recipeDir := t.TempDir()
	serverDir := t.TempDir()
	archivePath := filepath.Join(recipeDir, "module.zip")
	file, _ := os.Create(archivePath)
	writer := zip.NewWriter(file)
	for name, content := range map[string]string{
		"my-module/Package.toml":  "[meta]\ntitle = \"Shipped\"\n[c_module]\nmodules = [\"shipped\"]\n",
		"my-module/libshipped.so": "elf",
		"my-module/shipped.dll":   "pe",
	} {
		entry, _ := writer.Create(name)
		entry.Write([]byte(content))
	}
	writer.Close()
	file.Close()

	module := Module{Name: "shipped", Linux: "module.zip", Windows: "module.zip"}
	if err := installModules(context.Background(), recipeDir, serverDir, []Module{module}, silentReporter{}); err != nil {
		t.Fatal(err)
	}
	resource, err := server.FindResource(serverDir, server.PackageKind, "shipped")
	if err != nil || resource.Title != "Shipped" || resource.Type != "c_module" {
		t.Errorf("shipped package not kept: %+v %v", resource, err)
	}
}
