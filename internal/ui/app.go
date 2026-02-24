package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/danhigham/tg-tui/internal/domain"
	"github.com/danhigham/tg-tui/internal/state"
	"github.com/danhigham/tg-tui/internal/telegram"
)

type focusTarget int

const (
	focusChatList focusTarget = iota
	focusMessages
	focusInput
)

const chatListWidth = 36

// inputRenderedHeight is the total height of the input box (4 inner + 2 border).
const inputRenderedHeight = 6

// Model is the root Bubble Tea model.
type Model struct {
	chatList    ChatListModel
	messageView MessageViewModel
	input       InputModel
	auth        AuthModel
	status      statusModel

	store    *state.Store
	client   telegram.Client
	authFlow *telegram.TUIAuth

	focus  focusTarget
	width  int
	height int
}

// NewModel creates the root model with all sub-components.
func NewModel(store *state.Store, client telegram.Client, authFlow *telegram.TUIAuth) Model {
	m := Model{
		chatList:    NewChatListModel(),
		messageView: NewMessageViewModel(),
		input:       NewInputModel(),
		auth:        NewAuthModel(),
		status:      newStatusModel(),
		store:       store,
		client:      client,
		authFlow:    authFlow,
		focus:       focusChatList,
	}

	m.messageView = m.messageView.SetStatusPill(m.status.View())

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
	return tea.Batch(m.input.Init(), tea.WindowSize())
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
		return m, nil

	case AuthRequestMsg:
		m.auth = m.auth.Show(msg.Stage)
		return m, nil

	case ChatSelectedMsg:
		m.store.SetActiveChat(msg.ChatID)
		chats := m.store.GetChatList()
		for _, c := range chats {
			if c.ID == msg.ChatID {
				m.messageView = m.messageView.SetChatTitle(c.Title)
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
				history, err := client.GetHistory(context.Background(), chatID, 50)
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

	case sendMessageMsg:
		chatID := m.store.GetActiveChat()
		if chatID == 0 {
			return m, nil
		}
		client := m.client
		text := msg.text
		cmds = append(cmds, func() tea.Msg {
			err := client.SendMessage(context.Background(), chatID, text)
			if err != nil {
				return SendErrorMsg{Err: err}
			}
			return nil
		})
		return m, tea.Batch(cmds...)

	case StatusMsg:
		m.status.text = msg.Text
		m.status.connected = msg.Connected
		m.messageView = m.messageView.SetStatusPill(m.status.View())
		return m, nil

	case SendErrorMsg:
		m.status.text = fmt.Sprintf("Send error: %v", msg.Err)
		m.status.connected = false
		return m, nil

	case tea.KeyMsg:
		if m.auth.IsVisible() {
			var cmd tea.Cmd
			m.auth, cmd = m.auth.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.focus != focusInput {
				return m, tea.Quit
			}
		case "tab":
			m.focus = (m.focus + 1) % 3
			m = m.updateFocus()
			return m, nil
		case "shift+tab":
			m.focus = (m.focus + 2) % 3
			m = m.updateFocus()
			return m, nil
		case "esc":
			m.focus = focusChatList
			m = m.updateFocus()
			return m, nil
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
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	if m.auth.IsVisible() {
		return m.auth.View()
	}

	// Chat list on the left
	chatListView := m.chatList.View()

	// Right pane: messages + input stacked vertically
	messagesView := m.messageView.View()
	inputView := m.input.View()
	rightPane := lipgloss.JoinVertical(lipgloss.Left, messagesView, inputView)

	// Join horizontally
	full := lipgloss.JoinHorizontal(lipgloss.Top, chatListView, rightPane)

	// Clamp to terminal dimensions
	return lipgloss.NewStyle().
		MaxWidth(m.width).
		MaxHeight(m.height).
		Render(full)
}

func (m Model) distributeSize() Model {
	// Full height for content — no status bar row (it floats)
	contentHeight := m.height

	// Chat list: fixed width, full height
	clWidth := chatListWidth
	if clWidth > m.width {
		clWidth = m.width
	}
	m.chatList = m.chatList.SetSize(clWidth, contentHeight)

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
	p := tea.NewProgram(model, tea.WithAltScreen())
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
