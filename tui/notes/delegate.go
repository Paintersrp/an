package notes

import (
	"os"
	"path/filepath"

	"github.com/Paintersrp/an/internal/config"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func newItemDelegate(
	keys *delegateKeyMap,
	cfg *config.Config,
) list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	d.Styles.SelectedTitle = selectedItemStyle
	d.Styles.SelectedDesc = selectedItemStyle

	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		var (
			filename string
			path     string
		)

		if i, ok := m.SelectedItem().(ListItem); ok {
			filename = i.fileName
			path = i.path
		} else {
			return nil
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.archive):
				// Call ArchiveNote to move the note to the archive directory
				if err := archiveNote(path, cfg); err != nil {
					// Handle the error, perhaps set a status message indicating failure
					return m.NewStatusMessage(statusMessageStyle("Failed to archive " + filename))
				}
				i := m.Index()
				m.RemoveItem(i)
				return m.NewStatusMessage(statusMessageStyle("Archived " + filename))

			case key.Matches(msg, keys.delete):
				return m.NewStatusMessage(statusMessageStyle("You chose " + filename))

			case key.Matches(msg, keys.rename):
				return m.NewStatusMessage(statusMessageStyle("You chose " + filename))

			}
		}

		return nil
	}

	shortHelp := []key.Binding{keys.delete, keys.rename}
	longHelp := [][]key.Binding{{keys.archive, keys.delete, keys.rename}}

	d.ShortHelpFunc = func() []key.Binding {
		return shortHelp
	}

	d.FullHelpFunc = func() [][]key.Binding {
		return longHelp
	}
	return d
}

type delegateKeyMap struct {
	archive key.Binding
	delete  key.Binding
	rename  key.Binding
}

func newDelegateKeyMap() *delegateKeyMap {
	return &delegateKeyMap{
		archive: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "archive"),
		),
		delete: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "del"),
		),
		rename: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "rename"),
		),
	}
}

func archiveNote(path string, cfg *config.Config) error {
	archivePath := filepath.Join(cfg.VaultDir, "archive")

	// Check if the archive directory exists, if not create it
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		if err := os.MkdirAll(archivePath, os.ModePerm); err != nil {
			return err
		}
	}

	// Move the note to the archive directory
	newPath := filepath.Join(archivePath, filepath.Base(path))
	if err := os.Rename(path, newPath); err != nil {
		return err
	}

	return nil
}
