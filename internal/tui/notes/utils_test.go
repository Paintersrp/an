package notes

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/bubbles/list"

	"github.com/Paintersrp/an/internal/tui/notes/submodels"
)

func newTestNoteListModel(t *testing.T, item ListItem, inputValue string) NoteListModel {
	t.Helper()

	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{item}, delegate, 0, 0)
	l.Select(0)

	inputModel := submodels.NewInputModel()
	inputModel.Input.SetValue(inputValue)

	return NoteListModel{
		list:       l,
		inputModel: inputModel,
	}
}

func TestParseNoteFilesHandlesRootAndNestedPaths(t *testing.T) {
	dir := t.TempDir()

	rootFile := filepath.Join(dir, "root.md")
	nestedDir := filepath.Join(dir, "sub")
	deepDir := filepath.Join(dir, "sub", "deep")
	nestedFile := filepath.Join(nestedDir, "child.md")
	deepFile := filepath.Join(deepDir, "grand.md")

	if err := os.MkdirAll(deepDir, 0o755); err != nil {
		t.Fatalf("failed to create nested directories: %v", err)
	}

	files := map[string]string{
		rootFile:   "---\ntitle: Root\n---\n",
		nestedFile: "---\ntitle: Child\n---\n",
		deepFile:   "---\ntitle: Grand\n---\n",
	}

	for path, contents := range files {
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("failed to write file %s: %v", path, err)
		}
	}

	noteFiles := []string{rootFile, nestedFile, deepFile}
	items := ParseNoteFiles(noteFiles, dir, false)

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	item0 := items[0].(ListItem)
	if item0.subdirectory != "" {
		t.Fatalf("expected empty subdirectory for root file, got %q", item0.subdirectory)
	}
	if item0.fileName != "root.md" {
		t.Fatalf("expected root fileName 'root.md', got %q", item0.fileName)
	}

	item1 := items[1].(ListItem)
	if item1.subdirectory != "sub" {
		t.Fatalf("expected subdirectory 'sub', got %q", item1.subdirectory)
	}
	if item1.fileName != "child.md" {
		t.Fatalf("expected fileName 'child.md', got %q", item1.fileName)
	}

	item2 := items[2].(ListItem)
	if item2.subdirectory != "sub" {
		t.Fatalf("expected top-level subdirectory 'sub', got %q", item2.subdirectory)
	}
	if item2.fileName != filepath.Join("deep", "grand.md") {
		t.Fatalf("expected fileName 'deep/grand.md', got %q", item2.fileName)
	}
}

func TestRenameFileSuccess(t *testing.T) {
	dir := t.TempDir()

	originalPath := filepath.Join(dir, "original.md")
	if err := os.WriteFile(originalPath, []byte("---\ntitle: Original\n---\nbody"), 0o644); err != nil {
		t.Fatalf("failed to write original file: %v", err)
	}

	item := ListItem{
		fileName: "original.md",
		path:     originalPath,
	}

	model := newTestNoteListModel(t, item, "Renamed")

	if err := renameFile(model); err != nil {
		t.Fatalf("renameFile returned error: %v", err)
	}

	newPath := filepath.Join(dir, "Renamed.md")
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("expected renamed file to exist: %v", err)
	}

	if _, err := os.Stat(originalPath); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected original file to be removed, got: %v", err)
	}

	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("failed to read renamed file: %v", err)
	}

	if string(data) != "---\ntitle: Renamed\n---\nbody" {
		t.Fatalf("unexpected renamed file contents: %q", string(data))
	}
}

func TestRenameFilePreservesBodyTitleOccurrences(t *testing.T) {
	dir := t.TempDir()

	originalContent := "---\ntitle: Original\n---\nOriginal body mentions Original multiple times.\n"
	originalPath := filepath.Join(dir, "original.md")
	if err := os.WriteFile(originalPath, []byte(originalContent), 0o644); err != nil {
		t.Fatalf("failed to write original file: %v", err)
	}

	item := ListItem{
		fileName: "original.md",
		path:     originalPath,
	}

	model := newTestNoteListModel(t, item, "Renamed")

	if err := renameFile(model); err != nil {
		t.Fatalf("renameFile returned error: %v", err)
	}

	renamedPath := filepath.Join(dir, "Renamed.md")
	data, err := os.ReadFile(renamedPath)
	if err != nil {
		t.Fatalf("failed to read renamed file: %v", err)
	}

	expected := "---\ntitle: Renamed\n---\nOriginal body mentions Original multiple times.\n"
	if string(data) != expected {
		t.Fatalf("unexpected renamed file contents: %q", string(data))
	}
}

func TestRenameFileWithoutTitle(t *testing.T) {
	tests := []struct {
		name     string
		contents string
	}{
		{
			name:     "no front matter",
			contents: "body only",
		},
		{
			name:     "front matter without title",
			contents: "---\ntags:\n  - tag\n---\nbody",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			originalPath := filepath.Join(dir, "original.md")
			if err := os.WriteFile(originalPath, []byte(tt.contents), 0o644); err != nil {
				t.Fatalf("failed to write original file: %v", err)
			}

			item := ListItem{
				fileName: "original.md",
				path:     originalPath,
			}

			model := newTestNoteListModel(t, item, "Renamed")

			if err := renameFile(model); err != nil {
				t.Fatalf("renameFile returned error: %v", err)
			}

			newPath := filepath.Join(dir, "Renamed.md")
			data, err := os.ReadFile(newPath)
			if err != nil {
				t.Fatalf("failed to read renamed file: %v", err)
			}

			if string(data) != tt.contents {
				t.Fatalf("expected contents to remain unchanged, got %q", string(data))
			}
		})
	}
}

