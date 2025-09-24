package journal

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	svc "github.com/Paintersrp/an/internal/services/journal"
	"github.com/Paintersrp/an/internal/state"
)

type Model struct {
	service *svc.Service
	state   *state.State
	list    list.Model
	keys    keyMap
	active  string
	status  string
	width   int
	height  int
}

type keyMap struct {
	open    key.Binding
	today   key.Binding
	refresh key.Binding
	day     key.Binding
	week    key.Binding
	month   key.Binding
	year    key.Binding
}

type entryItem struct {
	entry svc.Entry
}

func NewModel(s *state.State) (*Model, error) {
	if s == nil || s.Handler == nil || s.Templater == nil {
		return nil, fmt.Errorf("journal model requires configured state dependencies")
	}

	service := svc.NewService(s.Templater, s.Handler)
	entries, err := service.List("day")
	if err != nil {
		return nil, err
	}

	delegate := list.NewDefaultDelegate()
	lm := list.New(toListItems(entries), delegate, 0, 0)
	lm.Title = "Day Journal"
	lm.DisableQuitKeybindings()

	return &Model{
		service: service,
		state:   s,
		list:    lm,
		keys:    newKeyMap(),
		active:  "day",
	}, nil
}

func newKeyMap() keyMap {
	return keyMap{
		open: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "open"),
		),
		today: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "today"),
		),
		refresh: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "refresh"),
		),
		day: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "day"),
		),
		week: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "week"),
		),
		month: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "month"),
		),
		year: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "year"),
		),
	}
}

func toListItems(entries []svc.Entry) []list.Item {
	items := make([]list.Item, 0, len(entries))
	for _, entry := range entries {
		items = append(items, entryItem{entry: entry})
	}
	return items
}

func (i entryItem) Title() string {
	if i.entry.Date.IsZero() {
		return strings.Title(i.entry.Title)
	}
	return fmt.Sprintf("%s", i.entry.Title)
}

func (i entryItem) Description() string {
	if i.entry.Date.IsZero() {
		return i.entry.Path
	}
	return fmt.Sprintf("%s", i.entry.Date.Format("2006-01-02"))
}

func (i entryItem) FilterValue() string {
	return i.entry.Title
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-2)
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.open):
			return m.handleOpen()
		case key.Matches(msg, m.keys.today):
			return m.handleToday()
		case key.Matches(msg, m.keys.refresh):
			return m, m.refresh()
		case key.Matches(msg, m.keys.day):
			return m.switchTemplate("day")
		case key.Matches(msg, m.keys.week):
			return m.switchTemplate("week")
		case key.Matches(msg, m.keys.month):
			return m.switchTemplate("month")
		case key.Matches(msg, m.keys.year):
			return m.switchTemplate("year")
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	if m.status != "" {
		return fmt.Sprintf("%s\n%s", m.list.View(), m.status)
	}
	return m.list.View()
}

func (m *Model) switchTemplate(template string) (tea.Model, tea.Cmd) {
	if m.active == template {
		return m, nil
	}
	m.active = template
	title := strings.Title(template) + " Journal"
	m.list.Title = title
	return m, m.refresh()
}

func (m *Model) refresh() tea.Cmd {
	entries, err := m.service.List(m.active)
	if err != nil {
		m.status = fmt.Sprintf("refresh failed: %v", err)
		return nil
	}
	m.status = fmt.Sprintf("showing %s entries", m.active)
	return m.list.SetItems(toListItems(entries))
}

func (m *Model) handleOpen() (tea.Model, tea.Cmd) {
	item, ok := m.list.SelectedItem().(entryItem)
	if !ok {
		return m, nil
	}
	if err := m.service.Open(item.entry.Path); err != nil {
		m.status = fmt.Sprintf("open failed: %v", err)
	} else {
		m.status = fmt.Sprintf("opened %s", item.entry.Title)
	}
	return m, nil
}

func (m *Model) handleToday() (tea.Model, tea.Cmd) {
	entry, err := m.service.EnsureEntry(m.active, 0, nil, nil, "")
	if err != nil {
		m.status = fmt.Sprintf("ensure failed: %v", err)
		return m, nil
	}
	if err := m.service.Open(entry.Path); err != nil {
		m.status = fmt.Sprintf("open failed: %v", err)
		return m, nil
	}
	m.status = fmt.Sprintf("opened %s", entry.Title)
	return m, m.refresh()
}
