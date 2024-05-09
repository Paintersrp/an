package pinList

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func newItemDelegate(keys *delegateKeyMap, cfg *config.Config) list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	d.Styles.SelectedTitle = selectedItemStyle
	d.Styles.SelectedDesc = selectedItemStyle

	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		var title string

		if i, ok := m.SelectedItem().(PinListItem); ok {
			title = i.Title()
		} else {
			return nil
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.rename):
				if title == "default" {
					return m.NewStatusMessage(statusMessageStyle("Cannot rename the default pin"))
				}
				pinType := ""

				if m.Title == "Available Task Pins" {
					pinType = "task"
				} else {
					pinType = "text"
				}

				cfg.RenamePin(title, "test", pinType, false)
				refreshItems(cfg, pinType, m)
			case key.Matches(msg, keys.makeDefault):
				if i, ok := m.SelectedItem().(PinListItem); ok {
					description := i.Description()
					// Determine the type of pin based on the title
					pinType := ""

					if m.Title == "Available Task Pins" {
						pinType = "task"
					} else {
						pinType = "text"
					}

					// Update the relevant default pin
					if pinType == "task" {
						cfg.PinnedTaskFile = description
					} else {
						cfg.PinnedFile = description
					}
					if err := cfg.ToFile(); err != nil {
						return m.NewStatusMessage(statusMessageStyle("Error saving the configuration: " + err.Error()))
					}

					return tea.Batch(
						m.NewStatusMessage(
							statusMessageStyle("Default "+pinType+" pin replaced with "+description),
						),
						refreshItems(cfg, pinType, m),
					)
				}

			case key.Matches(msg, keys.remove):
				if title == "default" {
					cfg.ClearPinnedFile("text", false)
					m.SelectedItem()
					m.SetItem(m.Index(), PinListItem{title: title, description: "No Default Pinned File"})
					return m.NewStatusMessage(statusMessageStyle("Cannot delete default pin. Cleared instead."))
				}

				cfg.DeleteNamedPin(title, "text", false)
				index := m.Index()
				m.RemoveItem(index)

				return m.NewStatusMessage(statusMessageStyle("Deleted " + title))
			}
		}

		return nil
	}

	help := []key.Binding{keys.makeDefault, keys.remove, keys.rename}

	d.ShortHelpFunc = func() []key.Binding {
		return help
	}

	d.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{help}
	}

	return d
}

type delegateKeyMap struct {
	makeDefault key.Binding
	remove      key.Binding
	rename      key.Binding
}

func newDelegateKeyMap() *delegateKeyMap {
	return &delegateKeyMap{
		makeDefault: key.NewBinding(
			key.WithKeys("M"),
			key.WithHelp("M", "make default"),
		),
		remove: key.NewBinding(
			key.WithKeys("D", "backspace", "delete"),
			key.WithHelp("D", "delete"),
		),
		rename: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "rename"),
		),
	}
}

func refreshItems(cfg *config.Config, pinType string, m *list.Model) tea.Cmd {
	items := getItemsByType(cfg, pinType)
	return m.SetItems(items)
}
