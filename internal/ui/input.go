package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InputModel wraps a bubbles textinput for message composition.
type InputModel struct {
	textinput textinput.Model
	focused   bool
	width     int
	height    int
}

func NewInputModel() InputModel {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Prompt = "> "
	ti.CharLimit = 4096

	return InputModel{textinput: ti}
}

func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			text := m.textinput.Value()
			if text != "" {
				m.textinput.SetValue("")
				return m, func() tea.Msg {
					return sendMessageMsg{text: text}
				}
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textinput, cmd = m.textinput.Update(msg)
	return m, cmd
}

func (m InputModel) View() string {
	innerW := m.width - 2
	innerH := m.height - 2
	if innerW < 0 {
		innerW = 0
	}
	if innerH < 0 {
		innerH = 0
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor(m.focused)).
		Width(innerW).
		Height(innerH)

	return style.Render(m.textinput.View())
}

func (m InputModel) Focus() InputModel {
	m.focused = true
	m.textinput.Focus()
	return m
}

func (m InputModel) Blur() InputModel {
	m.focused = false
	m.textinput.Blur()
	return m
}

func (m InputModel) SetSize(w, h int) InputModel {
	m.width = w
	m.height = h
	// Inner width: subtract border (2) and prompt "> " (2)
	m.textinput.Width = w - 4
	if m.textinput.Width < 1 {
		m.textinput.Width = 1
	}
	return m
}

func (m InputModel) SetFocused(f bool) InputModel {
	if f {
		return m.Focus()
	}
	return m.Blur()
}

func (m InputModel) Init() tea.Cmd {
	return textinput.Blink
}
