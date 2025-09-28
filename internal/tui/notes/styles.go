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

	previewMetadataSectionTitleStyle = lipgloss.NewStyle().
						Foreground(lipgloss.Color("#89dceb")).
						Bold(true)

	previewMetadataBulletStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#bac2de"))

	textPromptStyle = previewStyle.Copy()

	filterPaletteStyle = lipgloss.NewStyle().
				MarginLeft(1).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(lipgloss.Color("#334455"))

	previewPaletteStyle = lipgloss.NewStyle().
				MarginLeft(1).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(lipgloss.Color("#334455"))

	graphPaneStyle = lipgloss.NewStyle().
			MarginLeft(1).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("#334455"))

	previewPaletteTitleStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("#0AF"))

	graphPaneTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#0AF"))

	previewPaletteHeaderStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("#89dceb"))

	graphPaneHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#89dceb"))

	previewPaletteCursorStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#FFF")).
					Background(lipgloss.Color("#0AF"))

	graphPaneCursorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFF")).
				Background(lipgloss.Color("#0AF"))

	previewPaletteInactiveStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#cdd6f4"))

	graphPaneInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#cdd6f4"))

	graphPaneQueueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f9e2af"))

	previewPaletteEmptyStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#6c7086"))

	graphPaneEmptyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6c7086"))

	previewPaletteHelpStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#94e2d5"))

	graphPaneHelpStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#94e2d5"))

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
