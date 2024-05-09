package pinList

import (
	"fmt"
	"os"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/zet"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type PinListModel struct {
	list         list.Model
	keys         *listKeyMap
	delegateKeys *delegateKeyMap
	cfg          *config.Config
	pinType      string
}

func NewPinListModel(cfg *config.Config, pinType string) PinListModel {
	var (
		delegateKeys = newDelegateKeyMap()
		listKeys     = newListKeyMap()
	)

	delegate := newItemDelegate(delegateKeys, cfg)
	l := list.New(nil, delegate, 0, 0)
	l.Title = getTitleByType(pinType)
	l.Styles.Title = titleStyle
	l.AdditionalFullHelpKeys = func() []key.Binding { return fullHelp(listKeys) }

	items := getItemsByType(cfg, pinType)
	l.SetItems(items)

	return PinListModel{
		list:         l,
		keys:         listKeys,
		delegateKeys: delegateKeys,
		cfg:          cfg,
		pinType:      pinType,
	}
}

func (m PinListModel) Init() tea.Cmd {
	return nil
}

func (m PinListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
		// Don't match any of the keys below if we're actively filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.keys.openNote):
			if ok := m.openNote(); ok {
				return m, tea.Quit
			} else {
				return m, nil
			}
		case key.Matches(msg, m.keys.toggleTitleBar):
			v := !m.list.ShowTitle()
			m.list.SetShowTitle(v)
			m.list.SetShowFilter(v)
			m.list.SetFilteringEnabled(v)
			return m, nil

		case key.Matches(msg, m.keys.toggleStatusBar):
			m.list.SetShowStatusBar(!m.list.ShowStatusBar())
			return m, nil

		case key.Matches(msg, m.keys.togglePagination):
			m.list.SetShowPagination(!m.list.ShowPagination())
			return m, nil

		case key.Matches(msg, m.keys.toggleHelpMenu):
			m.list.SetShowHelp(!m.list.ShowHelp())
			return m, nil

		case key.Matches(msg, m.keys.swapToTextView):
			return m, m.refreshItems("text")

		case key.Matches(msg, m.keys.swapToTaskView):
			return m, m.refreshItems("task")

		case key.Matches(msg, m.keys.swapView):
			var cmd tea.Cmd

			switch m.pinType {
			case "text":
				m.pinType = "task"
				cmd = m.refreshItems(m.pinType)
			case "task":
				m.pinType = "text"
				cmd = m.refreshItems(m.pinType)
			}

			return m, cmd
		}
	}

	newListModel, cmd := m.list.Update(msg)
	m.list = newListModel
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m PinListModel) View() string {
	list := listStyle.Render(m.list.View())

	return appStyle.Render(list)
}

func Run(cfg *config.Config, pinType string) tea.Model {
	m, err := tea.NewProgram(NewPinListModel(cfg, pinType), tea.WithAltScreen()).Run()

	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	return m
}

func (m *PinListModel) openNote() bool {
	var p string

	if i, ok := m.list.SelectedItem().(PinListItem); ok {
		p = i.description
	} else {
		return false
	}

	err := zet.OpenFromPath(p)

	if err != nil {
		m.list.NewStatusMessage(statusMessageStyle(fmt.Sprintf("Open Error: %s", err)))
		return false
	}

	return true
}

func (m *PinListModel) refreshItems(pinType string) tea.Cmd {
	items := getItemsByType(m.cfg, pinType)
	title := getTitleByType(pinType)
	m.list.Title = title
	return m.list.SetItems(items)
}

func getItemsByType(cfg *config.Config, pinType string) []list.Item {
	var items []list.Item

	switch pinType {
	case "text":
		for name, path := range cfg.NamedPins {
			items = append(items, PinListItem{title: name, description: path})
		}

		if cfg.PinnedFile != "" {
			items = append(
				items,
				PinListItem{title: "default", description: cfg.PinnedFile},
			)
		} else {
			items = append(
				items,
				PinListItem{title: "default", description: "No Default Pinned File"},
			)
		}
	case "task":
		// Iterate over NamedPins and create a PinListItem for each entry
		for name, path := range cfg.NamedTaskPins {
			items = append(items, PinListItem{title: name, description: path})
		}

		// Add the default PinnedFile if it's set
		if cfg.PinnedTaskFile != "" {
			items = append(
				items,
				PinListItem{title: "default", description: cfg.PinnedTaskFile},
			)
		} else {
			items = append(
				items,
				PinListItem{title: "default", description: "No Default Pinned File"},
			)

		}
	}

	return items
}

func getTitleByType(pinType string) string {
	switch pinType {
	case "text":
		return "Available Text Pins"
	case "task":
		return "Available Task Pins"
	default:
		return "Available Text Pins"
	}
}
