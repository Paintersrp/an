package notes

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
			key.WithKeys("n"),
			key.WithHelp("n", "notes"),
		),
		tasks: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "tasks"),
		),
		journal: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "journal"),
		),
		next: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "next workspace"),
		),
	}
}

func NewRootModel(notes *NoteListModel, tasks *taskstui.Model, journal *journaltui.Model) *RootModel {
	model := &RootModel{
		notes:   notes,
		tasks:   tasks,
		journal: journal,
		active:  viewNotes,
		keys:    newRootKeyMap(),
	}
	model.updateRootStatus()
	return model
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
	m.updateRootStatus()
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

		if !editorActive && m.handleViewSwitch(msg) {
			return m, nil
		}

		if key.Matches(msg, m.keys.next) {
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
	sections := []string{}

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
	return padFrame(content, m.width, m.height)
}

func (m *RootModel) handleViewSwitch(msg tea.KeyMsg) bool {
	switch {
	case key.Matches(msg, m.keys.notes):
		m.active = viewNotes
		m.updateRootStatus()
		return true
	case key.Matches(msg, m.keys.tasks):
		m.active = viewTasks
		m.updateRootStatus()
		return true
	case key.Matches(msg, m.keys.journal):
		m.active = viewJournal
		m.updateRootStatus()
		return true
	}
	return false
}

func formatShortcut(binding key.Binding) string {
	help := binding.Help()
	keyStr := strings.TrimSpace(help.Key)
	desc := strings.TrimSpace(help.Desc)

	if desc != "" {
		desc = capitalize(desc)
	}

	switch {
	case keyStr != "" && desc != "":
		return fmt.Sprintf("%s %s", keyStr, desc)
	case keyStr != "":
		return keyStr
	case desc != "":
		return desc
	}

	keys := binding.Keys()
	if len(keys) > 0 {
		return keys[0]
	}
	return ""
}

func capitalize(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
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
		newNotes.previewFocused = m.notes.previewFocused
		if newNotes.previewFocused {
			newNotes.focusPreview()
		} else {
			newNotes.blurPreview()
		}
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
	m.updateRootStatus()

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
		return rootHeaderActiveStyle.Render(fmt.Sprintf("[%s]", label))
	}
	return rootHeaderStyle.Render(label)
}

func padFrame(content string, width, height int) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}

	if width > 0 {
		for i, line := range lines {
			pad := width - lipgloss.Width(line)
			if pad > 0 {
				lines[i] = line + strings.Repeat(" ", pad)
			}
		}
	}

	if height > len(lines) {
		blank := ""
		if width > 0 {
			blank = strings.Repeat(" ", width)
		}
		for len(lines) < height {
			lines = append(lines, blank)
		}
	}

	return strings.Join(lines, "\n")
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

func (m *RootModel) updateRootStatus() {
	status := state.RootStatus{
		ActiveView:    string(m.active),
		WorkspaceName: m.workspaceName(),
		WorkspaceHint: m.workspaceSwitchHint(),
		Shortcuts: []state.ViewShortcut{
			{View: string(viewNotes), Label: formatShortcut(m.keys.notes)},
			{View: string(viewTasks), Label: formatShortcut(m.keys.tasks)},
			{View: string(viewJournal), Label: formatShortcut(m.keys.journal)},
		},
	}

	status.Footer = m.renderFooter(status)
	m.applyRootStatus(status)
}

func (m *RootModel) applyRootStatus(status state.RootStatus) {
	if m.notes != nil && m.notes.state != nil {
		m.notes.state.RootStatus = status
	}
	if m.tasks != nil && m.tasks.State() != nil {
		m.tasks.State().RootStatus = status
	}
	if m.journal != nil && m.journal.State() != nil {
		m.journal.State().RootStatus = status
	}
}

func (m *RootModel) renderFooter(status state.RootStatus) string {
	sections := []string{}
	if status.WorkspaceName != "" {
		label := fmt.Sprintf("Workspace: [%s]", status.WorkspaceName)
		if status.WorkspaceHint != "" {
			label += " " + status.WorkspaceHint
		}
		sections = append(sections, rootHeaderWorkspaceStyle.Render(label))
	}

	sections = append(sections, rootHeaderStyle.Render("Views:"))
	for _, shortcut := range status.Shortcuts {
		view := rootView(shortcut.View)
		sections = append(sections, highlight(view, m.active, shortcut.Label))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, sections...)
}

func (m *RootModel) workspaceSwitchHint() string {
	if !m.hasMultipleWorkspaces() {
		return ""
	}

	nextHelp := m.keys.next.Help()
	nextKey := strings.TrimSpace(nextHelp.Key)
	if nextKey == "" {
		keys := m.keys.next.Keys()
		if len(keys) > 0 {
			nextKey = keys[0]
		} else {
			nextKey = "ctrl+w"
		}
	}
	return fmt.Sprintf("(%s to switch)", nextKey)
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
