// Package note handles the core note management functionality.
package notes

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/Paintersrp/an/internal/cache"
	"github.com/Paintersrp/an/internal/note"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/tui/notes/submodels"
	v "github.com/Paintersrp/an/internal/views"
	"github.com/Paintersrp/an/utils"
)

var maxCacheSizeMB int64 = 50

type NoteListModel struct {
	list         list.Model
	cache        *cache.Cache
	keys         *listKeyMap
	delegateKeys *delegateKeyMap
	state        *state.State
	preview      string
	viewName     string
	formModel    submodels.FormModel
	inputModel   submodels.InputModel
	width        int
	height       int
	renaming     bool
	showDetails  bool
	creating     bool
	copying      bool
	sortField    sortField
	sortOrder    sortOrder
}

func NewNoteListModel(
	s *state.State,
	viewName string,
) (*NoteListModel, error) {
	files, err := s.ViewManager.GetFilesByView(viewName, s.Vault)
	if err != nil {
		return nil, err
	}

	items := ParseNoteFiles(files, s.Vault, false)
	sortedItems := sortItems(castToListItems(items), sortByModifiedAt, descending)

	dkeys := newDelegateKeyMap()
	lkeys := newListKeyMap()
	title := v.GetTitleForView(viewName, int(sortByModifiedAt), int(descending))
	delegate := newItemDelegate(dkeys, s.Handler, viewName)

	l := list.New(sortedItems, delegate, 0, 0)
	l.Title = title
	l.Styles.Title = titleStyle

	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			lkeys.openNote,
			lkeys.changeView,
		}
	}

	l.AdditionalFullHelpKeys = lkeys.fullHelp
	c, err := cache.New(maxCacheSizeMB)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	i := submodels.NewInputModel()
	f := submodels.NewFormModel(s)

	return &NoteListModel{
		state:        s,
		cache:        c,
		list:         l,
		viewName:     viewName,
		keys:         lkeys,
		delegateKeys: dkeys,
		inputModel:   i,
		formModel:    f,
		renaming:     false,
		creating:     false,
		copying:      false,
		sortField:    sortByModifiedAt,
		sortOrder:    descending,
	}, nil
}

func (m NoteListModel) Init() tea.Cmd {
	return nil
}

func (m NoteListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case m.copying:
			return m.handleCopyUpdate(msg)
		case m.renaming:
			return m.handleRenameUpdate(msg)
		case m.creating:
			return m.handleCreationUpdate(msg)
		default:
			m.handleDefaultUpdate(msg)

			if m.state.Config.Editor == "vim" || m.state.Config.Editor == "nano" {
				if key.Matches(msg, m.keys.openNote) {
					return m, tea.Quit
				}
			}

		}
	}

	nl, cmd := m.list.Update(msg)
	m.list = nl
	cmds = append(cmds, cmd)

	// TODO: need to asyncronously stream in the markdown preview
	m.handlePreview()
	return m, tea.Batch(cmds...)
}

func (m NoteListModel) handleCopyUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if key.Matches(msg, m.keys.exitAltView) {
		m.toggleCopy()
		return m, nil
	}

	m.inputModel.Input, cmd = m.inputModel.Input.Update(msg)
	cmds = append(cmds, cmd)

	if key.Matches(msg, m.keys.submitAltView) {
		if err := copyFile(m); err != nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Error copying file: %v", err)),
			)
		} else {
			m.toggleCopy()
			m.refresh()
			return m, cmd
		}
	}

	return m, tea.Batch(cmds...)
}

