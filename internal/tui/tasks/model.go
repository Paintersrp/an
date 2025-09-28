package tasks

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"

	services "github.com/Paintersrp/an/internal/services/tasks"
	"github.com/Paintersrp/an/internal/state"
)

type Model struct {
	service    *services.Service
	state      *state.State
	list       list.Model
	keys       keyMap
	status     string
	width      int
	height     int
	items      []services.Item
	filters    filterState
	owners     []string
	projects   []string
	priorities []string
}

type keyMap struct {
	open          key.Binding
	toggle        key.Binding
	refresh       key.Binding
	cycleDue      key.Binding
	cycleOwner    key.Binding
	cycleProject  key.Binding
	cyclePriority key.Binding
	clearFilters  key.Binding
}

type listItem struct {
	item services.Item
}

type dueMode int

const (
	dueAll dueMode = iota
	dueAgenda
	dueUpcoming
	dueOverdue
	dueUnscheduled
	dueScheduled
)

type filterState struct {
	due         dueMode
	ownerIdx    int
	projectIdx  int
	priorityIdx int
}

func NewModel(s *state.State) (*Model, error) {
	if s == nil || s.Handler == nil {
		return nil, fmt.Errorf("task model requires a configured state handler")
	}

	svc := services.NewService(s.Handler)
	items, err := svc.List()
	if err != nil {
		return nil, err
	}

	delegate := list.NewDefaultDelegate()
	lm := list.New(nil, delegate, 0, 0)
	lm.Title = "Tasks"
	lm.DisableQuitKeybindings()

	model := &Model{
		service: svc,
		state:   s,
		list:    lm,
		keys:    newKeyMap(),
	}

	if cmd := model.setItems(items); cmd != nil {
		if msg := cmd(); msg != nil {
			model.list, _ = model.list.Update(msg)
		}
	}
	return model, nil
}

func newKeyMap() keyMap {
	return keyMap{
		open: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "open note"),
		),
		toggle: key.NewBinding(
			key.WithKeys("space"),
			key.WithHelp("space", "toggle"),
		),
		refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		cycleDue: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "cycle due filter"),
		),
		cycleOwner: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "cycle owner"),
		),
		cycleProject: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "cycle project"),
		),
		cyclePriority: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "cycle priority"),
		),
		clearFilters: key.NewBinding(
			key.WithKeys("ctrl+0"),
			key.WithHelp("ctrl+0", "clear filters"),
		),
	}
}

func toListItems(items []services.Item) []list.Item {
	ls := make([]list.Item, 0, len(items))
	for _, item := range items {
		ls = append(ls, listItem{item: item})
	}
	return ls
}

func (i listItem) Title() string {
	prefix := "[ ]"
	if i.item.Completed {
		prefix = "[x]"
	}
	if due := formatDuePrefix(i.item); due != "" {
		return fmt.Sprintf("%s %s %s", prefix, due, i.item.Content)
	}
	return fmt.Sprintf("%s %s", prefix, i.item.Content)
}

func (i listItem) Description() string {
	rel := i.item.RelPath
	if rel == "" {
		rel = filepath.Base(i.item.Path)
	}
	parts := make([]string, 0, 5)
	if desc := formatMetadataSummary(i.item); desc != "" {
		parts = append(parts, desc)
	}
	parts = append(parts, fmt.Sprintf("%s:%d", rel, i.item.Line))
	return strings.Join(parts, " | ")
}

func (i listItem) FilterValue() string {
	fields := []string{i.item.Content, i.item.Owner, i.item.Project, i.item.Priority}
	fields = append(fields, i.item.References...)
	return strings.Join(fields, " ")
}

