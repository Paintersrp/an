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
	rename              key.Binding
	link                key.Binding
	submitAltView       key.Binding
	exitAltView         key.Binding
	toggleDisplayMode   key.Binding
	switchToDefaultMode key.Binding
	switchToArchiveMode key.Binding
	switchToOrphanMode  key.Binding
	switchToTrashMode   key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		toggleTitleBar: key.NewBinding(
			key.WithKeys("B"),
			key.WithHelp("B", "toggle title"),
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
		rename: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "rename"),
		),
		link: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "link"),
		),
		submitAltView: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "submit"),
		),
		exitAltView: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "exit input mode"),
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
		switchToTrashMode: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "switch to trash mode"),
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
		m.rename,
		m.changeMode,
		m.switchToDefaultMode,
		m.switchToArchiveMode,
		m.switchToOrphanMode,
	}
}
