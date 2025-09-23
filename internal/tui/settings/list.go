package settings

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/erikgeiser/promptkit/selection"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/views"
)

// TODO: Clean and Organize
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
	list               list.Model
	keys               *listKeyMap
	delegateKeys       *delegateKeyMap
	config             *config.Config
	state              *state.State
	workspace          *config.Workspace
	configInput        ListInputModel
	inputActive        bool
	editorSelect       *selection.Model[string]
	editorSelectActive bool
	modeSelect         *selection.Model[string]
	modeSelectActive   bool
	pendingAction      string
}

const (
	viewActionAdd    = "add_view"
	viewActionRemove = "remove_view"
)

func NewListModel(s *state.State) ListModel {
	delegateKeys := newDelegateKeyMap()
	listKeys := newListKeyMap()
	configInput := initialInputModel()

	cfg := s.Config
	ws := s.Workspace
	if ws == nil {
		ws = cfg.MustWorkspace()
	}

	items := []list.Item{
		ListItem{title: "VaultDir", description: ws.VaultDir},
		ListItem{title: "Editor", description: ws.Editor},
		ListItem{title: "NvimArgs", description: ws.NvimArgs},
		ListItem{title: "FileSystemMode", description: ws.FileSystemMode},
		ListItem{title: "Add Custom View", description: "name|include|exclude|sortField|sortOrder|predicates"},
		ListItem{title: "Remove Custom View", description: "name"},
	}

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

	editorSel := selection.New(
		"Please select an editor option.",
		[]string{"nvim", "obsidian", "vscode", "vim", "nano"},
	)
	editorSel.Filter = nil
	editorSelect := selection.NewModel(editorSel)

	modeSel := selection.New(
		"Please a file system mode for your vault.",
		[]string{"strict", "confirm", "free"},
	)

	modeSel.Filter = nil
	modeSelect := selection.NewModel(modeSel)

	return ListModel{
		list:               configList,
		keys:               listKeys,
		delegateKeys:       delegateKeys,
		configInput:        configInput,
		inputActive:        false,
		config:             cfg,
		state:              s,
		workspace:          ws,
		editorSelect:       editorSelect,
		editorSelectActive: false,
		modeSelect:         modeSelect,
		modeSelectActive:   false,
		pendingAction:      "",
	}
}

