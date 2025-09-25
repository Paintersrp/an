package notes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/state"
	journaltui "github.com/Paintersrp/an/internal/tui/journal"
	taskstui "github.com/Paintersrp/an/internal/tui/tasks"
)

type rootView string

const (
	viewNotes   rootView = "notes"
	viewTasks   rootView = "tasks"
	viewJournal rootView = "journal"
)

type RootModel struct {
	notes   *NoteListModel
	tasks   *taskstui.Model
	journal *journaltui.Model
	active  rootView
	keys    rootKeyMap
	width   int
	height  int
}

type rootKeyMap struct {
	notes   key.Binding
	tasks   key.Binding
	journal key.Binding
	next    key.Binding
}

func newRootKeyMap() rootKeyMap {
	return rootKeyMap{
		notes: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "notes"),
		),
		tasks: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "tasks"),
		),
		journal: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "journal"),
		),
		next: key.NewBinding(
			key.WithKeys("ctrl+w"),
			key.WithHelp("ctrl+w", "next workspace"),
		),
	}
}

func NewRootModel(notes *NoteListModel, tasks *taskstui.Model, journal *journaltui.Model) *RootModel {
	return &RootModel{
		notes:   notes,
		tasks:   tasks,
		journal: journal,
		active:  viewNotes,
		keys:    newRootKeyMap(),
	}
}

func (m *RootModel) Init() tea.Cmd {
	cmds := []tea.Cmd{}
	if m.notes != nil {
		cmds = append(cmds, m.notes.Init())
	}
	if m.tasks != nil {
		cmds = append(cmds, m.tasks.Init())
	}
	if m.journal != nil {
		cmds = append(cmds, m.journal.Init())
	}
	return tea.Batch(cmds...)
}

func (m *RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateAll(msg)
		return m, nil
	case tea.QuitMsg:
		if m.notes != nil {
			model, _ := m.notes.Update(msg)
			m.notes = adoptNoteModel(model, m.notes)
		}
		return m, nil
	case tea.KeyMsg:
		editorActive := m.active == viewNotes && m.notes != nil && m.notes.editorActive()

		switch {
		case !editorActive && key.Matches(msg, m.keys.notes):
			m.active = viewNotes
			return m, nil
		case !editorActive && key.Matches(msg, m.keys.tasks):
			m.active = viewTasks
			return m, nil
		case !editorActive && key.Matches(msg, m.keys.journal):
			m.active = viewJournal
			return m, nil
		case key.Matches(msg, m.keys.next):
			if cmd := m.cycleWorkspace(); cmd != nil {
				return m, cmd
			}
			return m, nil
		}
	}

	switch m.active {
	case viewNotes:
		if m.notes == nil {
			return m, nil
		}
		model, cmd := m.notes.Update(msg)
		m.notes = adoptNoteModel(model, m.notes)
		return m, cmd
	case viewTasks:
		if m.tasks == nil {
			return m, nil
		}
		model, cmd := m.tasks.Update(msg)
		m.tasks = adoptTasksModel(model, m.tasks)
		return m, cmd
	case viewJournal:
		if m.journal == nil {
			return m, nil
		}
		model, cmd := m.journal.Update(msg)
		m.journal = adoptJournalModel(model, m.journal)
		return m, cmd
	}

	return m, nil
}

func (m *RootModel) View() string {
	sections := []string{
		m.header(),
	}

	switch m.active {
	case viewNotes:
		if m.notes != nil {
			sections = append(sections, m.notes.View())
		}
	case viewTasks:
		if m.tasks != nil {
			sections = append(sections, m.tasks.View())
		}
	case viewJournal:
		if m.journal != nil {
			sections = append(sections, m.journal.View())
		}
	}

	content := strings.Join(sections, "\n")
	return padToHeight(content, m.height)
}

func (m *RootModel) header() string {
	sections := []string{}
	if name := m.workspaceName(); name != "" {
		label := fmt.Sprintf("Workspace: [%s]", name)
		if m.hasMultipleWorkspaces() {
			label += " (ctrl+w to switch)"
		}
		sections = append(sections, label)
	}
	sections = append(sections, "Views:")
	sections = append(sections, highlight(viewNotes, m.active, "1. Notes"))
	sections = append(sections, highlight(viewTasks, m.active, "2. Tasks"))
	sections = append(sections, highlight(viewJournal, m.active, "3. Journal"))
	return strings.Join(sections, "  ")
}

