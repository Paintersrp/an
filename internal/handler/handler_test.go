package handler

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestWalkFilesDefaultIncludesRootAndNestedNotes(t *testing.T) {
	t.Parallel()

	vaultDir := t.TempDir()

	rootNote := filepath.Join(vaultDir, "root.md")
	nestedDir := filepath.Join(vaultDir, "project")
	nestedNote := filepath.Join(nestedDir, "nested.md")
	archivedNote := filepath.Join(vaultDir, "archive", "archived.md")
	trashedNote := filepath.Join(vaultDir, "trash", "trashed.md")

	mustWriteFile(t, rootNote)
	mustMkdirAll(t, nestedDir)
	mustWriteFile(t, nestedNote)
	mustWriteFile(t, archivedNote)
	mustWriteFile(t, trashedNote)

	h := NewFileHandler(vaultDir)

	files, err := h.WalkFiles([]string{"archive", "trash"}, nil, "default")
	if err != nil {
		t.Fatalf("WalkFiles returned error: %v", err)
	}

	slices.Sort(files)
	expected := []string{rootNote, nestedNote}
	slices.Sort(expected)

	if !slices.Equal(files, expected) {
		t.Fatalf("WalkFiles returned %v, want %v", files, expected)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}
