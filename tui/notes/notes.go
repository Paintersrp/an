// Package note handles the core note management functionality.
package notes

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Paintersrp/an/internal/cache"
	"github.com/Paintersrp/an/internal/config"
	v "github.com/Paintersrp/an/internal/views"
	"github.com/Paintersrp/an/pkg/fs/templater"
	"github.com/Paintersrp/an/pkg/fs/zet"
	"github.com/Paintersrp/an/utils"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// TODO: Replace Magic Number (Cache Size)
// TODO: Orphan as 2 not archive, archive as 3

type NoteListModel struct {
	config       *config.Config
	templater    *templater.Templater
	views        map[string]v.View
	cache        *cache.Cache
	list         list.Model
	keys         *listKeyMap
	delegateKeys *delegateKeyMap
	preview      string
	width        int
	height       int
	viewName     string
	showDetails  bool
	renaming     bool
	inputModel   InputModel
	formModel    FormModel
	creating     bool
}

func NewNoteListModel(
	cfg *config.Config,
	t *templater.Templater,
	views map[string]v.View,
	viewName string,
) (*NoteListModel, error) {
	files, err := v.GetFilesByView(views, viewName, cfg.VaultDir)
	if err != nil {
		return nil, err
	}

	items := ParseNoteFiles(files, cfg.VaultDir, false)
	dkeys := newDelegateKeyMap()
	lkeys := newListKeyMap()
	title := v.GetTitleForView(viewName)
	delegate := newItemDelegate(dkeys, cfg, viewName)

	l := list.New(items, delegate, 0, 0)
	l.Title = title
	l.Styles.Title = titleStyle

	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			lkeys.openNote,
			lkeys.changeView,
		}
	}

	l.AdditionalFullHelpKeys = lkeys.fullHelp
	c, err := cache.New(50)

	if err != nil {
		panic(err)
	}

	i := NewInputModel()
	f := NewFormModel(cfg, t)

	return &NoteListModel{
		config:       cfg,
		templater:    t,
		views:        views,
		cache:        c,
		list:         l,
		viewName:     viewName,
		keys:         lkeys,
		delegateKeys: dkeys,
		inputModel:   i,
		formModel:    f,
		renaming:     false,
		creating:     false,
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
		// Don't match any of the keys below if we're actively filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		if m.renaming {
			if key.Matches(msg, m.keys.exitAltView) {
				m.toggleRename()
				return m, nil
			}

			var cmd tea.Cmd
			m.inputModel.Input, cmd = m.inputModel.Input.Update(msg)
			cmds = append(cmds, cmd)

			if key.Matches(msg, m.keys.submitAltView) {
				err := renameFile(m)

				if err != nil {
					return m, nil
				}

				m.toggleRename()
				m.refresh()
				return m, cmd

			}

			return m, tea.Batch(cmds...)
		}

		if m.creating {
			if key.Matches(msg, m.keys.exitAltView) {
				m.toggleCreation()
				return m, nil
			}

			var cmd tea.Cmd
			m.formModel, cmd = m.formModel.Update(msg)
			cmds = append(cmds, cmd)

			return m, tea.Batch(cmds...)
		}

		switch {
		case key.Matches(msg, m.keys.openNote):
			if ok := m.openNote(); ok {
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

		case key.Matches(msg, m.keys.switchToArchiveView):
			return m, m.swapView("archive")

		case key.Matches(msg, m.keys.switchToOrphanView):
			return m, m.swapView("orphan")

		case key.Matches(msg, m.keys.switchToTrashView):
			return m, m.swapView("trash")

		case key.Matches(msg, m.keys.switchToUnfulfillView):
			return m, m.swapView("unfulfilled")

		case key.Matches(msg, m.keys.rename):
			m.toggleRename()

		case key.Matches(msg, m.keys.create):
			m.toggleCreation()
		}

	}

	nl, cmd := m.list.Update(msg)
	m.list = nl
	cmds = append(cmds, cmd)

	// we need to asyncronously generate the markdown preview, then
	// once we have the preview we update it which should update the display
	// while waiting, could just show preview as blank for now
	m.handlePreview()
	return m, tea.Batch(cmds...)
}

