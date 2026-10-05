package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRouterConfigHistory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Chdir(home)

	if got := lastRouterConfigPath(); got != "~/.lowkey/router.json" {
		t.Fatalf("unexpected first-run default: %q", got)
	}
	resolved, err := resolveRouterConfigPath(" ~/.lowkey/router.json ")
	if err != nil || resolved != filepath.Join(home, ".lowkey", "router.json") {
		t.Fatalf("home expansion: %q, %v", resolved, err)
	}
	if err := rememberRouterConfigPath(" ~/.lowkey/router.json "); err != nil {
		t.Fatal(err)
	}
	if got := lastRouterConfigPath(); got != resolved {
		t.Fatalf("path was not persisted: %q", got)
	}
	if err := rememberRouterConfigPath("custom-router.json"); err != nil {
		t.Fatal(err)
	}
	if got := lastRouterConfigPath(); got != filepath.Join(home, "custom-router.json") {
		t.Fatalf("relative path was not saved as absolute: %q", got)
	}
	if err := rememberRouterConfigPath(" "); err == nil {
		t.Fatal("accepted an empty path")
	}
	if got := lastRouterConfigPath(); got != filepath.Join(home, "custom-router.json") {
		t.Fatal("invalid path replaced saved history")
	}
	if err := os.WriteFile(filepath.Join(home, ".lowkey", "last-router-config"), []byte("\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := lastRouterConfigPath(); got != "~/.lowkey/router.json" {
		t.Fatalf("empty history did not fall back: %q", got)
	}
}
