package notes

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0AF")).
			Background(lipgloss.Color("transparent")).
			Padding(0, 1)

	inputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			Padding(0, 1).Width(100)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#0AF", Dark: "#0AF"}).
			Render

	focusedStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#0AF")).
			Foreground(lipgloss.Color("#FFF"))

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#0AF")).
				Background(lipgloss.Color("#224")).
				Padding(0, 0)

	listStyle = lipgloss.NewStyle().
			MarginRight(1).
			Border(lipgloss.NormalBorder(), false, false, false, false).
			BorderForeground(lipgloss.Color("#334455"))

	previewStyle = lipgloss.NewStyle().
			MarginLeft(1).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("#334455"))

	textPromptStyle = previewStyle.Copy()

	cursorStyle = focusedStyle.Copy()
	textStyle   = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCC"))
)
