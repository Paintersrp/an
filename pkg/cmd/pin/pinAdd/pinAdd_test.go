package pinAdd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/pkg/shared/flags"
)

func TestRunReturnsChangePinError(t *testing.T) {
	t.Helper()

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "note.md")
	if err := os.WriteFile(tempFile, []byte("example"), 0o644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	ws := &config.Workspace{NamedPins: config.PinMap{}, NamedTaskPins: config.PinMap{}}
	cfg := &config.Config{
		Workspaces:       map[string]*config.Workspace{"default": ws},
		CurrentWorkspace: "default",
	}
	if err := cfg.ActivateWorkspace("default"); err != nil {
		t.Fatalf("failed to activate workspace: %v", err)
	}

	st := &state.State{Config: cfg, Workspace: ws, WorkspaceName: "default"}

	cmd := &cobra.Command{}
	flags.AddPath(cmd)
	flags.AddName(cmd, "")
	if err := cmd.Flags().Set("path", tempFile); err != nil {
		t.Fatalf("failed to set path flag: %v", err)
	}

	err := run(cmd, nil, st, "invalid")
	if err == nil {
		t.Fatal("expected run to return error from ChangePin, but got nil")
	}

	const expected = "invalid pin file type. Valid options are text and task"
	if err.Error() != expected {
		t.Fatalf("unexpected error. want %q, got %q", expected, err)
	}
}
