package notes

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Paintersrp/an/internal/cache"
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/templater"
	"github.com/Paintersrp/an/pkg/fs/zet"
	"github.com/Paintersrp/an/utils"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// DONE: Would be nice to hold tab and see alt info like path and tertiary details
// DONE: don't include files in the base vault dir in archive
// TODO: cache view outputs
// TODO: Replace panics
// TODO: Replace Magic Number (Cache Size)
// TODO: Orphan as 2 not archive, archive as 3

type NoteListModel struct {
	views             map[string]ViewConfig
	config            *config.Config
	cache             *cache.Cache
	keys              *listKeyMap
	delegateKeys      *delegateKeyMap
	list              list.Model
	preview           string
	width             int
	height            int
	viewFlag          string
	showAsFileDetails bool
	renaming          bool
	input             ListInputModel
	form              FormModel
	creating          bool
}

func NewNoteListModel(
	cfg *config.Config,
	t *templater.Templater,
	views map[string]ViewConfig,
	viewFlag string,
) (NoteListModel, *cache.Cache) {
	files, _ := getFilesByView(views, viewFlag, cfg.VaultDir)
	items := parseNoteFiles(files, cfg.VaultDir, false)
	dkeys := newDelegateKeyMap()
	lkeys := newListKeyMap()
	title := getTitleForView(viewFlag)
	delegate := newItemDelegate(dkeys, cfg, viewFlag)

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

	i := initialInputModel()
	f := initialFormModel(cfg, t)

	return NoteListModel{
		list:         l,
		keys:         lkeys,
		delegateKeys: dkeys,
		config:       cfg,
		cache:        c,
		views:        views,
		viewFlag:     viewFlag,
		input:        i,
		form:         f,
		renaming:     false,
		creating:     false,
	}, c
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
			// Handle exiting input mode
			if key.Matches(msg, m.keys.exitAltView) {
				m.input.Input.Blur()
				m.renaming = false
				return m, nil
			}

			// Update the text input and handle its commands
			var cmd tea.Cmd
			m.input.Input, cmd = m.input.Input.Update(msg)
			cmds = append(cmds, cmd)

			// Handle the case when Enter is pressed and the input is submitted
			if key.Matches(msg, m.keys.submitAltView) {
				// Retrieve the new name from the input model
				err := renameFile(m)

				if err != nil {
					return m, nil
				}

				m.renaming = false
				m.refresh()
				return m, cmd

			}

			return m, tea.Batch(cmds...)
		}

		if m.creating {
			// Handle exiting input mode
			if key.Matches(msg, m.keys.exitAltView) {
				m.form.inputs[m.form.focused].Blur()
				m.creating = false
				return m, nil
			}

			var cmd tea.Cmd
			m.form, cmd = m.form.Update(msg)
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

		case key.Matches(msg, m.keys.toggleDisplayView):
			m.showAsFileDetails = !m.showAsFileDetails
			cmd := m.refreshItems()
			return m, cmd

		case key.Matches(msg, m.keys.changeView):
			m.cycleView()
			cmd := m.refresh()
			return m, cmd

		case key.Matches(msg, m.keys.switchToDefaultView):
			m.viewFlag = "default"
			cmd := m.refresh()
			return m, cmd

		case key.Matches(msg, m.keys.switchToArchiveView):
			m.viewFlag = "archive"
			cmd := m.refresh()
			return m, cmd

		case key.Matches(msg, m.keys.switchToOrphanView):
			m.viewFlag = "orphan"
			cmd := m.refresh()
			return m, cmd

		case key.Matches(msg, m.keys.switchToTrashView):
			m.viewFlag = "trash"
			cmd := m.refresh()
			return m, cmd

		case key.Matches(msg, m.keys.switchToUnfulfillView):
			m.viewFlag = "unfulfilled"
			cmd := m.refresh()
			return m, cmd
		}

		if key.Matches(msg, m.keys.rename) {
			m.renaming = true
			m.input.Input.Focus()
			if s, ok := m.list.SelectedItem().(ListItem); ok {
				m.input.Input.SetValue(s.title)
			}
		}
		if key.Matches(msg, m.keys.create) {
			m.creating = true
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
		return appStyle.Render(modelStyle.Render(m.form.View()))
	}

	if m.renaming {
		textPrompt := textPromptStyle.Render(
			lipgloss.NewStyle().
				Height(m.list.Height()).
				MaxHeight(m.list.Height()).
				Padding(0, 2).
				Render(fmt.Sprintf("%s\n\n%s", titleStyle.Render("Rename File"), m.input.View())),
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
	views map[string]ViewConfig,
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

	m, mCache := NewNoteListModel(c, t, views, viewFlag)

	if _, err := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithAltScreen()).Run(); err != nil {
		// handle error for instances where neovim/editor doesn't pass stdin back in time to close gracefully with bubbletea
		if strings.Contains(err.Error(), "resource temporarily unavailable") {
			os.Exit(0) // exit gracefully
		} else {
			log.Fatalf("Error running program: %v", err)
		}
	}

	fmt.Printf("Cache size: %s\n", cache.ReadableSize(int64(mCache.SizeOf())))
	fmt.Printf("Cache size: %d\n", mCache.SizeOf())

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

			// calculate the width and height for the preview pane
			w := m.width / 2
			h := m.list.Height()

			// render the preview
			r := utils.RenderMarkdownPreview(
				s.path,
				w,
				h,
			)

			// add item preview to cache
			if err := m.cache.Put(s.path, r); err != nil {
				// handle the error appropriately, e.g., log it or show an error message
				m.list.NewStatusMessage(statusStyle(fmt.Sprintf("Error updating cache: %s", err)))
			} else {
				// no errors, update preview with rendered content
				m.preview = r
			}
		}
	}
}

func (m *NoteListModel) refresh() tea.Cmd {
	m.list.Title = getTitleForView(m.viewFlag)
	m.refreshDelegate()
	cmd := m.refreshItems()
	m.handlePreview()
	return cmd
}

// refreshes the list items based on view conditions
func (m *NoteListModel) refreshItems() tea.Cmd {
	files, _ := getFilesByView(m.views, m.viewFlag, m.config.VaultDir)
	items := parseNoteFiles(files, m.config.VaultDir, m.showAsFileDetails)
	return m.list.SetItems(items)
}

func (m *NoteListModel) refreshDelegate() {
	dkeys := newDelegateKeyMap()
	delegate := newItemDelegate(dkeys, m.config, m.viewFlag)
	m.list.SetDelegate(delegate)
}

// cycles through views
// default -> archive
// archive -> orphan
// orphan -> trash
// trash -> default -> repeat
func (m *NoteListModel) cycleView() {
	switch m.viewFlag {
	case "default":
		m.viewFlag = "archive"
	case "archive":
		m.viewFlag = "orphan"
	case "orphan":
		m.viewFlag = "trash"
	case "trash":
		m.viewFlag = "default"
	default:
		m.viewFlag = "default"
	}
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