func TestRenameFileCollision(t *testing.T) {
	dir := t.TempDir()

	originalPath := filepath.Join(dir, "original.md")
	if err := os.WriteFile(originalPath, []byte("---\ntitle: Original\n---\nbody"), 0o644); err != nil {
		t.Fatalf("failed to write original file: %v", err)
	}

	existingPath := filepath.Join(dir, "Existing.md")
	if err := os.WriteFile(existingPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("failed to write existing file: %v", err)
	}

	item := ListItem{
		fileName: "original.md",
		path:     originalPath,
	}

	model := newTestNoteListModel(t, item, "Existing")

	err := renameFile(model)
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("expected fs.ErrExist, got: %v", err)
	}

	data, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatalf("failed to read original file after collision: %v", err)
	}

	if string(data) != "---\ntitle: Original\n---\nbody" {
		t.Fatalf("original file contents changed on collision: %q", string(data))
	}

	existingData, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("failed to read existing destination file: %v", err)
	}

	if string(existingData) != "existing" {
		t.Fatalf("destination file was modified on collision: %q", string(existingData))
	}
}

func TestCopyFileSuccess(t *testing.T) {
	dir := t.TempDir()

	originalPath := filepath.Join(dir, "original.md")
	if err := os.WriteFile(originalPath, []byte("---\ntitle: Original\n---\nbody"), 0o644); err != nil {
		t.Fatalf("failed to write original file: %v", err)
	}

	item := ListItem{
		fileName: "original.md",
		path:     originalPath,
	}

	model := newTestNoteListModel(t, item, "Copy")

	if err := copyFile(model); err != nil {
		t.Fatalf("copyFile returned error: %v", err)
	}

	copyPath := filepath.Join(dir, "Copy.md")
	data, err := os.ReadFile(copyPath)
	if err != nil {
		t.Fatalf("failed to read copied file: %v", err)
	}

	if string(data) != "---\ntitle: Copy\n---\nbody" {
		t.Fatalf("unexpected copied file contents: %q", string(data))
	}
}

func TestCopyFilePreservesBodyTitleOccurrences(t *testing.T) {
	dir := t.TempDir()

	originalContent := "---\ntitle: Original\n---\nBody keeps Original references intact.\n"
	originalPath := filepath.Join(dir, "original.md")
	if err := os.WriteFile(originalPath, []byte(originalContent), 0o644); err != nil {
		t.Fatalf("failed to write original file: %v", err)
	}

	item := ListItem{
		fileName: "original.md",
		path:     originalPath,
	}

	model := newTestNoteListModel(t, item, "Copy")

	if err := copyFile(model); err != nil {
		t.Fatalf("copyFile returned error: %v", err)
	}

	copyPath := filepath.Join(dir, "Copy.md")
	data, err := os.ReadFile(copyPath)
	if err != nil {
		t.Fatalf("failed to read copied file: %v", err)
	}

	expected := "---\ntitle: Copy\n---\nBody keeps Original references intact.\n"
	if string(data) != expected {
		t.Fatalf("unexpected copied file contents: %q", string(data))
	}

	originalData, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatalf("failed to read original file: %v", err)
	}

	if string(originalData) != originalContent {
		t.Fatalf("original file contents changed: %q", string(originalData))
	}
}

func TestCopyFileWithoutTitle(t *testing.T) {
	tests := []struct {
		name     string
		contents string
	}{
		{
			name:     "no front matter",
			contents: "body only",
		},
		{
			name:     "front matter without title",
			contents: "---\ntags:\n  - tag\n---\nbody",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			originalPath := filepath.Join(dir, "original.md")
			if err := os.WriteFile(originalPath, []byte(tt.contents), 0o644); err != nil {
				t.Fatalf("failed to write original file: %v", err)
			}

			item := ListItem{
				fileName: "original.md",
				path:     originalPath,
			}

			model := newTestNoteListModel(t, item, "Copy")

			if err := copyFile(model); err != nil {
				t.Fatalf("copyFile returned error: %v", err)
			}

			copyPath := filepath.Join(dir, "Copy.md")
			data, err := os.ReadFile(copyPath)
			if err != nil {
				t.Fatalf("failed to read copied file: %v", err)
			}

			if string(data) != tt.contents {
				t.Fatalf("expected contents to remain unchanged, got %q", string(data))
			}
		})
	}
}

func TestCopyFileCollision(t *testing.T) {
	dir := t.TempDir()

	originalPath := filepath.Join(dir, "original.md")
	if err := os.WriteFile(originalPath, []byte("---\ntitle: Original\n---\nbody"), 0o644); err != nil {
		t.Fatalf("failed to write original file: %v", err)
	}

	copyPath := filepath.Join(dir, "Copy.md")
	if err := os.WriteFile(copyPath, []byte("existing copy"), 0o644); err != nil {
		t.Fatalf("failed to write existing copy file: %v", err)
	}

	item := ListItem{
		fileName: "original.md",
		path:     originalPath,
	}

	model := newTestNoteListModel(t, item, "Copy")

	err := copyFile(model)
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("expected fs.ErrExist from copyFile, got: %v", err)
	}

	data, err := os.ReadFile(copyPath)
	if err != nil {
		t.Fatalf("failed to read existing copy file: %v", err)
	}

	if string(data) != "existing copy" {
		t.Fatalf("destination file was modified on collision: %q", string(data))
	}

	originalData, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatalf("failed to read original file after copy collision: %v", err)
	}

	if string(originalData) != "---\ntitle: Original\n---\nbody" {
		t.Fatalf("original file contents changed on copy collision: %q", string(originalData))
	}
}
