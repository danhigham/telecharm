package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/danhigham/tg-tui/internal/domain"
)

// chatItem implements list.Item for the chat list.
type chatItem struct {
	chatID      int64
	title       string
	unreadCount int
	lastMessage string
}

func (i chatItem) FilterValue() string { return i.title }

// chatItemDelegate renders a chatItem in the list.
type chatItemDelegate struct{}

func (d chatItemDelegate) Height() int                             { return 2 }
func (d chatItemDelegate) Spacing() int                            { return 1 }
func (d chatItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d chatItemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ci, ok := item.(chatItem)
	if !ok {
		return
	}

	title := ci.title
	if ci.unreadCount > 0 {
		title = fmt.Sprintf("%s (%d)", ci.title, ci.unreadCount)
	}

	desc := ci.lastMessage
	if len(desc) > 50 {
		desc = desc[:50] + "..."
	}

	isSelected := index == m.Index()
	rowWidth := m.Width()

	// Base styles — full row width so background fills entire line.
	// MaxHeight(1) prevents long text from wrapping to extra lines,
	// which would exceed the delegate's declared Height() of 2.
	titleStyle := lipgloss.NewStyle().Width(rowWidth).MaxHeight(1)
	descStyle := lipgloss.NewStyle().Width(rowWidth).MaxHeight(1).Foreground(lipgloss.Color("240"))

	if isSelected {
		titleStyle = titleStyle.Background(lipgloss.Color("237")).Foreground(lipgloss.Color("15")).Bold(true)
		descStyle = descStyle.Background(lipgloss.Color("237")).Foreground(lipgloss.Color("250"))
	}
	if ci.unreadCount > 0 {
		titleStyle = titleStyle.Bold(true)
	}

	fmt.Fprintf(w, "%s\n%s", titleStyle.Render(title), descStyle.Render(desc))
}

// ChatListModel wraps bubbles/list for the chat sidebar.
type ChatListModel struct {
	list    list.Model
	focused bool
	width   int
	height  int
}

func NewChatListModel() ChatListModel {
	delegate := chatItemDelegate{}
	l := list.New(nil, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()

	return ChatListModel{list: l}
}

func (m ChatListModel) Update(msg tea.Msg) (ChatListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			if item, ok := m.list.SelectedItem().(chatItem); ok {
				return m, func() tea.Msg {
					return ChatSelectedMsg{ChatID: item.chatID}
				}
			}
			return m, nil
		}
	}

	// Delegate all other keys (including j/k) to the list for native scrolling
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ChatListModel) View() string {
	innerW := m.width - 2
	innerH := m.height - 2
	if innerW < 0 {
		innerW = 0
	}
	if innerH < 0 {
		innerH = 0
	}

	// Truncate list output to exactly innerH lines to prevent overflow
	content := truncateHeight(m.list.View(), innerH)

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor(m.focused)).
		Width(innerW).
		Height(innerH)

	return style.Render(content)
}

func (m ChatListModel) WithItems(chats []domain.ChatInfo) ChatListModel {
	items := make([]list.Item, len(chats))
	for i, c := range chats {
		items[i] = chatItem{
			chatID:      c.ID,
			title:       c.Title,
			unreadCount: c.UnreadCount,
			lastMessage: c.LastMessage,
		}
	}
	m.list.SetItems(items)
	return m
}

func (m ChatListModel) SetSize(w, h int) ChatListModel {
	m.width = w
	m.height = h
	innerW := w - 2
	innerH := h - 2
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}
	m.list.SetSize(innerW, innerH)
	return m
}

func (m ChatListModel) SetFocused(f bool) ChatListModel {
	m.focused = f
	return m
}