func (m *Model) Init() tea.Cmd {
	if m.state == nil {
		return nil
	}

	if cmd := m.state.IndexHeartbeatCmd(); cmd != nil {
		return cmd
	}
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-2)
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.toggle):
			return m.handleToggle()
		case key.Matches(msg, m.keys.open):
			return m.handleOpen()
		case key.Matches(msg, m.keys.refresh):
			return m, m.refresh()
		case key.Matches(msg, m.keys.cycleDue):
			m.advanceDueFilter()
			return m, m.applyFilters()
		case key.Matches(msg, m.keys.cycleOwner):
			m.advanceOwnerFilter()
			return m, m.applyFilters()
		case key.Matches(msg, m.keys.cycleProject):
			m.advanceProjectFilter()
			return m, m.applyFilters()
		case key.Matches(msg, m.keys.cyclePriority):
			m.advancePriorityFilter()
			return m, m.applyFilters()
		case key.Matches(msg, m.keys.clearFilters):
			m.resetFilters()
			return m, m.applyFilters()
		}
	case state.IndexStatsMsg:
		if m.state != nil && m.state.Watcher != nil {
			return m, m.state.Watcher.Start()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	pinned := ""
	if m.state != nil && m.state.Config != nil {
		ws := m.state.Config.MustWorkspace()
		if ws != nil && ws.PinnedTaskFile != "" {
			pinned = ws.PinnedTaskFile
		}
	}

	listWidth := m.list.Width()
	if listWidth <= 0 {
		listWidth = m.width
	}

	view := m.list.View()
	if root := m.rootStatusLine(); root != "" {
		originalTitle := m.list.Title
		gap := "  "
		if originalTitle == "" {
			gap = ""
		}
		if suffix := rootStatusSuffix(view, listWidth, root, gap); suffix != "" {
			m.list.Title = originalTitle + gap + suffix
			view = m.list.View()
			m.list.Title = originalTitle
		}
	}

	var statuses []string
	if pinned != "" {
		statuses = append(statuses, fmt.Sprintf("Pinned: %s", pinned))
	}
	if summary := m.filterSummary(); summary != "" {
		statuses = append(statuses, summary)
	}
	if m.status != "" {
		statuses = append(statuses, m.status)
	}

	if len(statuses) > 0 {
		return fmt.Sprintf("%s\n%s", view, strings.Join(statuses, "\n"))
	}
	return view
}

func (m *Model) rootStatusLine() string {
	if m.state == nil || m.state.RootStatus == nil {
		return ""
	}
	return m.state.RootStatus.Value()
}

func rootStatusSuffix(view string, width int, status, gap string) string {
	if status == "" || width <= 0 {
		return ""
	}

	lines := strings.Split(view, "\n")
	if len(lines) == 0 {
		return ""
	}

	line := lines[0]
	available := width - lipgloss.Width(line)
	if available <= 0 {
		return ""
	}

	if gap != "" {
		available -= lipgloss.Width(gap)
		if available <= 0 {
			return ""
		}
	}

	trimmed := truncate.StringWithTail(status, uint(available), "")
	if trimmed == "" {
		return ""
	}

	return trimmed
}

func (m *Model) handleOpen() (tea.Model, tea.Cmd) {
	item, ok := m.list.SelectedItem().(listItem)
	if !ok {
		return m, nil
	}

	if err := m.service.Open(item.item.Path); err != nil {
		m.status = fmt.Sprintf("open failed: %v", err)
	} else {
		m.status = fmt.Sprintf("opened %s", item.Description())
	}
	return m, nil
}

func (m *Model) handleToggle() (tea.Model, tea.Cmd) {
	item, ok := m.list.SelectedItem().(listItem)
	if !ok {
		return m, nil
	}

	completed, err := m.service.Toggle(item.item.Path, item.item.Line)
	if err != nil {
		m.status = fmt.Sprintf("toggle failed: %v", err)
		return m, nil
	}

	if completed {
		m.status = "marked complete"
	} else {
		m.status = "marked incomplete"
	}

	return m, m.refresh()
}

func (m *Model) refresh() tea.Cmd {
	items, err := m.service.List()
	if err != nil {
		m.status = fmt.Sprintf("refresh failed: %v", err)
		return nil
	}

	return m.setItems(items)
}

func uniqueSorted(items []services.Item, selector func(services.Item) string) []string {
	seen := make(map[string]struct{})
	values := make([]string, 0)
	for _, item := range items {
		val := strings.TrimSpace(selector(item))
		if val == "" {
			continue
		}
		normalized := strings.ToLower(val)
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		values = append(values, val)
	}
	sort.Slice(values, func(i, j int) bool {
		return strings.ToLower(values[i]) < strings.ToLower(values[j])
	})
	return values
}

func pickValue(values []string, idx int) string {
	if idx <= 0 {
		return ""
	}
	actual := idx - 1
	if actual < 0 || actual >= len(values) {
		return ""
	}
	return values[actual]
}

func advanceIndex(current, length int) int {
	if length == 0 {
		return 0
	}
	current++
	if current > length {
		return 0
	}
	return current
}

