// Package note handles the core note management functionality.
package notes

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/Paintersrp/an/internal/cache"
	"github.com/Paintersrp/an/internal/note"
	"github.com/Paintersrp/an/internal/pathutil"
	"github.com/Paintersrp/an/internal/review"
	"github.com/Paintersrp/an/internal/search"
	"github.com/Paintersrp/an/internal/state"
	journaltui "github.com/Paintersrp/an/internal/tui/journal"
	"github.com/Paintersrp/an/internal/tui/notes/submodels"
	taskstui "github.com/Paintersrp/an/internal/tui/tasks"
	v "github.com/Paintersrp/an/internal/views"
	"github.com/Paintersrp/an/utils"
)

var maxCacheSizeMB int64 = 50

const searchCompactionInterval = time.Hour

type NoteListModel struct {
	list                list.Model
	cache               *cache.Cache
	keys                *listKeyMap
	delegateKeys        *delegateKeyMap
	state               *state.State
	preview             string
	previewSummary      string
	previewViewport     viewport.Model
	viewName            string
	formModel           submodels.FormModel
	filterModel         *submodels.FilterModel
	inputModel          submodels.InputModel
	width               int
	height              int
	previewWidth        int
	renaming            bool
	showDetails         bool
	creating            bool
	copying             bool
	filtering           bool
	editor              *editorSession
	sortField           sortField
	sortOrder           sortOrder
	searchIndex         *search.Index
	searchQuery         search.Query
	searchConfig        search.Config
	lastSearchRebuild   time.Time
	highlights          *highlightStore
	pendingIndexUpdates map[string]struct{}
	reviewQueue         []review.ResurfaceItem
	availableTags       []string
	availableMetadata   map[string][]string
	allItems            []list.Item
	searchInitialized   bool
	previewFocused      bool
}

type previewLoadedMsg struct {
	path       string
	markdown   string
	summary    string
	cacheErr   error
	complete   bool
	background *previewRequest
}

type previewCacheEntry struct {
	Markdown string
	Complete bool
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
	filterModel := submodels.NewFilterModel()

	m := &NoteListModel{
		state:               s,
		cache:               c,
		list:                l,
		viewName:            viewName,
		keys:                lkeys,
		delegateKeys:        dkeys,
		inputModel:          i,
		formModel:           f,
		filterModel:         filterModel,
		renaming:            false,
		creating:            false,
		copying:             false,
		filtering:           false,
		sortField:           sortField,
		sortOrder:           sortOrder,
		highlights:          highlightMatches,
		pendingIndexUpdates: make(map[string]struct{}),
		availableMetadata:   make(map[string][]string),
		previewViewport:     viewport.New(0, 0),
	}

	m.allItems = append([]list.Item(nil), sortedItems...)
	m.rebuildSearch(files)
	m.list.Filter = m.makeFilterFunc()
	m.updateFilterInventory()
	m.syncFilterPalette()
	m.updateFilterStatus()
	filtered := m.filteredItems()
	m.list.SetItems(filtered)
	m.blurPreview()

	return m, nil
}

func (m *NoteListModel) rebuildSearch(paths []string) {
	if m.highlights != nil {
		m.highlights.clear()
	}

	if m.pendingIndexUpdates == nil {
		m.pendingIndexUpdates = make(map[string]struct{})
	}

	if m.state == nil || m.state.Config == nil {
		m.searchIndex = nil
		m.searchQuery = search.Query{}
		m.searchConfig = search.Config{}
		m.lastSearchRebuild = time.Time{}
		m.pendingIndexUpdates = make(map[string]struct{})
		return
	}

	ws := m.state.Config.MustWorkspace()
	cfg := ws.Search
	searchCfg := search.Config{
		EnableBody:     cfg.EnableBody,
		IgnoredFolders: append([]string(nil), cfg.IgnoredFolders...),
	}

	metadata := cloneMetadataMap(cfg.DefaultMetadataFilters)

	if !m.searchInitialized {
		m.searchQuery = search.Query{
			Tags:     cloneStringSlice(cfg.DefaultTagFilters),
			Metadata: metadata,
		}
		m.searchInitialized = true
	} else {
		m.searchQuery.Tags = cloneStringSlice(m.searchQuery.Tags)
		if len(m.searchQuery.Metadata) == 0 {
			m.searchQuery.Metadata = make(map[string][]string)
		} else {
			m.searchQuery.Metadata = cloneMetadataMap(m.searchQuery.Metadata)
		}
	}

	forceRebuild := m.searchIndex == nil ||
		!configsEqual(m.searchConfig, searchCfg) ||
		time.Since(m.lastSearchRebuild) > searchCompactionInterval

	if forceRebuild {
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
		m.searchConfig = searchCfg
		m.lastSearchRebuild = time.Now()
		m.pendingIndexUpdates = make(map[string]struct{})
		return
	}

	m.searchConfig = searchCfg
	m.applyPendingIndexUpdates()
	m.pruneIndex(paths)
}

