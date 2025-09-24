package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/handler"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/templater"
	journaltui "github.com/Paintersrp/an/internal/tui/journal"
	taskstui "github.com/Paintersrp/an/internal/tui/tasks"
	"github.com/Paintersrp/an/internal/views"
)

func TestRootModelNavigation(t *testing.T) {
	dir := t.TempDir()
	atoms := filepath.Join(dir, "atoms")
	if err := os.MkdirAll(atoms, 0o755); err != nil {
		t.Fatalf("failed to create atoms directory: %v", err)
	}

	notePath := filepath.Join(atoms, "note.md")
	if err := os.WriteFile(notePath, []byte("---\ntitle: test\n---\n"), 0o644); err != nil {
		t.Fatalf("failed to write note: %v", err)
	}

	tasksPath := filepath.Join(atoms, "tasks.md")
	if err := os.WriteFile(tasksPath, []byte("- [ ] task"), 0o644); err != nil {
		t.Fatalf("failed to write tasks file: %v", err)
	}

	journalPath := filepath.Join(atoms, "day-20240101.md")
	if err := os.WriteFile(journalPath, []byte("journal"), 0o644); err != nil {
		t.Fatalf("failed to write journal file: %v", err)
	}

	ws := &config.Workspace{
		VaultDir:       dir,
		PinnedTaskFile: tasksPath,
		NamedTaskPins:  config.PinMap{},
	}
	cfg := &config.Config{
		Workspaces:       map[string]*config.Workspace{"default": ws},
		CurrentWorkspace: "default",
	}
	if err := cfg.ActivateWorkspace("default"); err != nil {
		t.Fatalf("failed to activate workspace: %v", err)
	}

	handler := handler.NewFileHandler(dir)
	templ, err := templater.NewTemplater(ws)
	if err != nil {
		t.Fatalf("failed to create templater: %v", err)
	}
	viewManager, err := views.NewViewManager(handler, cfg)
	if err != nil {
		t.Fatalf("failed to create view manager: %v", err)
	}

	st := &state.State{
		Config:        cfg,
		Workspace:     ws,
		WorkspaceName: "default",
		Templater:     templ,
		Handler:       handler,
		ViewManager:   viewManager,
		Views:         viewManager.Views,
		Vault:         dir,
	}

	noteModel, err := NewNoteListModel(st, "default")
	if err != nil {
		t.Fatalf("failed to create note model: %v", err)
	}
	tasksModel, err := taskstui.NewModel(st)
	if err != nil {
		t.Fatalf("failed to create tasks model: %v", err)
	}
	journalModel, err := journaltui.NewModel(st)
	if err != nil {
		t.Fatalf("failed to create journal model: %v", err)
	}

	root := NewRootModel(noteModel, tasksModel, journalModel)
	root.Init()
	root.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	if !strings.Contains(root.View(), "[1. Notes]") {
		t.Fatalf("expected notes view to be active")
	}

	root.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if root.active != viewTasks {
		t.Fatalf("expected tasks view after ctrl+2, got %v", root.active)
	}
	if !strings.Contains(root.View(), "Pinned:") {
		t.Fatalf("expected tasks view to render pinned status")
	}

	root.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	if root.active != viewJournal {
		t.Fatalf("expected journal view after ctrl+3, got %v", root.active)
	}
	if !strings.Contains(root.View(), "Journal") {
		t.Fatalf("expected journal view content in output")
	}
}
