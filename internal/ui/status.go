package ui

import "github.com/charmbracelet/lipgloss"

type statusModel struct {
	text      string
	connected bool
}

func newStatusModel() statusModel {
	return statusModel{
		text:      "Connecting...",
		connected: false,
	}
}

// View renders a compact status pill (not full-width).
func (m statusModel) View() string {
	bg := lipgloss.Color("58") // olive
	if m.connected {
		bg = lipgloss.Color("22") // dark green
	}

	style := lipgloss.NewStyle().
		Background(bg).
		Foreground(lipgloss.Color("15")).
		Padding(0, 1)

	return style.Render(m.text)
}
