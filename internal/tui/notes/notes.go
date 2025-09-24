// Package note handles the core note management functionality.
package notes

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/Paintersrp/an/internal/cache"
	"github.com/Paintersrp/an/internal/note"
	"github.com/Paintersrp/an/internal/pathutil"
	"github.com/Paintersrp/an/internal/search"
	"github.com/Paintersrp/an/internal/state"
	journaltui "github.com/Paintersrp/an/internal/tui/journal"
	"github.com/Paintersrp/an/internal/tui/notes/submodels"
	taskstui "github.com/Paintersrp/an/internal/tui/tasks"
	v "github.com/Paintersrp/an/internal/views"
	"github.com/Paintersrp/an/utils"
)

var maxCacheSizeMB int64 = 50

type NoteListModel struct {
	list         list.Model
	cache        *cache.Cache
	keys         *listKeyMap
	delegateKeys *delegateKeyMap
	state        *state.State
	preview      string
	viewName     string
	formModel    submodels.FormModel
	inputModel   submodels.InputModel
	width        int
	height       int
	renaming     bool
	showDetails  bool
	creating     bool
	copying      bool
	editor       *editorSession
	sortField    sortField
	sortOrder    sortOrder
	searchIndex  *search.Index
	searchQuery  search.Query
	highlights   *highlightStore
}

type previewLoadedMsg struct {
	path     string
	content  string
	cacheErr error
}

type editorFinishedMsg struct {
	path   string
	err    error
	waited bool
}

func NewNoteListModel(
	s *state.State,
	viewName string,
) (*NoteListModel, error) {
	view, err := s.ViewManager.GetView(viewName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve view %q: %w", viewName, err)
	}

	files, err := s.ViewManager.GetFilesByView(viewName)
	if err != nil {
		return nil, fmt.Errorf("failed to load files for view %q: %w", viewName, err)
	}

	items := ParseNoteFiles(files, s.Vault, false)
	sortField := sortFieldFromView(view.Sort.Field)
	sortOrder := sortOrderFromView(view.Sort.Order)
	sortedItems := sortItems(castToListItems(items), sortField, sortOrder)

	highlightMatches := newHighlightStore()
	attachHighlightStore(sortedItems, highlightMatches)

	dkeys := newDelegateKeyMap()
	lkeys := newListKeyMap()
	title := v.GetTitleForView(viewName, view.Sort.Field, view.Sort.Order)
	delegate := newItemDelegate(dkeys, s.Handler, viewName)

	l := list.New(sortedItems, delegate, 0, 0)
	l.Title = title
	l.Styles.Title = titleStyle

	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			lkeys.openNote,
			lkeys.editInline,
			lkeys.quickCapture,
			lkeys.changeView,
		}
	}

	l.AdditionalFullHelpKeys = lkeys.fullHelp
	c, err := cache.New(maxCacheSizeMB)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	i := submodels.NewInputModel()
	f := submodels.NewFormModel(s)

	m := &NoteListModel{
		state:        s,
		cache:        c,
		list:         l,
		viewName:     viewName,
		keys:         lkeys,
		delegateKeys: dkeys,
		inputModel:   i,
		formModel:    f,
		renaming:     false,
		creating:     false,
		copying:      false,
		sortField:    sortField,
		sortOrder:    sortOrder,
		highlights:   highlightMatches,
	}

	m.rebuildSearch(files)
	m.list.Filter = m.makeFilterFunc()

	return m, nil
}

func (m *NoteListModel) rebuildSearch(paths []string) {
	if m.highlights != nil {
		m.highlights.clear()
	}

	if m.state == nil || m.state.Config == nil {
		m.searchIndex = nil
		m.searchQuery = search.Query{}
		return
	}

	ws := m.state.Config.MustWorkspace()
	cfg := ws.Search
	searchCfg := search.Config{
		EnableBody:     cfg.EnableBody,
		IgnoredFolders: append([]string(nil), cfg.IgnoredFolders...),
	}

	metadata := make(map[string][]string, len(cfg.DefaultMetadataFilters))
	for key, values := range cfg.DefaultMetadataFilters {
		metadata[key] = append([]string(nil), values...)
	}

	m.searchQuery = search.Query{
		Tags:     append([]string(nil), cfg.DefaultTagFilters...),
		Metadata: metadata,
	}

	index := search.NewIndex(m.state.Vault, searchCfg)
	if err := index.Build(paths); err != nil {
		log.Printf("failed to rebuild search index: %v", err)
		m.list.NewStatusMessage(
			statusStyle(fmt.Sprintf("Search index error: %v", err)),
		)
		m.searchIndex = nil
		return
	}

	m.searchIndex = index
}

