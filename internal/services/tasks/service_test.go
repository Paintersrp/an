package tasks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Paintersrp/an/internal/handler"
	taskindex "github.com/Paintersrp/an/internal/services/tasks/index"
)

func TestServiceListIncludesPathAndStatus(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "atoms", "example.md")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	content := "- [ ] first task @due(2024-01-10) @owner(Alice)\n- [x] done task @priority(low)\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	svc := NewService(handler.NewFileHandler(dir), taskindex.NewService(dir))
	tasks, err := svc.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("expected two tasks, got %d", len(tasks))
	}

	if tasks[0].Path == "" || tasks[0].Line == 0 {
		t.Fatalf("expected task metadata to include path and line, got %#v", tasks[0])
	}

	if tasks[1].Completed != true {
		t.Fatalf("expected completed task to be marked complete")
	}
	if tasks[0].Due == nil {
		t.Fatalf("expected due date metadata to be carried to service item")
	}
	if tasks[0].Owner != "Alice" {
		t.Fatalf("expected owner metadata to be carried to service item, got %q", tasks[0].Owner)
	}
	if tasks[1].Priority != "low" {
		t.Fatalf("expected priority metadata to be carried to service item, got %q", tasks[1].Priority)
	}
}

func TestServiceToggleFlipsCompletion(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "atoms", "example.md")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	content := "- [ ] first task\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	svc := NewService(handler.NewFileHandler(dir), nil)
	completed, err := svc.Toggle(file, 1)
	if err != nil {
		t.Fatalf("Toggle returned error: %v", err)
	}
	if !completed {
		t.Fatalf("expected task to be marked complete after toggle")
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != "- [x] first task\n" {
		t.Fatalf("expected task to be toggled, got %q", string(data))
	}
}

func TestServiceToggleRefreshesIndex(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "atoms", "example.md")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	content := "- [ ] first task\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	idx := taskindex.NewService(dir)
	t.Cleanup(func() {
		_ = idx.Close()
	})

	svc := NewService(handler.NewFileHandler(dir), idx)

	tasks, err := svc.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one task, got %d", len(tasks))
	}
	if tasks[0].Completed {
		t.Fatalf("expected task to start incomplete")
	}

	completed, err := svc.Toggle(file, 1)
	if err != nil {
		t.Fatalf("Toggle returned error: %v", err)
	}
	if !completed {
		t.Fatalf("expected toggle to return completed state")
	}

	tasks, err = svc.List()
	if err != nil {
		t.Fatalf("List returned error after toggle: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one task after toggle, got %d", len(tasks))
	}
	if !tasks[0].Completed {
		t.Fatalf("expected task to be marked complete after toggle")
	}
}
