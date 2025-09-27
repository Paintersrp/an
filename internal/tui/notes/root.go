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
	state   *state.State
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
	var st *state.State
	if notes != nil {
		st = notes.state
	}

	return &RootModel{
		notes:   notes,
		tasks:   tasks,
		journal: journal,
		active:  viewNotes,
		keys:    newRootKeyMap(),
		state:   st,
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
	header := m.header()
	sections := []string{header}

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

func (m *RootModel) header() string {
	line := m.statusLine()
	m.setRootStatus(line)
	return line
}

func (m *RootModel) statusLine() string {
	sections := []string{}
	if name := m.workspaceName(); name != "" {
		label := fmt.Sprintf("Workspace: [%s]", name)
		if m.hasMultipleWorkspaces() {
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
			label += fmt.Sprintf(" (%s to switch)", nextKey)
		}
		sections = append(sections, rootHeaderWorkspaceStyle.Render(label))
	}

	sections = append(sections, rootHeaderStyle.Render("Views:"))
	sections = append(sections, highlight(viewNotes, m.active, formatShortcut(m.keys.notes)))
	sections = append(sections, highlight(viewTasks, m.active, formatShortcut(m.keys.tasks)))
	sections = append(sections, highlight(viewJournal, m.active, formatShortcut(m.keys.journal)))

	return lipgloss.JoinHorizontal(lipgloss.Left, sections...)
}

func (m *RootModel) setRootStatus(line string) {
	if m.state == nil && m.notes != nil {
		m.state = m.notes.state
	}

	if m.state == nil {
		return
	}

	if m.state.RootStatus == nil {
		m.state.RootStatus = &state.RootStatus{}
	}

	m.state.RootStatus.Line = line
}

func (m *RootModel) handleViewSwitch(msg tea.KeyMsg) bool {
	switch {
	case key.Matches(msg, m.keys.notes):
		m.active = viewNotes
		return true
	case key.Matches(msg, m.keys.tasks):
		m.active = viewTasks
		return true
	case key.Matches(msg, m.keys.journal):
		m.active = viewJournal
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
	m.state = newState

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
	var (
		adjustedMsg tea.WindowSizeMsg
		useAdjusted bool
	)
	if windowMsg, ok := msg.(tea.WindowSizeMsg); ok {
		adjustedMsg = windowMsg
		headerLines := lipgloss.Height(m.header())
		adjustedMsg.Height = windowMsg.Height - headerLines
		if adjustedMsg.Height < 0 {
			adjustedMsg.Height = 0
		}
		useAdjusted = true
	}

	if m.notes != nil {
		forward := msg
		if useAdjusted {
			forward = adjustedMsg
		}
		model, _ := m.notes.Update(forward)
		m.notes = adoptNoteModel(model, m.notes)
	}
	if m.tasks != nil {
		forward := msg
		if useAdjusted {
			forward = adjustedMsg
		}
		model, _ := m.tasks.Update(forward)
		m.tasks = adoptTasksModel(model, m.tasks)
	}
	if m.journal != nil {
		forward := msg
		if useAdjusted {
			forward = adjustedMsg
		}
		model, _ := m.journal.Update(forward)
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
