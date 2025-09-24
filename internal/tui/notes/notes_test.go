package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestRefreshItemsClampsSelectionWhenListShrinks(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	fileNames := []string{"one.md", "two.md", "three.md"}
	for _, name := range fileNames {
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
			t.Fatalf("failed to write file %s: %v", name, err)
		}
	}

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

	model, err := NewNoteListModel(&state.State{
		Config:        cfg,
		Workspace:     ws,
		WorkspaceName: cfg.CurrentWorkspace,
		Handler:       fileHandler,
		ViewManager:   viewManager,
		Vault:         tempDir,
	}, "default")
	if err != nil {
		t.Fatalf("NewNoteListModel returned error: %v", err)
	}

	if items := len(model.list.Items()); items != len(fileNames) {
		t.Fatalf("expected %d items, got %d", len(fileNames), items)
	}

	model.list.Select(2)

	removed := filepath.Join(tempDir, "three.md")
	if err := os.Remove(removed); err != nil {
		t.Fatalf("failed to remove %s: %v", removed, err)
	}

	if cmd := model.refreshItems(); cmd != nil {
		t.Fatalf("expected refreshItems command to be nil, got %T", cmd)
	}

	visible := model.list.VisibleItems()
	if idx := model.list.Index(); idx < 0 || idx >= len(visible) {
		t.Fatalf("expected selection to be within bounds, got index %d with %d visible items", idx, len(visible))
	}

	if _, ok := model.list.SelectedItem().(ListItem); !ok {
		t.Fatalf("expected a selected item after refreshing list")
	}
}

func TestSearchFilterIncludesBodyMatches(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	matchingBody := []byte("---\ntitle: Body Note\n---\nthis searchterm appears in the body\n")
	nonMatching := []byte("---\ntitle: Plain Note\n---\njust other content\n")

	bodyPath := filepath.Join(tempDir, "body.md")
	otherPath := filepath.Join(tempDir, "other.md")
	if err := os.WriteFile(bodyPath, matchingBody, 0o644); err != nil {
		t.Fatalf("failed to write body note: %v", err)
	}
	if err := os.WriteFile(otherPath, nonMatching, 0o644); err != nil {
		t.Fatalf("failed to write other note: %v", err)
	}

	fileHandler := handler.NewFileHandler(tempDir)
	ws := &config.Workspace{VaultDir: tempDir, Search: config.SearchConfig{EnableBody: true}}
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

	model, err := NewNoteListModel(&state.State{
		Config:        cfg,
		Workspace:     ws,
		WorkspaceName: cfg.CurrentWorkspace,
		Handler:       fileHandler,
		ViewManager:   viewManager,
		Vault:         tempDir,
	}, "default")
	if err != nil {
		t.Fatalf("NewNoteListModel returned error: %v", err)
	}

	items := model.list.Items()
	targets := make([]string, len(items))
	for i, item := range items {
		targets[i] = item.FilterValue()
	}

	ranks := model.makeFilterFunc()("searchterm", targets)
	if len(ranks) == 0 {
		t.Fatalf("expected at least one rank for body search")
	}

	found := false
	for _, rank := range ranks {
		item, ok := items[rank.Index].(ListItem)
		if !ok {
			t.Fatalf("expected ListItem at index %d", rank.Index)
		}
		if item.path == bodyPath {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected body note to be included in search results")
	}

	if res, ok := model.highlights.lookup(bodyPath); !ok {
		t.Fatalf("expected highlight entry for body note")
	} else if res.Snippet == "" {
		t.Fatalf("expected highlight snippet for body match")
	}

	bodyItem := items[ranks[0].Index].(ListItem)
	if desc := bodyItem.Description(); !strings.Contains(desc, "searchterm") {
		t.Fatalf("expected description to include snippet, got %q", desc)
	}
}

func TestSearchFilterPrefersSearchRankOrder(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	titleMatch := []byte("---\ntitle: Searchterm Heading\n---\ncontent without repeated term\n")
	bodyMatch := []byte("---\ntitle: Fresh Body\n---\nsearchterm appears multiple times: searchterm searchterm\n")

	titlePath := filepath.Join(tempDir, "title.md")
	bodyPath := filepath.Join(tempDir, "body.md")
	if err := os.WriteFile(titlePath, titleMatch, 0o644); err != nil {
		t.Fatalf("failed to write title note: %v", err)
	}
	if err := os.WriteFile(bodyPath, bodyMatch, 0o644); err != nil {
		t.Fatalf("failed to write body note: %v", err)
	}

	stale := time.Now().Add(-48 * time.Hour)
	fresh := time.Now()
	if err := os.Chtimes(titlePath, stale, stale); err != nil {
		t.Fatalf("failed to age title note: %v", err)
	}
	if err := os.Chtimes(bodyPath, fresh, fresh); err != nil {
		t.Fatalf("failed to refresh body note time: %v", err)
	}

	fileHandler := handler.NewFileHandler(tempDir)
	ws := &config.Workspace{VaultDir: tempDir, Search: config.SearchConfig{EnableBody: true}}
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

	model, err := NewNoteListModel(&state.State{
		Config:        cfg,
		Workspace:     ws,
		WorkspaceName: cfg.CurrentWorkspace,
		Handler:       fileHandler,
		ViewManager:   viewManager,
		Vault:         tempDir,
	}, "default")
	if err != nil {
		t.Fatalf("NewNoteListModel returned error: %v", err)
	}

	items := model.list.Items()
	targets := make([]string, len(items))
	for i, item := range items {
		targets[i] = item.FilterValue()
	}

	ranks := model.makeFilterFunc()("searchterm", targets)
	if len(ranks) < 2 {
		t.Fatalf("expected both notes to appear in results, got %d", len(ranks))
	}

	first := items[ranks[0].Index].(ListItem)
	second := items[ranks[1].Index].(ListItem)

	if first.path != bodyPath {
		t.Fatalf("expected body match to rank first, got %s", first.path)
	}
	if second.path != titlePath {
		t.Fatalf("expected title match second, got %s", second.path)
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
