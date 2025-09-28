package review

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	reviewsvc "github.com/Paintersrp/an/internal/review"
	"github.com/Paintersrp/an/internal/search"
	indexsvc "github.com/Paintersrp/an/internal/services/index"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/templater"
	"github.com/Paintersrp/an/internal/tui/textarea"
)

type Model struct {
	state          *state.State
	queue          []reviewsvc.ResurfaceItem
	graph          reviewsvc.Graph
	manifest       templater.TemplateManifest
	responses      map[string]string
	editor         *textarea.Model
	keys           keyMap
	width          int
	height         int
	step           int
	mode           reviewMode
	showGraph      bool
	status         string
	loading        bool
	ready          bool
	confirmingSave bool
}

type keyMap struct {
	refresh     key.Binding
	toggleGraph key.Binding
	nextStep    key.Binding
	prevStep    key.Binding
	complete    key.Binding
	daily       key.Binding
	weekly      key.Binding
	retro       key.Binding
	exit        key.Binding
}

type reviewMode struct {
	Name     string
	Template string
	Key      string
}

type queueLoadedMsg struct {
	queue []reviewsvc.ResurfaceItem
	graph reviewsvc.Graph
	err   error
}

type reviewSavedMsg struct {
	path string
	err  error
}

// ExitRequestedMsg indicates that the user requested to leave the review view.
type ExitRequestedMsg struct{}

var modes = []reviewMode{
	{Name: "Daily", Template: "review-daily", Key: "daily"},
	{Name: "Weekly", Template: "review-weekly", Key: "weekly"},
	{Name: "Retro", Template: "review-retro", Key: "retro"},
}

const (
	defaultQueueLimit = 12
	editorMinHeight   = 5
)

func NewModel(st *state.State) (*Model, error) {
	if st == nil || st.Templater == nil || st.Config == nil {
		return nil, fmt.Errorf("review model requires configured state dependencies")
	}

	editor := textarea.New(0, 0)
	mode := modes[0]
	manifest, err := st.Templater.Manifest(mode.Template)
	if err != nil {
		return nil, fmt.Errorf("load %s manifest: %w", mode.Template, err)
	}

	model := &Model{
		state:     st,
		manifest:  manifest,
		responses: make(map[string]string),
		editor:    editor,
		keys:      newKeyMap(),
		mode:      mode,
		showGraph: true,
	}
	model.applyCurrentFieldDefaults()
	return model, nil
}

func newKeyMap() keyMap {
	return keyMap{
		refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh queue"),
		),
		toggleGraph: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "toggle graph"),
		),
		nextStep: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "next step"),
		),
		prevStep: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("ctrl+p", "previous step"),
		),
		complete: key.NewBinding(
			key.WithKeys("ctrl+enter"),
			key.WithHelp("ctrl+enter", "complete checklist"),
		),
		daily: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "daily mode"),
		),
		weekly: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "weekly mode"),
		),
		retro: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "retro mode"),
		),
		exit: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "exit review"),
		),
	}
}

func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{}
	if m.editor != nil {
		cmds = append(cmds, m.editor.Init())
	}
	cmds = append(cmds, m.refreshQueue())
	return tea.Batch(cmds...)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeEditor()
		return m, nil
	case queueLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.status = fmt.Sprintf("refresh failed: %v", msg.err)
			return m, nil
		}
		m.queue = msg.queue
		m.graph = msg.graph
		m.ready = true
		if len(m.queue) == 0 {
			m.status = "No notes are due for resurfacing."
		} else {
			m.status = fmt.Sprintf("Loaded resurfacing queue (%d items)", len(m.queue))
		}
		return m, nil
	case reviewSavedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Failed to save review log: %v", msg.err)
		} else {
			m.status = fmt.Sprintf("Review log saved: %s", m.relativePath(msg.path))
		}
		m.confirmingSave = false
		return m, nil
	case tea.KeyMsg:
		if m.confirmingSave && msg.Type == tea.KeyEsc {
			m.confirmingSave = false
			m.status = "Canceled review save."
			return m, nil
		}
		if handled, cmd := m.handleKeys(msg); handled {
			return m, cmd
		}
	}

	if m.editor != nil {
		if cmd := m.editor.Update(msg); cmd != nil {
			return m, cmd
		}
	}
	return m, nil
}

func (m *Model) View() string {
	if !m.ready && m.loading {
		return "Loading review data..."
	}

	sections := []string{
		m.renderHeader(),
		m.renderQueue(),
	}
	if m.showGraph {
		if graph := m.renderGraph(); graph != "" {
			sections = append(sections, graph)
		}
	}
	sections = append(sections, m.renderChecklist())
	if m.status != "" {
		sections = append(sections, statusStyle.Render(m.status))
	}
	return strings.Join(filterEmpty(sections), "\n\n")
}

