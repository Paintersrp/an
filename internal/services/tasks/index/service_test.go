package index

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireSnapshotRebuildsTasks(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "tasks.md")
	if err := os.WriteFile(file, []byte("- [ ] first\n- [x] second"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	svc := NewService(dir)
	snapshot, err := svc.AcquireSnapshot()
	if err != nil {
		t.Fatalf("AcquireSnapshot returned error: %v", err)
	}

	tasks := snapshot.Tasks()
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	if tasks[0].Path == "" {
		t.Fatalf("expected task path to be recorded")
	}
}

func TestQueueUpdateReparsesFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "tasks.md")
	if err := os.WriteFile(file, []byte("- [ ] first"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	svc := NewService(dir)
	if _, err := svc.AcquireSnapshot(); err != nil {
		t.Fatalf("AcquireSnapshot returned error: %v", err)
	}

	if err := os.WriteFile(file, []byte("- [x] done"), 0o644); err != nil {
		t.Fatalf("rewrite file: %v", err)
	}
	svc.QueueUpdate("tasks.md")

	snapshot, err := svc.AcquireSnapshot()
	if err != nil {
		t.Fatalf("AcquireSnapshot after update returned error: %v", err)
	}

	tasks := snapshot.Tasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task after update, got %d", len(tasks))
	}
	if tasks[0].Status != "checked" {
		t.Fatalf("expected task status to update, got %q", tasks[0].Status)
	}
}

func TestQueueUpdateRemovesDeletedFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "tasks.md")
	if err := os.WriteFile(file, []byte("- [ ] first"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	svc := NewService(dir)
	if _, err := svc.AcquireSnapshot(); err != nil {
		t.Fatalf("AcquireSnapshot returned error: %v", err)
	}

	if err := os.Remove(file); err != nil {
		t.Fatalf("remove file: %v", err)
	}
	svc.QueueUpdate("tasks.md")

	snapshot, err := svc.AcquireSnapshot()
	if err != nil {
		t.Fatalf("AcquireSnapshot after delete returned error: %v", err)
	}

	if tasks := snapshot.Tasks(); len(tasks) != 0 {
		t.Fatalf("expected no tasks after deletion, got %d", len(tasks))
	}
}
