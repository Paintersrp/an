package review

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Paintersrp/an/internal/search"
)

func TestBuildResurfaceQueueAppliesBuckets(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	now := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)

	stale := writeNote(t, dir, "stale.md", now.Add(-10*24*time.Hour))
	mid := writeNote(t, dir, "mid.md", now.Add(-4*24*time.Hour))
	fresh := writeNote(t, dir, "fresh.md", now.Add(-6*time.Hour))

	idx := search.NewIndex(dir, search.Config{})
	if err := idx.Build([]string{stale, mid, fresh}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	queue := BuildResurfaceQueue(idx, ResurfaceOptions{
		Now: now,
		Buckets: []Bucket{
			{Name: "daily", After: 24 * time.Hour},
			{Name: "weekly", After: 7 * 24 * time.Hour},
		},
	})

	if len(queue) != 2 {
		t.Fatalf("expected 2 resurfaced notes, got %d", len(queue))
	}

	if queue[0].Path != filepath.Clean(stale) || queue[0].Bucket != "weekly" {
		t.Fatalf("unexpected first queue item: %+v", queue[0])
	}
	if queue[1].Path != filepath.Clean(mid) || queue[1].Bucket != "daily" {
		t.Fatalf("unexpected second queue item: %+v", queue[1])
	}
}

func TestFilterQueueFiltersByTagsAndMetadata(t *testing.T) {
	t.Parallel()

	now := time.Now()
	queue := []ResurfaceItem{
		{
			Path:       "a.md",
			Tags:       []string{"weekly", "project"},
			Metadata:   map[string][]string{"status": []string{"active"}},
			ModifiedAt: now.Add(-48 * time.Hour),
			Age:        48 * time.Hour,
			Bucket:     "daily",
		},
		{
			Path:       "b.md",
			Tags:       []string{"daily"},
			Metadata:   map[string][]string{"status": []string{"paused"}},
			ModifiedAt: now.Add(-120 * time.Hour),
			Age:        120 * time.Hour,
			Bucket:     "weekly",
		},
	}

	filtered := FilterQueue(queue, []string{"project"}, map[string][]string{"status": []string{"active"}})
	if len(filtered) != 1 || filtered[0].Path != "a.md" {
		t.Fatalf("unexpected filtered queue: %+v", filtered)
	}
}

func writeNote(t *testing.T, dir, name string, modTime time.Time) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("---\ntitle: Note\n---\nbody"), 0o644); err != nil {
		t.Fatalf("write note: %v", err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	return path
}
