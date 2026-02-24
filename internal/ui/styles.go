package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	daySeparatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	timeStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	outNameStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	inNameStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
	typingStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)

	highlightColor = lipgloss.Color("6")  // cyan
	dimColor       = lipgloss.Color("240") // gray
)

func borderColor(focused bool) lipgloss.Color {
	if focused {
		return highlightColor
	}
	return dimColor
}

// truncateHeight limits s to at most maxLines lines.
func truncateHeight(s string, maxLines int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n")
}
