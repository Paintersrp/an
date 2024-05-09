package notes

import "github.com/charmbracelet/bubbles/key"

type listKeyMap struct {
	toggleTitleBar        key.Binding
	toggleStatusBar       key.Binding
	togglePagination      key.Binding
	toggleHelpMenu        key.Binding
	openNote              key.Binding
	toggleFocus           key.Binding
	quit                  key.Binding
	changeView            key.Binding
	rename                key.Binding
	create                key.Binding
	link                  key.Binding
	submitAltView         key.Binding
	exitAltView           key.Binding
	toggleDisplayView     key.Binding
	switchToDefaultView   key.Binding
	switchToArchiveView   key.Binding
	switchToOrphanView    key.Binding
	switchToTrashView     key.Binding
	switchToUnfulfillView key.Binding
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
		toggleDisplayView: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "Display"),
		),
		openNote: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "open"),
		),
		changeView: key.NewBinding(
			key.WithKeys("V"),
			key.WithHelp("V", "view"),
		),
		rename: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "rename"),
		),
		create: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "create"),
		),
		submitAltView: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "submit (alt view)"),
		),
		exitAltView: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "exit alt view"),
		),
		switchToDefaultView: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "switch to default view"),
		),
		switchToArchiveView: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "switch to archive view"),
		),
		switchToOrphanView: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "switch to orphan view"),
		),
		switchToTrashView: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "switch to trash view"),
		),
		switchToUnfulfillView: key.NewBinding(
			key.WithKeys("5"),
			key.WithHelp("5", "switch to unfulfilled view"),
		),
	}
}

func (m listKeyMap) fullHelp() []key.Binding {
	return []key.Binding{
		m.toggleTitleBar,
		m.toggleStatusBar,
		m.togglePagination,
		m.toggleHelpMenu,
		m.toggleDisplayView,
		m.openNote,
		m.rename,
		m.changeView,
		m.switchToDefaultView,
		m.switchToArchiveView,
		m.switchToOrphanView,
		m.switchToTrashView,
		m.switchToUnfulfillView,
		m.exitAltView,
		m.submitAltView,
	}
}
