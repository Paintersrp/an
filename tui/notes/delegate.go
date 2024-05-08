package notes

import (
	"os"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/utils"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

var currView string

func newItemDelegate(
	keys *delegateKeyMap,
	cfg *config.Config,
	view string,
) list.DefaultDelegate {
	currView = view
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
				if currView == "default" || currView == "orphan" {
					if err := utils.Archive(p, cfg); err != nil {
						return m.NewStatusMessage(statusStyle("Failed to archive " + n))
					}
					i := m.Index()
					m.RemoveItem(i)
					return m.NewStatusMessage(statusStyle("Archived " + n))
				}

			case key.Matches(msg, keys.delete):
				if currView == "trash" { // Ensure we're in trash view
					if err := os.Remove(p); err != nil {
						return m.NewStatusMessage(statusStyle("Failed to delete " + n))
					}
					i := m.Index()
					m.RemoveItem(i)
					return m.NewStatusMessage(statusStyle("Deleted " + n))
				}

			case key.Matches(msg, keys.trash):
				if err := utils.Trash(p, cfg); err != nil {
					return m.NewStatusMessage(statusStyle("Failed to move " + n + " to trash"))
				}
				i := m.Index()
				m.RemoveItem(i)
				return m.NewStatusMessage(statusStyle("Moved " + n + " to trash"))

			case key.Matches(msg, keys.undo):
				switch currView {
				case "archive":
					if err := utils.Unarchive(p, cfg); err != nil {
						return m.NewStatusMessage(statusStyle("Failed to unarchive " + n))
					}
					i := m.Index()
					m.RemoveItem(i)
					return m.NewStatusMessage(statusStyle("Restored " + n))

				case "trash":
					if err := utils.Untrash(p, cfg); err != nil {
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

	var (
		longHelp  [][]key.Binding
		shortHelp []key.Binding
	)

	switch view {
	case "archive":
		shortHelp = []key.Binding{keys.trash, keys.undo}
		longHelp = [][]key.Binding{{keys.trash, keys.undo}}
	case "orphan":
		shortHelp = []key.Binding{keys.trash, keys.archive}
		longHelp = [][]key.Binding{{keys.trash, keys.archive}}
	case "trash":
		shortHelp = []key.Binding{keys.delete, keys.undo}
		longHelp = [][]key.Binding{{keys.delete, keys.undo}}
	default:
		shortHelp = []key.Binding{keys.trash, keys.archive}
		longHelp = [][]key.Binding{{keys.trash, keys.archive}}
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
			key.WithKeys("T"),
			key.WithHelp("T", "trash"),
		),
	}
}