func (m NoteListModel) handleRenameUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if key.Matches(msg, m.keys.exitAltView) {
		m.toggleRename()
		return m, nil
	}

	m.inputModel.Input, cmd = m.inputModel.Input.Update(msg)
	cmds = append(cmds, cmd)

	if key.Matches(msg, m.keys.submitAltView) {
		if err := renameFile(m); err != nil {
			m.list.NewStatusMessage(
				statusStyle(fmt.Sprintf("Error renaming file: %v", err)),
			)
		} else {
			m.toggleRename()
			m.refresh()
			return m, cmd
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *NoteListModel) handleCreationUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if key.Matches(msg, m.keys.exitAltView) {
		m.toggleCreation()
		return m, nil
	}

	m.formModel, cmd = m.formModel.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// TODO: returns are kinda unnecessary now
func (m *NoteListModel) handleDefaultUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.openNote):
		if ok := m.openNote(false); ok {
			return m, tea.Quit
		} else {
			return m, nil
		}

	case key.Matches(msg, m.keys.openNoteInObsidian):
		if ok := m.openNote(true); ok {
			return m, tea.Quit
		} else {
			return m, nil
		}

	case key.Matches(msg, m.keys.toggleTitleBar):
		m.toggleTitleBar()
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

	case key.Matches(msg, m.keys.toggleDisplayView):
		return m, m.toggleDetails()

	case key.Matches(msg, m.keys.changeView):
		return m, m.cycleView()

	case key.Matches(msg, m.keys.switchToDefaultView):
		return m, m.swapView("default")

	case key.Matches(msg, m.keys.switchToUnfulfillView):
		return m, m.swapView("unfulfilled")

	case key.Matches(msg, m.keys.switchToOrphanView):
		return m, m.swapView("orphan")

	case key.Matches(msg, m.keys.switchToArchiveView):
		return m, m.swapView("archive")

	case key.Matches(msg, m.keys.switchToTrashView):
		return m, m.swapView("trash")

	case key.Matches(msg, m.keys.rename):
		m.toggleRename()

	case key.Matches(msg, m.keys.create):
		m.toggleCreation()

	case key.Matches(msg, m.keys.copy):
		m.toggleCopy()

	case key.Matches(msg, m.keys.sortByTitle):
		m.sortField = sortByTitle
		return m, m.refreshSort()

	case key.Matches(msg, m.keys.sortBySubdir):
		m.sortField = sortBySubdir
		return m, m.refreshSort()

	case key.Matches(msg, m.keys.sortByModifiedAt):
		m.sortField = sortByModifiedAt
		return m, m.refreshSort()

	case key.Matches(msg, m.keys.sortAscending):
		m.sortOrder = ascending
		return m, m.refreshSort()

	case key.Matches(msg, m.keys.sortAscending):
		m.sortOrder = descending
		return m, m.refreshSort()

	case key.Matches(msg, m.keys.sortDescending):
		m.sortOrder = descending
		return m, m.refreshSort()
	}
	return m, nil
}

func (m NoteListModel) View() string {
	list := listStyle.MaxWidth(m.width / 2).Render(m.list.View())

	if m.copying {
		textPrompt := textPromptStyle.Render(
			lipgloss.NewStyle().
				Height(m.list.Height()).
				MaxHeight(m.list.Height()).
				Padding(0, 2).
				Render(fmt.Sprintf("%s\n\n%s\n\n%s", titleStyle.Render("Choose new name for the copy"), m.inputModel.View(), helpStyle.Render("do not include file extension"))),
		)

		layout := lipgloss.JoinHorizontal(lipgloss.Top, list, textPrompt)
		return appStyle.Render(layout)
	}

	if m.creating {
		modelStyle := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).Padding(0, 1)
		return appStyle.Render(modelStyle.Render(m.formModel.View()))
	}

	if m.renaming {
		textPrompt := textPromptStyle.Render(
			lipgloss.NewStyle().
				Height(m.list.Height()).
				MaxHeight(m.list.Height()).
				Padding(0, 2).
				Render(fmt.Sprintf("%s\n\n%s", titleStyle.Render("Rename File"), m.inputModel.View())),
		)

		layout := lipgloss.JoinHorizontal(lipgloss.Top, list, textPrompt)
		return appStyle.Render(layout)
	}

	preview := previewStyle.Render(
		lipgloss.NewStyle().
			Height(m.list.Height()).
			MaxHeight(m.list.Height()).
			Render(fmt.Sprintf("%s\n%s", titleStyle.Render("Preview"), m.preview)),
	)

	layout := lipgloss.JoinHorizontal(lipgloss.Top, list, preview)
	return appStyle.Render(layout)
}

func Run(s *state.State, views map[string]v.View, viewFlag string) error {
	originalState, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatalf("Failed to get original terminal state: %v", err)
	}

	defer func() {
		// we attempt to recover original state so that we can terminate gracefully
		if err := term.Restore(int(os.Stdin.Fd()), originalState); err != nil {
			log.Fatalf("Failed to restore original terminal state: %v", err)
		}
	}()

	m, err := NewNoteListModel(s, viewFlag)
	if err != nil {
		return err
	}

	if _, err := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithAltScreen()).Run(); err != nil {
		// handle error for instances where neovim/editor doesn't pass stdin back in time to close gracefully with bubbletea
		if strings.Contains(err.Error(), "resource temporarily unavailable") {
			os.Exit(0)
		} else {
			log.Fatalf("Error running program: %v", err)
		}
	}

	return nil
}