func (m *RootModel) currentConfig() *config.Config {
	if m.notes != nil && m.notes.state != nil {
		return m.notes.state.Config
	}
	return nil
}

func (m *RootModel) workspaceName() string {
	if cfg := m.currentConfig(); cfg != nil {
		return cfg.CurrentWorkspace
	}
	return ""
}

func (m *RootModel) hasMultipleWorkspaces() bool {
	if cfg := m.currentConfig(); cfg != nil {
		return len(cfg.WorkspaceNames()) > 1
	}
	return false
}

func (m *RootModel) cycleWorkspace() tea.Cmd {
	cfg := m.currentConfig()
	if cfg == nil {
		return nil
	}

	names := cfg.WorkspaceNames()
	if len(names) == 0 {
		return nil
	}

	current := cfg.CurrentWorkspace
	idx := 0
	for i, name := range names {
		if name == current {
			idx = i
			break
		}
	}

	next := names[(idx+1)%len(names)]
	if next == "" || next == current {
		if len(names) <= 1 {
			m.notifyWorkspaceStatus("Only one workspace configured")
		}
		return nil
	}

	if err := cfg.SwitchWorkspace(next); err != nil {
		m.notifyWorkspaceStatus(fmt.Sprintf("Workspace switch failed: %v", err))
		return nil
	}

	if m.notes != nil {
		_ = m.notes.closeWatcher()
	}

	newState, err := state.NewState(next)
	if err != nil {
		m.notifyWorkspaceStatus(fmt.Sprintf("Workspace switch failed: %v", err))
		return nil
	}

	viewName := "default"
	if m.notes != nil && m.notes.viewName != "" {
		viewName = m.notes.viewName
	}

	newNotes, err := NewNoteListModel(newState, viewName)
	if err != nil {
		newNotes, err = NewNoteListModel(newState, "default")
		if err != nil {
			m.notifyWorkspaceStatus(fmt.Sprintf("Workspace switch failed: %v", err))
			return nil
		}
	}

	if m.notes != nil {
		newNotes.width = m.notes.width
		newNotes.height = m.notes.height
		newNotes.showDetails = m.notes.showDetails
	}

	newTasks, err := taskstui.NewModel(newState)
	if err != nil {
		m.notifyWorkspaceStatus(fmt.Sprintf("Workspace switch failed: %v", err))
		return nil
	}

	newJournal, err := journaltui.NewModel(newState)
	if err != nil {
		m.notifyWorkspaceStatus(fmt.Sprintf("Workspace switch failed: %v", err))
		return nil
	}

	m.notes = newNotes
	m.tasks = newTasks
	m.journal = newJournal

	cmds := []tea.Cmd{}
	if cmd := m.notes.Init(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if m.tasks != nil {
		if cmd := m.tasks.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if m.journal != nil {
		if cmd := m.journal.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	m.notifyWorkspaceStatus(fmt.Sprintf("Switched to workspace %s", next))

	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m *RootModel) notifyWorkspaceStatus(message string) {
	if m.notes != nil {
		m.notes.list.NewStatusMessage(statusStyle(message))
	}
}

func highlight(view rootView, active rootView, label string) string {
	if view == active {
		return fmt.Sprintf("[%s]", label)
	}
	return label
}

func padToHeight(content string, height int) string {
	if height <= 0 {
		return content
	}

	lines := strings.Count(content, "\n") + 1
	if lines >= height {
		return content
	}

	return content + strings.Repeat("\n", height-lines)
}

func (m *RootModel) updateAll(msg tea.Msg) {
	if m.notes != nil {
		model, _ := m.notes.Update(msg)
		m.notes = adoptNoteModel(model, m.notes)
	}
	if m.tasks != nil {
		model, _ := m.tasks.Update(msg)
		m.tasks = adoptTasksModel(model, m.tasks)
	}
	if m.journal != nil {
		model, _ := m.journal.Update(msg)
		m.journal = adoptJournalModel(model, m.journal)
	}
}

func adoptNoteModel(model tea.Model, current *NoteListModel) *NoteListModel {
	switch m := model.(type) {
	case *NoteListModel:
		return m
	default:
		return current
	}
}

func adoptTasksModel(model tea.Model, current *taskstui.Model) *taskstui.Model {
	if m, ok := model.(*taskstui.Model); ok {
		return m
	}
	return current
}

func adoptJournalModel(model tea.Model, current *journaltui.Model) *journaltui.Model {
	if m, ok := model.(*journaltui.Model); ok {
		return m
	}
	return current
}
