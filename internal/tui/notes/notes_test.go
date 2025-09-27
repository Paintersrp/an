package notes

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"

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

func TestPreviewViewportUpdatesOnPreviewLoaded(t *testing.T) {
	model := newEditorTestModel(t, map[string]string{"note.md": "content"})

	model.previewViewport.Width = 80
	model.previewViewport.Height = 10

	selected, ok := model.list.SelectedItem().(ListItem)
	if !ok {
		t.Fatalf("expected selected list item")
	}

	msg := previewLoadedMsg{path: selected.path, markdown: "body", summary: "Links: 1 outbound"}
	updated, _ := model.Update(msg)

	noteModel, ok := updated.(*NoteListModel)
	if !ok {
		t.Fatalf("expected *NoteListModel, got %T", updated)
	}

	body := noteModel.previewViewport.View()
	if body == "" {
		t.Fatalf("expected viewport view to include content")
	}

	renderedSummary := previewSummaryStyle.Render(strings.TrimSpace(msg.summary))
	if !strings.Contains(body, renderedSummary) {
		t.Fatalf("expected viewport view to include summary %q, got %q", renderedSummary, body)
	}

	if !strings.Contains(body, msg.markdown) {
		t.Fatalf("expected viewport view to include markdown %q, got %q", msg.markdown, body)
	}
}

func TestHandleDefaultUpdateForwardsPreviewScrollKeys(t *testing.T) {
	model := newEditorTestModel(t, map[string]string{"note.md": "content"})
	model.previewViewport.Width = 10
	model.previewViewport.Height = 3
	model.setPreviewContent("line1\nline2\nline3\nline4", "")
	model.previewViewport.GotoTop()

	if cmd, handled := model.handleDefaultUpdate(tea.KeyMsg{Type: tea.KeyShiftTab}); handled {
		if cmd != nil {
			t.Fatalf("expected nil command when focusing preview, got %T", cmd)
		}
	} else {
		t.Fatalf("expected focus key to be handled")
	}

	if !model.previewFocused {
		t.Fatalf("expected preview to gain focus after focus key")
	}

	cmd, handled := model.handleDefaultUpdate(tea.KeyMsg{Type: tea.KeyDown})
	if cmd != nil {
		t.Fatalf("expected nil command, got %T", cmd)
	}
	if !handled {
		t.Fatalf("expected preview scroll key to be handled")
	}
	if model.previewViewport.YOffset == 0 {
		t.Fatalf("expected viewport y offset to change after scroll")
	}
}

func TestHandleDefaultUpdateSkipsPreviewScrollWhenUnfocused(t *testing.T) {
	model := newEditorTestModel(t, map[string]string{"note.md": "content"})
	model.previewViewport.Width = 10
	model.previewViewport.Height = 3
	model.setPreviewContent("line1\nline2\nline3\nline4", "")
	model.previewViewport.GotoTop()

	if cmd, handled := model.handleDefaultUpdate(tea.KeyMsg{Type: tea.KeyShiftTab}); !handled {
		t.Fatalf("expected focus key to be handled")
	} else if cmd != nil {
		t.Fatalf("expected nil command when focusing preview, got %T", cmd)
	}

	if !model.previewFocused {
		t.Fatalf("expected preview to gain focus before toggling off")
	}

	if cmd, handled := model.handleDefaultUpdate(tea.KeyMsg{Type: tea.KeyShiftTab}); !handled {
		t.Fatalf("expected focus key to toggle off")
	} else if cmd != nil {
		t.Fatalf("expected nil command when blurring preview, got %T", cmd)
	}

	if model.previewFocused {
		t.Fatalf("expected preview to lose focus after toggling off")
	}

	yBefore := model.previewViewport.YOffset
	cmd, handled := model.handleDefaultUpdate(tea.KeyMsg{Type: tea.KeyDown})
	if cmd != nil {
		t.Fatalf("expected nil command when preview is unfocused, got %T", cmd)
	}
	if handled {
		t.Fatalf("expected key to be handled by list when preview is unfocused")
	}
	if model.previewViewport.YOffset != yBefore {
		t.Fatalf("expected viewport y offset to remain unchanged")
	}
}

