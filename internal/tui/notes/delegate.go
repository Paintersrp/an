package notes

import (
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paintersrp/an/internal/handler"
)

var currView string

func newItemDelegate(
	keys *delegateKeyMap,
	h *handler.FileHandler,
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
				if err := h.Trash(p); err != nil {
					return m.NewStatusMessage(statusStyle("Failed to move " + n + " to trash"))
				}
				i := m.Index()
				m.RemoveItem(i)
				return m.NewStatusMessage(statusStyle("Moved " + n + " to trash"))

			case key.Matches(msg, keys.undo):
				switch currView {
				case "trash":
					if err := h.Untrash(p); err != nil {
						return m.NewStatusMessage(statusStyle("Failed to restore " + n))
					}
					i := m.Index()
					m.RemoveItem(i)
					return m.NewStatusMessage(statusStyle("Restored " + n))
				}

			case key.Matches(msg, keys.keypadDelete):
				switch currView {
				default:
					if err := h.Trash(p); err != nil {
						return m.NewStatusMessage(statusStyle("Failed to move " + n + " to trash"))
					}
					i := m.Index()
					m.RemoveItem(i)
					return m.NewStatusMessage(statusStyle("Moved " + n + " to trash"))

				case "trash":
					if err := os.Remove(p); err != nil {
						return m.NewStatusMessage(statusStyle("Failed to delete " + n))
					}
					i := m.Index()
					m.RemoveItem(i)
					return m.NewStatusMessage(statusStyle("Deleted " + n))
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
	case "orphan":
		shortHelp = []key.Binding{keys.trash}
		longHelp = [][]key.Binding{{keys.trash, keys.keypadDelete}}
	case "trash":
		shortHelp = []key.Binding{keys.delete, keys.undo}
		longHelp = [][]key.Binding{{keys.delete, keys.undo, keys.keypadDelete}}
	default:
		shortHelp = []key.Binding{keys.trash}
		longHelp = [][]key.Binding{{keys.trash, keys.keypadDelete}}
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
	undo         key.Binding
	delete       key.Binding
	trash        key.Binding
	keypadDelete key.Binding
}

func newDelegateKeyMap() *delegateKeyMap {
	return &delegateKeyMap{
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
		keypadDelete: key.NewBinding(
			key.WithKeys("delete"),
			key.WithHelp("delete", "delete (or trash)"),
		),
	}
}
