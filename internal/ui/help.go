package ui

import "charm.land/lipgloss/v2"

// HelpModel renders a centered help overlay listing keyboard shortcuts.
type HelpModel struct {
	visible       bool
	width, height int
}

// NewHelpModel creates a hidden help model.
func NewHelpModel() HelpModel {
	return HelpModel{}
}

// IsVisible reports whether the help overlay is showing.
func (h HelpModel) IsVisible() bool {
	return h.visible
}

// Toggle flips the help overlay visibility.
func (h HelpModel) Toggle() HelpModel {
	h.visible = !h.visible
	return h
}

// SetSize updates the terminal dimensions for centering.
func (h HelpModel) SetSize(w, ht int) HelpModel {
	h.width = w
	h.height = ht
	return h
}

const helpText = ` Keyboard Shortcuts

 General
   Ctrl+C        Quit
   h / F1        Toggle this help
   Tab           Next pane
   Shift+Tab     Previous pane
   Esc           Back to chat list
   Ctrl+B        Toggle sidebar
   [ / ]         Resize sidebar

 Chat List
   j/k / ↑/↓     Navigate chats
   Enter         Select chat
   /             Filter chats

 Messages
   j / k         Scroll down / up
   PgUp / PgDn   Page scroll

 Input
   Enter         Send message

 Press h, F1, or Esc to close`

// View renders the help box (without full-screen placement).
// Use BoxOffset to get the X/Y for centering via the Layer API.
func (h HelpModel) View() string {
	if !h.visible || h.width == 0 || h.height == 0 {
		return ""
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 3).
		BorderForegroundBlend(rainbowBlend...)

	return style.Render(helpText)
}

// BoxOffset returns the (x, y) needed to center the help box
// within the terminal dimensions.
func (h HelpModel) BoxOffset() (int, int) {
	box := h.View()
	bw := lipgloss.Width(box)
	bh := lipgloss.Height(box)
	x := (h.width - bw) / 2
	y := (h.height - bh) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}