func (m *NoteListModel) makeFilterFunc() list.FilterFunc {
	base := list.DefaultFilter

	return func(term string, targets []string) []list.Rank {
		trimmed := strings.TrimSpace(term)
		if m.highlights != nil {
			m.highlights.clear()
		}

		baseRanks := base(term, targets)

		if m.searchIndex == nil {
			return baseRanks
		}

		needsSearch := trimmed != "" || len(m.searchQuery.Tags) > 0 ||
			len(m.searchQuery.Metadata) > 0
		if !needsSearch {
			return baseRanks
		}

		query := m.searchQuery
		query.Term = trimmed
		results := m.searchIndex.Search(query)
		if len(results) == 0 {
			return baseRanks
		}

		highlightMap := make(map[string]search.Result, len(results))
		orderedPaths := make([]string, 0, len(results))
		for _, res := range results {
			normalized := pathutil.NormalizePath(res.Path)
			highlightMap[normalized] = res
			orderedPaths = append(orderedPaths, normalized)
		}

		if m.highlights != nil {
			m.highlights.setAll(highlightMap)
		}

		items := m.list.Items()
		indexByPath := make(map[string]int, len(items))
		for idx, item := range items {
			if li, ok := item.(ListItem); ok {
				indexByPath[pathutil.NormalizePath(li.path)] = idx
			}
		}

		highlightRanks := make([]list.Rank, 0, len(orderedPaths))
		for _, path := range orderedPaths {
			if idx, ok := indexByPath[path]; ok {
				highlightRanks = append(highlightRanks, list.Rank{Index: idx})
			}
		}

		if trimmed == "" &&
			(len(m.searchQuery.Tags) > 0 || len(m.searchQuery.Metadata) > 0) {
			return highlightRanks
		}

		existing := make(map[int]struct{}, len(baseRanks))
		for _, rank := range baseRanks {
			existing[rank.Index] = struct{}{}
		}

		for _, rank := range highlightRanks {
			if _, ok := existing[rank.Index]; !ok {
				baseRanks = append(baseRanks, rank)
			}
		}

		return baseRanks
	}
}

