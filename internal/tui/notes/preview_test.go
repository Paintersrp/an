package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Paintersrp/an/internal/search"
)

func TestFormatPreviewContextEmpty(t *testing.T) {
	t.Parallel()

	ctx := previewContext{}
	got := formatPreviewContext(ctx, "")
	want := "Links: 0 outbound · 0 backlinks"
	if got != want {
		t.Fatalf("unexpected summary: got %q, want %q", got, want)
	}
}

func TestFormatPreviewContextLargeList(t *testing.T) {
	t.Parallel()

	ctx := previewContext{
		Outbound: []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine"},
	}

	summary := formatPreviewContext(ctx, "")
	if !strings.Contains(summary, "Links: 9 outbound · 0 backlinks") {
		t.Fatalf("summary missing counts: %q", summary)
	}
	if !strings.Contains(summary, "one") || !strings.Contains(summary, "eight") {
		t.Fatalf("expected early entries to be listed: %q", summary)
	}
	if strings.Contains(summary, "nine") {
		t.Fatalf("expected trailing entries to be truncated: %q", summary)
	}
	if !strings.Contains(summary, "… and 1 more") {
		t.Fatalf("expected truncation notice, got %q", summary)
	}
}

func TestBuildPreviewContextResolvesAlias(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	alpha := filepath.Join(tempDir, "Alpha.md")
	bravo := filepath.Join(tempDir, "bravo.md")

	if err := os.WriteFile(alpha, []byte("[[Bravo]]"), 0o644); err != nil {
		t.Fatalf("failed to write alpha note: %v", err)
	}
	if err := os.WriteFile(bravo, []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to write bravo note: %v", err)
	}

	idx := search.NewIndex(tempDir, search.Config{})
	if err := idx.Build([]string{alpha, bravo}); err != nil {
		t.Fatalf("build index: %v", err)
	}

	ctx := buildPreviewContext("Bravo", idx, nil, nil)
	if ctx.Path != filepath.Clean(bravo) {
		t.Fatalf("expected canonical path %q, got %q", filepath.Clean(bravo), ctx.Path)
	}
	if len(ctx.Backlinks) != 1 {
		t.Fatalf("expected one backlink, got %d", len(ctx.Backlinks))
	}

	summary := formatPreviewContext(ctx, tempDir)
	if !strings.Contains(summary, "Alpha.md") {
		t.Fatalf("expected relative backlink path in summary: %q", summary)
	}
	if !strings.Contains(summary, "Links: 0 outbound · 1 backlinks") {
		t.Fatalf("expected backlink count in summary: %q", summary)
	}
}
