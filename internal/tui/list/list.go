package list

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paintersrp/an/internal/config"
)

// TODO: Clean and Organize
// TODO: Input Validation
// TODO: Molecule Adding
// TODO: Skip saving file and making changes if no changes to input occured

type ListItem struct {
	title       string
	description string
}

func (i ListItem) Title() string       { return i.title }
func (i ListItem) Description() string { return i.description }
func (i ListItem) FilterValue() string { return i.title }

type listKeyMap struct {
	toggleTitleBar   key.Binding
	toggleStatusBar  key.Binding
	togglePagination key.Binding
	toggleHelpMenu   key.Binding
	toggleEditItem   key.Binding
	quit             key.Binding
	exitInputMode    key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		toggleTitleBar: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "toggle title"),
		),
		toggleStatusBar: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "toggle status"),
		),
		togglePagination: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "toggle pagination"),
		),
		toggleHelpMenu: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "toggle help"),
		),
		toggleEditItem: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "edit item"),
		),
		exitInputMode: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "exit input mode"),
		),
	}
}

type ListModel struct {
	list         list.Model
	keys         *listKeyMap
	delegateKeys *delegateKeyMap
	config       *config.Config
	configInput  ListInputModel
	inputActive  bool
}

func NewListModel(cfg *config.Config) ListModel {
	delegateKeys := newDelegateKeyMap()
	listKeys := newListKeyMap()
	configInput := initialInputModel()

	// Create list items from the config
	items := []list.Item{
		ListItem{title: "VaultDir", description: cfg.VaultDir},
		ListItem{title: "Editor", description: cfg.Editor},
		ListItem{title: "NvimArgs", description: cfg.NvimArgs},
		ListItem{title: "FileSystemMode", description: cfg.FileSystemMode},
		ListItem{title: "PinnedFile", description: cfg.PinnedFile},
		ListItem{title: "PinnedTaskFile", description: cfg.PinnedTaskFile},
	}

	// Setup list
	delegate := newItemDelegate(delegateKeys)
	configList := list.New(items, delegate, 0, 0)
	configList.Title = "Configuration"
	configList.Styles.Title = titleStyle
	configList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.toggleTitleBar,
			listKeys.toggleStatusBar,
			listKeys.togglePagination,
			listKeys.toggleHelpMenu,
		}
	}

	return ListModel{
		list:         configList,
		keys:         listKeys,
		delegateKeys: delegateKeys,
		configInput:  configInput,
		config:       cfg,
	}
}

func (m ListModel) Init() tea.Cmd {
	return nil
}

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		if m.inputActive {
			// Handle exiting input mode
			if key.Matches(msg, m.keys.exitInputMode) {
				m.configInput.Input.Blur()
				m.inputActive = false
				return m, nil
			}

			// Update the text input and handle its commands
			var cmd tea.Cmd
			m.configInput.Input, cmd = m.configInput.Input.Update(msg)
			cmds = append(cmds, cmd)

			// Handle the case when Enter is pressed and the input is submitted
			if key.Matches(msg, m.keys.toggleEditItem) {
				var title string

				if i, ok := m.list.SelectedItem().(ListItem); ok {
					title = i.Title()
				} else {
					return m, nil
				}

				inputValue := m.configInput.Input.Value()
				switch title {
				case "VaultDir":
					m.config.VaultDir = inputValue
				case "Editor":
					m.config.Editor = inputValue
				case "NvimArgs":
					m.config.NvimArgs = inputValue
				case "MoleculeMode":
					m.config.FileSystemMode = inputValue
				case "PinnedFile":
					m.config.PinnedFile = inputValue
				case "PinnedTaskFile":
					m.config.PinnedTaskFile = inputValue
				default:
					// Handle unknown title
				}

				// Save the updated config
				err := m.config.Save()
				if err != nil {
					fmt.Println("Failed to save config file, exiting...")
					os.Exit(1)
				}

				// Update the description of the selected item
				index := m.list.Index()
				items := m.list.Items()
				items[index] = ListItem{title: title, description: inputValue}
				m.list.SetItems(items)

				// Reset and unfocus the text input
				m.configInput.Input.Reset()
				m.configInput.Input.Blur()
				m.inputActive = false
				m.list.NewStatusMessage(statusMessageStyle("Updated and Saved: " + title))
			}

			return m, tea.Batch(cmds...)
		}

		switch {
		case key.Matches(msg, m.keys.toggleEditItem):
			var title, value string

			if i, ok := m.list.SelectedItem().(ListItem); ok {
				title = i.Title()
				value = i.Description()
			} else {
				return m, nil
			}

			m.inputActive = true
			m.configInput.Title = title
			m.configInput.Input.Focus()
			m.configInput.Input.SetValue(value)
			return m, nil

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

		}
	}

	// This will also call our delegate's update function.
	newListModel, cmd := m.list.Update(msg)
	m.list = newListModel
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m ListModel) View() string {
	if m.inputActive {
		return appStyle.Render(inputStyle.Render(m.configInput.View()))

	}
	return appStyle.Render(m.list.View())
}

func Run(c *config.Config) error {
	if _, err := tea.NewProgram(NewListModel(c), tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	return nil
}
