package ui

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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

	isSelected := index == m.Index()
	// Account for the cursor prefix ("  " or "> ") in available width.
	contentWidth := m.Width() - 2
	if contentWidth < 1 {
		contentWidth = 1
	}

	titleStyle := lipgloss.NewStyle().MaxWidth(contentWidth).MaxHeight(1)
	descStyle := lipgloss.NewStyle().MaxWidth(contentWidth).MaxHeight(1).Foreground(lipgloss.Color("240"))

	cursor := "  "
	if isSelected {
		cursor = "> "
		titleStyle = titleStyle.Foreground(lipgloss.Color("170")).Bold(true)
		descStyle = descStyle.Foreground(lipgloss.Color("250"))
	}
	if ci.unreadCount > 0 {
		titleStyle = titleStyle.Bold(true)
	}

	fmt.Fprintf(w, "%s%s\n%s%s", cursor, titleStyle.Render(title), "  ", descStyle.Render(desc))
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
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	return ChatListModel{list: l}
}

func (m ChatListModel) Update(msg tea.Msg) (ChatListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Only handle enter for chat selection when not filtering.
		if msg.String() == "enter" && m.list.FilterState() != list.Filtering {
			if item, ok := m.list.SelectedItem().(chatItem); ok {
				return m, func() tea.Msg {
					return ChatSelectedMsg{ChatID: item.chatID}
				}
			}
			return m, nil
		}
	}

	// Delegate all other keys (including j/k and filter '/') to the list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ChatListModel) View() string {
	contentH := m.height - 2
	if contentH < 0 {
		contentH = 0
	}

	// Truncate list output to content area inside border
	content := truncateHeight(m.list.View(), contentH)

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(m.width).
		Height(m.height)
	style = applyBorderColor(style, m.focused)

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
