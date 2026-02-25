package ui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	daySeparatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	timeStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	outNameStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	inNameStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
	typingStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)

	dimColor = lipgloss.Color("240") // gray

	// Rainbow gradient colors for focused borders (wraps back to start).
	rainbowBlend = []color.Color{
		lipgloss.Color("#FF6B9D"), // pink
		lipgloss.Color("#9B59B6"), // purple
		lipgloss.Color("#3498DB"), // blue
		lipgloss.Color("#2ECC71"), // green
		lipgloss.Color("#FF6B9D"), // pink (wrap)
	}
)

// applyBorderColor applies either the rainbow blend (focused) or dim border color.
func applyBorderColor(s lipgloss.Style, focused bool) lipgloss.Style {
	if focused {
		return s.BorderForegroundBlend(rainbowBlend...)
	}
	return s.BorderForeground(dimColor)
}

// truncateHeight limits s to at most maxLines lines.
func truncateHeight(s string, maxLines int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n")
}
