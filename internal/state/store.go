package state

import (
	"sort"
	"sync"
	"time"

	"github.com/danhigham/tg-tui/internal/domain"
)

const maxMessages = 500
const typingTimeout = 6 * time.Second

type typingInfo struct {
	userName string
	timer    *time.Timer
}

type Store struct {
	mu         sync.RWMutex
	chatList   []domain.ChatInfo
	messages   map[int64][]domain.Message
	typing     map[int64]*typingInfo
	activeChat int64
	authState  domain.AuthState
	drawFunc   func()
}

func New(drawFunc func()) *Store {
	return &Store{
		messages: make(map[int64][]domain.Message),
		typing:   make(map[int64]*typingInfo),
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
	s.mu.Unlock()
	s.draw()
}

func (s *Store) OnChatListUpdate(chats []domain.ChatInfo) {
	s.mu.Lock()
	s.chatList = chats
	s.sortChatList()
	s.mu.Unlock()
	s.draw()
}

func (s *Store) OnMessageRead(chatID int64, maxID int) {
	s.mu.Lock()
	for i, c := range s.chatList {
		if c.ID == chatID {
			s.chatList[i].UnreadCount = 0
			break
		}
	}
	s.mu.Unlock()
	s.draw()
}

func (s *Store) OnUserStatus(userID int64, online bool) {
	// Future: update online indicators
}

func (s *Store) OnUserTyping(chatID int64, userName string) {
	s.mu.Lock()
	if existing, ok := s.typing[chatID]; ok {
		existing.timer.Stop()
		existing.userName = userName
		existing.timer = time.AfterFunc(typingTimeout, func() {
			s.clearTyping(chatID)
		})
	} else {
		s.typing[chatID] = &typingInfo{
			userName: userName,
			timer: time.AfterFunc(typingTimeout, func() {
				s.clearTyping(chatID)
			}),
		}
	}
	s.mu.Unlock()
	s.draw()
}

func (s *Store) OnUserTypingStop(chatID int64) {
	s.clearTyping(chatID)
}

func (s *Store) clearTyping(chatID int64) {
	s.mu.Lock()
	if info, ok := s.typing[chatID]; ok {
		info.timer.Stop()
		delete(s.typing, chatID)
	}
	s.mu.Unlock()
	s.draw()
}

func (s *Store) GetTypingUser(chatID int64) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if info, ok := s.typing[chatID]; ok {
		return info.userName
	}
	return ""
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
	s.messages[chatID] = msgs
	s.mu.Unlock()
	s.draw()
}

// PrependMessages adds older messages to the front of the existing slice,
// deduplicating by message ID and respecting maxMessages.
func (s *Store) PrependMessages(chatID int64, msgs []domain.Message) {
	s.mu.Lock()
	existing := s.messages[chatID]

	// Build a set of existing message IDs for deduplication.
	idSet := make(map[int]struct{}, len(existing))
	for _, m := range existing {
		idSet[m.ID] = struct{}{}
	}

	// Filter out duplicates from the new messages.
	var unique []domain.Message
	for _, m := range msgs {
		if _, ok := idSet[m.ID]; !ok {
			unique = append(unique, m)
		}
	}

	combined := append(unique, existing...)
	if len(combined) > maxMessages {
		combined = combined[len(combined)-maxMessages:]
	}
	s.messages[chatID] = combined
	s.mu.Unlock()
}

// GetOldestMessageID returns the ID of the oldest cached message for a chat,
// or 0 if no messages are cached.
func (s *Store) GetOldestMessageID(chatID int64) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := s.messages[chatID]
	if len(msgs) == 0 {
		return 0
	}
	return msgs[0].ID
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
	s.authState = as
	s.mu.Unlock()
	s.draw()
}

func (s *Store) GetAuthState() domain.AuthState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.authState
}
