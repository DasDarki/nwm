package recipe

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DasDarki/nwm/internal/server"
)

func TestParseResourceWeb(t *testing.T) {
	manifest, err := ParseDSL("package game enable path . web frontend\n")
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Packages[0].Web != "frontend" {
		t.Fatalf("web: %+v", manifest.Packages[0])
	}
	if !strings.Contains(RenderDSL(manifest), "web frontend") {
		t.Errorf("web option is missing from the rendered recipe")
	}
}

func TestParseResourceWebNeedsLocalSource(t *testing.T) {
	if _, err := ParseDSL("package game web frontend\n"); err == nil {
		t.Error("expected an error for a vault package")
	}
	if _, err := ParseDSL("assets pack path . web frontend\n"); err == nil {
		t.Error("expected an error for an asset pack")
	}
}

func TestPlanRejectsPackageRootAsWeb(t *testing.T) {
	if _, err := PlanBuild(Resource{Name: "game", Web: "."}, t.TempDir()); err == nil {
		t.Error("expected an error")
	}
}

func TestWebFolderIsNeitherMountedNorWatched(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Package.toml"), "")
	writeFile(t, filepath.Join(root, "Client", "Index.lua"), "")
	writeFile(t, filepath.Join(root, "frontend", "src", "main.ts"), "")
	plan, err := PlanBuild(Resource{Name: "game", Web: "frontend"}, root)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "game")
	mount, err := Assemble("game", server.PackageKind, plan, target, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "frontend")); !os.IsNotExist(err) {
		t.Errorf("web folder was mounted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "Client", "Index.lua")); err != nil {
		t.Errorf("package content is missing: %v", err)
	}
	if change, _ := mount.Classify(filepath.Join(root, "frontend", "src", "main.ts")); change != ChangeNone {
		t.Errorf("web change was classified as %v", change)
	}
}

func TestBuildWebInstallsAndBuilds(t *testing.T) {
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm is not installed")
	}
	root := t.TempDir()
	web := filepath.Join(root, "frontend")
	writeFile(t, filepath.Join(web, "package.json"), `{"name":"web","version":"1.0.0","scripts":{"build":"node -e \"require('fs').writeFileSync('built.txt','ok')\""}}`)
	plan := &BuildPlan{Root: root, Web: web}
	if err := BuildWeb(context.Background(), "game", plan, ApplyOptions{}, silentReporter{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(web, "built.txt")); err != nil {
		t.Errorf("build did not run: %v", err)
	}
}

func TestBuildWebAsksBeforeBuilding(t *testing.T) {
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm is not installed")
	}
	plan := &BuildPlan{Web: t.TempDir()}
	options := ApplyOptions{ConfirmBuild: func(string, []string) bool { return false }}
	if err := BuildWeb(context.Background(), "game", plan, options, silentReporter{}); err == nil {
		t.Error("expected the declined build to fail")
	}
}
