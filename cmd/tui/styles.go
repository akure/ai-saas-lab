package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Color palette (Dracula / Cyberpunk Dark Terminal Aesthetic)
	ColorCyan    = lipgloss.Color("#00F5FF")
	ColorViolet  = lipgloss.Color("#BD93F9")
	ColorPink    = lipgloss.Color("#FF79C6")
	ColorGreen   = lipgloss.Color("#50FA7B")
	ColorAmber   = lipgloss.Color("#FFB86C")
	ColorRed     = lipgloss.Color("#FF5555")
	ColorSlate   = lipgloss.Color("#6272A4")
	ColorBgDark  = lipgloss.Color("#191A21")
	ColorBgPanel = lipgloss.Color("#282A36")

	// UI Layout & Borders
	BannerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorCyan).
			Background(ColorBgPanel).
			Padding(0, 1).
			MarginBottom(1)

	ActiveTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBgDark).
			Background(ColorCyan).
			Padding(0, 2)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(ColorSlate).
				Background(ColorBgPanel).
				Padding(0, 2)

	TabGapStyle = lipgloss.NewStyle().
			Foreground(ColorSlate).
			Background(ColorBgPanel)

	ContainerBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorViolet).
			Padding(1, 2)

	StatusBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBgDark).
			Background(ColorGreen).
			Padding(0, 1)

	PlanBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBgDark).
			Background(ColorPink).
			Padding(0, 1)

	PersonaBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBgDark).
			Background(ColorAmber).
			Padding(0, 1)

	UserMsgStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorCyan)

	AssistantMsgStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorGreen)

	SystemMsgStyle = lipgloss.NewStyle().
			Italic(true).
			Foreground(ColorSlate)

	HelpFooterStyle = lipgloss.NewStyle().
			Foreground(ColorSlate).
			MarginTop(1)
)

func renderProgressBar(used, total int, width int) string {
	if total <= 0 {
		total = 100
	}
	pct := float64(used) / float64(total)
	if pct > 1.0 {
		pct = 1.0
	}

	fillWidth := int(pct * float64(width))
	emptyWidth := width - fillWidth

	filled := strings.Repeat("█", fillWidth)
	empty := strings.Repeat("░", emptyWidth)

	color := ColorGreen
	if pct > 0.8 {
		color = ColorRed
	} else if pct > 0.5 {
		color = ColorAmber
	}

	barStyle := lipgloss.NewStyle().Foreground(color)
	emptyStyle := lipgloss.NewStyle().Foreground(ColorSlate)

	return fmt.Sprintf("[%s%s] %d/%d (%d%%)", barStyle.Render(filled), emptyStyle.Render(empty), used, total, int(pct*100))
}
