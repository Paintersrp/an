package review

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paintersrp/an/internal/config"
	reviewsvc "github.com/Paintersrp/an/internal/review"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/templater"
)

func TestCompleteConfirmationCancel(t *testing.T) {
	t.Helper()
	tempDir := t.TempDir()
	st := newTestState(t, tempDir)

	model, err := NewModel(st)
	if err != nil {
		t.Fatalf("NewModel returned error: %v", err)
	}

	model.editor.SetValue("draft response")

	updated, cmd := model.Update(ctrlEnterMsg())
	if cmd != nil {
		t.Fatalf("expected no command after first confirmation, got %T", cmd)
	}
	m := adoptTestModel(updated)
	if !m.confirmingSave {
		t.Fatalf("expected model to enter confirmation state")
	}
	if !strings.Contains(m.status, "ctrl+enter") {
		t.Fatalf("expected confirmation status message, got %q", m.status)
	}

	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Fatalf("expected no command after cancel, got %T", cmd)
	}
	m = adoptTestModel(updated)
	if m.confirmingSave {
		t.Fatalf("expected confirmation to be cleared after cancel")
	}
	if !strings.Contains(strings.ToLower(m.status), "canceled") {
		t.Fatalf("expected cancel status message, got %q", m.status)
	}

	reviewDir := filepath.Join(tempDir, ".an", "review")
	if _, err := os.Stat(reviewDir); !os.IsNotExist(err) {
		entries, err := os.ReadDir(reviewDir)
		if err != nil {
			t.Fatalf("failed to inspect review directory: %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("expected review directory to remain empty on cancel, found %d entries", len(entries))
		}
	}
}

func TestCompleteConfirmationSavesLog(t *testing.T) {
	tempDir := t.TempDir()
	st := newTestState(t, tempDir)

	prevOpen := openReviewNote
	defer func() {
		openReviewNote = prevOpen
	}()

	var openedPath string
	openReviewNote = func(path string, _ bool) error {
		openedPath = path
		return nil
	}

	model, err := NewModel(st)
	if err != nil {
		t.Fatalf("NewModel returned error: %v", err)
	}

	model.editor.SetValue("completed checklist")
	model.queue = []reviewsvc.ResurfaceItem{
		{
			Path:       filepath.Join(tempDir, "atoms", "sample.md"),
			ModifiedAt: time.Now().Add(-48 * time.Hour),
			Bucket:     "weekly",
		},
	}

	updated, cmd := model.Update(ctrlEnterMsg())
	if cmd != nil {
		t.Fatalf("expected no command after first confirmation, got %T", cmd)
	}
	m := adoptTestModel(updated)

	updated, cmd = m.Update(ctrlEnterMsg())
	if cmd == nil {
		t.Fatal("expected save command after confirmation")
	}
	m = adoptTestModel(updated)
	if status := strings.ToLower(m.status); !strings.Contains(status, "saving") {
		t.Fatalf("expected saving status, got %q", m.status)
	}

	msg := cmd()
	updated, _ = m.Update(msg)
	m = adoptTestModel(updated)

	if openedPath == "" {
		t.Fatalf("expected review note to be opened")
	}
	if _, err := os.Stat(openedPath); err != nil {
		t.Fatalf("expected review log to exist: %v", err)
	}
	content, err := os.ReadFile(openedPath)
	if err != nil {
		t.Fatalf("failed to read review log: %v", err)
	}
	if !strings.Contains(string(content), "Checklist responses") {
		t.Fatalf("expected checklist section in log: %s", string(content))
	}
	if !strings.Contains(string(content), "Resurfacing queue") {
		t.Fatalf("expected queue section in log: %s", string(content))
	}
	if !strings.Contains(m.status, filepath.Base(openedPath)) {
		t.Fatalf("expected status to include saved filename, got %q", m.status)
	}
	if m.confirmingSave {
		t.Fatalf("expected confirmation state to be cleared after save")
	}
}

func newTestState(t *testing.T, vault string) *state.State {
	t.Helper()

	ws := &config.Workspace{
		VaultDir:       vault,
		Editor:         "nvim",
		FileSystemMode: "strict",
	}
	cfg := &config.Config{
		Workspaces:       map[string]*config.Workspace{"test": ws},
		CurrentWorkspace: "test",
	}
	if err := cfg.ActivateWorkspace("test"); err != nil {
		t.Fatalf("failed to activate workspace: %v", err)
	}

	tmpl, err := templater.NewTemplater(ws)
	if err != nil {
		t.Fatalf("failed to create templater: %v", err)
	}

	return &state.State{
		Config:    cfg,
		Workspace: ws,
		Templater: tmpl,
		Vault:     ws.VaultDir,
	}
}

func adoptTestModel(model tea.Model) *Model {
	m, ok := model.(*Model)
	if !ok {
		panic("unexpected model type")
	}
	return m
}

func ctrlEnterMsg() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ctrl+enter")}
}
