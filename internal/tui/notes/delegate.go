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
			case key.Matches(msg, keys.archive):
				if currView == "default" || currView == "orphan" {
					if err := h.Archive(p); err != nil {
						return m.NewStatusMessage(statusStyle("Failed to archive " + n))
					}
					return batchStatusWithRemoval(m, p, statusStyle("Archived "+n))
				}

			case key.Matches(msg, keys.delete):
				if currView == "trash" { // Ensure we're in trash view
					if err := os.Remove(p); err != nil {
						return m.NewStatusMessage(statusStyle("Failed to delete " + n))
					}
					return batchStatusWithRemoval(m, p, statusStyle("Deleted "+n))
				}

			case key.Matches(msg, keys.trash):
				if err := h.Trash(p); err != nil {
					return m.NewStatusMessage(statusStyle("Failed to move " + n + " to trash"))
				}
				return batchStatusWithRemoval(m, p, statusStyle("Moved "+n+" to trash"))

			case key.Matches(msg, keys.undo):
				switch currView {
				case "archive":
					if err := h.Unarchive(p); err != nil {
						return m.NewStatusMessage(statusStyle("Failed to unarchive " + n))
					}
					return batchStatusWithRemoval(m, p, statusStyle("Restored "+n))

				case "trash":
					if err := h.Untrash(p); err != nil {
						return m.NewStatusMessage(statusStyle("Failed to restore " + n))
					}
					return batchStatusWithRemoval(m, p, statusStyle("Restored "+n))
				}

			case key.Matches(msg, keys.keypadDelete):
				switch currView {
				default:
					if err := h.Trash(p); err != nil {
						return m.NewStatusMessage(statusStyle("Failed to move " + n + " to trash"))
					}
					return batchStatusWithRemoval(m, p, statusStyle("Moved "+n+" to trash"))

				case "trash":
					if err := os.Remove(p); err != nil {
						return m.NewStatusMessage(statusStyle("Failed to delete " + n))
					}
					return batchStatusWithRemoval(m, p, statusStyle("Deleted "+n))
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
		longHelp = [][]key.Binding{{keys.trash, keys.undo, keys.keypadDelete}}
	case "orphan":
		shortHelp = []key.Binding{keys.trash, keys.archive}
		longHelp = [][]key.Binding{{keys.trash, keys.archive, keys.keypadDelete}}
	case "trash":
		shortHelp = []key.Binding{keys.delete, keys.undo}
		longHelp = [][]key.Binding{{keys.delete, keys.undo, keys.keypadDelete}}
	default:
		shortHelp = []key.Binding{keys.trash, keys.archive}
		longHelp = [][]key.Binding{{keys.trash, keys.archive, keys.keypadDelete}}
	}

	d.ShortHelpFunc = func() []key.Binding {
		return shortHelp
	}

	d.FullHelpFunc = func() [][]key.Binding {
		return longHelp
	}
	return d
}

func batchStatusWithRemoval(m *list.Model, path, status string) tea.Cmd {
	removeCmd := removeItemByPath(m, path)
	statusCmd := m.NewStatusMessage(status)

	if removeCmd == nil {
		return statusCmd
	}

	return tea.Batch(removeCmd, statusCmd)
}

func removeItemByPath(m *list.Model, path string) tea.Cmd {
	if path == "" {
		return nil
	}

	items := m.Items()
	for idx, item := range items {
		li, ok := item.(ListItem)
		if !ok {
			continue
		}
		if li.path == path {
			newItems := make([]list.Item, 0, len(items)-1)
			newItems = append(newItems, items[:idx]...)
			newItems = append(newItems, items[idx+1:]...)
			return m.SetItems(newItems)
		}
	}

	return nil
}

type delegateKeyMap struct {
	archive      key.Binding
	undo         key.Binding
	delete       key.Binding
	trash        key.Binding
	keypadDelete key.Binding
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
		keypadDelete: key.NewBinding(
			key.WithKeys("delete"),
			key.WithHelp("delete", "delete (or trash)"),
		),
	}
}
