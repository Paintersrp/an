package handler

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestWalkFilesExcludesArchiveAndTrash(t *testing.T) {
	vaultDir := t.TempDir()

	keepPath := filepath.Join(vaultDir, "keep.md")
	if err := os.WriteFile(keepPath, []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to write keep file: %v", err)
	}

	archiveDir := filepath.Join(vaultDir, "archive", "nested")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatalf("failed to create archive directory: %v", err)
	}

	archivePaths := []string{
		filepath.Join(vaultDir, "archive", "note.md"),
		filepath.Join(archiveDir, "deep.md"),
	}

	for _, path := range archivePaths {
		if err := os.WriteFile(path, []byte("archived"), 0o644); err != nil {
			t.Fatalf("failed to write archive file %s: %v", path, err)
		}
	}

	trashDir := filepath.Join(vaultDir, "trash", "nested")
	if err := os.MkdirAll(trashDir, 0o755); err != nil {
		t.Fatalf("failed to create trash directory: %v", err)
	}

	trashPaths := []string{
		filepath.Join(vaultDir, "trash", "note.md"),
		filepath.Join(trashDir, "deep.md"),
	}

	for _, path := range trashPaths {
		if err := os.WriteFile(path, []byte("trashed"), 0o644); err != nil {
			t.Fatalf("failed to write trash file %s: %v", path, err)
		}
	}

	handler := NewFileHandler(vaultDir)

	files, err := handler.WalkFiles([]string{"archive", "trash"}, nil, "")
	if err != nil {
		t.Fatalf("WalkFiles returned error: %v", err)
	}

	sort.Strings(files)

	if len(files) != 1 || files[0] != keepPath {
		t.Fatalf("expected only keep file %s, got %v", keepPath, files)
	}

	disallowed := append([]string{}, archivePaths...)
	disallowed = append(disallowed, trashPaths...)

	for _, path := range disallowed {
		for _, file := range files {
			if file == path {
				t.Fatalf("unexpected file from excluded directories: %s", file)
			}
		}
	}
}
