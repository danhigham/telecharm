package telegram

import (
	"context"

	"github.com/danhigham/tg-tui/internal/domain"
)

// EventHandler receives events from the Telegram client.
type EventHandler interface {
	OnNewMessage(msg domain.Message)
	OnChatListUpdate(chats []domain.ChatInfo)
	OnMessageRead(chatID int64, maxID int)
	OnUserStatus(userID int64, online bool)
	OnUserTyping(chatID int64, userName string)
	OnUserTypingStop(chatID int64)
}

// Client is the interface for Telegram operations.
type Client interface {
	Run(ctx context.Context) error
	SendMessage(ctx context.Context, chatID int64, text string) error
	GetHistory(ctx context.Context, chatID int64, limit int) ([]domain.Message, error)
	GetDialogs(ctx context.Context) ([]domain.ChatInfo, error)
	MarkAsRead(ctx context.Context, chatID int64, maxID int) error
}
