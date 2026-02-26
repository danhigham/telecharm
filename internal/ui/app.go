package ui

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/danhigham/telecharm/internal/domain"
	"github.com/danhigham/telecharm/internal/state"
	"github.com/danhigham/telecharm/internal/telegram"
)

type focusTarget int

const (
	focusChatList focusTarget = iota
	focusMessages
	focusInput
)

const (
	defaultSplitPos = 36
	minSplitPos     = 20
	maxSplitFrac    = 0.5 // chat list never exceeds half the terminal
	splitStep       = 4
)

// inputRenderedHeight is the total height of the input box (4 inner + 2 border).
const inputRenderedHeight = 6

// Model is the root Bubble Tea model.
type Model struct {
	chatList    ChatListModel
	messageView MessageViewModel
	input       InputModel
	auth        AuthModel
	status      statusModel
	splash      SplashModel
	help        HelpModel

	store    *state.Store
	client   telegram.Client
	authFlow *telegram.TUIAuth

	focus           focusTarget
	splitPos        int // width of the chat list pane (resizable)
	chatListVisible bool
	width           int
	height          int
}

// NewModel creates the root model with all sub-components.
func NewModel(store *state.Store, client telegram.Client, authFlow *telegram.TUIAuth) Model {
	m := Model{
		chatList:    NewChatListModel(),
		messageView: NewMessageViewModel(),
		input:       NewInputModel(),
		auth:        NewAuthModel(),
		status:      newStatusModel(),
		splash:      NewSplashModel(),
		help:        NewHelpModel(),
		store:       store,
		client:      client,
		authFlow:    authFlow,
		focus:           focusChatList,
		splitPos:        defaultSplitPos,
		chatListVisible: true,
	}

	m.auth = m.auth.SetOnSubmit(func(stage domain.AuthState, value string) {
		switch stage {
		case domain.AuthStatePhone:
			authFlow.PhoneCh <- value
		case domain.AuthStateCode:
			authFlow.CodeCh <- value
		case domain.AuthState2FA:
			authFlow.PasswordCh <- value
		}
	})

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.input.Init(),
		tea.Tick(3*time.Second, func(time.Time) tea.Msg { return SplashDoneMsg{} }),
		tea.Tick(30*time.Second, func(time.Time) tea.Msg { return clockTickMsg{} }),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.distributeSize()
		return m, nil

	case StoreUpdatedMsg:
		m = m.refreshFromStore()
		// Auto-select the first chat if none is active yet.
		if m.store.GetActiveChat() == 0 {
			chats := m.store.GetChatList()
			if len(chats) > 0 {
				return m, func() tea.Msg {
					return ChatSelectedMsg{ChatID: chats[0].ID}
				}
			}
		}
		return m, nil

	case AuthRequestMsg:
		m.auth = m.auth.Show(msg.Stage)
		return m, nil

	case ChatSelectedMsg:
		m.store.SetActiveChat(msg.ChatID)
		chats := m.store.GetChatList()
		for _, c := range chats {
			if c.ID == msg.ChatID {
				m.status = m.status.SetChatTitle(c.Title)
				break
			}
		}
		msgs := m.store.GetMessages(msg.ChatID)
		if len(msgs) > 0 {
			m.messageView = m.messageView.SetMessages(msgs)
		}
		m.focus = focusInput
		m = m.updateFocus()
		if len(msgs) == 0 {
			chatID := msg.ChatID
			client := m.client
			cmds = append(cmds, func() tea.Msg {
				history, err := client.GetHistory(context.Background(), chatID, 50, 0)
				if err != nil {
					return SendErrorMsg{Err: err}
				}
				return HistoryLoadedMsg{ChatID: chatID, Messages: history}
			})
		}
		return m, tea.Batch(cmds...)

	case HistoryLoadedMsg:
		m.store.SetMessages(msg.ChatID, msg.Messages)
		if m.store.GetActiveChat() == msg.ChatID {
			m.messageView = m.messageView.SetMessages(msg.Messages)
		}
		return m, nil

	case LoadOlderHistoryMsg:
		if m.store.GetActiveChat() != msg.ChatID {
			return m, nil
		}
		oldestID := m.store.GetOldestMessageID(msg.ChatID)
		if oldestID == 0 {
			return m, nil
		}
		m.messageView = m.messageView.SetLoading(true)
		chatID := msg.ChatID
		client := m.client
		cmds = append(cmds, func() tea.Msg {
			history, err := client.GetHistory(context.Background(), chatID, 50, oldestID)
			if err != nil {
				return SendErrorMsg{Err: err}
			}
			return OlderHistoryLoadedMsg{ChatID: chatID, Messages: history}
		})
		return m, tea.Batch(cmds...)

	case OlderHistoryLoadedMsg:
		m.store.PrependMessages(msg.ChatID, msg.Messages)
		if m.store.GetActiveChat() == msg.ChatID {
			m.messageView = m.messageView.PrependMessages(msg.Messages)
		}
		return m, nil

	case sendMessageMsg:
		chatID := m.store.GetActiveChat()
		if chatID == 0 {
			return m, nil
		}
		client := m.client
		store := m.store
		text := msg.text
		cmds = append(cmds, func() tea.Msg {
			sentMsg, err := client.SendMessage(context.Background(), chatID, text)
			if err != nil {
				return SendErrorMsg{Err: err}
			}
			store.OnNewMessage(sentMsg)
			return nil
		})
		return m, tea.Batch(cmds...)

	case SplashDoneMsg:
		m.splash = m.splash.TimerDone()
		return m, nil

	case clockTickMsg:
		// Re-tick to keep the clock updated
		return m, tea.Tick(30*time.Second, func(time.Time) tea.Msg { return clockTickMsg{} })

	case StatusMsg:
		m.status.text = msg.Text
		m.status.connected = msg.Connected
		if msg.Connected {
			m.splash = m.splash.ConnReady()
			if name := m.client.GetSelfName(); name != "" {
				m.status = m.status.SetUserName(name)
			}
		}
		return m, nil

	case SendErrorMsg:
		m.status.text = fmt.Sprintf("Send error: %v", msg.Err)
		m.status.connected = false
		return m, nil

	case tea.KeyMsg:
		if m.splash.IsVisible() {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}

		if m.auth.IsVisible() {
			var cmd tea.Cmd
			m.auth, cmd = m.auth.Update(msg)
			return m, cmd
		}

		if m.help.IsVisible() {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "h", "f1", "esc", "q":
				m.help = m.help.Toggle()
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "f1":
			m.help = m.help.Toggle()
			return m, nil
		case "q":
			if m.focus != focusInput {
				return m, tea.Quit
			}
		case "h":
			if m.focus != focusInput {
				m.help = m.help.Toggle()
				return m, nil
			}
		case "ctrl+b":
			m.chatListVisible = !m.chatListVisible
			if !m.chatListVisible && m.focus == focusChatList {
				m.focus = focusMessages
				m = m.updateFocus()
			}
			m = m.distributeSize()
			return m, nil
		case "tab":
			if m.chatListVisible {
				m.focus = (m.focus + 1) % 3
			} else {
				// Toggle between messages and input only
				if m.focus == focusMessages {
					m.focus = focusInput
				} else {
					m.focus = focusMessages
				}
			}
			m = m.updateFocus()
			return m, nil
		case "shift+tab":
			if m.chatListVisible {
				m.focus = (m.focus + 2) % 3
			} else {
				if m.focus == focusMessages {
					m.focus = focusInput
				} else {
					m.focus = focusMessages
				}
			}
			m = m.updateFocus()
			return m, nil
		case "esc":
			if m.chatListVisible {
				m.focus = focusChatList
			} else {
				m.focus = focusMessages
			}
			m = m.updateFocus()
			return m, nil
		case "[":
			if m.focus != focusInput && m.chatListVisible {
				m.splitPos -= splitStep
				m = m.clampSplitPos()
				m = m.distributeSize()
				return m, nil
			}
		case "]":
			if m.focus != focusInput && m.chatListVisible {
				m.splitPos += splitStep
				m = m.clampSplitPos()
				m = m.distributeSize()
				return m, nil
			}
		}

		switch m.focus {
		case focusChatList:
			var cmd tea.Cmd
			m.chatList, cmd = m.chatList.Update(msg)
			cmds = append(cmds, cmd)
		case focusMessages:
			var cmd tea.Cmd
			m.messageView, cmd = m.messageView.Update(msg)
			cmds = append(cmds, cmd)
		case focusInput:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.PasteMsg:
		if m.focus == focusInput {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}

func (m Model) View() tea.View {
	v := tea.NewView("")
	v.AltScreen = true

	if m.width == 0 || m.height == 0 {
		return v
	}

	if m.auth.IsVisible() {
		v.SetContent(m.auth.View())
		return v
	}

	// Status bar across the top
	statusBar := m.status.View()

	// Right pane: messages + input stacked vertically
	messagesView := m.messageView.View()
	inputView := m.input.View()
	rightPane := lipgloss.JoinVertical(lipgloss.Left, messagesView, inputView)

	// Join panes horizontally, then stack status bar on top
	var panes string
	if m.chatListVisible {
		chatListView := m.chatList.View()
		panes = lipgloss.JoinHorizontal(lipgloss.Top, chatListView, rightPane)
	} else {
		panes = rightPane
	}
	full := lipgloss.JoinVertical(lipgloss.Left, statusBar, panes)

	// Clamp to terminal dimensions
	mainContent := lipgloss.NewStyle().
		MaxWidth(m.width).
		MaxHeight(m.height).
		Render(full)

	if m.splash.IsVisible() {
		x, y := m.splash.BoxOffset()
		bg := lipgloss.NewLayer(mainContent)
		fg := lipgloss.NewLayer(m.splash.View()).X(x).Y(y).Z(1)
		comp := lipgloss.NewCompositor(bg, fg)
		v.SetContent(comp.Render())
	} else if m.help.IsVisible() {
		x, y := m.help.BoxOffset()
		bg := lipgloss.NewLayer(mainContent)
		fg := lipgloss.NewLayer(m.help.View()).X(x).Y(y).Z(1)
		comp := lipgloss.NewCompositor(bg, fg)
		v.SetContent(comp.Render())
	} else {
		v.SetContent(mainContent)
	}
	return v
}

func (m Model) clampSplitPos() Model {
	maxPos := int(float64(m.width) * maxSplitFrac)
	if m.splitPos < minSplitPos {
		m.splitPos = minSplitPos
	}
	if m.splitPos > maxPos {
		m.splitPos = maxPos
	}
	return m
}

func (m Model) distributeSize() Model {
	// Status bar takes 1 row at the top
	m.status = m.status.SetWidth(m.width)
	contentHeight := m.height - 1
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Chat list: resizable width, full content height
	var clWidth int
	if m.chatListVisible {
		m = m.clampSplitPos()
		clWidth = m.splitPos
		if clWidth > m.width {
			clWidth = m.width
		}
		m.chatList = m.chatList.SetSize(clWidth, contentHeight)
	}

	// Right pane: remaining width
	rightWidth := m.width - clWidth
	if rightWidth < 1 {
		rightWidth = 1
	}

	// Input gets fixed height, messages get the rest
	messagesHeight := contentHeight - inputRenderedHeight
	if messagesHeight < 1 {
		messagesHeight = 1
	}

	m.messageView = m.messageView.SetSize(rightWidth, messagesHeight)
	m.input = m.input.SetSize(rightWidth, inputRenderedHeight)

	m.auth = m.auth.SetSize(m.width, m.height)
	m.splash = m.splash.SetSize(m.width, m.height)
	m.help = m.help.SetSize(m.width, m.height)

	return m
}

func (m Model) updateFocus() Model {
	m.chatList = m.chatList.SetFocused(m.focus == focusChatList)
	m.messageView = m.messageView.SetFocused(m.focus == focusMessages)
	m.input = m.input.SetFocused(m.focus == focusInput)
	return m
}

func (m Model) refreshFromStore() Model {
	chats := m.store.GetChatList()
	m.chatList = m.chatList.WithItems(chats)

	activeChat := m.store.GetActiveChat()
	if activeChat != 0 {
		m.messageView = m.messageView.SetTypingUser(m.store.GetTypingUser(activeChat))
		msgs := m.store.GetMessages(activeChat)
		m.messageView = m.messageView.SetMessages(msgs)
	}

	return m
}

// App wraps the Bubble Tea program for external use.
type App struct {
	program *tea.Program
}

// NewApp creates a new App ready to Run.
func NewApp(store *state.Store, client telegram.Client, authFlow *telegram.TUIAuth) *App {
	model := NewModel(store, client, authFlow)
	p := tea.NewProgram(model)
	return &App{program: p}
}

// Run starts the Bubble Tea event loop (blocks until quit).
func (a *App) Run() error {
	_, err := a.program.Run()
	return err
}

// Send sends a message into the Bubble Tea event loop from external goroutines.
func (a *App) Send(msg tea.Msg) {
	go a.program.Send(msg)
}

// DrawFunc returns a function suitable for state.Store that triggers a re-render.
func (a *App) DrawFunc() func() {
	return func() {
		a.Send(StoreUpdatedMsg{})
	}
}
