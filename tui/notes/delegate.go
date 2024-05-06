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
				if err := archive(p, cfg); err != nil {
					return m.NewStatusMessage(statusStyle("Failed to archive " + n))
				}
				i := m.Index()
				m.RemoveItem(i)
				return m.NewStatusMessage(statusStyle("Archived " + n))

			case key.Matches(msg, keys.delete):
				return m.NewStatusMessage(statusStyle("You chose " + n))

			case key.Matches(msg, keys.rename):
				return m.NewStatusMessage(statusStyle("You chose " + n))

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

func archive(path string, cfg *config.Config) error {
	p := filepath.Join(cfg.VaultDir, "archive")

	// Check if the archive directory exists, if not create it
	if _, err := os.Stat(p); os.IsNotExist(err) {
		if err := os.MkdirAll(p, os.ModePerm); err != nil {
			return err
		}
	}

	// Move the note to the archive directory
	new := filepath.Join(p, filepath.Base(path))
	if err := os.Rename(path, new); err != nil {
		return err
	}

	return nil
}
