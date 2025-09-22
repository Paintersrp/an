package sublist

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/tui/notes"
)

type SubListModel struct {
	List         list.Model
	keys         *listKeyMap
	delegateKeys *delegateKeyMap
	state        *state.State
	cfg          *config.Config
	width        int
	height       int
}

func NewSubListModel(s *state.State) SubListModel {
	var (
		delegateKeys = newDelegateKeyMap()
		listKeys     = newListKeyMap()
	)

	delegate := newItemDelegate(delegateKeys, s.Config)
	l := list.New(nil, delegate, 0, 0)
	l.Title = "Sublist"
	l.Styles.Title = titleStyle
	// l.DisableQuitKeybindings()
	l.AdditionalFullHelpKeys = func() []key.Binding { return fullHelp(listKeys) }

	files, err := s.ViewManager.GetFilesByView("default", s.Vault)
	if err != nil {
		l.NewStatusMessage(
			statusMessageStyle(fmt.Sprintf("Failed to load default view: %v", err)),
		)
	} else {
		items := notes.ParseNoteFiles(files, s.Vault, false)
		l.SetItems(items)
	}

	return SubListModel{
		List:         l,
		keys:         listKeys,
		delegateKeys: delegateKeys,
		state:        s,
	}
}

func (m SubListModel) Init() tea.Cmd {
	return nil
}

func (m SubListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.List.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.keys.toggleTitleBar):
			v := !m.List.ShowTitle()
			m.List.SetShowTitle(v)
			m.List.SetShowFilter(v)
			m.List.SetFilteringEnabled(v)
			return m, nil

		case key.Matches(msg, m.keys.toggleStatusBar):
			m.List.SetShowStatusBar(!m.List.ShowStatusBar())
			return m, nil

		case key.Matches(msg, m.keys.togglePagination):
			m.List.SetShowPagination(!m.List.ShowPagination())
			return m, nil

		case key.Matches(msg, m.keys.toggleHelpMenu):
			m.List.SetShowHelp(!m.List.ShowHelp())
			return m, nil

		}
	}

	newListModel, cmd := m.List.Update(msg)
	m.List = newListModel
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m SubListModel) View() string {
	return m.List.View()
}

func Run(s *state.State) tea.Model {
	m, err := tea.NewProgram(NewSubListModel(s), tea.WithAltScreen()).Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	return m
}
