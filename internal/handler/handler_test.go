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

func TestUnarchiveRecreatesOriginalDir(t *testing.T) {
	vaultDir := t.TempDir()
	handler := NewFileHandler(vaultDir)

	originalDir := filepath.Join(vaultDir, "projects", "sub")
	if err := os.MkdirAll(originalDir, 0o755); err != nil {
		t.Fatalf("failed to create original directory: %v", err)
	}

	originalPath := filepath.Join(originalDir, "note.md")
	content := []byte("archived content")
	if err := os.WriteFile(originalPath, content, 0o644); err != nil {
		t.Fatalf("failed to write original file: %v", err)
	}

	archivePath := filepath.Join(vaultDir, "archive", "projects", "sub", "note.md")
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		t.Fatalf("failed to create archive directory: %v", err)
	}

	if err := os.Rename(originalPath, archivePath); err != nil {
		t.Fatalf("failed to move file to archive: %v", err)
	}

	if err := os.RemoveAll(originalDir); err != nil {
		t.Fatalf("failed to remove original directory: %v", err)
	}

	if err := handler.Unarchive(archivePath); err != nil {
		t.Fatalf("Unarchive returned error: %v", err)
	}

	if _, err := os.Stat(originalDir); err != nil {
		t.Fatalf("expected original directory to exist: %v", err)
	}

	restored, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatalf("failed to read restored file: %v", err)
	}

	if string(restored) != string(content) {
		t.Fatalf("restored file content mismatch: got %q, want %q", string(restored), string(content))
	}
}

func TestUntrashRecreatesOriginalDir(t *testing.T) {
	vaultDir := t.TempDir()
	handler := NewFileHandler(vaultDir)

	originalDir := filepath.Join(vaultDir, "projects", "sub")
	if err := os.MkdirAll(originalDir, 0o755); err != nil {
		t.Fatalf("failed to create original directory: %v", err)
	}

	originalPath := filepath.Join(originalDir, "note.md")
	content := []byte("trashed content")
	if err := os.WriteFile(originalPath, content, 0o644); err != nil {
		t.Fatalf("failed to write original file: %v", err)
	}

	trashPath := filepath.Join(vaultDir, "trash", "projects", "sub", "note.md")
	if err := os.MkdirAll(filepath.Dir(trashPath), 0o755); err != nil {
		t.Fatalf("failed to create trash directory: %v", err)
	}

	if err := os.Rename(originalPath, trashPath); err != nil {
		t.Fatalf("failed to move file to trash: %v", err)
	}

	if err := os.RemoveAll(originalDir); err != nil {
		t.Fatalf("failed to remove original directory: %v", err)
	}

	if err := handler.Untrash(trashPath); err != nil {
		t.Fatalf("Untrash returned error: %v", err)
	}

	if _, err := os.Stat(originalDir); err != nil {
		t.Fatalf("expected original directory to exist: %v", err)
	}

	restored, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatalf("failed to read restored file: %v", err)
	}

	if string(restored) != string(content) {
		t.Fatalf("restored file content mismatch: got %q, want %q", string(restored), string(content))
	}
}
