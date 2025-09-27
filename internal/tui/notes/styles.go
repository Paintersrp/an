package notes

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0AF")).
			Background(lipgloss.Color("transparent")).
			Bold(true).
			Padding(0, 1)

	inputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			Padding(0, 1).Width(100)

	statusBannerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#0AF", Dark: "#0AF"})

	previewSummaryStyle = statusBannerStyle.Copy().
				Padding(0, 1)

	statusStyle = statusBannerStyle.Render

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

	filterPaletteStyle = lipgloss.NewStyle().
				MarginLeft(1).
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#334455"))

	linkSelectStyle = lipgloss.NewStyle().MarginLeft(1).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("#334455"))

	cursorStyle = focusedStyle.Copy()
	textStyle   = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCC"))
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cba6f7"))
)

func renderHelpWithinWidth(width int, content string) string {
	if width <= 0 {
		return helpStyle.Render(content)
	}

	return helpStyle.Copy().
		Width(width).
		MaxWidth(width).
		Render(content)
}
