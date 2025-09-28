package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/handler"
	"github.com/Paintersrp/an/internal/search"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/views"
)

func newEditorTestModel(t *testing.T, files map[string]string) *NoteListModel {
	t.Helper()

	vault := t.TempDir()
	for name, content := range files {
		path := filepath.Join(vault, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("failed to create directory for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write note %s: %v", name, err)
		}
	}

	captureDir := filepath.Join(vault, "captures")
	if err := os.MkdirAll(captureDir, 0o755); err != nil {
		t.Fatalf("failed to create capture directory: %v", err)
	}

	cfg := &config.Config{
		Workspaces: map[string]*config.Workspace{
			"default": {
				VaultDir: vault,
				Editor:   "nvim",
				SubDirs:  []string{"captures"},
			},
		},
		CurrentWorkspace: "default",
	}

	activateWorkspace(t, cfg, "default")

	fileHandler := handler.NewFileHandler(vault)
	viewManager, err := views.NewViewManager(fileHandler, cfg)
	if err != nil {
		t.Fatalf("failed to create view manager: %v", err)
	}

	st := &state.State{
		Config:        cfg,
		Workspace:     cfg.MustWorkspace(),
		WorkspaceName: "default",
		Handler:       fileHandler,
		ViewManager:   viewManager,
		Vault:         vault,
	}

	paths := make([]string, 0, len(files))
	for name := range files {
		paths = append(paths, filepath.Join(vault, name))
	}
	searchCfg := search.Config{}
	if ws := cfg.MustWorkspace(); ws != nil {
		searchCfg.EnableBody = ws.Search.EnableBody
		searchCfg.IgnoredFolders = append([]string(nil), ws.Search.IgnoredFolders...)
	}
	idx := search.NewIndex(vault, searchCfg)
	if err := idx.Build(paths); err != nil {
		t.Fatalf("failed to build search index: %v", err)
	}
	st.Index = &stubIndexService{idx: idx}

	model, err := NewNoteListModel(st, "default")
	if err != nil {
		t.Fatalf("failed to create note list model: %v", err)
	}

	model.width = 100
	model.height = 40
	model.list.SetShowStatusBar(false)
	if len(model.list.Items()) > 0 {
		model.list.Select(0)
	}

	return model
}

func TestStartInlineEditLoadsContent(t *testing.T) {
	model := newEditorTestModel(t, map[string]string{"note.md": "original content"})

	_ = model.startInlineEdit()

	if model.editor == nil {
		t.Fatalf("expected editor session to be active")
	}

	if got := model.editor.value(); got != "original content" {
		t.Fatalf("unexpected editor contents: %q", got)
	}
}

func TestSaveExistingEditorDetectsConflict(t *testing.T) {
	model := newEditorTestModel(t, map[string]string{"note.md": "original"})
	_ = model.startInlineEdit()

	if model.editor == nil {
		t.Fatalf("expected editor session to be active")
	}

	path := model.editor.path
	model.editor.setValue("updated")

	time.Sleep(time.Second)
	if err := os.WriteFile(path, []byte("external"), 0o644); err != nil {
		t.Fatalf("failed to write external change: %v", err)
	}

	_ = model.saveEditor()

	if model.editor == nil {
		t.Fatalf("expected editor to remain open after conflict")
	}

	if !model.editor.allowOverwrite {
		t.Fatalf("expected allowOverwrite flag to be set")
	}

	if !strings.Contains(model.editor.status, "External changes") {
		t.Fatalf("expected conflict status message, got %q", model.editor.status)
	}
}

func TestSaveExistingEditorWritesFile(t *testing.T) {
	model := newEditorTestModel(t, map[string]string{"note.md": "original"})
	_ = model.startInlineEdit()

	path := model.editor.path
	model.editor.setValue("updated")

	_ = model.saveEditor()

	if model.editor != nil {
		t.Fatalf("expected editor session to close after save")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read note after save: %v", err)
	}

	if string(data) != "updated" {
		t.Fatalf("expected file to contain updated content, got %q", string(data))
	}
}

func TestSaveExistingEditorHandlesWriteError(t *testing.T) {
	model := newEditorTestModel(t, map[string]string{"note.md": "original"})
	_ = model.startInlineEdit()

	model.editor.setValue("updated")
	model.state.Handler = nil

	_ = model.saveEditor()

	if model.editor == nil {
		t.Fatalf("expected editor to remain open after error")
	}

	if !strings.Contains(model.editor.status, "File handler unavailable") {
		t.Fatalf("expected handler error message, got %q", model.editor.status)
	}
}

func TestQuickCaptureCreatesFile(t *testing.T) {
	model := newEditorTestModel(t, map[string]string{})
	_ = model.startScratchCapture()

	if model.editor == nil {
		t.Fatalf("expected scratch editor to be active")
	}

	path := model.editor.path
	model.editor.setValue("scratch body")

	_ = model.saveEditor()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected scratch file to exist: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read scratch file: %v", err)
	}

	if string(data) != "scratch body" {
		t.Fatalf("unexpected scratch file contents: %q", string(data))
	}
}

func TestScratchCaptureViewUpdatesWithInput(t *testing.T) {
	model := newEditorTestModel(t, map[string]string{})
	_ = model.startScratchCapture()

	if model.editor == nil {
		t.Fatalf("expected scratch editor to be active")
	}

	_, cmd := model.handleEditorUpdate(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		_ = cmd()
	}

	view := model.View()
	if !strings.Contains(view, "a") {
		t.Fatalf("expected scratch view to contain input, got %q", view)
	}
}
