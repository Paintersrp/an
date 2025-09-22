package taskNewEchoFile

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestFindHighestIncrement(t *testing.T) {
	t.Parallel()

	vaultDir := t.TempDir()
	subDir := "tasks"
	subDirPath := filepath.Join(vaultDir, subDir)

	if err := os.MkdirAll(subDirPath, 0o755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	for i := 1; i <= 2; i++ {
		filename := fmt.Sprintf("task-echo-%02d.md", i)
		if err := os.WriteFile(filepath.Join(subDirPath, filename), []byte(""), 0o644); err != nil {
			t.Fatalf("failed to seed test file %s: %v", filename, err)
		}
	}

	highest := findHighestIncrement(vaultDir, subDir)
	if highest != 2 {
		t.Fatalf("expected highest increment of 2, got %d", highest)
	}

	nextTitle := fmt.Sprintf("task-echo-%02d", highest+1)
	if nextTitle != "task-echo-03" {
		t.Fatalf("expected next title to be task-echo-03, got %s", nextTitle)
	}
}

func TestFindHighestIncrementNoMatches(t *testing.T) {
	t.Parallel()

	vaultDir := t.TempDir()
	subDir := "tasks"
	subDirPath := filepath.Join(vaultDir, subDir)

	if err := os.MkdirAll(subDirPath, 0o755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	highest := findHighestIncrement(vaultDir, subDir)
	if highest != 0 {
		t.Fatalf("expected highest increment of 0, got %d", highest)
	}
}
