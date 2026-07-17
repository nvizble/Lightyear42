// Package tui renders CLI output (cards, tables) with Lip Gloss and will
// host the Bubble Tea dashboard in Milestone 5.
package tui

import "github.com/charmbracelet/lipgloss"

// Shared color palette. Adaptive colors keep output readable on both
// light and dark terminals.
var (
	colorPrimary = lipgloss.AdaptiveColor{Light: "25", Dark: "39"}   // blue
	colorAccent  = lipgloss.AdaptiveColor{Light: "130", Dark: "214"} // orange
	colorMuted   = lipgloss.AdaptiveColor{Light: "243", Dark: "245"} // gray
	colorGood    = lipgloss.AdaptiveColor{Light: "28", Dark: "42"}   // green
	colorFail    = lipgloss.AdaptiveColor{Light: "124", Dark: "196"} // red
)

var (
	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	styleLabel = lipgloss.NewStyle().Foreground(colorMuted)
	styleValue = lipgloss.NewStyle()
	styleLevel = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleGood  = lipgloss.NewStyle().Foreground(colorGood)
	styleFail  = lipgloss.NewStyle().Foreground(colorFail)

	styleCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(0, 2)

	styleTableHeader = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	styleTableCell   = lipgloss.NewStyle().Padding(0, 1)
)
