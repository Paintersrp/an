package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParserWalkExtractsTasksAndTags(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "note.md")
	content := `# Title
- [ ] first task
- [x] completed task
- [ ]    
tags:
- project/foo
- project/foo
- weekly
`

	if err := os.WriteFile(notePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write note: %v", err)
	}

	p := NewParser(dir)
	if err := p.Walk(); err != nil {
		t.Fatalf("Walk returned error: %v", err)
	}

	var (
		foundUnchecked bool
		foundChecked   bool
	)

	for _, task := range p.TaskHandler.Tasks {
		switch task.Content {
		case "first task":
			if task.Status != "unchecked" {
				t.Fatalf("expected first task to be unchecked, got %q", task.Status)
			}
			foundUnchecked = true
		case "completed task":
			if task.Status != "checked" {
				t.Fatalf("expected completed task to be checked, got %q", task.Status)
			}
			foundChecked = true
		}
	}

	if !foundUnchecked || !foundChecked {
		t.Fatalf("expected to find both tracked tasks, got %#v", p.TaskHandler.Tasks)
	}

	if got := p.TagHandler.TagCounts["project/foo"]; got != 2 {
		t.Fatalf("expected tag 'project/foo' to be counted twice, got %d", got)
	}

	if got := p.TagHandler.TagCounts["weekly"]; got != 1 {
		t.Fatalf("expected tag 'weekly' to be counted once, got %d", got)
	}

	if len(p.TagHandler.TagList) != 2 {
		t.Fatalf("expected TagList to only contain unique tags, got %#v", p.TagHandler.TagList)
	}
}

func TestTaskHandlerParseTaskIgnoresEmptyContent(t *testing.T) {
	handler := NewTaskHandler()
	handler.ParseTask("[ ]   ")

	if len(handler.Tasks) != 0 {
		t.Fatalf("expected no tasks to be added for empty content, got %#v", handler.Tasks)
	}
}

func TestTagHandlerParseTagIncrementsCounts(t *testing.T) {
	handler := NewTagHandler()
	handler.ParseTag("project/foo")
	handler.ParseTag("project/foo")

	if handler.TagCounts["project/foo"] != 2 {
		t.Fatalf("expected tag count to increment to 2, got %d", handler.TagCounts["project/foo"])
	}

	if len(handler.TagList) != 1 {
		t.Fatalf("expected TagList to deduplicate entries, got %#v", handler.TagList)
	}
}
