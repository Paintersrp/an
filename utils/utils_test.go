package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderMarkdownPreview_AppliesWrapWidth(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")

	markdown := `---
title: Example Note
---

This is a sentence with enough words to require wrapping when rendered into a preview panel.
`

	if err := os.WriteFile(path, []byte(markdown), 0o600); err != nil {
		t.Fatalf("failed to write markdown: %v", err)
	}

	const previewWidth = 20

	rendered, _ := RenderMarkdownPreview(path, previewWidth, 0, 0)

	wrapWidth := previewWidth - previewHorizontalSpace
	if wrapWidth <= 0 {
		wrapWidth = defaultWrapWidth
	}

	for i, line := range strings.Split(rendered, "\n") {
		trimmed := strings.TrimRight(line, " ")
		if trimmed == "" {
			continue
		}

		if width := lipgloss.Width(trimmed); width > wrapWidth {
			t.Fatalf("line %d exceeds wrap width: got %d, want <= %d: %q", i, width, wrapWidth, trimmed)
		}
	}
}
