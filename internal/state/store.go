package state

import (
	"sort"
	"sync"

	"github.com/danhigham/tg-tui/internal/domain"
)

const maxMessages = 500

type Store struct {
	mu         sync.RWMutex
	chatList   []domain.ChatInfo
	messages   map[int64][]domain.Message
	activeChat int64
	authState  domain.AuthState
	drawFunc   func()
}

func New(drawFunc func()) *Store {
	return &Store{
		messages: make(map[int64][]domain.Message),
		drawFunc: drawFunc,
	}
}

func (s *Store) SetDrawFunc(f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.drawFunc = f
}

func (s *Store) draw() {
	if s.drawFunc != nil {
		s.drawFunc()
	}
}

func (s *Store) OnNewMessage(msg domain.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	msgs := s.messages[msg.ChatID]
	msgs = append(msgs, msg)
	if len(msgs) > maxMessages {
		msgs = msgs[len(msgs)-maxMessages:]
	}
	s.messages[msg.ChatID] = msgs

	// Update chat list: bump unread count and move to top
	for i, c := range s.chatList {
		if c.ID == msg.ChatID {
			if msg.ChatID != s.activeChat || msg.Out {
				s.chatList[i].UnreadCount++
			}
			s.chatList[i].LastMessage = msg.Text
			s.chatList[i].LastTime = msg.Timestamp
			break
		}
	}
	s.sortChatList()
	s.draw()
}

func (s *Store) OnChatListUpdate(chats []domain.ChatInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.chatList = chats
	s.sortChatList()
	s.draw()
}

func (s *Store) OnMessageRead(chatID int64, maxID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, c := range s.chatList {
		if c.ID == chatID {
			s.chatList[i].UnreadCount = 0
			break
		}
	}
	s.draw()
}

func (s *Store) OnUserStatus(userID int64, online bool) {
	// Future: update online indicators
}

func (s *Store) sortChatList() {
	sort.Slice(s.chatList, func(i, j int) bool {
		return s.chatList[i].LastTime.After(s.chatList[j].LastTime)
	})
}

func (s *Store) GetChatList() []domain.ChatInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.ChatInfo, len(s.chatList))
	copy(out, s.chatList)
	return out
}

func (s *Store) GetMessages(chatID int64) []domain.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := s.messages[chatID]
	out := make([]domain.Message, len(msgs))
	copy(out, msgs)
	return out
}

func (s *Store) SetMessages(chatID int64, msgs []domain.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[chatID] = msgs
	s.draw()
}

func (s *Store) SetActiveChat(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeChat = chatID
}

func (s *Store) GetActiveChat() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeChat
}

func (s *Store) SetAuthState(as domain.AuthState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authState = as
	s.draw()
}

func (s *Store) GetAuthState() domain.AuthState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.authState
}
