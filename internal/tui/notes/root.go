package notes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

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
}

type rootKeyMap struct {
	notes   key.Binding
	tasks   key.Binding
	journal key.Binding
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
		m.updateAll(msg)
		return m, nil
	case tea.QuitMsg:
		if m.notes != nil {
			model, _ := m.notes.Update(msg)
			m.notes = adoptNoteModel(model, m.notes)
		}
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.notes):
			m.active = viewNotes
			return m, nil
		case key.Matches(msg, m.keys.tasks):
			m.active = viewTasks
			return m, nil
		case key.Matches(msg, m.keys.journal):
			m.active = viewJournal
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

	return strings.Join(sections, "\n")
}

func (m *RootModel) header() string {
	sections := []string{"Views:"}
	sections = append(sections, highlight(viewNotes, m.active, "1. Notes"))
	sections = append(sections, highlight(viewTasks, m.active, "2. Tasks"))
	sections = append(sections, highlight(viewJournal, m.active, "3. Journal"))
	return strings.Join(sections, "  ")
}

func highlight(view rootView, active rootView, label string) string {
	if view == active {
		return fmt.Sprintf("[%s]", label)
	}
	return label
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
	case NoteListModel:
		return &m
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
