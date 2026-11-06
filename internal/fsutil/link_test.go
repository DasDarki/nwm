package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinkDirAndSafeRemove(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "repo", "Server")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "Index.lua"), []byte("print('v1')"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "server", "Packages", "game", "Server")
	if err := LinkDir(source, link); err != nil {
		t.Fatalf("LinkDir: %v", err)
	}
	if !IsLink(link) {
		t.Fatal("link is not detected as a link")
	}
	content, err := os.ReadFile(filepath.Join(link, "Index.lua"))
	if err != nil || string(content) != "print('v1')" {
		t.Fatalf("reading through the link: %q %v", content, err)
	}
	os.WriteFile(filepath.Join(source, "Index.lua"), []byte("print('v2')"), 0o644)
	content, _ = os.ReadFile(filepath.Join(link, "Index.lua"))
	if string(content) != "print('v2')" {
		t.Fatalf("changes are not visible through the link: %q", content)
	}

	if err := RemoveAll(filepath.Join(root, "server")); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "server")); !os.IsNotExist(err) {
		t.Error("server folder still exists")
	}
	if _, err := os.Stat(filepath.Join(source, "Index.lua")); err != nil {
		t.Fatalf("RemoveAll deleted the link target: %v", err)
	}
}

func TestRemoveAllRemovesPlainTrees(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b", "c")
	os.MkdirAll(nested, 0o755)
	os.WriteFile(filepath.Join(nested, "file.txt"), []byte("x"), 0o644)
	if err := RemoveAll(filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "a")); !os.IsNotExist(err) {
		t.Error("tree was not removed")
	}
	if err := RemoveAll(filepath.Join(root, "missing")); err != nil {
		t.Errorf("removing a missing path should not fail: %v", err)
	}
}