func (m *Model) handleKeys(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.refresh):
		return true, m.refreshQueue()
	case key.Matches(msg, m.keys.toggleGraph):
		m.showGraph = !m.showGraph
		if m.showGraph && len(m.graph.Nodes) == 0 && len(m.queue) > 0 {
			// Recompute graph lazily when toggled on.
			return true, m.computeGraph()
		}
		return true, nil
	case key.Matches(msg, m.keys.nextStep):
		m.advanceStep(1)
		return true, nil
	case key.Matches(msg, m.keys.prevStep):
		m.advanceStep(-1)
		return true, nil
	case key.Matches(msg, m.keys.complete):
		m.persistCurrentResponse()
		if !m.confirmingSave {
			m.confirmingSave = true
			m.status = "Press ctrl+enter again to save the review log, or esc to cancel."
			return true, nil
		}
		m.confirmingSave = false
		m.status = "Saving review log..."
		responses := cloneStringMap(m.responses)
		queue := append([]reviewsvc.ResurfaceItem(nil), m.queue...)
		return true, m.saveReviewLog(responses, m.manifest, queue, time.Now().UTC())
	case key.Matches(msg, m.keys.daily):
		return true, m.switchMode(modes[0])
	case key.Matches(msg, m.keys.weekly):
		return true, m.switchMode(modes[1])
	case key.Matches(msg, m.keys.retro):
		return true, m.switchMode(modes[2])
	case key.Matches(msg, m.keys.exit):
		return true, exitRequestedCmd
	}
	return false, nil
}

func (m *Model) switchMode(mode reviewMode) tea.Cmd {
	if mode.Template == m.mode.Template {
		return nil
	}
	manifest, err := m.state.Templater.Manifest(mode.Template)
	if err != nil {
		m.status = fmt.Sprintf("load %s manifest failed: %v", mode.Template, err)
		return nil
	}
	m.mode = mode
	m.manifest = manifest
	m.responses = make(map[string]string)
	m.step = 0
	m.applyCurrentFieldDefaults()
	m.status = fmt.Sprintf("Switched to %s review", mode.Name)
	return nil
}

func (m *Model) renderHeader() string {
	title := lipgloss.NewStyle().Bold(true).Render("Review")
	info := fmt.Sprintf("Mode: %s — Press 1/2/3 to switch modes", m.mode.Name)
	if m.loading {
		info += " (refreshing...)"
	}
	return fmt.Sprintf("%s\n%s", title, info)
}

