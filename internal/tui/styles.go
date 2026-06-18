package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// Charmy, adaptive palette that reads well on both light and dark terminals.
var (
	colPrimary = lipgloss.Color("212") // pink
	colDim     = lipgloss.AdaptiveColor{Light: "246", Dark: "245"}
	colWarn    = lipgloss.Color("214") // orange
	colErr     = lipgloss.Color("203") // red
	colOK      = lipgloss.Color("42")  // green
	colSelBg   = lipgloss.Color("57")  // indigo
	colSelFg   = lipgloss.Color("231") // near-white
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("231")).
			Background(colPrimary).
			Padding(0, 1)

	subtleStyle = lipgloss.NewStyle().Foreground(colDim)
	okStyle     = lipgloss.NewStyle().Foreground(colOK)
	errStyle    = lipgloss.NewStyle().Foreground(colErr)
	warnStyle   = lipgloss.NewStyle().Foreground(colWarn)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colPrimary).
			Padding(1, 3)

	boxTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(colPrimary)
)

// tableStyles styles the bubbles table to match the rest of the UI.
func tableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colDim).
		BorderBottom(true).
		Bold(true).
		Foreground(colPrimary)
	s.Selected = s.Selected.
		Foreground(colSelFg).
		Background(colSelBg).
		Bold(true)
	return s
}