func (m *NoteListModel) queueIndexUpdate(rel string) {
	cleaned := strings.TrimSpace(rel)
	if cleaned == "" {
		return
	}
	if m.pendingIndexUpdates == nil {
		m.pendingIndexUpdates = make(map[string]struct{})
	}
	m.pendingIndexUpdates[filepath.ToSlash(cleaned)] = struct{}{}
}

func (m *NoteListModel) applyPendingIndexUpdates() {
	if len(m.pendingIndexUpdates) == 0 || m.searchIndex == nil || m.state == nil {
		return
	}

	for rel := range m.pendingIndexUpdates {
		abs := filepath.Join(m.state.Vault, filepath.FromSlash(rel))
		normalized := pathutil.NormalizePath(abs)
		if normalized == "" {
			continue
		}

		info, err := os.Stat(normalized)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				if removeErr := m.searchIndex.Remove(normalized); removeErr != nil {
					log.Printf("failed to remove %s from search index: %v", normalized, removeErr)
				}
				continue
			}
			log.Printf("stat for %s failed: %v", normalized, err)
			continue
		}

		if info.IsDir() {
			if removeErr := m.searchIndex.Remove(normalized); removeErr != nil {
				log.Printf("failed to remove directory %s from search index: %v", normalized, removeErr)
			}
			continue
		}

		if err := m.searchIndex.Update(normalized); err != nil {
			log.Printf("failed to update search index for %s: %v", normalized, err)
		}
	}

	m.pendingIndexUpdates = make(map[string]struct{})
}

func (m *NoteListModel) pruneIndex(paths []string) {
	if m.searchIndex == nil {
		return
	}

	desired := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		normalized := pathutil.NormalizePath(p)
		if normalized != "" {
			desired[normalized] = struct{}{}
		}
	}

	existingMeta := m.searchIndex.Documents()
	existing := make(map[string]struct{}, len(existingMeta))
	for _, meta := range existingMeta {
		normalized := pathutil.NormalizePath(meta.Path)
		if normalized == "" {
			continue
		}
		existing[normalized] = struct{}{}
		if _, ok := desired[normalized]; ok {
			continue
		}
		if err := m.searchIndex.Remove(meta.Path); err != nil {
			log.Printf("failed to remove stale search document %s: %v", meta.Path, err)
		}
	}

	for normalized := range desired {
		if _, ok := existing[normalized]; ok {
			continue
		}
		if err := m.searchIndex.Update(normalized); err != nil {
			log.Printf("failed to index new note %s: %v", normalized, err)
		}
	}
}

func configsEqual(a, b search.Config) bool {
	if a.EnableBody != b.EnableBody {
		return false
	}
	if len(a.IgnoredFolders) != len(b.IgnoredFolders) {
		return false
	}
	return reflect.DeepEqual(a.IgnoredFolders, b.IgnoredFolders)
}

func (m *NoteListModel) makeFilterFunc() list.FilterFunc {
	base := list.DefaultFilter

	return func(term string, targets []string) []list.Rank {
		trimmed := strings.TrimSpace(term)
		if m.highlights != nil {
			m.highlights.clear()
		}

		baseRanks := base(term, targets)
		matchedIndexes := make(map[int][]int, len(baseRanks))
		for _, rank := range baseRanks {
			matchedIndexes[rank.Index] = rank.MatchedIndexes
		}

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

		searchRanks := make([]list.Rank, 0, len(orderedPaths))
		for _, path := range orderedPaths {
			if idx, ok := indexByPath[path]; ok {
				rank := list.Rank{Index: idx}
				if matches, ok := matchedIndexes[idx]; ok {
					rank.MatchedIndexes = matches
				}
				searchRanks = append(searchRanks, rank)
			}
		}

		if trimmed == "" &&
			(len(m.searchQuery.Tags) > 0 || len(m.searchQuery.Metadata) > 0) {
			return searchRanks
		}

		if trimmed != "" && len(searchRanks) > 0 {
			ordered := make([]list.Rank, 0, len(searchRanks)+len(baseRanks))
			seen := make(map[int]struct{}, len(searchRanks))
			for _, rank := range searchRanks {
				ordered = append(ordered, rank)
				seen[rank.Index] = struct{}{}
			}
			for _, rank := range baseRanks {
				if _, ok := seen[rank.Index]; ok {
					continue
				}
				ordered = append(ordered, rank)
			}
			return ordered
		}

		existing := make(map[int]struct{}, len(baseRanks))
		for _, rank := range baseRanks {
			existing[rank.Index] = struct{}{}
		}

		for _, rank := range searchRanks {
			if _, ok := existing[rank.Index]; !ok {
				baseRanks = append(baseRanks, rank)
			}
		}

		return baseRanks
	}
}