func (m ListModel) Init() tea.Cmd {
	return tea.Batch(m.editorSelect.Init(), m.modeSelect.Init())
}

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		if m.editorSelectActive {
			if key.Matches(msg, m.keys.exitInputMode) {
				m.editorSelectActive = false
				return m, nil
			}

			var cmd tea.Cmd
			_, cmd = m.editorSelect.Update(msg)
			cmds = append(cmds, cmd)

			if key.Matches(msg, m.keys.toggleEditItem) {
				c, err := m.editorSelect.Value()
				if err != nil {
					return m, nil
				}

				if err := m.config.ChangeEditor(c); err != nil {
					m.list.NewStatusMessage(statusMessageStyle(err.Error()))
					return m, nil
				}

				m.editorSelectActive = false

				index := m.list.Index()
				items := m.list.Items()
				items[index] = ListItem{title: "Editor", description: c}
				m.list.SetItems(items)
				m.list.NewStatusMessage(statusMessageStyle("Updated and Saved: Editor"))

				editorSel := selection.New(
					"Please select an editor option.",
					[]string{"nvim", "obsidian", "vscode", "vim", "nano"},
				)
				editorSel.Filter = nil
				m.editorSelect = selection.NewModel(editorSel)
				return m, m.editorSelect.Init()
			}

			return m, tea.Batch(cmds...)
		}

		if m.modeSelectActive {
			if key.Matches(msg, m.keys.exitInputMode) {
				m.modeSelectActive = false
				return m, nil
			}

			var cmd tea.Cmd
			_, cmd = m.modeSelect.Update(msg)
			cmds = append(cmds, cmd)

			if key.Matches(msg, m.keys.toggleEditItem) {
				c, err := m.modeSelect.Value()
				if err != nil {
					return m, nil
				}

				if err := m.config.ChangeMode(c); err != nil {
					m.list.NewStatusMessage(statusMessageStyle(err.Error()))
					return m, nil
				}
				m.modeSelectActive = false
				m.workspace = m.config.MustWorkspace()

				if saveErr := m.config.Save(); saveErr != nil {
					fmt.Println("Failed to save config file, exiting...")
					os.Exit(1)
				}

				index := m.list.Index()
				items := m.list.Items()
				items[index] = ListItem{title: "FileSystemMode", description: c}
				m.list.SetItems(items)
				m.list.NewStatusMessage(statusMessageStyle("Updated and Saved: FileSystemMode"))

				modeSel := selection.New(
					"Please a file system mode for your vault.",
					[]string{"strict", "confirm", "free"},
				)
				modeSel.Filter = nil
				m.modeSelect = selection.NewModel(modeSel)
				return m, m.modeSelect.Init()
			}

			return m, tea.Batch(cmds...)
		}

		if m.inputActive {
			if key.Matches(msg, m.keys.exitInputMode) {
				m.configInput.Input.Blur()
				m.inputActive = false
				m.pendingAction = ""
				return m, nil
			}

			var cmd tea.Cmd
			m.configInput.Input, cmd = m.configInput.Input.Update(msg)
			cmds = append(cmds, cmd)

			if key.Matches(msg, m.keys.toggleEditItem) {
				var title string

				if i, ok := m.list.SelectedItem().(ListItem); ok {
					title = i.Title()
				} else {
					return m, nil
				}

				inputValue := m.configInput.Input.Value()
				handledCustomAction := false
				updateDescription := true

				switch {
				case m.pendingAction == viewActionAdd:
					name, def, err := parseViewDefinition(inputValue)
					if err != nil {
						m.list.NewStatusMessage(statusMessageStyle(err.Error()))
						return m, nil
					}

					if err := m.state.ViewManager.AddCustomView(name, def); err != nil {
						m.list.NewStatusMessage(statusMessageStyle(err.Error()))
						return m, nil
					}

					handledCustomAction = true
					updateDescription = false
					m.list.NewStatusMessage(statusMessageStyle(fmt.Sprintf("Added view: %s", name)))
				case m.pendingAction == viewActionRemove:
					name := strings.TrimSpace(inputValue)
					if name == "" {
						m.list.NewStatusMessage(statusMessageStyle("View name is required"))
						return m, nil
					}

					if err := m.state.ViewManager.RemoveCustomView(name); err != nil {
						m.list.NewStatusMessage(statusMessageStyle(err.Error()))
						return m, nil
					}

					handledCustomAction = true
					updateDescription = false
					m.list.NewStatusMessage(statusMessageStyle(fmt.Sprintf("Removed view: %s", name)))
				case title == "Editor":
					if err := m.config.ChangeEditor(inputValue); err != nil {
						m.list.NewStatusMessage(statusMessageStyle(err.Error()))
						return m, nil
					}
					m.workspace = m.config.MustWorkspace()
				default:
					switch title {
					case "VaultDir":
						m.workspace.VaultDir = inputValue
					case "NvimArgs":
						m.workspace.NvimArgs = inputValue
					case "MoleculeMode":
						m.workspace.FileSystemMode = inputValue
					}

					if err := m.config.Save(); err != nil {
						fmt.Println("Failed to save config file, exiting...")
						os.Exit(1)
					}
					m.workspace = m.config.MustWorkspace()
				}

				index := m.list.Index()
				if updateDescription {
					items := m.list.Items()
					items[index] = ListItem{title: title, description: inputValue}
					m.list.SetItems(items)
				}

				m.configInput.Input.Reset()
				m.configInput.Input.Blur()
				m.inputActive = false
				m.pendingAction = ""
				if !handledCustomAction {
					m.list.NewStatusMessage(statusMessageStyle("Updated and Saved: " + title))
				}
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

			switch title {
			case "Editor":
				m.editorSelectActive = true
			case "FileSystemMode":
				m.modeSelectActive = true
			case "Add Custom View":
				m.pendingAction = viewActionAdd
				m.inputActive = true
				m.configInput.Title = "Add Custom View"
				m.configInput.Input.Focus()
				m.configInput.Input.SetValue("")
				return m, nil
			case "Remove Custom View":
				m.pendingAction = viewActionRemove
				m.inputActive = true
				m.configInput.Title = "Remove Custom View"
				m.configInput.Input.Focus()
				m.configInput.Input.SetValue("")
				return m, nil
			default:
				m.inputActive = true
			}

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

	newListModel, cmd := m.list.Update(msg)
	m.list = newListModel
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m ListModel) View() string {
	if m.inputActive {
		return appStyle.Render(inputStyle.Render(m.configInput.View()))
	}
	if m.editorSelectActive {
		return appStyle.Render(m.editorSelect.View())
	}
	if m.modeSelectActive {
		return appStyle.Render(m.modeSelect.View())
	}
	return appStyle.Render(m.list.View())
}

func Run(s *state.State) error {
	if _, err := tea.NewProgram(NewListModel(s), tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	return nil
}

func parseViewDefinition(input string) (string, config.ViewDefinition, error) {
	parts := strings.Split(input, "|")
	if len(parts) == 0 {
		return "", config.ViewDefinition{}, fmt.Errorf("input cannot be empty")
	}

	name := strings.TrimSpace(parts[0])
	if name == "" {
		return "", config.ViewDefinition{}, fmt.Errorf("view name is required")
	}

	def := config.ViewDefinition{}

	if len(parts) > 1 {
		def.Include = splitAndTrimCSV(parts[1])
	}
	if len(parts) > 2 {
		def.Exclude = splitAndTrimCSV(parts[2])
	}
	if len(parts) > 3 {
		field := strings.ToLower(strings.TrimSpace(parts[3]))
		if field != "" {
			sortField := views.SortField(field)
			if !views.IsValidSortField(sortField) {
				return "", config.ViewDefinition{}, fmt.Errorf("invalid sort field: %s", field)
			}
			def.Sort.Field = field
		}
	}
	if len(parts) > 4 {
		order := strings.ToLower(strings.TrimSpace(parts[4]))
		if order != "" {
			sortOrder := views.SortOrder(order)
			if !views.IsValidSortOrder(sortOrder) {
				return "", config.ViewDefinition{}, fmt.Errorf("invalid sort order: %s", order)
			}
			def.Sort.Order = order
		}
	}
	if len(parts) > 5 {
		predicates := splitAndTrimCSV(parts[5])
		for i, predicate := range predicates {
			normalized := strings.ToLower(predicate)
			if !views.IsValidPredicate(views.Predicate(normalized)) {
				return "", config.ViewDefinition{}, fmt.Errorf("invalid predicate: %s", predicate)
			}
			predicates[i] = normalized
		}
		def.Predicates = predicates
	}

	return name, def, nil
}

func splitAndTrimCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}
