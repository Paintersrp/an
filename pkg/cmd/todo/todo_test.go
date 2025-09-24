package todo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Paintersrp/an/internal/config"
)

func TestCollectTODOsFindsEntries(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte("package main\n// TODO: refactor handler\n"), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	entries, err := collectTODOs(dir)
	if err != nil {
		t.Fatalf("collectTODOs returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(entries))
	}
	if entries[0].Text != "refactor handler" {
		t.Fatalf("expected TODO text to be trimmed, got %q", entries[0].Text)
	}
	if entries[0].RelPath != "main.go" {
		t.Fatalf("expected relative path, got %q", entries[0].RelPath)
	}
}

func TestResolveDestinationUsesPinnedTaskFile(t *testing.T) {
	ws := &config.Workspace{VaultDir: "/tmp/vault", PinnedTaskFile: "/tmp/vault/tasks.md"}

	dest, err := resolveDestination(ws, "")
	if err != nil {
		t.Fatalf("resolveDestination returned error: %v", err)
	}
	if dest != "/tmp/vault/tasks.md" {
		t.Fatalf("expected pinned task file to be used, got %q", dest)
	}
}

func TestResolveDestinationRejectsOutsideVault(t *testing.T) {
	ws := &config.Workspace{VaultDir: "/tmp/vault"}
	if _, err := resolveDestination(ws, "../other.md"); err == nil {
		t.Fatalf("expected resolveDestination to reject paths outside the vault")
	}
}
