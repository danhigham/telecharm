package state_test

import (
	"testing"
	"time"

	"github.com/danhigham/tg-tui/internal/domain"
	"github.com/danhigham/tg-tui/internal/state"
)

func TestStore_OnNewMessage(t *testing.T) {
	s := state.New(nil) // nil drawFunc for testing

	msg := domain.Message{
		ID:         1,
		ChatID:     100,
		SenderName: "Alice",
		Text:       "Hello",
		Timestamp:  time.Now(),
	}

	s.OnNewMessage(msg)

	msgs := s.GetMessages(100)
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}
	if msgs[0].Text != "Hello" {
		t.Errorf("Text = %q, want %q", msgs[0].Text, "Hello")
	}
}

func TestStore_OnChatListUpdate(t *testing.T) {
	s := state.New(nil)

	chats := []domain.ChatInfo{
		{ID: 1, Title: "Alice", LastTime: time.Now()},
		{ID: 2, Title: "Bob", LastTime: time.Now().Add(-time.Hour)},
	}

	s.OnChatListUpdate(chats)

	got := s.GetChatList()
	if len(got) != 2 {
		t.Fatalf("got %d chats, want 2", len(got))
	}
	if got[0].Title != "Alice" {
		t.Errorf("first chat = %q, want Alice", got[0].Title)
	}
}

func TestStore_ActiveChat(t *testing.T) {
	s := state.New(nil)

	s.SetActiveChat(42)
	if s.GetActiveChat() != 42 {
		t.Errorf("ActiveChat = %d, want 42", s.GetActiveChat())
	}
}

func TestStore_OnNewMessage_UpdatesChatList(t *testing.T) {
	s := state.New(nil)

	chats := []domain.ChatInfo{
		{ID: 1, Title: "Alice", UnreadCount: 0},
		{ID: 2, Title: "Bob", UnreadCount: 0},
	}
	s.OnChatListUpdate(chats)

	msg := domain.Message{
		ID:         1,
		ChatID:     2,
		SenderName: "Bob",
		Text:       "Hey",
		Timestamp:  time.Now(),
	}
	s.OnNewMessage(msg)

	updated := s.GetChatList()
	// Bob's chat should now be first (most recent) and have unread=1
	if updated[0].ID != 2 {
		t.Errorf("first chat ID = %d, want 2 (Bob)", updated[0].ID)
	}
	if updated[0].UnreadCount != 1 {
		t.Errorf("UnreadCount = %d, want 1", updated[0].UnreadCount)
	}
}

func TestStore_MessageLimit(t *testing.T) {
	s := state.New(nil)

	for i := 0; i < 600; i++ {
		s.OnNewMessage(domain.Message{
			ID:     i,
			ChatID: 1,
			Text:   "msg",
		})
	}

	msgs := s.GetMessages(1)
	if len(msgs) > 500 {
		t.Errorf("messages = %d, want <= 500", len(msgs))
	}
}
