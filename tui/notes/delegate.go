package notes

import (
	"os"
	"path/filepath"

	"github.com/Paintersrp/an/internal/config"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

var currMode string

func newItemDelegate(
	keys *delegateKeyMap,
	cfg *config.Config,
	mode string,
) list.DefaultDelegate {
	currMode = mode
	d := list.NewDefaultDelegate()

	d.Styles.SelectedTitle = selectedItemStyle
	d.Styles.SelectedDesc = selectedItemStyle

	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		var (
			n string
			p string
		)

		if i, ok := m.SelectedItem().(ListItem); ok {
			n = i.fileName
			p = i.path
		} else {
			return nil
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.archive):
				if currMode == "default" {
					if err := archive(p, cfg); err != nil {
						return m.NewStatusMessage(statusStyle("Failed to archive " + n))
					}
					i := m.Index()
					m.RemoveItem(i)
					return m.NewStatusMessage(statusStyle("Archived " + n))
				}

			case key.Matches(msg, keys.delete):
				if currMode == "trash" { // Ensure we're in trash mode
					if err := os.Remove(p); err != nil {
						return m.NewStatusMessage(statusStyle("Failed to delete " + n))
					}
					i := m.Index()
					m.RemoveItem(i)
					return m.NewStatusMessage(statusStyle("Deleted " + n))
				}

			case key.Matches(msg, keys.trash):
				if err := trash(p, cfg); err != nil {
					return m.NewStatusMessage(statusStyle("Failed to move " + n + " to trash"))
				}
				i := m.Index()
				m.RemoveItem(i)
				return m.NewStatusMessage(statusStyle("Moved " + n + " to trash"))

			case key.Matches(msg, keys.undo):
				switch currMode {
				case "archive":
					if err := unarchive(p, cfg); err != nil {
						return m.NewStatusMessage(statusStyle("Failed to unarchive " + n))
					}
					i := m.Index()
					m.RemoveItem(i)
					return m.NewStatusMessage(statusStyle("Restored " + n))

				case "trash":
					if err := untrash(p, cfg); err != nil {
						return m.NewStatusMessage(statusStyle("Failed to restore " + n))
					}
					i := m.Index()
					m.RemoveItem(i)
					return m.NewStatusMessage(statusStyle("Restored " + n))

				}
			}
		}

		return nil
	}

	var shortHelp []key.Binding

	switch mode {
	case "archive":
		shortHelp = []key.Binding{keys.trash, keys.undo}
	case "orphan":
		shortHelp = []key.Binding{keys.trash, keys.link}
	case "trash":
		shortHelp = []key.Binding{keys.delete, keys.undo}
	default:
		shortHelp = []key.Binding{keys.trash, keys.archive}
	}

	// should swap longhelps too probably
	longHelp := [][]key.Binding{
		{keys.archive, keys.undo, keys.delete, keys.trash},
	}

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
	undo    key.Binding
	delete  key.Binding
	trash   key.Binding
	link    key.Binding
}

func newDelegateKeyMap() *delegateKeyMap {
	return &delegateKeyMap{
		archive: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "archive"),
		),
		undo: key.NewBinding(
			key.WithKeys("U"),
			key.WithHelp("U", "undo"),
		),
		delete: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "del"),
		),
		trash: key.NewBinding(
			key.WithKeys("V"),
			key.WithHelp("T", "trash"),
		),
		// for help text only really...
		link: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "link"),
		),
	}
}

func archive(path string, cfg *config.Config) error {
	// Get the subdirectory path relative to the vault directory
	subDir, err := filepath.Rel(cfg.VaultDir, filepath.Dir(path))
	if err != nil {
		return err
	}

	// Create the archive subdirectory path
	archiveSubDir := filepath.Join(cfg.VaultDir, "archive", subDir)
	if _, err := os.Stat(archiveSubDir); os.IsNotExist(err) {
		if err := os.MkdirAll(archiveSubDir, os.ModePerm); err != nil {
			return err
		}
	}

	// Move the note to the archive subdirectory
	newPath := filepath.Join(archiveSubDir, filepath.Base(path))
	if err := os.Rename(path, newPath); err != nil {
		return err
	}

	return nil
}

func unarchive(path string, cfg *config.Config) error {
	// Infer the original subdirectory from the archive path
	subDir, err := filepath.Rel(
		filepath.Join(cfg.VaultDir, "archive"),
		filepath.Dir(path),
	)
	if err != nil {
		return err
	}

	// Define the original directory where the notes should be restored
	originalDir := filepath.Join(cfg.VaultDir, subDir)

	// Move the note from the archive directory back to the original directory
	newPath := filepath.Join(originalDir, filepath.Base(path))
	if err := os.Rename(path, newPath); err != nil {
		return err
	}

	return nil
}

// Function to move a note to the trash directory
func trash(path string, cfg *config.Config) error {
	// Get the subdirectory path relative to the vault directory
	subDir, err := filepath.Rel(cfg.VaultDir, filepath.Dir(path))
	if err != nil {
		return err
	}

	// Define the trash directory path
	trashDir := filepath.Join(cfg.VaultDir, "trash", subDir)
	if _, err := os.Stat(trashDir); os.IsNotExist(err) {
		if err := os.MkdirAll(trashDir, os.ModePerm); err != nil {
			return err
		}
	}

	// Move the note to the trash directory
	newPath := filepath.Join(trashDir, filepath.Base(path))
	if err := os.Rename(path, newPath); err != nil {
		return err
	}

	return nil
}

// Function to restore a note from the trash directory
func untrash(path string, cfg *config.Config) error {
	// Infer the original subdirectory from the archive path
	subDir, err := filepath.Rel(
		filepath.Join(cfg.VaultDir, "trash"),
		filepath.Dir(path),
	)
	if err != nil {
		return err
	}

	// Define the original directory where the notes should be restored
	originalDir := filepath.Join(cfg.VaultDir, subDir)

	// Move the note from the trash directory back to the original directory
	newPath := filepath.Join(originalDir, filepath.Base(path))
	if err := os.Rename(path, newPath); err != nil {
		return err
	}

	return nil
}
