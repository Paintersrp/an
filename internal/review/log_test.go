package review

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Paintersrp/an/internal/templater"
)

func TestListReviewLogsFiltersAndSorts(t *testing.T) {
	dir := t.TempDir()
	manifest := templater.TemplateManifest{Name: "review-daily"}

	// Log that should be filtered out (weekly mode).
	writeLogFile(t, dir, "review-weekly-2024-01-01.md", "## review-weekly — 2024-01-01T10:00:00Z\n\nweekly content\n")

	// Oldest daily entry with embedded timestamp.
	oldest := "## review-daily — 2024-01-01T09:00:00Z\n\n### Checklist responses\n\n- **Clear capture inbox:** done\n"
	writeLogFile(t, dir, "review-daily-2024-01-01.md", oldest)

	// Entry without timestamp should fall back to filename-derived date.
	mid := "## review-daily\n\nFollow up summary\n"
	midPath := writeLogFile(t, dir, "review-daily-2024-01-02.md", mid)

	// Manually edited entry without slug in filename but with front matter and heading.
	manual := "---\ntitle: custom\n---\n## Daily Review Ritual — 2024-01-03T08:00:00Z\n\nFirst highlight\n- bullet line\n"
	manualPath := writeLogFile(t, dir, "ritual-notes.md", manual)
	// Ensure the file's mod time differs from embedded timestamp to ensure timestamp sorting is used.
	olderMod := time.Date(2023, 12, 31, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(manualPath, olderMod, olderMod); err != nil {
		t.Fatalf("failed to set manual log mod time: %v", err)
	}

	// Entry without timestamp or dated filename should fall back to mod time.
	fallback := "Daily reflection summary\nCaptured a free-form insight.\n"
	fallbackPath := writeLogFile(t, dir, "daily-notes.md", fallback)
	fallbackTime := time.Date(2024, 1, 4, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(fallbackPath, fallbackTime, fallbackTime); err != nil {
		t.Fatalf("failed to set fallback log mod time: %v", err)
	}

	logs, err := ListReviewLogs(dir, manifest, "daily")
	if err != nil {
		t.Fatalf("ListReviewLogs returned error: %v", err)
	}

	if len(logs) != 4 {
		t.Fatalf("expected 4 daily logs, got %d", len(logs))
	}

	if logs[0].Path != fallbackPath {
		t.Fatalf("expected log without timestamp to sort by mod time first, got %q", logs[0].Path)
	}
	if !logs[0].Timestamp.Equal(fallbackTime) {
		t.Fatalf("expected fallback log timestamp to use mod time, got %v", logs[0].Timestamp)
	}

	if logs[1].Path != manualPath {
		t.Fatalf("expected manual log to be second by timestamp, got %q", logs[1].Path)
	}
	if logs[1].Title != "Daily Review Ritual" {
		t.Fatalf("expected manual log title to be derived from heading, got %q", logs[1].Title)
	}
	if len(logs[1].Preview) == 0 || logs[1].Preview[0] != "First highlight" {
		t.Fatalf("expected preview to include manual content, got %#v", logs[1].Preview)
	}
	if len(logs[1].Preview) < 2 || logs[1].Preview[1] != "- bullet line" {
		t.Fatalf("expected preview to preserve bullet formatting, got %#v", logs[1].Preview)
	}

	if logs[2].Path != midPath {
		t.Fatalf("expected filename-dated log to sort next, got %q", logs[2].Path)
	}
	expectedDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	if !logs[2].Timestamp.Equal(expectedDate) {
		t.Fatalf("expected timestamp derived from filename, got %v", logs[2].Timestamp)
	}

	if logs[3].Title == "" {
		t.Fatalf("expected oldest entry to have derived title, got empty string")
	}
	if logs[3].Timestamp.Year() != 2024 || logs[3].Timestamp.Day() != 1 {
		t.Fatalf("expected oldest entry timestamp from embedded date, got %v", logs[3].Timestamp)
	}
}

func writeLogFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write log %s: %v", name, err)
	}
	return path
}
