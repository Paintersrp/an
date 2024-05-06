package notes

import "github.com/charmbracelet/bubbles/key"

type listKeyMap struct {
	toggleTitleBar      key.Binding
	toggleStatusBar     key.Binding
	togglePagination    key.Binding
	toggleHelpMenu      key.Binding
	openNote            key.Binding
	toggleFocus         key.Binding
	quit                key.Binding
	changeMode          key.Binding
	toggleDisplayMode   key.Binding
	switchToDefaultMode key.Binding
	switchToArchiveMode key.Binding
	switchToOrphanMode  key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		toggleTitleBar: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "toggle title"),
		),
		toggleStatusBar: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "toggle status"),
		),
		togglePagination: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "toggle pagination"),
		),
		toggleHelpMenu: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "toggle help"),
		),
		toggleDisplayMode: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "Display"),
		),
		openNote: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "open"),
		),
		changeMode: key.NewBinding(
			key.WithKeys("M"),
			key.WithHelp("M", "mode"),
		),
		switchToDefaultMode: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "switch to default mode"),
		),
		switchToArchiveMode: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "switch to archive mode"),
		),
		switchToOrphanMode: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "switch to orphan mode"),
		),
	}
}

func (m listKeyMap) fullHelp() []key.Binding {
	return []key.Binding{
		m.toggleTitleBar,
		m.toggleStatusBar,
		m.togglePagination,
		m.toggleHelpMenu,
		m.toggleDisplayMode,
		m.openNote,
		m.changeMode,
		m.switchToDefaultMode,
		m.switchToArchiveMode,
		m.switchToOrphanMode,
	}
}
