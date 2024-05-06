package notes

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Paintersrp/an/internal/cache"
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/fs/zet"
	"github.com/Paintersrp/an/utils"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// DONE: Would be nice to hold tab and see alt info like path and tertiary details
// TODO: cache mode outputs
// TODO: don't include files in the base vault dir in archive
// TODO: Replace panics
// TODO: Replace Magic Number (Cache Size)

type NoteListModel struct {
	modes             map[string]ModeConfig
	config            *config.Config
	cache             *cache.Cache
	keys              *listKeyMap
	delegateKeys      *delegateKeyMap
	list              list.Model
	preview           string
	width             int
	height            int
	modeFlag          string
	orphansFlag       bool
	showAsFileDetails bool
}

func NewNoteListModel(
	cfg *config.Config,
	modes map[string]ModeConfig,
	modeFlag string,
) NoteListModel {
	files, _ := getFilesByMode(modes, modeFlag, cfg.VaultDir)
	items := parseNoteFiles(files, cfg.VaultDir, false)

	dkeys := newDelegateKeyMap()
	lkeys := newListKeyMap()
	t := getTitleForMode(modeFlag)

	// Setup list
	delegate := newItemDelegate(dkeys, cfg)
	l := list.New(items, delegate, 0, 0)
	l.Title = t
	l.Styles.Title = titleStyle

	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			lkeys.openNote,
			lkeys.changeMode,
		}
	}

	l.AdditionalFullHelpKeys = lkeys.fullHelp
	c, err := cache.New(100)

	if err != nil {
		panic(err)
	}

	return NoteListModel{
		list:         l,
		keys:         lkeys,
		delegateKeys: dkeys,
		config:       cfg,
		cache:        c,
		modes:        modes,
		modeFlag:     modeFlag,
	}
}

func (m NoteListModel) Init() tea.Cmd {
	return nil
}

func (m NoteListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	nl, cmd := m.list.Update(msg)
	m.list = nl
	cmds = append(cmds, cmd)

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

		case key.Matches(msg, m.keys.toggleDisplayMode):
			m.showAsFileDetails = !m.showAsFileDetails
			cmd := m.refreshItems()
			return m, cmd

		case key.Matches(msg, m.keys.changeMode):
			m.cycleMode()
			cmd := m.refreshItems()
			return m, cmd

		case key.Matches(msg, m.keys.switchToDefaultMode):
			m.modeFlag = "default"
			m.list.Title = getTitleForMode(m.modeFlag)
			cmd := m.refreshItems()
			return m, cmd

		case key.Matches(msg, m.keys.switchToArchiveMode):
			m.modeFlag = "archive"
			m.list.Title = getTitleForMode(m.modeFlag)
			cmd := m.refreshItems()
			return m, cmd

		case key.Matches(msg, m.keys.switchToOrphanMode):
			m.modeFlag = "orphan"
			m.list.Title = getTitleForMode(m.modeFlag)
			cmd := m.refreshItems()
			return m, cmd
		}
	}

	m.handlePreview()
	return m, tea.Batch(cmds...)
}

func (m NoteListModel) View() string {
	list := listStyle.MaxWidth(m.width / 2).Render(m.list.View())
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
	modes map[string]ModeConfig,
	modeFlag string,
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

	if _, err := tea.NewProgram(NewNoteListModel(c, modes, modeFlag), tea.WithInput(os.Stdin), tea.WithAltScreen()).Run(); err != nil {
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
// caches up to 100 previews to avoid reprocessing when navigating the list
func (m *NoteListModel) handlePreview() {
	if s, ok := m.list.SelectedItem().(ListItem); ok {
		// check if the preview is already in the cache
		if p, exists, err := m.cache.Get(s.path); err == nil && exists {
			m.list.NewStatusMessage(statusStyle("Cache Hit"))
			m.preview = p.(string)
		} else {
			// cache tries to recover from errors internally, so we *should* only see errors from
			// nil values and improper size on New (negative values)
			if err != nil {
				m.list.NewStatusMessage(statusStyle(fmt.Sprintf("Error accessing cache: %s", err)))
			} else {
				m.list.NewStatusMessage(statusStyle("Cache Miss"))
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

// refreshes the list items based on mode conditions
func (m *NoteListModel) refreshItems() tea.Cmd {
	files, _ := getFilesByMode(m.modes, m.modeFlag, m.config.VaultDir)
	items := parseNoteFiles(files, m.config.VaultDir, m.showAsFileDetails)
	return m.list.SetItems(items)
}

// cycles through modes
// default -> archive
// archive -> orphan
// orphan -> default -> repeat
func (m *NoteListModel) cycleMode() {
	switch m.modeFlag {
	case "default":
		m.modeFlag = "archive"
	case "archive":
		m.modeFlag = "orphan"
	case "orphan":
		m.modeFlag = "default"
	default:
		m.modeFlag = "default"
	}

	m.list.Title = getTitleForMode(m.modeFlag)
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