func (m *NoteListModel) handlePreview() {
	if s, ok := m.list.SelectedItem().(ListItem); ok {
		if p, exists, err := m.cache.Get(s.path); err == nil && exists {
			m.preview = p.(string)
		} else {
			if err != nil {
				m.list.NewStatusMessage(statusStyle(fmt.Sprintf("Error accessing cache: %s", err)))
			}

			w := m.width / 2
			h := m.list.Height()
			r := utils.RenderMarkdownPreview(s.path, w, h)

			if err := m.cache.Put(s.path, r); err != nil {
				m.list.NewStatusMessage(statusStyle(fmt.Sprintf("Error updating cache: %s", err)))
			} else {
				m.preview = r
			}
		}
	}
}

func (m *NoteListModel) refresh() tea.Cmd {
	m.list.Title = v.GetTitleForView(m.viewName, int(m.sortField), int(m.sortOrder))
	m.refreshDelegate()
	cmd := m.refreshItems()
	m.list.ResetSelected()
	m.handlePreview()
	return cmd
}

func (m *NoteListModel) refreshItems() tea.Cmd {
	files, _ := m.state.ViewManager.GetFilesByView(m.viewName, m.state.Vault)
	items := ParseNoteFiles(files, m.state.Vault, m.showDetails)
	sortedItems := sortItems(castToListItems(items), m.sortField, m.sortOrder)
	return m.list.SetItems(sortedItems)
}

func (m *NoteListModel) refreshDelegate() {
	dkeys := newDelegateKeyMap()
	delegate := newItemDelegate(dkeys, m.state.Handler, m.viewName)
	m.list.SetDelegate(delegate)
}

func (m *NoteListModel) refreshSort() tea.Cmd {
	m.list.Title = v.GetTitleForView(m.viewName, int(m.sortField), int(m.sortOrder))
	items := castToListItems(m.list.Items())
	sortedItems := sortItems(items, m.sortField, m.sortOrder)
	m.list.ResetSelected()
	cmd := m.list.SetItems(sortedItems)
	m.handlePreview()
	return cmd
}

// TODO: should prob use an error over a bool but a "success" flag sort of feels more natural for the context.
// TODO: unsuccessful opens provide a status message and the program stays live
// TODO: successful opens return true which trigger graceful stdin passing and closing of the program
func (m *NoteListModel) openNote(obsidian bool) bool {
	var p string

	if i, ok := m.list.SelectedItem().(ListItem); ok {
		p = i.path
	} else {
		return false
	}

	err := note.OpenFromPath(p, obsidian)
	if err != nil {
		m.list.NewStatusMessage(statusStyle(fmt.Sprintf("Open Error: %s", err)))
		return false
	}

	return true
}

func (m *NoteListModel) toggleTitleBar() {
	v := !m.list.ShowTitle()
	m.list.SetShowTitle(v)
	m.list.SetShowFilter(v)
	m.list.SetFilteringEnabled(v)
}

func (m *NoteListModel) toggleDetails() tea.Cmd {
	m.showDetails = !m.showDetails
	cmd := m.refreshItems()
	m.list.ResetSelected()
	return cmd
}

func (m *NoteListModel) cycleView() tea.Cmd {
	switch m.viewName {
	case "default":
		m.viewName = "archive"
	case "archive":
		m.viewName = "orphan"
	case "orphan":
		m.viewName = "trash"
	case "trash":
		m.viewName = "default"
	default:
		m.viewName = "default"
	}

	return m.refresh()
}

func (m *NoteListModel) swapView(newView string) tea.Cmd {
	m.viewName = newView
	return m.refresh()
}

func (m *NoteListModel) toggleCopy() {
	switch m.copying {
	case true:
		m.copying = false
		m.inputModel.Input.Blur()
	case false:
		m.copying = true
		m.inputModel.Input.Focus()
		if s, ok := m.list.SelectedItem().(ListItem); ok {
			m.inputModel.Input.SetValue(s.title + "-copy")
		}
	}
}

func (m *NoteListModel) toggleRename() {
	switch m.renaming {
	case true:
		m.renaming = false
		m.inputModel.Input.Blur()
	case false:
		m.renaming = true
		m.inputModel.Input.Focus()
		if s, ok := m.list.SelectedItem().(ListItem); ok {
			m.inputModel.Input.SetValue(s.title)
		}
	}
}

// TODO: clear?
func (m *NoteListModel) toggleCreation() {
	switch m.creating {
	case true:
		m.formModel.Inputs[m.formModel.Focused].Blur()
		m.creating = false
	case false:
		m.formModel.Inputs[m.formModel.Focused].Focus()
		m.creating = true
	}
}
