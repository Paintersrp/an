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
	filterPalette         key.Binding
	previewPalette        key.Binding
	toggleGraphPane       key.Binding
	rename                key.Binding
	create                key.Binding
	copy                  key.Binding
	editInline            key.Binding
	quickCapture          key.Binding
	link                  key.Binding
	submitAltView         key.Binding
	exitAltView           key.Binding
	toggleDisplayView     key.Binding
	switchToDefaultView   key.Binding
	switchToArchiveView   key.Binding
	switchToOrphanView    key.Binding
	switchToTrashView     key.Binding
	switchToUnfulfillView key.Binding
	updatePreview         key.Binding
	openNoteInObsidian    key.Binding
	sortByTitle           key.Binding
	sortBySubdir          key.Binding
	sortByModifiedAt      key.Binding
	sortAscending         key.Binding
	sortDescending        key.Binding
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
		toggleFocus: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("⇧+tab", "focus preview"),
		),
		openNoteInObsidian: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "open in obsidian"),
		),
		changeView: key.NewBinding(
			key.WithKeys("V"),
			key.WithHelp("V", "view"),
		),
		filterPalette: key.NewBinding(
			key.WithKeys("F"),
			key.WithHelp("F", "filters"),
		),
		previewPalette: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "preview links"),
		),
		toggleGraphPane: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "backlink graph"),
		),
		rename: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "rename"),
		),
		create: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "create"),
		),
		copy: key.NewBinding(
			key.WithKeys("Y"),
			key.WithHelp("Y", "copy"),
		),
		editInline: key.NewBinding(
			key.WithKeys("E"),
			key.WithHelp("E", "inline edit"),
		),
		quickCapture: key.NewBinding(
			key.WithKeys("Q"),
			key.WithHelp("Q", "scratch capture"),
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
		switchToOrphanView: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "switch to orphan view"),
		),
		switchToUnfulfillView: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "switch to unfulfilled view"),
		),
		switchToArchiveView: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "switch to archive view"),
		),
		switchToTrashView: key.NewBinding(
			key.WithKeys("5"),
			key.WithHelp("5", "switch to trash view"),
		),
		updatePreview: key.NewBinding(
			key.WithKeys("f9"),
			key.WithHelp("f9", "update preview"),
		),
		sortByTitle: key.NewBinding(
			key.WithKeys("f1"),
			key.WithHelp("f1", "sort by title"),
		),
		sortBySubdir: key.NewBinding(
			key.WithKeys("f2"),
			key.WithHelp("f2", "sort by subdirectory"),
		),
		sortByModifiedAt: key.NewBinding(
			key.WithKeys("f3"),
			key.WithHelp("f3", "sort by modified"),
		),
		sortAscending: key.NewBinding(
			key.WithKeys("f5"),
			key.WithHelp("f5", "ascending sort"),
		),
		sortDescending: key.NewBinding(
			key.WithKeys("f6"),
			key.WithHelp("f6", "descending sort"),
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
		m.toggleFocus,
		m.openNote,
		m.editInline,
		m.quickCapture,
		m.filterPalette,
		m.previewPalette,
		m.toggleGraphPane,
		m.rename,
		m.copy,
		m.changeView,
		m.switchToDefaultView,
		m.switchToArchiveView,
		m.switchToOrphanView,
		m.switchToTrashView,
		m.switchToUnfulfillView,
		m.exitAltView,
		m.submitAltView,
		m.openNoteInObsidian,
		m.sortByTitle,
		m.sortBySubdir,
	}
}
