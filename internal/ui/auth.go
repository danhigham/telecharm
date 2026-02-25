package ui

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/danhigham/tg-tui/internal/domain"
)

// AuthModel handles authentication input (phone, code, 2FA password).
type AuthModel struct {
	textinput  textinput.Model
	visible    bool
	stage      domain.AuthState
	onSubmit   func(stage domain.AuthState, value string)
	width      int
	height     int
}

func NewAuthModel() AuthModel {
	ti := textinput.New()
	ti.CharLimit = 64

	return AuthModel{
		textinput: ti,
	}
}

func (m AuthModel) Update(msg tea.Msg) (AuthModel, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			value := m.textinput.Value()
			if value != "" && m.onSubmit != nil {
				m.onSubmit(m.stage, value)
				m.visible = false
				m.textinput.SetValue("")
				m.textinput.Blur()
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.textinput, cmd = m.textinput.Update(msg)
	return m, cmd
}

func (m AuthModel) View() string {
	if !m.visible {
		return ""
	}

	var title string
	switch m.stage {
	case domain.AuthStatePhone:
		title = "Enter Phone Number"
	case domain.AuthStateCode:
		title = "Enter Verification Code"
	case domain.AuthState2FA:
		title = "Enter 2FA Password"
	default:
		title = "Authentication"
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForegroundBlend(rainbowBlend...).
		Padding(1, 2).
		Width(50).
		Render(title + "\n\n" + m.textinput.View() + "\n\nPress Enter to submit")

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		box)
}

func (m AuthModel) Show(stage domain.AuthState) AuthModel {
	m.visible = true
	m.stage = stage
	m.textinput.SetValue("")

	switch stage {
	case domain.AuthStatePhone:
		m.textinput.Placeholder = "+1234567890"
		m.textinput.EchoMode = textinput.EchoNormal
	case domain.AuthStateCode:
		m.textinput.Placeholder = "12345"
		m.textinput.EchoMode = textinput.EchoNormal
	case domain.AuthState2FA:
		m.textinput.Placeholder = "password"
		m.textinput.EchoMode = textinput.EchoPassword
	}

	_ = m.textinput.Focus()
	return m
}

func (m AuthModel) SetOnSubmit(fn func(stage domain.AuthState, value string)) AuthModel {
	m.onSubmit = fn
	return m
}

func (m AuthModel) SetSize(w, h int) AuthModel {
	m.width = w
	m.height = h
	return m
}

func (m AuthModel) IsVisible() bool {
	return m.visible
}
