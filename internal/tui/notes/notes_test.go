package notes

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/handler"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/views"
)

func TestCycleViewOrder(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	fileHandler := handler.NewFileHandler(tempDir)
	ws := &config.Workspace{VaultDir: tempDir}
	cfg := &config.Config{
		Workspaces:       map[string]*config.Workspace{"default": ws},
		CurrentWorkspace: "default",
	}
	if err := cfg.ActivateWorkspace("default"); err != nil {
		t.Fatalf("failed to activate workspace: %v", err)
	}
	viewManager, err := views.NewViewManager(fileHandler, cfg)
	if err != nil {
		t.Fatalf("NewViewManager returned error: %v", err)
	}

	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 0, 0)

	model := &NoteListModel{
		list:       l,
		state:      &state.State{Config: cfg, Workspace: ws, WorkspaceName: cfg.CurrentWorkspace, Handler: fileHandler, ViewManager: viewManager, Vault: tempDir},
		viewName:   "default",
		sortField:  sortByTitle,
		sortOrder:  ascending,
		highlights: newHighlightStore(),
	}

	expectedOrder := []string{"unfulfilled", "archive", "orphan", "trash", "default"}

	for i, want := range expectedOrder {
		model.cycleView()
		if got := model.viewName; got != want {
			t.Fatalf("step %d: expected view %q, got %q", i+1, want, got)
		}
	}
}

func TestEnsureSelectionInBoundsResetsOutOfRangeCursor(t *testing.T) {
	t.Parallel()

	delegate := list.NewDefaultDelegate()
	items := []list.Item{
		ListItem{fileName: "one.md", path: "one"},
		ListItem{fileName: "two.md", path: "two"},
		ListItem{fileName: "three.md", path: "three"},
	}

	l := list.New(items, delegate, 0, 0)
	l.SetSize(80, 30)
	l.Select(2)

	model := &NoteListModel{list: l}

	reduced := []list.Item{items[0]}
	model.list.SetItems(reduced)

	if idx := model.list.Index(); idx == 0 {
		t.Fatalf("expected index to remain out of bounds before enforcing selection, got %d", idx)
	}

	model.ensureSelectionInBounds()

	if idx := model.list.Index(); idx != 0 {
		t.Fatalf("expected index to reset to 0, got %d", idx)
	}

	if _, ok := model.list.SelectedItem().(ListItem); !ok {
		t.Fatalf("expected a selected item after resetting selection")
	}
}

func TestToggleRenameSeedsInputValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		item ListItem
		want string
	}{
		{
			name: "with title",
			item: ListItem{title: "Front Matter Title", fileName: "front-matter-title.md"},
			want: "Front Matter Title",
		},
		{
			name: "without title",
			item: ListItem{title: "", fileName: "plain-note.md"},
			want: "plain-note",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			model := newTestNoteListModel(t, tc.item, "")

			model.toggleRename()

			if !model.renaming {
				t.Fatalf("expected renaming to be true")
			}

			if got := model.inputModel.Input.Value(); got != tc.want {
				t.Fatalf("expected input value %q, got %q", tc.want, got)
			}
		})
	}
}

func TestToggleCopySeedsInputValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		item ListItem
		want string
	}{
		{
			name: "with title",
			item: ListItem{title: "Front Matter Title", fileName: "front-matter-title.md"},
			want: "Front Matter Title-copy",
		},
		{
			name: "without title",
			item: ListItem{title: "", fileName: "plain-note.md"},
			want: "plain-note-copy",
		},
		{
			name: "already suffixed",
			item: ListItem{title: "Front Matter Title-copy", fileName: "front-matter-title-copy.md"},
			want: "Front Matter Title-copy",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			model := newTestNoteListModel(t, tc.item, "")

			model.toggleCopy()

			if !model.copying {
				t.Fatalf("expected copying to be true")
			}

			if got := model.inputModel.Input.Value(); got != tc.want {
				t.Fatalf("expected input value %q, got %q", tc.want, got)
			}
		})
	}
}