func truncateToDay(t time.Time) time.Time {
	loc := t.Location()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func formatDuePrefix(item services.Item) string {
	if item.Due == nil {
		return ""
	}
	due := item.Due.In(time.Now().Location())
	today := truncateToDay(time.Now())
	switch {
	case truncateToDay(due).Before(today):
		return "(OVERDUE)"
	case truncateToDay(due).Equal(today):
		return "(due today)"
	default:
		return fmt.Sprintf("(due %s)", due.Format("Jan 02"))
	}
}

func formatMetadataSummary(item services.Item) string {
	var parts []string
	if item.Owner != "" {
		parts = append(parts, fmt.Sprintf("owner %s", item.Owner))
	}
	if item.Priority != "" {
		parts = append(parts, fmt.Sprintf("priority %s", item.Priority))
	}
	if item.Project != "" {
		parts = append(parts, fmt.Sprintf("project %s", item.Project))
	}
	if item.Scheduled != nil {
		parts = append(parts, fmt.Sprintf("scheduled %s", item.Scheduled.Format("Jan 02")))
	}
	if len(item.References) > 0 {
		parts = append(parts, "refs "+strings.Join(item.References, ", "))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " • ")
}

func (m *Model) setItems(items []services.Item) tea.Cmd {
	m.items = items
	m.owners = uniqueSorted(items, func(it services.Item) string { return it.Owner })
	m.projects = uniqueSorted(items, func(it services.Item) string { return it.Project })
	m.priorities = uniqueSorted(items, func(it services.Item) string { return it.Priority })
	if m.filters.ownerIdx > len(m.owners) {
		m.filters.ownerIdx = 0
	}
	if m.filters.projectIdx > len(m.projects) {
		m.filters.projectIdx = 0
	}
	if m.filters.priorityIdx > len(m.priorities) {
		m.filters.priorityIdx = 0
	}
	return m.applyFilters()
}

func (m *Model) applyFilters() tea.Cmd {
	filtered := make([]services.Item, 0, len(m.items))
	now := time.Now()
	owner := pickValue(m.owners, m.filters.ownerIdx)
	project := pickValue(m.projects, m.filters.projectIdx)
	priority := pickValue(m.priorities, m.filters.priorityIdx)
	for _, item := range m.items {
		if owner != "" && !strings.EqualFold(item.Owner, owner) {
			continue
		}
		if project != "" && !strings.EqualFold(item.Project, project) {
			continue
		}
		if priority != "" && !strings.EqualFold(item.Priority, priority) {
			continue
		}
		if !m.matchesDueFilter(item, now) {
			continue
		}
		filtered = append(filtered, item)
	}
	return m.list.SetItems(toListItems(filtered))
}

func (m *Model) matchesDueFilter(item services.Item, now time.Time) bool {
	switch m.filters.due {
	case dueAll:
		return true
	case dueAgenda:
		if item.Due == nil {
			return false
		}
		today := truncateToDay(now)
		due := truncateToDay(item.Due.In(now.Location()))
		return !due.After(today)
	case dueUpcoming:
		if item.Due == nil {
			return false
		}
		today := truncateToDay(now)
		due := truncateToDay(item.Due.In(now.Location()))
		return due.After(today)
	case dueOverdue:
		if item.Due == nil {
			return false
		}
		today := truncateToDay(now)
		due := truncateToDay(item.Due.In(now.Location()))
		return due.Before(today)
	case dueUnscheduled:
		return item.Due == nil
	case dueScheduled:
		return item.Scheduled != nil
	default:
		return true
	}
}

func (m *Model) advanceDueFilter() {
	m.filters.due++
	if m.filters.due > dueScheduled {
		m.filters.due = dueAll
	}
}

func (m *Model) advanceOwnerFilter() {
	m.filters.ownerIdx = advanceIndex(m.filters.ownerIdx, len(m.owners))
}

func (m *Model) advanceProjectFilter() {
	m.filters.projectIdx = advanceIndex(m.filters.projectIdx, len(m.projects))
}

func (m *Model) advancePriorityFilter() {
	m.filters.priorityIdx = advanceIndex(m.filters.priorityIdx, len(m.priorities))
}

func (m *Model) resetFilters() {
	m.filters = filterState{}
}

func (m *Model) filterSummary() string {
	var parts []string
	switch m.filters.due {
	case dueAgenda:
		parts = append(parts, "due: agenda")
	case dueUpcoming:
		parts = append(parts, "due: upcoming")
	case dueOverdue:
		parts = append(parts, "due: overdue")
	case dueUnscheduled:
		parts = append(parts, "due: unscheduled")
	case dueScheduled:
		parts = append(parts, "due: scheduled")
	}
	if owner := pickValue(m.owners, m.filters.ownerIdx); owner != "" {
		parts = append(parts, fmt.Sprintf("owner: %s", owner))
	}
	if project := pickValue(m.projects, m.filters.projectIdx); project != "" {
		parts = append(parts, fmt.Sprintf("project: %s", project))
	}
	if priority := pickValue(m.priorities, m.filters.priorityIdx); priority != "" {
		parts = append(parts, fmt.Sprintf("priority: %s", priority))
	}
	if len(parts) == 0 {
		return ""
	}
	return "Filters → " + strings.Join(parts, ", ")
}
