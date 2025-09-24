package tasks

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	services "github.com/Paintersrp/an/internal/services/tasks"
	"github.com/Paintersrp/an/internal/state"
)

type Model struct {
	service *services.Service
	state   *state.State
	list    list.Model
	keys    keyMap
	status  string
	width   int
	height  int
}

type keyMap struct {
	open    key.Binding
	toggle  key.Binding
	refresh key.Binding
}

type listItem struct {
	item services.Item
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
	lm := list.New(toListItems(items), delegate, 0, 0)
	lm.Title = "Tasks"
	lm.DisableQuitKeybindings()

	return &Model{
		service: svc,
		state:   s,
		list:    lm,
		keys:    newKeyMap(),
	}, nil
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
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "refresh"),
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
	return fmt.Sprintf("%s %s", prefix, i.item.Content)
}

func (i listItem) Description() string {
	rel := i.item.RelPath
	if rel == "" {
		rel = filepath.Base(i.item.Path)
	}
	return fmt.Sprintf("%s:%d", rel, i.item.Line)
}

func (i listItem) FilterValue() string {
	return i.item.Content
}

func (m *Model) Init() tea.Cmd {
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
		}
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

	status := m.status
	if pinned != "" {
		status = fmt.Sprintf("Pinned: %s", pinned)
		if m.status != "" {
			status = status + "\n" + m.status
		}
	}

	if status != "" {
		return fmt.Sprintf("%s\n%s", m.list.View(), status)
	}
	return m.list.View()
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

	return m.list.SetItems(toListItems(items))
}
