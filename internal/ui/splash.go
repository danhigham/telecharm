package ui

import "charm.land/lipgloss/v2"

const splashArt = `
 _                  _         _
| |_ __ _       __ | |_ _   _(_)
| __/ _` + "`" + ` |____ / _` + "`" + ` | __| | | | |
| || (_| |____| (_| | |_| |_| | |
 \__\__, |     \__,_|\__|\__,_|_|
    |___/
`

// SplashModel renders a centered splash overlay on startup.
// It stays visible for at least the minimum duration even if
// the connection becomes ready sooner.
type SplashModel struct {
	visible       bool
	timerDone     bool
	connReady     bool
	width, height int
}

// NewSplashModel creates a visible splash.
func NewSplashModel() SplashModel {
	return SplashModel{visible: true}
}

// SetSize updates the terminal dimensions for centering.
func (s SplashModel) SetSize(w, h int) SplashModel {
	s.width = w
	s.height = h
	return s
}

// IsVisible reports whether the splash is still showing.
func (s SplashModel) IsVisible() bool {
	return s.visible
}

// TimerDone marks the minimum display duration as elapsed.
// The splash dismisses only once both the timer is done and
// the connection is ready (or the timer alone suffices).
func (s SplashModel) TimerDone() SplashModel {
	s.timerDone = true
	if s.connReady {
		s.visible = false
	}
	return s
}

// ConnReady marks the connection as established.
// The splash dismisses only if the minimum timer has also elapsed.
func (s SplashModel) ConnReady() SplashModel {
	s.connReady = true
	if s.timerDone {
		s.visible = false
	}
	return s
}

// View renders the splash box centered to the full terminal.
func (s SplashModel) View() string {
	if !s.visible || s.width == 0 || s.height == 0 {
		return ""
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(highlightColor).
		Padding(1, 3).
		Render(splashArt)

	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, box)
}
