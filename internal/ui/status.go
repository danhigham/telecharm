package ui

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

var (
	// Dark gray background matching the lipgloss example
	statusBarBg = lipgloss.Color("#353533")
	// Bright magenta for the status pill and time highlight
	statusPillBg    = lipgloss.Color("#FF5FAF")
	statusPillBgOff = lipgloss.Color("#6C5098")
	// Teal/cyan for the time pill
	statusTimeBg = lipgloss.Color("#6124DF")
)

type statusModel struct {
	text      string
	connected bool
	chatTitle string
	userName  string
	width     int
}

func newStatusModel() statusModel {
	return statusModel{
		text:      "Connecting...",
		connected: false,
	}
}

// SetWidth sets the full terminal width for the status bar.
func (m statusModel) SetWidth(w int) statusModel {
	m.width = w
	return m
}

// SetChatTitle updates the active chat name shown on the left.
func (m statusModel) SetChatTitle(title string) statusModel {
	m.chatTitle = title
	return m
}

// SetUserName updates the logged-in user name shown on the right.
func (m statusModel) SetUserName(name string) statusModel {
	m.userName = name
	return m
}

// View renders a full-width status bar:
// [STATUS pill] [chat title] ... [time pill] [user name]
func (m statusModel) View() string {
	// Connection status pill
	pillBg := statusPillBgOff
	if m.connected {
		pillBg = statusPillBg
	}
	pillStyle := lipgloss.NewStyle().
		Background(pillBg).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Padding(0, 1)
	pill := pillStyle.Render(strings.ToUpper(m.text))

	// Chat title
	titleStyle := lipgloss.NewStyle().
		Background(statusBarBg).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Padding(0, 1)
	title := titleStyle.Render(m.chatTitle)

	// Current time pill
	timeStyle := lipgloss.NewStyle().
		Background(statusTimeBg).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Padding(0, 1)
	timePill := timeStyle.Render(time.Now().Format("15:04"))

	// User name — medium purple highlight, distinct from bar background
	userStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#7B5EA7")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Padding(0, 1)
	userPill := userStyle.Render(m.userName)

	// Left side: status + title
	left := pill + title

	// Right side: user + time
	right := userPill + timePill

	// Fill gap between left and right
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	filler := lipgloss.NewStyle().
		Background(statusBarBg).
		Render(strings.Repeat(" ", gap))

	barStyle := lipgloss.NewStyle().
		Background(statusBarBg).
		Width(m.width)

	return barStyle.Render(left + filler + right)
}