func TestApplyViewReplacesListItems(t *testing.T) {
	tempDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(tempDir, "trash"), 0o755); err != nil {
		t.Fatalf("failed to create trash directory: %v", err)
	}

	defaultFiles := []string{"one.md", "two.md"}
	for _, name := range defaultFiles {
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, []byte("default"), 0o644); err != nil {
			t.Fatalf("failed to write default file %s: %v", name, err)
		}
	}

	trashPath := filepath.Join(tempDir, "trash", "trashed.md")
	if err := os.WriteFile(trashPath, []byte("trashed"), 0o644); err != nil {
		t.Fatalf("failed to write trashed file: %v", err)
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

	state := &state.State{
		Config:        cfg,
		Workspace:     ws,
		WorkspaceName: cfg.CurrentWorkspace,
		Handler:       fileHandler,
		ViewManager:   viewManager,
		Vault:         tempDir,
	}

	model, err := NewNoteListModel(state, "default")
	if err != nil {
		t.Fatalf("NewNoteListModel returned error: %v", err)
	}

	if got, want := len(model.list.Items()), len(defaultFiles); got != want {
		t.Fatalf("expected %d default items, got %d", want, got)
	}

	// Switching to the trash view should replace the current list contents
	// with only the items present in the trash directory.
	model.applyView("trash")

	items := model.list.Items()
	if len(items) != 1 {
		t.Fatalf("expected 1 trashed item, got %d", len(items))
	}

	item, ok := items[0].(ListItem)
	if !ok {
		t.Fatalf("expected ListItem type, got %T", items[0])
	}

	expectedPath := filepath.Clean(trashPath)
	if item.path != expectedPath {
		t.Fatalf("expected trashed item path %q, got %q", expectedPath, item.path)
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

	items := model.list.Items()
	if idx := model.list.Index(); idx < 0 || idx >= len(items) {
		t.Fatalf("expected selection to be within bounds, got index %d with %d items", idx, len(items))
	}

	if _, ok := model.list.SelectedItem().(ListItem); !ok {
		t.Fatalf("expected a selected item after refreshing list")
	}
}

func TestFilterSummaryFormatting(t *testing.T) {
	t.Parallel()

	summary := filterSummary([]string{"beta", "alpha"}, map[string][]string{
		"status": []string{"draft", "active"},
		"owner":  []string{"alice"},
	})

	want := "• tags: alpha, beta • owner: alice • status: active, draft"
	if summary != want {
		t.Fatalf("expected summary %q, got %q", want, summary)
	}

	if summary := filterSummary(nil, nil); summary != "" {
		t.Fatalf("expected empty summary when no filters, got %q", summary)
	}
}

func TestApplyActiveFiltersRestrictsItems(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	active := "---\ntitle: Active\ntags:\n  - project\nstatus: active\n---\nbody"
	done := "---\ntitle: Done\ntags:\n  - project\nstatus: done\n---\nbody"

	activePath := filepath.Join(tempDir, "active.md")
	donePath := filepath.Join(tempDir, "done.md")

	if err := os.WriteFile(activePath, []byte(active), 0o644); err != nil {
		t.Fatalf("failed to write active note: %v", err)
	}
	if err := os.WriteFile(donePath, []byte(done), 0o644); err != nil {
		t.Fatalf("failed to write done note: %v", err)
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

	model.searchQuery.Tags = []string{"project"}
	model.searchQuery.Metadata = map[string][]string{"status": []string{"active"}}
	model.updateFilterStatus()
	if cmd := model.applyActiveFilters(); cmd != nil {
		_ = cmd()
	}

	items := model.list.Items()
	if len(items) != 1 {
		t.Fatalf("expected 1 filtered item, got %d", len(items))
	}

	filtered, ok := items[0].(ListItem)
	if !ok {
		t.Fatalf("expected ListItem type, got %T", items[0])
	}

	if filtered.path != filepath.Clean(activePath) {
		t.Fatalf("expected filtered path %q, got %q", filepath.Clean(activePath), filtered.path)
	}
}

func TestPadAreaPadsViewToRequestedBounds(t *testing.T) {
	t.Parallel()

	view := "item one\nitem two"
	width := 12
	height := 5

	padded := padArea(view, width, height)
	lines := strings.Split(padded, "\n")

	if len(lines) != height {
		t.Fatalf("expected %d lines, got %d", height, len(lines))
	}

	for i, line := range lines {
		if got := lipgloss.Width(line); got != width {
			t.Fatalf("line %d: expected width %d, got %d", i, width, got)
		}
	}

	for i := 2; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			t.Fatalf("expected padded line %d to be empty, got %q", i, lines[i])
		}
	}
}

type stubMsg struct{}

func TestSequenceWithClearWrapsCommand(t *testing.T) {
	t.Parallel()

	called := false
	inner := func() tea.Msg {
		called = true
		return stubMsg{}
	}

	cmd := sequenceWithClear(inner)
	if cmd == nil {
		t.Fatalf("expected command, got nil")
	}

	msg := cmd()
	seq := reflect.ValueOf(msg)
	if seq.Kind() != reflect.Slice {
		t.Fatalf("expected sequence message slice, got %T", msg)
	}

	if seq.Len() < 1 {
		t.Fatalf("expected at least one command in sequence, got %d", seq.Len())
	}

	first, ok := seq.Index(0).Interface().(tea.Cmd)
	if !ok || first == nil {
		t.Fatalf("expected clear command to be non-nil")
	}
	firstMsg := first()
	if got := fmt.Sprintf("%T", firstMsg); got != "tea.clearScreenMsg" {
		t.Fatalf("expected first message to clear screen, got %s", got)
	}

	if seq.Len() < 2 {
		t.Fatalf("expected wrapped command to be present, got %d entries", seq.Len())
	}

	second, ok := seq.Index(1).Interface().(tea.Cmd)
	if !ok || second == nil {
		t.Fatalf("expected wrapped command to be non-nil")
	}

	produced := second()
	if produced == nil {
		t.Fatalf("expected wrapped command to produce a message")
	}
	if _, ok := produced.(stubMsg); !ok {
		t.Fatalf("expected wrapped command to return stubMsg, got %T", produced)
	}

	if !called {
		t.Fatalf("expected wrapped command to run")
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

func TestRootStatusSuffixTruncatesToAvailableWidth(t *testing.T) {
	t.Parallel()

	baseLine := "Checkmark – All View"
	status := "Workspace: Extremely Long Name"
	gap := "  "

	baseWidth := lipgloss.Width(baseLine)
	available := 5
	width := baseWidth + lipgloss.Width(gap) + available

	suffix := rootStatusSuffix(baseLine, width, status, gap)
	if suffix == "" {
		t.Fatalf("expected non-empty suffix")
	}

	want := truncate.StringWithTail(status, uint(available), "")
	if suffix != want {
		t.Fatalf("expected suffix %q, got %q", want, suffix)
	}

	if narrow := rootStatusSuffix(baseLine, baseWidth+lipgloss.Width(gap), status, gap); narrow != "" {
		t.Fatalf("expected no suffix when width only covers gap, got %q", narrow)
	}

	if noGap := rootStatusSuffix("", available, status, ""); noGap != want {
		t.Fatalf("expected suffix %q without gap, got %q", want, noGap)
	}
}
