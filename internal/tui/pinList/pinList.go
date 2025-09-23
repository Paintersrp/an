package pinList

import (
	"fmt"
	"os"
	"sort"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/note"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/tui/notes"
	"github.com/Paintersrp/an/internal/tui/pinList/submodels/input"
	"github.com/Paintersrp/an/internal/tui/pinList/submodels/sublist"
)

type PinListModel struct {
	list         list.Model
	keys         *listKeyMap
	delegateKeys *delegateKeyMap
	state        *state.State
	pinType      string
	findingFor   string
	renamingFor  string
	sublist      sublist.SubListModel
	input        input.NameInputModel
	finding      bool
	renaming     bool
	adding       bool
}

func NewPinListModel(s *state.State, pinType string) PinListModel {
	var (
		delegateKeys = newDelegateKeyMap()
		listKeys     = newListKeyMap()
	)

	delegate := newItemDelegate(delegateKeys, s.Config, pinType)
	l := list.New(nil, delegate, 0, 0)
	l.Title = getTitleByType(pinType)
	l.Styles.Title = titleStyle
	l.AdditionalFullHelpKeys = func() []key.Binding { return fullHelp(listKeys) }
	l.AdditionalShortHelpKeys = func() []key.Binding { return shortHelp(listKeys) }

	items := getItemsByType(s.Config, pinType)
	l.SetItems(items)
	l.SetHeight(20)

	sl := sublist.NewSubListModel(s)
	i := input.NewNameInput()

	return PinListModel{
		list:         l,
		keys:         listKeys,
		delegateKeys: delegateKeys,
		state:        s,
		pinType:      pinType,
		sublist:      sl,
		input:        i,
		finding:      false,
		findingFor:   "",
		renaming:     false,
		renamingFor:  "",
		adding:       false,
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
		if m.list.FilterState() == list.Filtering {
			break
		}

		if m.adding {
			if m.renaming {
				if key.Matches(msg, m.keys.backToMain) {
					m.input.Input.Blur()
					m.renaming = false
					m.adding = false
					return m, nil
				}

				var cmd tea.Cmd
				m.input.Input, cmd = m.input.Input.Update(msg)
				cmds = append(cmds, cmd)

				if key.Matches(msg, m.keys.findSelect) {
					nv := m.input.Input.Value()

					if nv == "" {
						m.renaming = false
						return m, m.list.NewStatusMessage("given name was empty, please try again.")
					}

					m.input.Input.Blur()
					m.renaming = false
					m.finding = true
					m.sublist.List.SetSize(m.list.Width(), int(float64(m.list.Height())*0.7))
					m.sublist.List.Title = "Select a file to pin under the new name"

					return m, nil
				}

				return m, tea.Batch(cmds...)
			}

			if m.finding {
				if key.Matches(msg, m.keys.backToMain) {
					m.finding = false
					return m, nil
				}

				if key.Matches(msg, m.keys.findSelect) {
					if i, ok := m.sublist.List.SelectedItem().(notes.ListItem); ok {

						path := i.Path()
						name := m.input.Input.Value()
						if err := m.state.Config.AddPin(name, path, m.pinType); err != nil {
							return m, m.list.NewStatusMessage(
								statusMessageStyle(
									fmt.Sprintf("Failed to add pin: %v", err),
								),
							)
						}
						m.finding = false
						m.adding = false
						return m, m.refreshItems(m.pinType)
					} else {
						return m, nil
					}
				}

				var cmd tea.Cmd
				m.sublist.List, cmd = m.sublist.List.Update(msg)
				cmds = append(cmds, cmd)

				return m, tea.Batch(cmds...)
			}
		}

		if m.renaming {
			if key.Matches(msg, m.keys.backToMain) {
				m.input.Input.Blur()
				m.renaming = false
				return m, nil
			}

			var cmd tea.Cmd
			m.input.Input, cmd = m.input.Input.Update(msg)
			cmds = append(cmds, cmd)

			if key.Matches(msg, m.keys.findSelect) {
				nv := m.input.Input.Value()
				if nv == "" {
					m.renaming = false
					return m, m.list.NewStatusMessage("given name was empty, please try again.")
				}
				if nv == m.renamingFor {
					m.renaming = false
					return m, m.list.NewStatusMessage("new name and old name matched.")
				}

				if err := m.state.Config.RenamePin(m.renamingFor, nv, m.pinType); err != nil {
					return m, m.list.NewStatusMessage(
						statusMessageStyle(
							fmt.Sprintf("Failed to rename pin: %v", err),
						),
					)
				}
				m.input.Input.Blur()
				m.renaming = false

				return m, m.refreshItems(m.pinType)
			}

			return m, tea.Batch(cmds...)
		}

		if m.finding {
			if key.Matches(msg, m.keys.backToMain) {
				m.finding = false
				return m, nil
			}

			if key.Matches(msg, m.keys.findSelect) {
				if i, ok := m.sublist.List.SelectedItem().(notes.ListItem); ok {
					if err := m.state.Config.ChangePin(i.Path(), m.pinType, m.findingFor); err != nil {
						return m, m.list.NewStatusMessage(
							statusMessageStyle(
								fmt.Sprintf("Failed to change pin: %v", err),
							),
						)
					}
					m.finding = false
					return m, m.refreshItems(m.pinType)
				} else {
					return m, nil
				}
			}

			var cmd tea.Cmd
			m.sublist.List, cmd = m.sublist.List.Update(msg)
			cmds = append(cmds, cmd)

			return m, tea.Batch(cmds...)
		}

		switch {
		case key.Matches(msg, m.keys.find):
			if i, ok := m.list.SelectedItem().(PinListItem); ok {
				m.sublist.List.SetSize(m.list.Width(), int(float64(m.list.Height())*0.7))
				m.findingFor = i.Title()
				m.finding = true
			}

			return m, nil

		case key.Matches(msg, m.keys.rename):
			if i, ok := m.list.SelectedItem().(PinListItem); ok {
				if i.Title() == "default" {
					return m, m.list.NewStatusMessage(statusMessageStyle("Cannot rename the default pin"))
				}
				m.renaming = true
				m.renamingFor = i.title
				m.input.Input.Focus()
				m.input.Input.SetValue(i.title)
			}
			return m, nil

		case key.Matches(msg, m.keys.create):
			m.adding = true
			m.renaming = true
			m.input.Input.Focus()
			m.input.Input.SetValue("")
			return m, nil

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
			m.pinType = "text"
			return m, m.refreshItems("text")

		case key.Matches(msg, m.keys.swapToTaskView):
			m.pinType = "task"
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
	sublist := listStyle.Render(m.sublist.View())
	prompt := listStyle.Render(m.input.View())

	if m.renaming {
		return appStyle.Render(prompt)
	}

	if m.finding {
		return appStyle.Height(40).MaxHeight(40).
			Render(sublist)
	}

	return appStyle.Render(list)
}

func Run(s *state.State, pinType string) tea.Model {
	m, err := tea.NewProgram(NewPinListModel(s, pinType), tea.WithAltScreen()).Run()
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

		if p == "No Default Pinned File" {
			m.list.NewStatusMessage(statusMessageStyle("Item has no path"))
			return false
		}
	} else {
		return false
	}

	err := note.OpenFromPath(p, false)
	if err != nil {
		m.list.NewStatusMessage(statusMessageStyle(fmt.Sprintf("Open Error: %s", err)))
		return false
	}

	return true
}

func (m *PinListModel) refreshItems(pinType string) tea.Cmd {
	items := getItemsByType(m.state.Config, pinType)
	title := getTitleByType(pinType)
	m.list.Title = title
	m.refreshDelegate(pinType)
	return m.list.SetItems(items)
}

func (m *PinListModel) refreshDelegate(pinType string) {
	dkeys := newDelegateKeyMap()
	delegate := newItemDelegate(dkeys, m.state.Config, pinType)
	m.list.SetDelegate(delegate)
}

func getItemsByType(cfg *config.Config, pinType string) []list.Item {
	var items []list.Item
	ws := cfg.MustWorkspace()

	switch pinType {
	case "text":
		names := make([]string, 0, len(ws.NamedPins))
		for name := range ws.NamedPins {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			path := ws.NamedPins[name]
			items = append(items, PinListItem{title: name, description: path})
		}

		if ws.PinnedFile != "" {
			items = append(
				items,
				PinListItem{title: "default", description: ws.PinnedFile},
			)
		} else {
			items = append(
				items,
				PinListItem{title: "default", description: "No Default Pinned File"},
			)
		}
	case "task":
		names := make([]string, 0, len(ws.NamedTaskPins))
		for name := range ws.NamedTaskPins {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			path := ws.NamedTaskPins[name]
			items = append(items, PinListItem{title: name, description: path})
		}

		if ws.PinnedTaskFile != "" {
			items = append(
				items,
				PinListItem{title: "default", description: ws.PinnedTaskFile},
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
