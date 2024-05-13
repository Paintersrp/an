package pinList

import "github.com/charmbracelet/bubbles/key"

type listKeyMap struct {
	toggleTitleBar   key.Binding
	toggleStatusBar  key.Binding
	togglePagination key.Binding
	toggleHelpMenu   key.Binding
	swapView         key.Binding
	swapToTextView   key.Binding
	swapToTaskView   key.Binding
	openNote         key.Binding
	findSelect       key.Binding
	rename           key.Binding
	backToMain       key.Binding
	create           key.Binding
	find             key.Binding
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
		swapView: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "swap list view"),
		),
		swapToTextView: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "swap directly to text view"),
		),
		swapToTaskView: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "swap directly to task view"),
		),
		openNote: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "open"),
		),
		find: key.NewBinding(
			key.WithKeys("F"),
			key.WithHelp("F", "find"),
		),
		findSelect: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select (find)"),
		),
		rename: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "rename"),
		),
		create: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "create"),
		),
		backToMain: key.NewBinding(
			key.WithKeys("esc", "q"),
			key.WithHelp("esc", "back to main list"),
		),
	}
}

func fullHelp(keys *listKeyMap) []key.Binding {
	return []key.Binding{
		keys.toggleTitleBar,
		keys.toggleStatusBar,
		keys.togglePagination,
		keys.toggleHelpMenu,
		keys.swapView,
		keys.swapToTextView,
		keys.swapToTaskView,
		keys.openNote,
		keys.find,
		keys.findSelect,
		keys.rename,
		keys.backToMain,
		keys.create,
		keys.find,
	}
}

func shortHelp(keys *listKeyMap) []key.Binding {
	return []key.Binding{
		keys.openNote,
		keys.find,
		keys.create,
	}
}
