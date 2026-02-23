package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/danhigham/tg-tui/internal/domain"
)

type ChatList struct {
	*tview.List
	onSelect func(chatID int64)
}

func NewChatList() *ChatList {
	cl := &ChatList{
		List: tview.NewList(),
	}
	cl.ShowSecondaryText(true)
	cl.SetBorder(true).SetTitle(" Chats ")
	cl.SetHighlightFullLine(true)
	cl.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDarkCyan).
		Foreground(tcell.ColorWhite))

	// vim-style navigation
	cl.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'j':
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		case 'k':
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		}
		return event
	})

	return cl
}

func (cl *ChatList) SetOnSelect(fn func(chatID int64)) {
	cl.onSelect = fn
}

func (cl *ChatList) Update(chats []domain.ChatInfo) {
	currentIdx := cl.GetCurrentItem()
	cl.Clear()

	for _, chat := range chats {
		title := chat.Title
		if chat.UnreadCount > 0 {
			title = fmt.Sprintf("[::b]%s (%d)[::-]", chat.Title, chat.UnreadCount)
		}

		secondary := chat.LastMessage
		if len(secondary) > 40 {
			secondary = secondary[:40] + "..."
		}

		chatID := chat.ID // capture for closure
		cl.AddItem(title, secondary, 0, func() {
			if cl.onSelect != nil {
				cl.onSelect(chatID)
			}
		})
	}

	if currentIdx < cl.GetItemCount() {
		cl.SetCurrentItem(currentIdx)
	}
}
