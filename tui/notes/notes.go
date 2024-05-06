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

type NoteListModel struct {
	list              list.Model
	keys              *listKeyMap
	delegateKeys      *delegateKeyMap
	config            *config.Config
	previewContent    string
	previewCache      *cache.LRUCache
	width             int
	height            int
	modes             map[string]ModeConfig
	modeFlag          string
	orphansFlag       bool
	showAsFileDetails bool
}

func NewNoteListModel(
	cfg *config.Config,
	modes map[string]ModeConfig,
	modeFlag string,
) NoteListModel {
	noteFiles, _ := getFilesByMode(modes, modeFlag, cfg.VaultDir)
	items := parseNoteFiles(noteFiles, cfg.VaultDir, false)

	delegateKeys := newDelegateKeyMap()
	listKeys := newListKeyMap()
	listTitle := getTitleForMode(modeFlag)

	// Setup list
	delegate := newItemDelegate(delegateKeys, cfg)
	configList := list.New(items, delegate, 0, 0)
	configList.Title = listTitle
	configList.Styles.Title = titleStyle

	configList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.openNote,
			listKeys.changeMode,
		}
	}

	configList.AdditionalFullHelpKeys = listKeys.fullHelp
	return NoteListModel{
		list:         configList,
		keys:         listKeys,
		delegateKeys: delegateKeys,
		config:       cfg,
		previewCache: cache.NewLRUCache(100),
		modes:        modes,
		modeFlag:     modeFlag,
	}
}

func (m NoteListModel) Init() tea.Cmd {
	return nil
}

func (m NoteListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	newListModel, cmd := m.list.Update(msg)
	m.list = newListModel
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
	// Render the list and preview sections with borders
	listSection := listStyle.Render(m.list.View())
	previewSection := previewStyle.Render(
		lipgloss.NewStyle().
			Height(m.list.Height()).
			MaxHeight(m.list.Height()).
			Render(fmt.Sprintf("%s\n%s", titleStyle.Render("Preview"), m.previewContent)),
	)

	// Join the list and preview sections side by side
	layout := lipgloss.JoinHorizontal(lipgloss.Top, listSection, previewSection)

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

	return nil
}

func (m *NoteListModel) handlePreview() {
	if selectedItem, ok := m.list.SelectedItem().(ListItem); ok {
		// Check if the preview is already in the cache
		if cachedPreview, exists := m.previewCache.Get(selectedItem.path); exists {
			m.previewContent = cachedPreview.(string)
		} else {
			// Calculate the width and height for the preview pane
			previewWidth := m.width / 2
			previewHeight := m.list.Height()

			// Render the preview and store it in the cache
			renderedPreview := utils.RenderMarkdownPreview(
				selectedItem.path,
				previewWidth,
				previewHeight,
			)

			// Add item preview to cache
			m.previewCache.Put(selectedItem.path, renderedPreview)
			m.previewContent = renderedPreview
		}
	}

}

func (m *NoteListModel) refreshItems() tea.Cmd {
	noteFiles, _ := getFilesByMode(m.modes, m.modeFlag, m.config.VaultDir)
	items := parseNoteFiles(noteFiles, m.config.VaultDir, m.showAsFileDetails)
	return m.list.SetItems(items)
}

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

func (m *NoteListModel) openNote() bool {
	var filePath string

	if i, ok := m.list.SelectedItem().(ListItem); ok {
		filePath = i.path
	} else {
		return false
	}

	err := zet.OpenFromPath(filePath)

	if err != nil {
		m.list.NewStatusMessage(statusMessageStyle(fmt.Sprintf("Open Error: %s", err)))
		return false
	}

	return true
}