func (m NoteListModel) Init() tea.Cmd {
	var cmds []tea.Cmd

	if m.state != nil && m.state.Watcher != nil {
		cmds = append(cmds, m.state.Watcher.Start())
	}

	if cmd := m.handlePreview(false); cmd != nil {
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

func (m NoteListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case previewLoadedMsg:
		if msg.cacheErr != nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Error updating cache: %s", msg.cacheErr)),
			)
		}

		if s, ok := m.list.SelectedItem().(ListItem); ok && s.path == msg.path {
			m.preview = msg.content
		}

		return m, nil

	case editorFinishedMsg:
		if msg.err != nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Open Error: %v", msg.err)),
			)
			return m, nil
		}

		if msg.waited {
			return m, m.afterExternalEditor()
		}

		return m, nil

	case state.VaultNoteChangedMsg:
		var force bool
		if m.cache != nil && m.state != nil {
			abs := filepath.Join(m.state.Vault, filepath.FromSlash(msg.Path))
			normalized := pathutil.NormalizePath(abs)
			m.cache.Delete(normalized)

			if m.editor != nil && pathutil.NormalizePath(m.editor.path) == normalized {
				m.editor.status = "Note changed on disk. Press ctrl+r to reload or ctrl+s to overwrite."
				m.editor.allowOverwrite = true
			}

			if s, ok := m.list.SelectedItem().(ListItem); ok && pathutil.NormalizePath(s.path) == normalized {
				force = true
			}
		}

		cmds = append(cmds, m.refreshItems())

		if cmd := m.handlePreview(force); cmd != nil {
			cmds = append(cmds, cmd)
		}

		if m.state != nil && m.state.Watcher != nil {
			cmds = append(cmds, m.state.Watcher.Start())
		}

		return m, tea.Batch(cmds...)

	case state.VaultWatcherErrMsg:
		if msg.Err != nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Watcher error: %v", msg.Err)),
			)
		}

		if m.state != nil && m.state.Watcher != nil {
			cmds = append(cmds, m.state.Watcher.Start())
		}

		return m, tea.Batch(cmds...)

	case tea.QuitMsg:
		if err := m.closeWatcher(); err != nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Watcher shutdown error: %v", err)),
			)
		}

		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

		if m.editor != nil {
			width, height := m.editorSize()
			m.editor.setSize(width, height)
		}

		if cmd := m.handlePreview(true); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		if m.editor != nil {
			return m.handleEditorUpdate(msg)
		}

		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case m.copying:
			return m.handleCopyUpdate(msg)
		case m.renaming:
			return m.handleRenameUpdate(msg)
		case m.creating:
			return m.handleCreationUpdate(msg)
		default:
			if cmd, handled := m.handleDefaultUpdate(msg); handled {
				if key.Matches(msg, m.keys.openNote) {
					if ws := m.state.Config.MustWorkspace(); ws.Editor == "vim" || ws.Editor == "nano" {
						if cmd != nil {
							return m, tea.Batch(cmd, tea.Quit)
						}
						return m, tea.Quit
					}
				}

				if cmd != nil {
					return m, cmd
				}

				return m, nil
			}

		}
	}

	if m.editor != nil {
		if cmd := m.editor.area.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	previousSelection := m.currentSelectionPath()

	nl, cmd := m.list.Update(msg)
	m.list = nl
	cmds = append(cmds, cmd)

	if nextSelection := m.currentSelectionPath(); nextSelection != previousSelection {
		if nextSelection == "" {
			m.preview = ""
		}

		if cmd := m.handlePreview(false); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m NoteListModel) currentSelectionPath() string {
	if s, ok := m.list.SelectedItem().(ListItem); ok {
		return s.path
	}

	return ""
}

func (m NoteListModel) handleCopyUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if key.Matches(msg, m.keys.exitAltView) {
		m.toggleCopy()
		return m, nil
	}

	m.inputModel.Input, cmd = m.inputModel.Input.Update(msg)
	cmds = append(cmds, cmd)

	if key.Matches(msg, m.keys.submitAltView) {
		if err := copyFile(m); err != nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Error copying file: %v", err)),
			)
		} else {
			m.toggleCopy()
			m.refresh()
			return m, cmd
		}
	}

	return m, tea.Batch(cmds...)
}

func (m NoteListModel) handleRenameUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if key.Matches(msg, m.keys.exitAltView) {
		m.toggleRename()
		return m, nil
	}

	m.inputModel.Input, cmd = m.inputModel.Input.Update(msg)
	cmds = append(cmds, cmd)

	if key.Matches(msg, m.keys.submitAltView) {
		if err := renameFile(m); err != nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Error renaming file: %v", err)),
			)
		} else {
			m.toggleRename()
			m.refresh()
			return m, cmd
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *NoteListModel) handleCreationUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if key.Matches(msg, m.keys.exitAltView) {
		m.toggleCreation()
		return m, nil
	}

	m.formModel, cmd = m.formModel.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m NoteListModel) handleEditorUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.editor == nil {
		return m, nil
	}

	switch msg.Type {
	case tea.KeyCtrlS:
		cmd := m.saveEditor()
		return m, cmd
	case tea.KeyCtrlR:
		if m.editor.mode == editorModeExisting {
			return m, m.reloadEditor()
		}
		return m, nil
	case tea.KeyEsc:
		if !m.editor.hasChanges() || m.editor.pendingDiscard {
			cmd := m.closeEditor()
			if cmd2 := m.handlePreview(true); cmd2 != nil {
				return m, tea.Batch(cmd, cmd2)
			}
			return m, cmd
		}

		m.editor.pendingDiscard = true
		m.editor.status = "Discard changes? Press esc again to confirm."
		return m, nil
	}

	m.editor.pendingDiscard = false
	cmd := m.editor.area.Update(msg)
	return m, cmd
}

func (m *NoteListModel) startInlineEdit() tea.Cmd {
	if m.editor != nil {
		return nil
	}

	if m.state == nil || m.state.Handler == nil {
		m.list.NewStatusMessage(statusStyle("Editor unavailable: missing file handler"))
		return nil
	}

	selected, ok := m.list.SelectedItem().(ListItem)
	if !ok {
		return nil
	}

	data, err := m.state.Handler.ReadFile(selected.path)
	if err != nil {
		m.list.NewStatusMessage(statusStyle(fmt.Sprintf("Failed to open note: %v", err)))
		return nil
	}

	info, err := os.Stat(selected.path)
	if err != nil {
		m.list.NewStatusMessage(statusStyle(fmt.Sprintf("Failed to stat note: %v", err)))
		return nil
	}

	width, height := m.editorSize()
	session := newEditorSession(width, height)
	session.setMetadata(selected.path, selected.Title(), editorModeExisting)
	session.setValue(string(data))
	session.setOriginal(string(data), info.ModTime())
	session.status = ""

	m.editor = session
	return session.focus()
}

