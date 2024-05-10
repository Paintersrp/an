package sublist

import (
	"fmt"
	"os"

	"github.com/Paintersrp/an/internal/config"
	v "github.com/Paintersrp/an/internal/views"
	"github.com/Paintersrp/an/tui/notes"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type SubListModel struct {
	List         list.Model
	keys         *listKeyMap
	delegateKeys *delegateKeyMap
	cfg          *config.Config
	width        int
	height       int
}

func NewSubListModel(cfg *config.Config) SubListModel {
	var (
		delegateKeys = newDelegateKeyMap()
		listKeys     = newListKeyMap()
	)

	delegate := newItemDelegate(delegateKeys, cfg)
	l := list.New(nil, delegate, 0, 0)
	l.Title = "Sublist"
	// l.SetHeight(40)
	// l.SetWidth(40)
	l.Styles.Title = titleStyle
	// l.DisableQuitKeybindings()
	l.AdditionalFullHelpKeys = func() []key.Binding { return fullHelp(listKeys) }

	views := v.GenerateViews(cfg.VaultDir)
	files, _ := v.GetFilesByView(views, "default", cfg.VaultDir)
	items := notes.ParseNoteFiles(files, cfg.VaultDir, false)
	l.SetItems(items)

	return SubListModel{
		List:         l,
		keys:         listKeys,
		delegateKeys: delegateKeys,
		cfg:          cfg,
	}
}

func (m SubListModel) Init() tea.Cmd {
	return nil
}

func (m SubListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't match any of the keys below if we're actively filtering.
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

func Run(cfg *config.Config) tea.Model {
	m, err := tea.NewProgram(NewSubListModel(cfg), tea.WithAltScreen()).Run()

	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	return m
}
