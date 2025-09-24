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
- [ ] first task @due(2024-05-20) @owner(Alice) [[Project Hub]]
- [x] completed task @priority(high)
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
			if task.Metadata.Owner != "Alice" {
				t.Fatalf("expected owner metadata to be Alice, got %q", task.Metadata.Owner)
			}
			if task.Metadata.DueDate == nil {
				t.Fatalf("expected due date metadata to be captured")
			}
			if len(task.Metadata.References) != 1 || task.Metadata.References[0] != "Project Hub" {
				t.Fatalf("expected backlink metadata to be captured, got %#v", task.Metadata.References)
			}
		case "completed task":
			if task.Status != "checked" {
				t.Fatalf("expected completed task to be checked, got %q", task.Status)
			}
			foundChecked = true
			if task.Metadata.Priority != "high" {
				t.Fatalf("expected completed task priority metadata, got %#v", task.Metadata)
			}
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
	handler.ParseTask("[ ]   ", "", 0)

	if len(handler.Tasks) != 0 {
		t.Fatalf("expected no tasks to be added for empty content, got %#v", handler.Tasks)
	}
}

func TestTaskHandlerRecordsPathAndLine(t *testing.T) {
	handler := NewTaskHandler()
	handler.ParseTask("[ ] example", "/tmp/note.md", 42)

	if len(handler.Tasks) != 1 {
		t.Fatalf("expected a single task to be recorded, got %d", len(handler.Tasks))
	}

	task := handler.Tasks[1]
	if task.Path != "/tmp/note.md" {
		t.Fatalf("expected path to be recorded, got %q", task.Path)
	}
	if task.Line != 42 {
		t.Fatalf("expected line number 42, got %d", task.Line)
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
