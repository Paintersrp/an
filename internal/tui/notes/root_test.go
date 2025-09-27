package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

	view := root.View()
	for _, want := range []string{"alt+1 Notes", "alt+2 Tasks", "alt+3 Journal"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected header to include %q, got %q", want, view)
		}
	}
	if !strings.Contains(view, "[alt+1 Notes]") {
		t.Fatalf("expected notes view to be active")
	}

	root.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}, Alt: true})
	if root.active != viewTasks {
		t.Fatalf("expected tasks view after alt+2, got %v", root.active)
	}
	view = root.View()
	if !strings.Contains(view, "Pinned:") {
		t.Fatalf("expected tasks view to render pinned status")
	}
	if !strings.Contains(view, "[alt+2 Tasks]") {
		t.Fatalf("expected tasks shortcut to be highlighted in header: %q", view)
	}

	root.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}, Alt: true})
	if root.active != viewJournal {
		t.Fatalf("expected journal view after alt+3, got %v", root.active)
	}
	view = root.View()
	if !strings.Contains(view, "Journal") {
		t.Fatalf("expected journal view content in output")
	}
	if !strings.Contains(view, "[alt+3 Journal]") {
		t.Fatalf("expected journal shortcut to be highlighted in header: %q", view)
	}

	root.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}, Alt: true})
	if root.active != viewNotes {
		t.Fatalf("expected notes view after alt+1 chord, got %v", root.active)
	}
}

func TestRootModelKeepsNotesViewWhenEditorActive(t *testing.T) {
	noteModel := newEditorTestModel(t, map[string]string{})
	if noteModel == nil {
		t.Fatalf("expected note model")
	}

	root := NewRootModel(noteModel, nil, nil)

	_ = noteModel.startScratchCapture()
	if !noteModel.editorActive() {
		t.Fatalf("expected scratch editor to be active")
	}

	_, _ = root.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}, Alt: true})

	if root.active != viewNotes {
		t.Fatalf("expected to remain on notes view, got %v", root.active)
	}

	if got := noteModel.editor.value(); got != "2" {
		t.Fatalf("expected editor to capture input, got %q", got)
	}
}

func TestRootModelViewHeightMatchesWindowSize(t *testing.T) {
	noteModel := newEditorTestModel(t, map[string]string{"note.md": "content"})
	root := NewRootModel(noteModel, nil, nil)
	root.Init()

	const height = 12

	root.Update(tea.WindowSizeMsg{Width: 80, Height: height})

	view := root.View()
	lines := strings.Split(view, "\n")

	if len(lines) != height {
		t.Fatalf(
			"expected %d lines in view, got %d (note height %d, note view lines %d):\n%s",
			height,
			len(lines),
			root.notes.height,
			lipgloss.Height(root.notes.View()),
			view,
		)
	}

	if len(lines) == 0 || !strings.Contains(lines[0], "Views:") {
		t.Fatalf("expected header to be visible in view, got %q", view)
	}
}

func TestRootViewFillsFrame(t *testing.T) {
	root := &RootModel{active: viewNotes}
	root.width = 10
	root.height = 5

	view := root.View()
	lines := strings.Split(view, "\n")

	if len(lines) != 5 {
		t.Fatalf("expected view to render 5 lines, got %d", len(lines))
	}

	for i, line := range lines {
		if width := lipgloss.Width(line); width < 10 {
			t.Fatalf("line %d width mismatch: want at least 10, got %d", i, width)
		}
	}
}

func TestPadFrame(t *testing.T) {
	cases := []struct {
		name    string
		content string
		width   int
		height  int
		want    []string
	}{
		{
			name:    "no padding when tall",
			content: "a\nb\nc",
			width:   1,
			height:  2,
			want:    []string{"a", "b", "c"},
		},
		{
			name:    "pads shorter content",
			content: "a\nb",
			width:   3,
			height:  5,
			want: []string{
				"a  ",
				"b  ",
				"   ",
				"   ",
				"   ",
			},
		},
		{
			name:    "handles empty",
			content: "",
			width:   4,
			height:  3,
			want: []string{
				"    ",
				"    ",
				"    ",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := padFrame(tc.content, tc.width, tc.height)
			lines := strings.Split(got, "\n")
			if len(lines) != len(tc.want) {
				t.Fatalf("expected %d lines, got %d", len(tc.want), len(lines))
			}
			for i := range lines {
				if lines[i] != tc.want[i] {
					t.Fatalf("line %d mismatch: want %q, got %q", i, tc.want[i], lines[i])
				}
			}
		})
	}
}
