package taskEcho

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/state"
)

func TestRun_InvalidPriorityFallsBackToLow(t *testing.T) {
	tmpDir := t.TempDir()
	taskFile := filepath.Join(tmpDir, "tasks.md")
	initialContent := "## Tasks\n### Low Priority\n### Medium Priority\n### High Priority\n"

	if err := os.WriteFile(taskFile, []byte(initialContent), 0o644); err != nil {
		t.Fatalf("failed to write initial task file: %v", err)
	}

	st := &state.State{Config: &config.Config{PinnedTaskFile: taskFile}}

	cmd := &cobra.Command{}
	cmd.Flags().StringP("name", "n", "", "")

	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = w

	runErr := run(cmd, []string{"New task"}, st, "urgent")

	w.Close()
	os.Stdout = originalStdout
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to copy stdout: %v", err)
	}
	r.Close()
	output := buf.String()

	if runErr != nil {
		t.Fatalf("run returned error: %v", runErr)
	}

	trimmedOutput := strings.TrimSpace(output)
	expectedMessage := "Task appended to the pinned task file under the \"low\" section."
	if trimmedOutput != expectedMessage {
		t.Fatalf("unexpected output message: got %q, want %q", trimmedOutput, expectedMessage)
	}

	content, err := os.ReadFile(taskFile)
	if err != nil {
		t.Fatalf("failed to read task file: %v", err)
	}
	contentStr := string(content)

	entry := "- [ ] New task\n"
	if !strings.Contains(contentStr, entry) {
		t.Fatalf("task entry %q not found in task file", entry)
	}

	lowIndex := strings.Index(contentStr, "### Low Priority")
	entryIndex := strings.Index(contentStr, entry)
	mediumIndex := strings.Index(contentStr, "### Medium Priority")

	if lowIndex == -1 || mediumIndex == -1 {
		t.Fatalf("priority sections missing from task file: low=%d medium=%d", lowIndex, mediumIndex)
	}

	if !(lowIndex < entryIndex && entryIndex < mediumIndex) {
		t.Fatalf("task entry not placed under low priority section: low=%d entry=%d medium=%d", lowIndex, entryIndex, mediumIndex)
	}
}
