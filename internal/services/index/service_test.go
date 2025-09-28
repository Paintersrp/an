package index

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Paintersrp/an/internal/search"
)

func writeTestNote(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestServiceAcquireSnapshotAppliesPendingUpdates(t *testing.T) {
	dir := t.TempDir()
	note := writeTestNote(t, dir, "note.md", "---\ntitle: First\n---\nOriginal content")

	svc := NewService(dir, search.Config{EnableBody: true})
	idx, err := svc.AcquireSnapshot()
	if err != nil {
		t.Fatalf("AcquireSnapshot returned error: %v", err)
	}
	if idx == nil {
		t.Fatalf("expected snapshot index")
	}

	results := idx.Search(search.Query{Term: "original"})
	if len(results) != 1 {
		t.Fatalf("expected initial content to be indexed, got %+v", results)
	}

	updated := "---\ntitle: First\n---\nUpdated content"
	if err := os.WriteFile(note, []byte(updated), 0o644); err != nil {
		t.Fatalf("rewrite note: %v", err)
	}

	svc.QueueUpdate("note.md")
	if got := svc.Stats().Pending; got != 1 {
		t.Fatalf("expected pending queue size 1, got %d", got)
	}

	idx, err = svc.AcquireSnapshot()
	if err != nil {
		t.Fatalf("AcquireSnapshot with pending returned error: %v", err)
	}

	results = idx.Search(search.Query{Term: "updated"})
	if len(results) != 1 {
		t.Fatalf("expected updated content to be indexed, got %+v", results)
	}

	if got := svc.Stats().Pending; got != 0 {
		t.Fatalf("expected pending queue to be drained, got %d", got)
	}
}

func TestServiceClosePreventsSnapshots(t *testing.T) {
	dir := t.TempDir()
	_ = writeTestNote(t, dir, "note.md", "content")

	svc := NewService(dir, search.Config{})
	if err := svc.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	if _, err := svc.AcquireSnapshot(); err != ErrClosed {
		t.Fatalf("expected ErrClosed after Close, got %v", err)
	}
}