func (m *NoteListModel) startScratchCapture() tea.Cmd {
	if m.editor != nil {
		return nil
	}

	width, height := m.editorSize()
	path, err := m.nextScratchPath()
	if err != nil {
		m.list.NewStatusMessage(statusStyle(fmt.Sprintf("Capture error: %v", err)))
		return nil
	}

	session := newEditorSession(width, height)
	session.setMetadata(path, filepath.Base(path), editorModeScratch)
	session.setValue("")
	session.setOriginal("", time.Time{})
	session.status = ""

	m.editor = session
	return session.focus()
}

func (m *NoteListModel) saveEditor() tea.Cmd {
	if m.editor == nil {
		return nil
	}

	content := m.editor.value()
	switch m.editor.mode {
	case editorModeExisting:
		return m.saveExistingEditor(content)
	case editorModeScratch:
		return m.saveScratchEditor(content)
	default:
		return nil
	}
}

func (m *NoteListModel) saveExistingEditor(content string) tea.Cmd {
	if m.editor == nil {
		return nil
	}

	path := m.editor.path
	if path == "" {
		m.editor.status = "Unable to determine note path"
		return nil
	}

	if m.state == nil || m.state.Handler == nil {
		m.editor.status = "File handler unavailable"
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		m.editor.status = fmt.Sprintf("Save failed: %v", err)
		return nil
	}

	if !m.editor.allowOverwrite && !info.ModTime().Equal(m.editor.originalModTime) {
		disk, err := m.state.Handler.ReadFile(path)
		if err != nil {
			m.editor.status = fmt.Sprintf("Save blocked: %v", err)
			return nil
		}

		if !m.editor.checksumMatches(disk) {
			m.editor.status = "External changes detected. Press ctrl+r to reload or ctrl+s again to overwrite."
			m.editor.allowOverwrite = true
			return nil
		}
	}

	if err := m.state.Handler.WriteFile(path, []byte(content)); err != nil {
		m.editor.status = fmt.Sprintf("Save failed: %v", err)
		return nil
	}

	if info, err := os.Stat(path); err == nil {
		m.editor.setOriginal(content, info.ModTime())
	} else {
		m.editor.setOriginal(content, time.Now())
	}

	cmdBlur := m.closeEditor()

	cmds := []tea.Cmd{}
	if cmdBlur != nil {
		cmds = append(cmds, cmdBlur)
	}
	if cmd := m.handlePreview(true); cmd != nil {
		cmds = append(cmds, cmd)
	}

	m.list.NewStatusMessage(statusStyle("Note saved"))

	return tea.Batch(cmds...)
}