func (m *NoteListModel) filteredItems() []list.Item {
	if len(m.allItems) == 0 {
		return nil
	}

	if m.searchIndex == nil || (len(m.searchQuery.Tags) == 0 && len(m.searchQuery.Metadata) == 0) {
		return append([]list.Item(nil), m.allItems...)
	}

	query := search.Query{
		Tags:     cloneStringSlice(m.searchQuery.Tags),
		Metadata: cloneMetadataMap(m.searchQuery.Metadata),
	}

	matches := m.searchIndex.FilteredDocuments(query)
	if len(matches) == 0 {
		return []list.Item{}
	}

	allowed := make(map[string]struct{}, len(matches))
	for _, doc := range matches {
		normalized := pathutil.NormalizePath(doc.Path)
		if normalized == "" {
			continue
		}
		allowed[normalized] = struct{}{}
	}

	filtered := make([]list.Item, 0, len(allowed))
	for _, item := range m.allItems {
		listItem, ok := item.(ListItem)
		if !ok {
			filtered = append(filtered, item)
			continue
		}
		normalized := pathutil.NormalizePath(listItem.path)
		if _, ok := allowed[normalized]; ok {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

func (m *NoteListModel) applyActiveFilters() tea.Cmd {
	filtered := m.filteredItems()
	cmd := m.list.SetItems(filtered)
	m.ensureSelectionInBounds()
	return cmd
}

func (m *NoteListModel) updateFilterInventory() {
	tags := make(map[string]struct{})
	metadataValues := make(map[string]map[string]struct{})

	if m.searchIndex != nil {
		for _, doc := range m.searchIndex.Documents() {
			for _, tag := range doc.Tags {
				trimmed := strings.TrimSpace(tag)
				if trimmed == "" {
					continue
				}
				tags[trimmed] = struct{}{}
			}
			for key, values := range doc.FrontMatter {
				trimmedKey := strings.TrimSpace(key)
				if trimmedKey == "" {
					continue
				}
				if _, ok := metadataValues[trimmedKey]; !ok {
					metadataValues[trimmedKey] = make(map[string]struct{})
				}
				for _, value := range values {
					trimmedValue := strings.TrimSpace(value)
					if trimmedValue == "" {
						continue
					}
					metadataValues[trimmedKey][trimmedValue] = struct{}{}
				}
			}
		}
	} else {
		for _, item := range m.allItems {
			if listItem, ok := item.(ListItem); ok {
				for _, tag := range listItem.tags {
					trimmed := strings.TrimSpace(tag)
					if trimmed == "" {
						continue
					}
					tags[trimmed] = struct{}{}
				}
			}
		}
	}

	sortedTags := make([]string, 0, len(tags))
	for tag := range tags {
		sortedTags = append(sortedTags, tag)
	}
	sort.Strings(sortedTags)

	sortedMetadata := make(map[string][]string, len(metadataValues))
	keys := make([]string, 0, len(metadataValues))
	for key := range metadataValues {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		values := metadataValues[key]
		sortedValues := make([]string, 0, len(values))
		for value := range values {
			sortedValues = append(sortedValues, value)
		}
		sort.Strings(sortedValues)
		sortedMetadata[key] = sortedValues
	}

	m.availableTags = sortedTags
	m.availableMetadata = sortedMetadata

	if m.filterModel != nil {
		m.filterModel.SetOptions(sortedTags, sortedMetadata)
	}
}

func (m *NoteListModel) syncFilterPalette() {
	if m.filterModel == nil {
		return
	}
	m.filterModel.SetSelection(m.searchQuery.Tags, m.searchQuery.Metadata)
}

func (m *NoteListModel) updateFilterStatus() {
	summary := filterSummary(m.searchQuery.Tags, m.searchQuery.Metadata)
	singular := "note"
	plural := "notes"
	if summary != "" {
		singular = fmt.Sprintf("note %s", summary)
		plural = fmt.Sprintf("notes %s", summary)
	}
	m.list.SetStatusBarItemName(singular, plural)
}

func (m *NoteListModel) toggleFilterPalette() tea.Cmd {
	if m.filtering {
		m.filtering = false
		return nil
	}

	if m.filterModel == nil {
		m.filterModel = submodels.NewFilterModel()
	}

	m.filterModel.SetOptions(m.availableTags, m.availableMetadata)
	m.filterModel.SetSelection(m.searchQuery.Tags, m.searchQuery.Metadata)
	m.filtering = true
	return nil
}

func filterSummary(tags []string, metadata map[string][]string) string {
	sections := make([]string, 0, 1+len(metadata))

	normalizedTags := cloneStringSlice(tags)
	if len(normalizedTags) > 0 {
		sections = append(sections, fmt.Sprintf("tags: %s", strings.Join(normalizedTags, ", ")))
	}

	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		keys = append(keys, trimmed)
	}
	sort.Strings(keys)

	for _, key := range keys {
		values := cloneStringSlice(metadata[key])
		if len(values) == 0 {
			continue
		}
		sections = append(sections, fmt.Sprintf("%s: %s", key, strings.Join(values, ", ")))
	}

	if len(sections) == 0 {
		return ""
	}
	return "• " + strings.Join(sections, " • ")
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}

	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneMetadataMap(src map[string][]string) map[string][]string {
	if len(src) == 0 {
		return make(map[string][]string)
	}

	out := make(map[string][]string, len(src))
	for key, values := range src {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		cloned := cloneStringSlice(values)
		if len(cloned) == 0 {
			continue
		}
		out[trimmedKey] = cloned
	}

	if len(out) == 0 {
		return make(map[string][]string)
	}
	return out
}

func (m *NoteListModel) Init() tea.Cmd {
	var cmds []tea.Cmd

	if m.state != nil && m.state.Watcher != nil {
		cmds = append(cmds, m.state.Watcher.Start())
	}

	if cmd := m.handlePreview(false); cmd != nil {
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

func (m *NoteListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case submodels.FilterSelectionChangedMsg:
		m.searchQuery.Tags = cloneStringSlice(msg.Tags)
		m.searchQuery.Metadata = cloneMetadataMap(msg.Metadata)
		m.syncFilterPalette()
		m.updateFilterStatus()
		return m, m.applyActiveFilters()

	case submodels.FilterClosedMsg:
		m.filtering = false
		return m, nil

	case previewLoadedMsg:
		if msg.cacheErr != nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Error updating cache: %s", msg.cacheErr)),
			)
		}

		if s, ok := m.list.SelectedItem().(ListItem); ok && s.path == msg.path {
			m.setPreviewContent(msg.markdown, msg.summary)
		}

		if msg.background != nil {
			cmds = append(cmds, renderPreviewCmd(*msg.background))
		}

		if len(cmds) == 0 {
			return m, nil
		}

		return m, tea.Batch(cmds...)

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
		m.queueIndexUpdate(msg.Path)
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

	case noteListRefreshMsg:
		return m, batchCmds(m.refreshItems(), m.handlePreview(true))

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
		contentWidth := msg.Width - h
		contentHeight := msg.Height - v
		if contentWidth < 0 {
			contentWidth = msg.Width
		}
		if contentWidth < 0 {
			contentWidth = 0
		}
		if contentHeight < 0 {
			contentHeight = msg.Height
		}
		if contentHeight < 0 {
			contentHeight = 0
		}

		listFrameWidth, _ := listStyle.GetFrameSize()
		previewFrameWidth, _ := previewStyle.GetFrameSize()
		promptFrameWidth, _ := textPromptStyle.GetFrameSize()
		filterFrameWidth, _ := filterPaletteStyle.GetFrameSize()

		sideFrameWidth := previewFrameWidth
		if promptFrameWidth > sideFrameWidth {
			sideFrameWidth = promptFrameWidth
		}
		if filterFrameWidth > sideFrameWidth {
			sideFrameWidth = filterFrameWidth
		}

		available := contentWidth - listFrameWidth - sideFrameWidth
		if available < 0 {
			available = 0
		}

		listContentWidth := (available * 3) / 5
		previewContentWidth := available - listContentWidth

		const (
			minListContent    = 30
			minPreviewContent = 25
		)

		if available >= minListContent+minPreviewContent {
			if listContentWidth < minListContent {
				listContentWidth = minListContent
				previewContentWidth = available - listContentWidth
			}
			if previewContentWidth < minPreviewContent {
				previewContentWidth = minPreviewContent
				listContentWidth = available - previewContentWidth
			}
		} else {
			if previewContentWidth < 0 {
				previewContentWidth = 0
			}
			if listContentWidth < 0 {
				listContentWidth = 0
			}
		}

		if listContentWidth < 0 {
			listContentWidth = 0
		}
		if previewContentWidth < 0 {
			previewContentWidth = 0
		}

		// If there isn't enough room for the preview content, allocate the
		// remaining horizontal space to the list.
		if previewContentWidth == 0 && available > 0 {
			listContentWidth = available
		}

		m.previewWidth = previewContentWidth
		m.list.SetSize(listContentWidth, contentHeight)

		if previewContentWidth < 0 {
			previewContentWidth = 0
		}
		if contentHeight < 0 {
			contentHeight = 0
		}
		m.previewViewport.Width = previewContentWidth
		m.previewViewport.Height = contentHeight
		m.previewViewport.SetYOffset(m.previewViewport.YOffset)

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

		if m.filtering {
			model, cmd := m.handleFilterUpdate(msg)
			return model, cmd
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

	m.ensureSelectionInBounds()

	if nextSelection := m.currentSelectionPath(); nextSelection != previousSelection {
		if nextSelection == "" {
			m.setPreviewContent("", "")
			m.previewViewport.GotoTop()
		} else {
			m.previewViewport.GotoTop()
		}

		if cmd := m.handlePreview(false); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *NoteListModel) currentSelectionPath() string {
	if s, ok := m.list.SelectedItem().(ListItem); ok {
		return s.path
	}

	return ""
}

func (m *NoteListModel) ensureSelectionInBounds() {
	items := m.list.Items()
	if len(items) == 0 {
		m.list.ResetSelected()
		return
	}

	if idx := m.list.Index(); idx >= len(items) {
		m.list.ResetSelected()
	}
}

func (m *NoteListModel) handleCopyUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if key.Matches(msg, m.keys.exitAltView) {
		m.toggleCopy()
		return m, nil
	}

	m.inputModel.Input, cmd = m.inputModel.Input.Update(msg)
	cmds = append(cmds, cmd)

	if key.Matches(msg, m.keys.submitAltView) {
		if err := copyFile(*m); err != nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Error copying file: %v", err)),
			)
		} else {
			m.toggleCopy()
			if refreshCmd := m.refresh(); refreshCmd != nil {
				cmds = append(cmds, refreshCmd)
			}
			return m, tea.Batch(cmds...)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *NoteListModel) handleRenameUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if key.Matches(msg, m.keys.exitAltView) {
		m.toggleRename()
		return m, nil
	}

	m.inputModel.Input, cmd = m.inputModel.Input.Update(msg)
	cmds = append(cmds, cmd)

	if key.Matches(msg, m.keys.submitAltView) {
		if err := renameFile(*m); err != nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Error renaming file: %v", err)),
			)
		} else {
			m.toggleRename()
			if refreshCmd := m.refresh(); refreshCmd != nil {
				cmds = append(cmds, refreshCmd)
			}
			return m, tea.Batch(cmds...)
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

func (m *NoteListModel) handleFilterUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filterModel == nil {
		m.filtering = false
		return m, nil
	}

	cmd, handled := m.filterModel.Update(msg)
	if handled {
		return m, cmd
	}

	if key.Matches(msg, m.keys.exitAltView) {
		m.filtering = false
		return m, nil
	}

	return m, nil
}

func (m *NoteListModel) handleEditorUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	if m.previewHasFocus() && isPreviewScrollKey(msg) {
		var cmd tea.Cmd
		m.previewViewport, cmd = m.previewViewport.Update(msg)
		return cmd, true
	}

	switch {
	case key.Matches(msg, m.keys.toggleFocus):
		if m.previewHasFocus() {
			return m.blurPreview(), true
		}
		return m.focusPreview(), true

	case key.Matches(msg, m.keys.openNote):
		return batchCmds(m.blurPreview(), m.openNote(false)), true

	case key.Matches(msg, m.keys.openNoteInObsidian):
		return batchCmds(m.blurPreview(), m.openNote(true)), true

	case key.Matches(msg, m.keys.toggleTitleBar):
		m.toggleTitleBar()
		return batchCmds(m.blurPreview()), true

	case key.Matches(msg, m.keys.toggleStatusBar):
		m.list.SetShowStatusBar(!m.list.ShowStatusBar())
		return batchCmds(m.blurPreview()), true

	case key.Matches(msg, m.keys.togglePagination):
		m.list.SetShowPagination(!m.list.ShowPagination())
		return batchCmds(m.blurPreview()), true

	case key.Matches(msg, m.keys.toggleHelpMenu):
		m.list.SetShowHelp(!m.list.ShowHelp())
		return batchCmds(m.blurPreview()), true

	case key.Matches(msg, m.keys.toggleDisplayView):
		return batchCmds(m.blurPreview(), m.toggleDetails()), true

	case key.Matches(msg, m.keys.changeView):
		return batchCmds(m.blurPreview(), m.cycleView()), true

	case key.Matches(msg, m.keys.filterPalette):
		return batchCmds(m.blurPreview(), m.toggleFilterPalette()), true

	case key.Matches(msg, m.keys.switchToDefaultView):
		return batchCmds(m.blurPreview(), m.swapView("default")), true

	case key.Matches(msg, m.keys.switchToUnfulfillView):
		return batchCmds(m.blurPreview(), m.swapView("unfulfilled")), true

	case key.Matches(msg, m.keys.switchToOrphanView):
		return batchCmds(m.blurPreview(), m.swapView("orphan")), true

	case key.Matches(msg, m.keys.switchToArchiveView):
		return batchCmds(m.blurPreview(), m.swapView("archive")), true

	case key.Matches(msg, m.keys.switchToTrashView):
		return batchCmds(m.blurPreview(), m.swapView("trash")), true

	case key.Matches(msg, m.keys.rename):
		m.toggleRename()
		return batchCmds(m.blurPreview()), true

	case key.Matches(msg, m.keys.create):
		m.toggleCreation()
		return batchCmds(m.blurPreview()), true

	case key.Matches(msg, m.keys.copy):
		m.toggleCopy()
		return batchCmds(m.blurPreview()), true

	case key.Matches(msg, m.keys.editInline):
		return batchCmds(m.blurPreview(), m.startInlineEdit()), true

	case key.Matches(msg, m.keys.quickCapture):
		return batchCmds(m.blurPreview(), m.startScratchCapture()), true

	case key.Matches(msg, m.keys.sortByTitle):
		m.sortField = sortByTitle
		return batchCmds(m.blurPreview(), m.refreshSort()), true

	case key.Matches(msg, m.keys.sortBySubdir):
		m.sortField = sortBySubdir
		return batchCmds(m.blurPreview(), m.refreshSort()), true

	case key.Matches(msg, m.keys.sortByModifiedAt):
		m.sortField = sortByModifiedAt
		return batchCmds(m.blurPreview(), m.refreshSort()), true

	case key.Matches(msg, m.keys.sortAscending):
		m.sortOrder = ascending
		return batchCmds(m.blurPreview(), m.refreshSort()), true

	case key.Matches(msg, m.keys.sortAscending):
		m.sortOrder = descending
		return batchCmds(m.blurPreview(), m.refreshSort()), true

	case key.Matches(msg, m.keys.sortDescending):
		m.sortOrder = descending
		return batchCmds(m.blurPreview(), m.refreshSort()), true
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
			renderHelpWithinWidth(width, m.editorInstructions()),
		}

		if msg := m.editor.status; msg != "" {
			sections = append(sections, statusStyle(msg))
		}

		layout := lipgloss.JoinVertical(lipgloss.Left, sections...)
		return appStyle.Render(layout)
	}

	listWidth := m.list.Width()
	if listWidth <= 0 {
		listWidth = m.width / 2
	}

	listHeight := m.list.Height()
	if listHeight <= 0 {
		listHeight = m.height
	}

	listContent := padArea(m.list.View(), listWidth, listHeight)

	list := listStyle.Width(listWidth).Render(listContent)

	sideWidth := m.sidePanelWidth()
	if sideWidth < 0 {
		sideWidth = 0
	}

	if m.copying {
		promptContent := lipgloss.NewStyle().
			Width(sideWidth).
			MaxWidth(sideWidth).
			Height(listHeight).
			MaxHeight(listHeight).
			Padding(0, 2).
			Render(fmt.Sprintf("%s\n\n%s\n\n%s", titleStyle.Render("Choose new name for the copy"), m.inputModel.View(), helpStyle.Render("do not include file extension")))

		textPrompt := textPromptStyle.Render(promptContent)

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
		promptContent := lipgloss.NewStyle().
			Width(sideWidth).
			MaxWidth(sideWidth).
			Height(listHeight).
			MaxHeight(listHeight).
			Padding(0, 2).
			Render(fmt.Sprintf("%s\n\n%s", titleStyle.Render("Rename File"), m.inputModel.View()))

		textPrompt := textPromptStyle.Render(promptContent)

		layout := lipgloss.JoinHorizontal(lipgloss.Top, list, textPrompt)
		return appStyle.Render(layout)
	}

	if m.filtering && m.filterModel != nil {
		paletteContent := lipgloss.NewStyle().
			Width(sideWidth).
			MaxWidth(sideWidth).
			Height(listHeight).
			MaxHeight(listHeight).
			Padding(0, 2).
			Render(m.filterModel.View())

		palette := filterPaletteStyle.Render(paletteContent)

		layout := lipgloss.JoinHorizontal(lipgloss.Top, list, palette)
		return appStyle.Render(layout)
	}

	previewBody := m.previewViewport.View()
	if m.previewViewport.Width <= 0 || m.previewViewport.Height <= 0 {
		previewBody = renderPreviewContent(m.preview, m.previewSummary)
	}

	previewContent := lipgloss.NewStyle().
		Width(sideWidth).
		MaxWidth(sideWidth).
		Height(listHeight).
		MaxHeight(listHeight).
		Render(fmt.Sprintf("%s\n%s", titleStyle.Render("Preview"), previewBody))

	preview := previewStyle.Render(previewContent)

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
		m.setPreviewContent("", "")
		m.previewViewport.GotoTop()
		return nil
	}

	width := m.sidePanelWidth()
	if width <= 0 {
		width = m.width / 2
	}
	height := m.list.Height()

	var override *search.RelatedNotes
	if m.highlights != nil {
		if related, ok := m.highlights.related(selectedPath); ok {
			override = &related
		}
	}

	vault := ""
	if m.state != nil {
		vault = m.state.Vault
	}

	queuePaths := m.queuePaths()

	cache := m.cache
	const previewCutoff = 1000

	if cache == nil {
		req := previewRequest{
			path:     selectedPath,
			width:    width,
			height:   height,
			full:     false,
			cutoff:   previewCutoff,
			index:    m.searchIndex,
			vault:    vault,
			queue:    queuePaths,
			override: override,
		}
		return renderPreviewCmd(req)
	}

	if !force {
		if cached, exists, err := cache.Get(selectedPath); err == nil && exists {
			switch entry := cached.(type) {
			case previewCacheEntry:
				req := previewRequest{
					path:        selectedPath,
					cache:       cache,
					index:       m.searchIndex,
					vault:       vault,
					queue:       queuePaths,
					override:    override,
					preRendered: entry.Markdown,
					complete:    entry.Complete,
				}
				return renderPreviewCmd(req)
			case string:
				req := previewRequest{
					path:        selectedPath,
					cache:       cache,
					index:       m.searchIndex,
					vault:       vault,
					queue:       queuePaths,
					override:    override,
					preRendered: entry,
					complete:    false,
				}
				return renderPreviewCmd(req)
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

	req := previewRequest{
		path:     selectedPath,
		width:    width,
		height:   height,
		cache:    cache,
		full:     false,
		index:    m.searchIndex,
		vault:    vault,
		queue:    queuePaths,
		override: override,
		cutoff:   previewCutoff,
	}
	return renderPreviewCmd(req)
}

type previewRequest struct {
	path        string
	width       int
	height      int
	cache       *cache.Cache
	index       *search.Index
	vault       string
	queue       []string
	override    *search.RelatedNotes
	preRendered string
	complete    bool
	full        bool
	cutoff      int
}

func renderPreviewCmd(req previewRequest) tea.Cmd {
	queueCopy := append([]string(nil), req.queue...)

	var overrideCopy *search.RelatedNotes
	if req.override != nil {
		related := search.RelatedNotes{}
		if len(req.override.Outbound) > 0 {
			related.Outbound = append([]string(nil), req.override.Outbound...)
		}
		if len(req.override.Backlinks) > 0 {
			related.Backlinks = append([]string(nil), req.override.Backlinks...)
		}
		overrideCopy = &related
	}

	return func() tea.Msg {
		rendered := req.preRendered
		complete := req.complete
		var cacheErr error

		if rendered == "" {
			cutoff := req.cutoff
			if req.full {
				cutoff = 0
			}

			var trimmed bool
			rendered, trimmed = utils.RenderMarkdownPreview(req.path, req.width, req.height, cutoff)
			if req.full {
				complete = true
			} else {
				complete = !trimmed
			}

			if req.cache != nil {
				entry := previewCacheEntry{Markdown: rendered, Complete: complete}
				cacheErr = req.cache.Put(req.path, entry)
			}
		}

		ctx := buildPreviewContext(req.path, req.index, queueCopy, overrideCopy)
		summary := formatPreviewContext(ctx, req.vault)

		var background *previewRequest
		if !complete && !req.full {
			backgroundReq := previewRequest{
				path:     req.path,
				width:    req.width,
				height:   req.height,
				cache:    req.cache,
				index:    req.index,
				vault:    req.vault,
				queue:    queueCopy,
				override: overrideCopy,
				full:     true,
			}
			background = &backgroundReq
		}

		return previewLoadedMsg{
			path:       req.path,
			markdown:   rendered,
			summary:    summary,
			cacheErr:   cacheErr,
			complete:   complete,
			background: background,
		}
	}
}

func (m *NoteListModel) queuePaths() []string {
	if len(m.reviewQueue) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(m.reviewQueue))
	paths := make([]string, 0, len(m.reviewQueue))
	for _, item := range m.reviewQueue {
		cleaned := strings.TrimSpace(item.Path)
		if cleaned == "" {
			continue
		}
		normalized := pathutil.NormalizePath(cleaned)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		paths = append(paths, normalized)
	}
	return paths
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
	return sequenceWithClear(tea.Batch(cmd, m.handlePreview(true)))
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
	m.allItems = append([]list.Item(nil), sortedItems...)
	m.rebuildSearch(files)
	m.updateFilterInventory()
	m.syncFilterPalette()
	m.updateFilterStatus()
	return m.applyActiveFilters()
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
	items := castToListItems(m.allItems)
	sortedItems := sortItems(items, m.sortField, m.sortOrder)
	attachHighlightStore(sortedItems, m.highlights)
	m.allItems = append([]list.Item(nil), sortedItems...)
	m.list.ResetSelected()
	cmd := m.applyActiveFilters()
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

func padArea(view string, width, height int) string {
	if width <= 0 && height <= 0 {
		return view
	}

	lines := strings.Split(view, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}

	for i, line := range lines {
		if width <= 0 {
			continue
		}

		lineWidth := lipgloss.Width(line)
		if lineWidth < width {
			lines[i] = line + strings.Repeat(" ", width-lineWidth)
		}
	}

	if height > len(lines) {
		pad := ""
		if width > 0 {
			pad = strings.Repeat(" ", width)
		}

		for len(lines) < height {
			lines = append(lines, pad)
		}
	}

	return strings.Join(lines, "\n")
}

func (m *NoteListModel) setPreviewContent(markdown, summary string) {
	m.preview = markdown
	m.previewSummary = summary
	m.previewViewport.SetContent(renderPreviewContent(markdown, summary))
}

func renderPreviewContent(markdown, summary string) string {
	trimmedSummary := strings.TrimSpace(summary)
	content := markdown
	trimmedContent := strings.TrimSpace(content)
	if trimmedSummary != "" {
		renderedSummary := statusStyle(trimmedSummary)
		if trimmedContent != "" {
			return fmt.Sprintf("%s\n\n%s", renderedSummary, content)
		}
		return renderedSummary
	}
	return content
}

func (m *NoteListModel) previewHasFocus() bool {
	if m == nil {
		return false
	}
	return m.previewFocused
}

func (m *NoteListModel) focusPreview() tea.Cmd {
	if m == nil {
		return nil
	}
	if m.previewFocused {
		return nil
	}
	m.previewFocused = true
	return focusViewport(&m.previewViewport)
}

func (m *NoteListModel) blurPreview() tea.Cmd {
	if m == nil {
		return nil
	}
	if !m.previewFocused {
		return blurViewport(&m.previewViewport)
	}
	m.previewFocused = false
	return blurViewport(&m.previewViewport)
}

func focusViewport(v *viewport.Model) tea.Cmd {
	if v == nil {
		return nil
	}
	setViewportBindingsEnabled(&v.KeyMap, true)
	v.MouseWheelEnabled = true
	return nil
}

func blurViewport(v *viewport.Model) tea.Cmd {
	if v == nil {
		return nil
	}
	setViewportBindingsEnabled(&v.KeyMap, false)
	v.MouseWheelEnabled = false
	return nil
}

func setViewportBindingsEnabled(m *viewport.KeyMap, enabled bool) {
	if m == nil {
		return
	}
	m.PageDown.SetEnabled(enabled)
	m.PageUp.SetEnabled(enabled)
	m.HalfPageDown.SetEnabled(enabled)
	m.HalfPageUp.SetEnabled(enabled)
	m.Down.SetEnabled(enabled)
	m.Up.SetEnabled(enabled)
}

func isPreviewScrollKey(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown, tea.KeySpace, tea.KeyCtrlU, tea.KeyCtrlD:
		return true
	case tea.KeyRunes:
		if len(msg.Runes) != 1 {
			return false
		}
		switch msg.Runes[0] {
		case 'j', 'k', 'f', 'b', 'd', 'u':
			return true
		}
	}
	return false
}

func (m *NoteListModel) toggleTitleBar() {
	v := !m.list.ShowTitle()
	m.list.SetShowTitle(v)
	m.list.SetShowFilter(v)
	m.list.SetFilteringEnabled(v)
}

func (m *NoteListModel) toggleDetails() tea.Cmd {
	blur := m.blurPreview()
	m.showDetails = !m.showDetails
	cmd := m.refreshItems()
	m.list.ResetSelected()
	return batchCmds(blur, cmd, m.handlePreview(true))
}

func (m *NoteListModel) cycleView() tea.Cmd {
	next := m.state.ViewManager.NextView(m.viewName)
	return m.applyView(next)
}

func (m NoteListModel) sidePanelWidth() int {
	if m.previewWidth > 0 {
		return m.previewWidth
	}

	if m.width <= 0 {
		return 0
	}

	fallback := m.width / 2
	if fallback < 0 {
		return 0
	}

	return fallback
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

	return batchCmds(m.blurPreview(), m.refresh())
}

func sequenceWithClear(cmd tea.Cmd) tea.Cmd {
	return tea.Sequence(tea.ClearScreen, cmd)
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
