package search

import (
	"os"
	"path/filepath"
	"testing"
)

func writeNote(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
	return path
}

func TestIndexSearchBodyRespectsToggle(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	note := writeNote(t, dir, "note.md", "---\ntitle: Example\n---\nbody term here")

	idx := NewIndex(dir, Config{EnableBody: true})
	if err := idx.Build([]string{note}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	results := idx.Search(Query{Term: "term"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result with body search enabled, got %d", len(results))
	}

	idx = NewIndex(dir, Config{EnableBody: false})
	if err := idx.Build([]string{note}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	results = idx.Search(Query{Term: "term"})
	if len(results) != 0 {
		t.Fatalf("expected 0 results with body search disabled, got %d", len(results))
	}
}

func TestIndexSearchIgnoredFolders(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	included := writeNote(t, dir, "keep/note.md", "---\ntitle: Keep\n---\nbody")
	ignored := writeNote(t, dir, "archive/skip.md", "---\ntitle: Skip\n---\nbody skip")

	idx := NewIndex(dir, Config{EnableBody: true, IgnoredFolders: []string{"archive"}})
	if err := idx.Build([]string{included, ignored}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	results := idx.Search(Query{Term: "skip"})
	if len(results) != 0 {
		t.Fatalf("expected ignored folder to be skipped, got %d results", len(results))
	}

	results = idx.Search(Query{Term: "keep"})
	if len(results) != 1 {
		t.Fatalf("expected included note to be searchable, got %d results", len(results))
	}
}

func TestIndexSearchSupportsMetadataAndTags(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	matched := writeNote(t, dir, "projects/plan.md", "---\ntitle: Plan\ntags:\n  - project\n  - urgent\nstatus: active\n---\nMilestone body\n")
	archived := writeNote(t, dir, "projects/archive.md", "---\ntitle: Old\ntags:\n  - project\nstatus: done\n---\nFinished body\n")

	idx := NewIndex(dir, Config{EnableBody: true})
	if err := idx.Build([]string{matched, archived}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	query := Query{
		Term: "milestone",
		Tags: []string{"project"},
		Metadata: map[string][]string{
			"status": []string{"active"},
		},
	}

	results := idx.Search(query)
	if len(results) != 1 || results[0].Path != filepath.Clean(matched) {
		t.Fatalf("expected matching note, got %+v", results)
	}

	// Metadata-only queries should still filter results.
	query.Term = ""
	results = idx.Search(query)
	if len(results) != 1 || results[0].Path != filepath.Clean(matched) {
		t.Fatalf("expected metadata filters to return match, got %+v", results)
	}

	query.Metadata["status"] = []string{"missing"}
	results = idx.Search(query)
	if len(results) != 0 {
		t.Fatalf("expected metadata filters to exclude non-matching notes, got %+v", results)
	}
}
