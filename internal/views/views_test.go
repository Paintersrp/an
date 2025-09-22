package views

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Paintersrp/an/internal/handler"
)

func TestGetFilesByView_DefaultAndArchive(t *testing.T) {
	vaultDir := t.TempDir()

	mustMkdirAll(t, filepath.Join(vaultDir, "archive", "project"))
	mustMkdirAll(t, filepath.Join(vaultDir, "trash", "project"))
	mustMkdirAll(t, filepath.Join(vaultDir, "notes"))

	keepPath := filepath.Join(vaultDir, "notes", "keep.md")
	skipPath := filepath.Join(vaultDir, "notes", "skip.md")
	archivedPath := filepath.Join(vaultDir, "archive", "project", "archived.md")
	trashedPath := filepath.Join(vaultDir, "trash", "project", "trashed.md")

	mustWriteFile(t, keepPath)
	mustWriteFile(t, skipPath)
	mustWriteFile(t, archivedPath)
	mustWriteFile(t, trashedPath)

	h := handler.NewFileHandler(vaultDir)
	vm := NewViewManager(h, vaultDir)

	vm.Views["custom"] = View{
		ExcludeDirs:  []string{},
		ExcludeFiles: []string{"skip.md"},
		OrphanOnly:   false,
	}

	t.Run("default view excludes archive and trash", func(t *testing.T) {
		files, err := vm.GetFilesByView("default", vaultDir)
		if err != nil {
			t.Fatalf("GetFilesByView returned error: %v", err)
		}

		if !contains(files, keepPath) {
			t.Fatalf("default view missing expected file %s", keepPath)
		}

		if contains(files, archivedPath) {
			t.Fatalf("default view unexpectedly contained archived file %s", archivedPath)
		}

		if contains(files, trashedPath) {
			t.Fatalf("default view unexpectedly contained trashed file %s", trashedPath)
		}
	})

	t.Run("archive view returns archived notes", func(t *testing.T) {
		files, err := vm.GetFilesByView("archive", vaultDir)
		if err != nil {
			t.Fatalf("GetFilesByView returned error: %v", err)
		}

		if !contains(files, archivedPath) {
			t.Fatalf("archive view missing expected file %s", archivedPath)
		}

		if contains(files, keepPath) {
			t.Fatalf("archive view unexpectedly contained active file %s", keepPath)
		}
	})

	t.Run("custom view excludes configured files", func(t *testing.T) {
		files, err := vm.GetFilesByView("custom", vaultDir)
		if err != nil {
			t.Fatalf("GetFilesByView returned error: %v", err)
		}

		if contains(files, skipPath) {
			t.Fatalf("custom view unexpectedly contained excluded file %s", skipPath)
		}

		if !contains(files, keepPath) {
			t.Fatalf("custom view missing expected file %s", keepPath)
		}
	})

	t.Run("invalid view returns error", func(t *testing.T) {
		if _, err := vm.GetFilesByView("unknown", vaultDir); err == nil {
			t.Fatal("expected error for unknown view, got nil")
		}
	})
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