func (m *NoteListModel) saveScratchEditor(content string) tea.Cmd {
	if m.editor == nil {
		return nil
	}

	if content == "" {
		m.editor.status = "Nothing to save"
		return nil
	}

	path := m.editor.path
	if path == "" {
		m.editor.status = "Unable to determine capture destination"
		return nil
	}

	if m.state == nil || m.state.Handler == nil {
		m.editor.status = "File handler unavailable"
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		m.editor.status = fmt.Sprintf("Save failed: %v", err)
		return nil
	}

	if _, err := os.Stat(path); err == nil && !m.editor.allowOverwrite {
		m.editor.status = "Capture already exists. Press ctrl+s again to overwrite."
		m.editor.allowOverwrite = true
		return nil
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		m.editor.status = fmt.Sprintf("Save failed: %v", err)
		return nil
	}

	if err := m.state.Handler.WriteFile(path, []byte(content)); err != nil {
		m.editor.status = fmt.Sprintf("Save failed: %v", err)
		return nil
	}

	if err := note.RunPostCreateHooks(path); err != nil {
		m.editor.status = fmt.Sprintf("Save failed: %v", err)
		return nil
	}

	m.list.NewStatusMessage(
		statusStyle(fmt.Sprintf("Captured note %s", filepath.Base(path))),
	)

	cmdBlur := m.closeEditor()
	cmds := []tea.Cmd{}
	if cmdBlur != nil {
		cmds = append(cmds, cmdBlur)
	}
	if cmd := m.refresh(); cmd != nil {
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

func (m *NoteListModel) reloadEditor() tea.Cmd {
	if m.editor == nil || m.editor.mode != editorModeExisting {
		return nil
	}

	if m.state == nil || m.state.Handler == nil {
		m.editor.status = "File handler unavailable"
		return nil
	}

	data, err := m.state.Handler.ReadFile(m.editor.path)
	if err != nil {
		m.editor.status = fmt.Sprintf("Reload failed: %v", err)
		return nil
	}

	info, err := os.Stat(m.editor.path)
	if err != nil {
		m.editor.status = fmt.Sprintf("Reload failed: %v", err)
		return nil
	}

	m.editor.setValue(string(data))
	m.editor.setOriginal(string(data), info.ModTime())
	m.editor.status = "Reloaded from disk"
	return nil
}

func (m *NoteListModel) closeEditor() tea.Cmd {
	if m.editor == nil {
		return nil
	}

	cmd := m.editor.blur()
	m.editor = nil
	return cmd
}

func (m NoteListModel) editorSize() (int, int) {
	h, v := appStyle.GetFrameSize()
	width := m.width - h
	height := m.height - v
	if width <= 0 {
		width = m.width
	}
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = m.height
	}
	if height <= 0 {
		if listHeight := m.list.Height(); listHeight > 0 {
			height = listHeight
		} else {
			height = 24
		}
	}
	return width, height
}

func (m *NoteListModel) nextScratchPath() (string, error) {
	if m.state == nil || m.state.Config == nil {
		return "", fmt.Errorf("workspace not configured")
	}

	ws := m.state.Config.MustWorkspace()
	base := m.state.Vault
	if base == "" {
		return "", fmt.Errorf("vault path not configured")
	}

	var targetDir string
	if len(ws.SubDirs) > 0 && ws.SubDirs[0] != "" {
		targetDir = filepath.Join(base, ws.SubDirs[0])
	} else {
		targetDir = base
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}

	for i := 0; i < 10; i++ {
		timestamp := time.Now().Format("20060102-150405")
		name := fmt.Sprintf("scratch-%s", timestamp)
		if i > 0 {
			name = fmt.Sprintf("%s-%d", name, i)
		}
		path := filepath.Join(targetDir, name+".md")
		if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
			return path, nil
		}
	}

	return "", fmt.Errorf("unable to allocate capture filename")
}

func (m NoteListModel) editorInstructions() string {
	if m.editor == nil {
		return ""
	}

	switch m.editor.mode {
	case editorModeExisting:
		return "ctrl+s save • ctrl+r reload • esc discard"
	case editorModeScratch:
		return "ctrl+s save • esc discard"
	default:
		return ""
	}
}

func (m *NoteListModel) editorActive() bool {
	if m == nil {
		return false
	}
	return m.editor != nil
}

// TODO: returns are kinda unnecessary now
func (m *NoteListModel) handleDefaultUpdate(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.openNote):
		if cmd := m.openNote(false); cmd != nil {
			return cmd, true
		}
		return nil, true

	case key.Matches(msg, m.keys.openNoteInObsidian):
		if cmd := m.openNote(true); cmd != nil {
			return cmd, true
		}
		return nil, true

	case key.Matches(msg, m.keys.toggleTitleBar):
		m.toggleTitleBar()
		return nil, true

	case key.Matches(msg, m.keys.toggleStatusBar):
		m.list.SetShowStatusBar(!m.list.ShowStatusBar())
		return nil, true

	case key.Matches(msg, m.keys.togglePagination):
		m.list.SetShowPagination(!m.list.ShowPagination())
		return nil, true

	case key.Matches(msg, m.keys.toggleHelpMenu):
		m.list.SetShowHelp(!m.list.ShowHelp())
		return nil, true

	case key.Matches(msg, m.keys.toggleDisplayView):
		return m.toggleDetails(), true

	case key.Matches(msg, m.keys.changeView):
		return m.cycleView(), true

	case key.Matches(msg, m.keys.switchToDefaultView):
		return m.swapView("default"), true

	case key.Matches(msg, m.keys.switchToUnfulfillView):
		return m.swapView("unfulfilled"), true

	case key.Matches(msg, m.keys.switchToOrphanView):
		return m.swapView("orphan"), true

	case key.Matches(msg, m.keys.switchToArchiveView):
		return m.swapView("archive"), true

	case key.Matches(msg, m.keys.switchToTrashView):
		return m.swapView("trash"), true

	case key.Matches(msg, m.keys.rename):
		m.toggleRename()
		return nil, true

	case key.Matches(msg, m.keys.create):
		m.toggleCreation()
		return nil, true

	case key.Matches(msg, m.keys.copy):
		m.toggleCopy()
		return nil, true

	case key.Matches(msg, m.keys.editInline):
		return m.startInlineEdit(), true

	case key.Matches(msg, m.keys.quickCapture):
		return m.startScratchCapture(), true

	case key.Matches(msg, m.keys.sortByTitle):
		m.sortField = sortByTitle
		return m.refreshSort(), true

	case key.Matches(msg, m.keys.sortBySubdir):
		m.sortField = sortBySubdir
		return m.refreshSort(), true

	case key.Matches(msg, m.keys.sortByModifiedAt):
		m.sortField = sortByModifiedAt
		return m.refreshSort(), true

	case key.Matches(msg, m.keys.sortAscending):
		m.sortOrder = ascending
		return m.refreshSort(), true

	case key.Matches(msg, m.keys.sortAscending):
		m.sortOrder = descending
		return m.refreshSort(), true

	case key.Matches(msg, m.keys.sortDescending):
		m.sortOrder = descending
		return m.refreshSort(), true
	}

	return nil, false
}