func (m NoteListModel) View() string {
	list := listStyle.MaxWidth(m.width / 2).Render(m.list.View())

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

func Run(
	c *config.Config,
	t *templater.Templater,
	views map[string]v.View,
	viewFlag string,
) error {
	// Save the current terminal state
	originalState, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatalf("Failed to get original terminal state: %v", err)
	}

	defer func() {
		// in the event that the editor we open into does not terminate gracefully,
		// we attempt to recover original state so that we can terminate gracefully (aka reach the return nil)
		if err := term.Restore(int(os.Stdin.Fd()), originalState); err != nil {
			log.Fatalf("Failed to restore original terminal state: %v", err)
		}
	}()

	m, err := NewNoteListModel(c, t, views, viewFlag)
	if err != nil {
		return err
	}

	if _, err := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithAltScreen()).Run(); err != nil {
		// handle error for instances where neovim/editor doesn't pass stdin back in time to close gracefully with bubbletea
		if strings.Contains(err.Error(), "resource temporarily unavailable") {
			os.Exit(0) // exit gracefully
		} else {
			log.Fatalf("Error running program: %v", err)
		}
	}

	// ran with no errors*, terminal gracefully
	return nil
}

// handles markdown preview generation for selected (highlighted) items
// caches up to 50mb of previews to avoid reprocessing when navigating the list
func (m *NoteListModel) handlePreview() {
	if s, ok := m.list.SelectedItem().(ListItem); ok {
		// check if the preview is already in the cache
		if p, exists, err := m.cache.Get(s.path); err == nil && exists {
			m.preview = p.(string)
		} else {
			// cache tries to recover from errors internally, so we *should* only see errors from
			// nil values and improper size on New (negative values)
			if err != nil {
				m.list.NewStatusMessage(statusStyle(fmt.Sprintf("Error accessing cache: %s", err)))
			}

			w := m.width / 2
			h := m.list.Height()

			r := utils.RenderMarkdownPreview(
				s.path,
				w,
				h,
			)

			if err := m.cache.Put(s.path, r); err != nil {
				m.list.NewStatusMessage(statusStyle(fmt.Sprintf("Error updating cache: %s", err)))
			} else {
				m.preview = r
			}
		}
	}
}

func (m *NoteListModel) refresh() tea.Cmd {
	m.list.Title = v.GetTitleForView(m.viewName)
	m.refreshDelegate()
	cmd := m.refreshItems()
	m.handlePreview()
	return cmd
}

// refreshes the list items based on view conditions
func (m *NoteListModel) refreshItems() tea.Cmd {
	files, _ := v.GetFilesByView(m.views, m.viewName, m.config.VaultDir)
	items := ParseNoteFiles(files, m.config.VaultDir, m.showDetails)
	return m.list.SetItems(items)
}

func (m *NoteListModel) refreshDelegate() {
	dkeys := newDelegateKeyMap()
	delegate := newItemDelegate(dkeys, m.config, m.viewName)
	m.list.SetDelegate(delegate)
}

// should prob use an error over a bool but a "success" flag sort of feels more natural for the context.
// unsuccessful opens provide a status message and the program stays live
// successful opens return true which trigger graceful stdin passing and closing of the program
func (m *NoteListModel) openNote() bool {
	var p string

	if i, ok := m.list.SelectedItem().(ListItem); ok {
		p = i.path
	} else {
		return false
	}

	err := zet.OpenFromPath(p)

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
	return m.refreshItems()
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

func (m *NoteListModel) toggleRename() {
	if !m.renaming {
		m.renaming = true
		m.inputModel.Input.Focus()
		if s, ok := m.list.SelectedItem().(ListItem); ok {
			m.inputModel.Input.SetValue(s.title)
		}
	} else {
		m.renaming = false
		m.inputModel.Input.Blur()
	}
}

// clear?
func (m *NoteListModel) toggleCreation() {
	if !m.creating {
		m.formModel.inputs[m.formModel.focused].Focus()
		m.creating = true
	} else {
		m.formModel.inputs[m.formModel.focused].Blur()
		m.creating = false
	}
}