func (m *Model) renderQueue() string {
	if len(m.queue) == 0 {
		return "Resurfacing queue: No notes are currently due."
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Resurfacing queue (%d items):\n", len(m.queue)))
	limit := len(m.queue)
	if limit > defaultQueueLimit {
		limit = defaultQueueLimit
	}
	for i := 0; i < limit; i++ {
		item := m.queue[i]
		fmt.Fprintf(&b, "%2d. %s — last touched %s (%s)\n", i+1, m.relativePath(item.Path), humanizeAge(item.Age), item.Bucket)
	}
	if len(m.queue) > limit {
		fmt.Fprintf(&b, "…and %d more", len(m.queue)-limit)
	}
	return strings.TrimSpace(b.String())
}

func (m *Model) renderGraph() string {
	if len(m.graph.Nodes) == 0 {
		return ""
	}
	var paths []string
	for path := range m.graph.Nodes {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var b strings.Builder
	b.WriteString("Backlink graph preview:\n")
	for _, path := range paths {
		node := m.graph.Nodes[path]
		fmt.Fprintf(&b, "- %s\n", m.relativePath(node.Path))
		if len(node.Backlinks) > 0 {
			fmt.Fprintf(&b, "    ← %s\n", m.joinPaths(node.Backlinks))
		}
		if len(node.Outbound) > 0 {
			fmt.Fprintf(&b, "    → %s\n", m.joinPaths(node.Outbound))
		}
	}
	return strings.TrimSpace(b.String())
}

func (m *Model) renderChecklist() string {
	if len(m.manifest.Fields) == 0 {
		return "Checklist: no steps configured for this mode."
	}
	field := m.manifest.Fields[m.step]
	title := field.Label
	if title == "" {
		title = humanizeKey(field.Key)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s checklist — Step %d of %d: %s\n", m.manifest.Name, m.step+1, len(m.manifest.Fields), title)
	if field.Prompt != "" {
		fmt.Fprintf(&b, "%s\n", field.Prompt)
	}
	if len(field.Options) > 0 {
		fmt.Fprintf(&b, "Options: %s\n", strings.Join(field.Options, ", "))
	}
	if len(field.Defaults) > 0 {
		fmt.Fprintf(&b, "Suggested focus tags: %s\n", strings.Join(field.Defaults, ", "))
		suggestions := reviewsvc.FilterQueue(m.queue, field.Defaults, nil)
		if len(suggestions) > 0 {
			max := suggestions
			if len(max) > 3 {
				max = suggestions[:3]
			}
			fmt.Fprintln(&b, "Related resurfacing candidates:")
			for _, item := range max {
				fmt.Fprintf(&b, "  • %s (%s)\n", m.relativePath(item.Path), item.ModifiedAt.Format("2006-01-02"))
			}
		}
	}
	if m.editor != nil {
		b.WriteString("\n")
		b.WriteString(m.editor.View())
	}
	b.WriteString("\nControls: ctrl+n next · ctrl+p previous · ctrl+enter complete · esc exit")
	return strings.TrimSpace(b.String())
}

func (m *Model) refreshQueue() tea.Cmd {
	if m == nil || m.state == nil {
		return nil
	}
	m.loading = true
	m.status = "Refreshing resurfacing queue..."
	state := m.state
	return func() tea.Msg {
		queue, graph, err := buildArtifacts(state)
		return queueLoadedMsg{queue: queue, graph: graph, err: err}
	}
}

func (m *Model) computeGraph() tea.Cmd {
	if len(m.queue) == 0 {
		return nil
	}
	state := m.state
	queue := append([]reviewsvc.ResurfaceItem(nil), m.queue...)
	return func() tea.Msg {
		graph, err := buildGraph(state, queue)
		if err != nil {
			return queueLoadedMsg{queue: queue, graph: reviewsvc.Graph{}, err: err}
		}
		return queueLoadedMsg{queue: queue, graph: graph, err: nil}
	}
}

func (m *Model) advanceStep(delta int) {
	if len(m.manifest.Fields) == 0 {
		return
	}
	m.persistCurrentResponse()
	next := m.step + delta
	if next < 0 {
		next = 0
	}
	if next >= len(m.manifest.Fields) {
		next = len(m.manifest.Fields) - 1
	}
	if next == m.step {
		return
	}
	m.step = next
	m.applyCurrentFieldDefaults()
}

func (m *Model) applyCurrentFieldDefaults() {
	if len(m.manifest.Fields) == 0 {
		if m.editor != nil {
			m.editor.SetValue("")
		}
		return
	}
	field := m.manifest.Fields[m.step]
	value := m.responses[field.Key]
	if value == "" {
		value = field.Default
	}
	if m.editor != nil {
		m.editor.SetValue(value)
		m.editor.CursorEnd()
	}
}

func (m *Model) persistCurrentResponse() {
	if len(m.manifest.Fields) == 0 || m.editor == nil {
		return
	}
	field := m.manifest.Fields[m.step]
	m.responses[field.Key] = strings.TrimSpace(m.editor.Value())
}

func (m *Model) saveReviewLog(responses map[string]string, manifest templater.TemplateManifest, queue []reviewsvc.ResurfaceItem, ts time.Time) tea.Cmd {
	state := m.state
	return func() tea.Msg {
		path, err := persistReviewLog(state, manifest, responses, queue, ts)
		return reviewSavedMsg{path: path, err: err}
	}
}

func (m *Model) resizeEditor() {
	if m.editor == nil {
		return
	}
	width := m.width
	if width <= 0 {
		width = 80
	}
	height := m.height / 4
	if height < editorMinHeight {
		height = editorMinHeight
	}
	m.editor.SetSize(width, height)
}

func (m *Model) relativePath(path string) string {
	if m.state == nil || m.state.Vault == "" {
		return path
	}
	rel, err := filepath.Rel(m.state.Vault, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func (m *Model) joinPaths(paths []string) string {
	converted := make([]string, 0, len(paths))
	for _, p := range paths {
		converted = append(converted, m.relativePath(p))
	}
	return strings.Join(converted, ", ")
}

func buildArtifacts(st *state.State) ([]reviewsvc.ResurfaceItem, reviewsvc.Graph, error) {
	queue, idx, err := buildQueue(st)
	if err != nil {
		return nil, reviewsvc.Graph{}, err
	}
	graph := reviewsvc.Graph{}
	if len(queue) > 0 {
		seeds := make([]string, len(queue))
		for i, item := range queue {
			seeds[i] = item.Path
		}
		graph = reviewsvc.BuildBacklinkGraph(idx, seeds)
	}
	return queue, graph, nil
}

func buildQueue(st *state.State) ([]reviewsvc.ResurfaceItem, *search.Index, error) {
	if st == nil || st.Config == nil {
		return nil, nil, errors.New("state is not configured")
	}
	ws := st.Config.MustWorkspace()
	searchCfg := search.Config{
		EnableBody:     ws.Search.EnableBody,
		IgnoredFolders: append([]string(nil), ws.Search.IgnoredFolders...),
	}
	query := search.Query{
		Tags:     append([]string(nil), ws.Search.DefaultTagFilters...),
		Metadata: cloneMetadata(ws.Search.DefaultMetadataFilters),
	}

	if st.Index != nil {
		snapshot, err := st.Index.AcquireSnapshot()
		if err == nil && snapshot != nil {
			queue := reviewsvc.BuildResurfaceQueue(snapshot, reviewsvc.ResurfaceOptions{
				Now:        time.Now(),
				MinimumAge: 0,
				Limit:      defaultQueueLimit,
				Buckets:    reviewsvc.DefaultBuckets(),
				Query:      query,
			})
			return queue, snapshot, nil
		}
		if err != nil && !errors.Is(err, indexsvc.ErrUnavailable) && !errors.Is(err, indexsvc.ErrClosed) {
			return nil, nil, err
		}
	}

	paths, err := collectNotePaths(st.Vault, searchCfg.IgnoredFolders)
	if err != nil {
		return nil, nil, err
	}
	idx := search.NewIndex(st.Vault, searchCfg)
	if err := idx.Build(paths); err != nil {
		return nil, nil, fmt.Errorf("build search index: %w", err)
	}
	queue := reviewsvc.BuildResurfaceQueue(idx, reviewsvc.ResurfaceOptions{
		Now:        time.Now(),
		MinimumAge: 0,
		Limit:      defaultQueueLimit,
		Buckets:    reviewsvc.DefaultBuckets(),
		Query:      query,
	})
	return queue, idx, nil
}

func buildGraph(st *state.State, queue []reviewsvc.ResurfaceItem) (reviewsvc.Graph, error) {
	if st == nil {
		return reviewsvc.Graph{}, errors.New("state is not configured")
	}
	if st.Index != nil {
		snapshot, err := st.Index.AcquireSnapshot()
		if err == nil && snapshot != nil {
			if len(queue) == 0 {
				return reviewsvc.Graph{}, nil
			}
			seeds := make([]string, len(queue))
			for i, item := range queue {
				seeds[i] = item.Path
			}
			return reviewsvc.BuildBacklinkGraph(snapshot, seeds), nil
		}
		if err != nil && !errors.Is(err, indexsvc.ErrUnavailable) && !errors.Is(err, indexsvc.ErrClosed) {
			return reviewsvc.Graph{}, err
		}
	}

	searchCfg := search.Config{}
	if st.Config != nil {
		ws := st.Config.MustWorkspace()
		searchCfg.EnableBody = ws.Search.EnableBody
		searchCfg.IgnoredFolders = append([]string(nil), ws.Search.IgnoredFolders...)
	}
	paths, err := collectNotePaths(st.Vault, searchCfg.IgnoredFolders)
	if err != nil {
		return reviewsvc.Graph{}, err
	}
	idx := search.NewIndex(st.Vault, searchCfg)
	if err := idx.Build(paths); err != nil {
		return reviewsvc.Graph{}, fmt.Errorf("build search index: %w", err)
	}
	if len(queue) == 0 {
		return reviewsvc.Graph{}, nil
	}
	seeds := make([]string, len(queue))
	for i, item := range queue {
		seeds[i] = item.Path
	}
	return reviewsvc.BuildBacklinkGraph(idx, seeds), nil
}

func collectNotePaths(root string, ignored []string) ([]string, error) {
	normalized := make(map[string]struct{}, len(ignored))
	for _, dir := range ignored {
		normalized[strings.ToLower(dir)] = struct{}{}
	}

	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := strings.ToLower(d.Name())
			if strings.HasPrefix(name, ".") && path != root {
				return filepath.SkipDir
			}
			if _, skip := normalized[name]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func cloneMetadata(values map[string][]string) map[string][]string {
	if len(values) == 0 {
		return make(map[string][]string)
	}
	cloned := make(map[string][]string, len(values))
	for key, vals := range values {
		cloned[key] = append([]string(nil), vals...)
	}
	return cloned
}

func humanizeAge(age time.Duration) string {
	if age < time.Hour {
		minutes := int(age.Minutes())
		if minutes <= 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	}
	days := int(age.Hours() / 24)
	if days == 0 {
		hours := int(age.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

func humanizeKey(key string) string {
	replaced := strings.ReplaceAll(key, "-", " ")
	replaced = strings.ReplaceAll(replaced, "_", " ")
	if replaced == "" {
		return ""
	}
	parts := strings.Fields(replaced)
	for i, part := range parts {
		if part == "" {
			continue
		}
		r := []rune(part)
		if len(r) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(string(r[0])) + string(r[1:])
	}
	return strings.Join(parts, " ")
}

func filterEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return make(map[string]string)
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

var statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

var exitRequestedCmd tea.Cmd = func() tea.Msg { return ExitRequestedMsg{} }