func (m NoteListModel) View() string {
	if m.editor != nil {
		width, height := m.editorSize()
		m.editor.setSize(width, height)

		sections := []string{
			titleStyle.Render(m.editor.viewHeader()),
			m.editor.area.View(),
			helpStyle.Render(m.editorInstructions()),
		}

		if msg := m.editor.status; msg != "" {
			sections = append(sections, statusStyle(msg))
		}

		layout := lipgloss.JoinVertical(lipgloss.Left, sections...)
		return appStyle.Render(layout)
	}

	list := listStyle.MaxWidth(m.width / 2).Render(m.list.View())

	if m.copying {
		textPrompt := textPromptStyle.Render(
			lipgloss.NewStyle().
				Height(m.list.Height()).
				MaxHeight(m.list.Height()).
				Padding(0, 2).
				Render(fmt.Sprintf("%s\n\n%s\n\n%s", titleStyle.Render("Choose new name for the copy"), m.inputModel.View(), helpStyle.Render("do not include file extension"))),
		)

		layout := lipgloss.JoinHorizontal(lipgloss.Top, list, textPrompt)
		return appStyle.Render(layout)
	}

	if m.creating {
		modelStyle := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).Padding(0, 1)
		return appStyle.Render(modelStyle.Render(m.formModel.View()))
	}

	if m.renaming {
		textPrompt := textPromptStyle.Render(
			lipgloss.NewStyle().
				Height(m.list.Height()).
				MaxHeight(m.list.Height()).
				Padding(0, 2).
				Render(fmt.Sprintf("%s\n\n%s", titleStyle.Render("Rename File"), m.inputModel.View())),
		)

		layout := lipgloss.JoinHorizontal(lipgloss.Top, list, textPrompt)
		return appStyle.Render(layout)
	}

	preview := previewStyle.Render(
		lipgloss.NewStyle().
			Height(m.list.Height()).
			MaxHeight(m.list.Height()).
			Render(fmt.Sprintf("%s\n%s", titleStyle.Render("Preview"), m.preview)),
	)

	layout := lipgloss.JoinHorizontal(lipgloss.Top, list, preview)
	return appStyle.Render(layout)
}

func Run(s *state.State, views map[string]v.View, viewFlag string) error {
	originalState, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatalf("Failed to get original terminal state: %v", err)
	}

	defer func() {
		// we attempt to recover original state so that we can terminate gracefully
		if err := term.Restore(int(os.Stdin.Fd()), originalState); err != nil {
			log.Fatalf("Failed to restore original terminal state: %v", err)
		}
	}()

	noteModel, err := NewNoteListModel(s, viewFlag)
	if err != nil {
		return err
	}

	tasksModel, err := taskstui.NewModel(s)
	if err != nil {
		return err
	}

	journalModel, err := journaltui.NewModel(s)
	if err != nil {
		return err
	}

	root := NewRootModel(noteModel, tasksModel, journalModel)

	if _, err := tea.NewProgram(root, tea.WithInput(os.Stdin), tea.WithAltScreen()).Run(); err != nil {
		// handle error for instances where neovim/editor doesn't pass stdin back in time to close gracefully with bubbletea
		if strings.Contains(err.Error(), "resource temporarily unavailable") {
			os.Exit(0)
		} else {
			log.Fatalf("Error running program: %v", err)
		}
	}

	return nil
}

