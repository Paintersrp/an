package tasks

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/handler"
	"github.com/Paintersrp/an/internal/state"
)

func TestToggleUpdatesTaskFile(t *testing.T) {
	dir := t.TempDir()
	atoms := filepath.Join(dir, "atoms")
	if err := os.MkdirAll(atoms, 0o755); err != nil {
		t.Fatalf("failed to create atoms directory: %v", err)
	}
	taskPath := filepath.Join(atoms, "tasks.md")
	if err := os.WriteFile(taskPath, []byte("- [ ] example"), 0o644); err != nil {
		t.Fatalf("failed to write tasks file: %v", err)
	}

	ws := &config.Workspace{VaultDir: dir, PinnedTaskFile: taskPath, NamedTaskPins: config.PinMap{}}
	cfg := &config.Config{Workspaces: map[string]*config.Workspace{"default": ws}, CurrentWorkspace: "default"}
	if err := cfg.ActivateWorkspace("default"); err != nil {
		t.Fatalf("failed to activate workspace: %v", err)
	}

	st := &state.State{Handler: handler.NewFileHandler(dir), Config: cfg}
	model, err := NewModel(st)
	if err != nil {
		t.Fatalf("failed to create tasks model: %v", err)
	}

	model.Init()
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.list.Select(0)
	model.handleToggle()

	data, err := os.ReadFile(taskPath)
	if err != nil {
		t.Fatalf("failed to read tasks file: %v", err)
	}
	if string(data) != "- [x] example" {
		t.Fatalf("expected task to be toggled, got %q", string(data))
	}
}
