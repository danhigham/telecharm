package ui

import (
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// InputModel wraps a bubbles textarea for message composition.
type InputModel struct {
	textarea textarea.Model
	focused  bool
	width    int
	height   int
}

func NewInputModel() InputModel {
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.Prompt = ""
	ta.CharLimit = 4096
	ta.ShowLineNumbers = false

	// Remove cursor-line background highlight.
	s := ta.Styles()
	s.Focused.CursorLine = lipgloss.NewStyle()
	ta.SetStyles(s)

	return InputModel{textarea: ta}
}

func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			text := m.textarea.Value()
			if text != "" {
				m.textarea.Reset()
				return m, func() tea.Msg {
					return sendMessageMsg{text: text}
				}
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m InputModel) View() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(m.width).
		Height(m.height)
	style = applyBorderColor(style, m.focused)

	return style.Render(m.textarea.View())
}

func (m InputModel) Focus() InputModel {
	m.focused = true
	_ = m.textarea.Focus()
	return m
}

func (m InputModel) Blur() InputModel {
	m.focused = false
	m.textarea.Blur()
	return m
}

func (m InputModel) SetSize(w, h int) InputModel {
	m.width = w
	m.height = h
	// Inner dimensions: subtract the outer border (1 each side).
	taWidth := w - 2
	if taWidth < 1 {
		taWidth = 1
	}
	taHeight := h - 2
	if taHeight < 1 {
		taHeight = 1
	}
	m.textarea.SetWidth(taWidth)
	m.textarea.SetHeight(taHeight)
	return m
}

func (m InputModel) SetFocused(f bool) InputModel {
	if f {
		return m.Focus()
	}
	return m.Blur()
}

func (m InputModel) Init() tea.Cmd {
	return textarea.Blink
}