func (m *NoteListModel) closeWatcher() error {
	if m.state == nil || m.state.Watcher == nil {
		return nil
	}

	err := m.state.Watcher.Close()
	m.state.Watcher = nil

	return err
}

func (m *NoteListModel) handlePreview(force bool) tea.Cmd {
	selectedPath := ""
	if s, ok := m.list.SelectedItem().(ListItem); ok {
		selectedPath = s.path
	} else {
		m.preview = ""
		return nil
	}

	cache := m.cache
	if cache == nil {
		width := m.width / 2
		height := m.list.Height()
		return renderPreviewCmd(selectedPath, width, height, nil)
	}

	if !force {
		if cached, exists, err := cache.Get(selectedPath); err == nil && exists {
			if preview, ok := cached.(string); ok {
				m.preview = preview
				return nil
			}

			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Unexpected cache type: %T", cached)),
			)
		} else if err != nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Error accessing cache: %s", err)),
			)
		}
	}

	width := m.width / 2
	height := m.list.Height()

	return renderPreviewCmd(selectedPath, width, height, cache)
}

func renderPreviewCmd(path string, width, height int, cache *cache.Cache) tea.Cmd {
	return func() tea.Msg {
		rendered := utils.RenderMarkdownPreview(path, width, height)

		var cacheErr error
		if cache != nil {
			cacheErr = cache.Put(path, rendered)
		}

		return previewLoadedMsg{
			path:     path,
			content:  rendered,
			cacheErr: cacheErr,
		}
	}
}

func (m *NoteListModel) refresh() tea.Cmd {
	m.list.Title = v.GetTitleForView(
		m.viewName,
		viewSortField(m.sortField),
		viewSortOrder(m.sortOrder),
	)
	m.refreshDelegate()
	cmd := m.refreshItems()
	m.list.ResetSelected()
	return tea.Batch(cmd, m.handlePreview(true))
}

func (m *NoteListModel) refreshItems() tea.Cmd {
	files, err := m.state.ViewManager.GetFilesByView(m.viewName)
	if err != nil {
		m.list.NewStatusMessage(
			statusStyle(fmt.Sprintf("Failed to load %s view: %v", m.viewName, err)),
		)
		return nil
	}
	items := ParseNoteFiles(files, m.state.Vault, m.showDetails)
	sortedItems := sortItems(castToListItems(items), m.sortField, m.sortOrder)
	attachHighlightStore(sortedItems, m.highlights)
	m.rebuildSearch(files)
	return m.list.SetItems(sortedItems)
}

func (m *NoteListModel) refreshDelegate() {
	dkeys := newDelegateKeyMap()
	delegate := newItemDelegate(dkeys, m.state.Handler, m.viewName)
	m.list.SetDelegate(delegate)
}

func (m *NoteListModel) refreshSort() tea.Cmd {
	m.list.Title = v.GetTitleForView(
		m.viewName,
		viewSortField(m.sortField),
		viewSortOrder(m.sortOrder),
	)
	items := castToListItems(m.list.Items())
	sortedItems := sortItems(items, m.sortField, m.sortOrder)
	m.list.ResetSelected()
	cmd := m.list.SetItems(sortedItems)
	return tea.Batch(cmd, m.handlePreview(true))
}

// TODO: should prob use an error over a bool but a "success" flag sort of feels more natural for the context.
// TODO: unsuccessful opens provide a status message and the program stays live
// openNote reports success so callers can trigger follow-up updates (like refreshing the preview)
func (m *NoteListModel) openNote(obsidian bool) tea.Cmd {
	item, ok := m.list.SelectedItem().(ListItem)
	if !ok {
		return nil
	}

	launch, err := note.EditorLaunchForPath(item.path, obsidian)
	if err != nil {
		m.list.NewStatusMessage(statusStyle(fmt.Sprintf("Open Error: %v", err)))
		return nil
	}
	if hookErr := note.RunPreOpenHooks(item.path); hookErr != nil {
		m.list.NewStatusMessage(statusStyle(fmt.Sprintf("Pre-open hook error: %v", hookErr)))
		return nil
	}
	if !launch.Wait {
		return func() tea.Msg {
			startErr := launch.Cmd.Start()
			if startErr != nil {
				return editorFinishedMsg{path: item.path, err: startErr}
			}
			if err := note.RunPostOpenHooks(item.path); err != nil {
				return editorFinishedMsg{path: item.path, err: err, waited: false}
			}
			return editorFinishedMsg{path: item.path, waited: false}
		}
	}

	return tea.ExecProcess(launch.Cmd, func(execErr error) tea.Msg {
		if execErr == nil {
			if err := note.RunPostOpenHooks(item.path); err != nil {
				return editorFinishedMsg{path: item.path, err: err, waited: true}
			}
		}
		return editorFinishedMsg{path: item.path, err: execErr, waited: true}
	})
}

