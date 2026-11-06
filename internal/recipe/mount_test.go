package recipe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DasDarki/nwm/internal/fsutil"
	"github.com/DasDarki/nwm/internal/server"
)

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPlanDetectsNebra(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "nebra.toml"), "target = \"5.4\"\nsource = \"code\"\noutput = \"build/lua\"\n")
	plan, err := PlanBuild(Resource{Name: "game"}, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Tool != "nebra" || len(plan.Build) != 2 || plan.Watch[1] != "watch" {
		t.Errorf("nebra commands: %+v", plan)
	}
	if plan.Output != filepath.Join(root, "build", "lua") || plan.Sources[0] != filepath.Join(root, "code") {
		t.Errorf("nebra folders: %+v", plan)
	}
	for _, excluded := range []string{"code", "build", "nebra.toml"} {
		found := false
		for _, entry := range plan.Exclude {
			found = found || entry == excluded
		}
		if !found {
			t.Errorf("%s should be excluded from the package: %v", excluded, plan.Exclude)
		}
	}
	explicit, _ := PlanBuild(Resource{Name: "game", Build: "make lua", Watch: "make watch"}, root)
	if explicit.Build[0] != "make" || explicit.Watch[1] != "watch" || explicit.Tool != "" {
		t.Errorf("explicit commands should win: %+v", explicit)
	}
}

func TestAssembleLinksAndMergesNebraPackage(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Package.toml"), "[meta]\ntitle = \"game\"\n[game_mode]\n")
	writeFile(t, filepath.Join(root, "nebra.toml"), "target = \"5.4\"\n")
	writeFile(t, filepath.Join(root, "src", "Server", "Index.neb"), "print 1")
	writeFile(t, filepath.Join(root, "out", "Server", "Index.lua"), "print(1)")
	writeFile(t, filepath.Join(root, "out", "Client", "Index.lua"), "print(2)")
	writeFile(t, filepath.Join(root, "Client", "ui", "index.html"), "<html>")
	writeFile(t, filepath.Join(root, "README.md"), "docs")

	plan, err := PlanBuild(Resource{Name: "game"}, root)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "Packages", "game")
	mount, err := Assemble("game", server.PackageKind, plan, target, true)
	if err != nil {
		t.Fatal(err)
	}
	if !fsutil.IsLink(filepath.Join(target, "Server")) {
		t.Error("Server only exists in the output and should be linked")
	}
	if fsutil.IsLink(filepath.Join(target, "Client")) {
		t.Error("Client exists in the root and the output and must be merged, not linked")
	}
	for _, expected := range []string{"Package.toml", "Server/Index.lua", "Client/Index.lua", "Client/ui/index.html", "README.md"} {
		if _, err := os.Stat(filepath.Join(target, expected)); err != nil {
			t.Errorf("%s missing in the package: %v", expected, err)
		}
	}
	for _, unexpected := range []string{"src", "out", "nebra.toml"} {
		if _, err := os.Stat(filepath.Join(target, unexpected)); err == nil {
			t.Errorf("%s should not be part of the package", unexpected)
		}
	}

	os.WriteFile(filepath.Join(target, "Package.toml"), []byte("rewritten by the server"), 0o644)
	original, _ := os.ReadFile(filepath.Join(root, "Package.toml"))
	if string(original) == "rewritten by the server" {
		t.Fatal("the server must not be able to rewrite the source Package.toml")
	}

	cases := map[string]Change{
		filepath.Join(root, "Package.toml"):               ChangeManifest,
		filepath.Join(root, "src", "Server", "Index.neb"): ChangeNone,
		filepath.Join(root, "out", "Server", "Index.lua"): ChangeContent,
		filepath.Join(root, "Client", "ui", "index.html"): ChangeContent,
		filepath.Join(root, "out", "Shared", "Util.lua"):  ChangeContent,
		filepath.Join(root, ".git", "HEAD"):               ChangeNone,
	}
	for path, expected := range cases {
		if change, _ := mount.Classify(path); change != expected {
			t.Errorf("Classify(%s) = %v, want %v", path, change, expected)
		}
	}

	writeFile(t, filepath.Join(root, "Client", "ui", "index.html"), "<html>v2")
	if err := mount.Resync("Client"); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(target, "Client", "ui", "index.html"))
	if string(content) != "<html>v2" {
		t.Errorf("resync did not update merged entry: %q", content)
	}

	if err := fsutil.RemoveAll(filepath.Dir(target)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "out", "Server", "Index.lua")); err != nil {
		t.Fatal("removing the server packages deleted source files")
	}
}

func TestAssembleCopyModeAndBuildWithoutWatch(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Package.toml"), "[meta]\ntitle = \"lib\"\n[script]\n")
	writeFile(t, filepath.Join(root, "Server", "Index.lua"), "print(1)")
	plan, _ := PlanBuild(Resource{Name: "lib", Build: "make"}, root)
	target := filepath.Join(t.TempDir(), "lib")
	mount, err := Assemble("lib", server.PackageKind, plan, target, false)
	if err != nil {
		t.Fatal(err)
	}
	if mount.Linked() {
		t.Error("copy mode must not create links")
	}
	if change, _ := mount.Classify(filepath.Join(root, "Server", "Index.lua")); change != ChangeSource {
		t.Errorf("a build without watch should rebuild on source changes, got %v", change)
	}
}