func (m *NoteListModel) afterExternalEditor() tea.Cmd {
	if cmd := m.handlePreview(true); cmd != nil {
		return cmd
	}

	return nil
}

func (m *NoteListModel) toggleTitleBar() {
	v := !m.list.ShowTitle()
	m.list.SetShowTitle(v)
	m.list.SetShowFilter(v)
	m.list.SetFilteringEnabled(v)
}

func (m *NoteListModel) toggleDetails() tea.Cmd {
	m.showDetails = !m.showDetails
	cmd := m.refreshItems()
	m.list.ResetSelected()
	return tea.Batch(cmd, m.handlePreview(true))
}

func (m *NoteListModel) cycleView() tea.Cmd {
	next := m.state.ViewManager.NextView(m.viewName)
	return m.applyView(next)
}

func (m *NoteListModel) swapView(newView string) tea.Cmd {
	return m.applyView(newView)
}

func (m *NoteListModel) applyView(viewName string) tea.Cmd {
	view, err := m.state.ViewManager.GetView(viewName)
	if err != nil {
		m.list.NewStatusMessage(statusStyle(fmt.Sprintf("Invalid view %s", viewName)))
		return nil
	}

	m.viewName = viewName
	m.sortField = sortFieldFromView(view.Sort.Field)
	m.sortOrder = sortOrderFromView(view.Sort.Order)

	return m.refresh()
}

func sortFieldFromView(field v.SortField) sortField {
	switch field {
	case v.SortFieldTitle:
		return sortByTitle
	case v.SortFieldSubdirectory:
		return sortBySubdir
	case v.SortFieldModified:
		fallthrough
	default:
		return sortByModifiedAt
	}
}

func sortOrderFromView(order v.SortOrder) sortOrder {
	switch order {
	case v.SortOrderAscending:
		return ascending
	case v.SortOrderDescending:
		fallthrough
	default:
		return descending
	}
}

func viewSortField(field sortField) v.SortField {
	switch field {
	case sortByTitle:
		return v.SortFieldTitle
	case sortBySubdir:
		return v.SortFieldSubdirectory
	case sortByModifiedAt:
		fallthrough
	default:
		return v.SortFieldModified
	}
}

func viewSortOrder(order sortOrder) v.SortOrder {
	switch order {
	case ascending:
		return v.SortOrderAscending
	case descending:
		fallthrough
	default:
		return v.SortOrderDescending
	}
}

func (m *NoteListModel) toggleCopy() {
	switch m.copying {
	case true:
		m.copying = false
		m.inputModel.Input.Blur()
	case false:
		m.copying = true
		m.inputModel.Input.Focus()
		if s, ok := m.list.SelectedItem().(ListItem); ok {
			base := s.Title()
			if base == "" {
				base = strings.TrimSuffix(s.fileName, ".md")
			}
			if base == "" {
				break
			}

			const suffix = "-copy"
			if !strings.HasSuffix(base, suffix) {
				base += suffix
			}
			m.inputModel.Input.SetValue(base)
		}
	}
}

func (m *NoteListModel) toggleRename() {
	switch m.renaming {
	case true:
		m.renaming = false
		m.inputModel.Input.Blur()
	case false:
		m.renaming = true
		m.inputModel.Input.Focus()
		if s, ok := m.list.SelectedItem().(ListItem); ok {
			value := s.Title()
			if value == "" {
				value = strings.TrimSuffix(s.fileName, ".md")
			}
			m.inputModel.Input.SetValue(value)
		}
	}
}

// TODO: clear?
func (m *NoteListModel) toggleCreation() {
	switch m.creating {
	case true:
		m.formModel.Inputs[m.formModel.Focused].Blur()
		m.creating = false
	case false:
		m.formModel.Inputs[m.formModel.Focused].Focus()
		m.creating = true
	}
}
