# tg-tui Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a curses-based Telegram client in Go using tview/tcell and gotd/td.

**Architecture:** Three-layer event-driven: Telegram wrapper (gotd/td) emits domain events to a mutex-protected state store, which triggers tview redraws. TUI components read state to render.

**Tech Stack:** Go, rivo/tview, gdamore/tcell/v2, gotd/td, go.uber.org/zap, gopkg.in/yaml.v3

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `cmd/tg-tui/main.go`

**Step 1: Initialize Go module and install dependencies**

```bash
cd /home/danhigham/workspace/tg-tui
go mod init github.com/danhigham/tg-tui
go get github.com/rivo/tview@latest
go get github.com/gdamore/tcell/v2@latest
go get github.com/gotd/td@latest
go get github.com/gotd/contrib@latest
go get go.uber.org/zap@latest
go get gopkg.in/yaml.v3@latest
```

**Step 2: Create minimal main.go that starts tview**

Create `cmd/tg-tui/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()

	textView := tview.NewTextView().
		SetText("tg-tui - Telegram Client\n\nPress Ctrl+C to quit.")
	textView.SetBorder(true).SetTitle(" tg-tui ")

	if err := app.SetRoot(textView, true).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 3: Verify it compiles and runs**

```bash
cd /home/danhigham/workspace/tg-tui
go build ./cmd/tg-tui/
```

Expected: builds without errors, produces `tg-tui` binary.

**Step 4: Commit**

```bash
git add go.mod go.sum cmd/
git commit -m "feat: project scaffolding with tview hello world"
```

---

### Task 2: Configuration Loading

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write the failing test**

Create `internal/config/config_test.go`:

```go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/danhigham/tg-tui/internal/config"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := []byte(`telegram:
  api_id: 12345
  api_hash: "abcdef0123456789"
log_level: debug
`)
	if err := os.WriteFile(cfgPath, content, 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Telegram.APIID != 12345 {
		t.Errorf("APIID = %d, want 12345", cfg.Telegram.APIID)
	}
	if cfg.Telegram.APIHash != "abcdef0123456789" {
		t.Errorf("APIHash = %q, want %q", cfg.Telegram.APIHash, "abcdef0123456789")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := config.Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestConfigDir(t *testing.T) {
	dir := config.Dir()
	if dir == "" {
		t.Error("Dir() returned empty string")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd /home/danhigham/workspace/tg-tui
go test ./internal/config/
```

Expected: compilation failure — package doesn't exist yet.

**Step 3: Write implementation**

Create `internal/config/config.go`:

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Telegram TelegramConfig `yaml:"telegram"`
	LogLevel string         `yaml:"log_level"`
}

type TelegramConfig struct {
	APIID   int    `yaml:"api_id"`
	APIHash string `yaml:"api_hash"`
}

func Dir() string {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		cfgDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(cfgDir, "tg-tui")
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	return &cfg, nil
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/config/ -v
```

Expected: all 3 tests PASS.

**Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add configuration loading from YAML"
```

---

### Task 3: Domain Types

**Files:**
- Create: `internal/domain/types.go`

**Step 1: Create domain types**

Create `internal/domain/types.go`:

```go
package domain

import "time"

type ChatInfo struct {
	ID          int64
	Title       string
	UnreadCount int
	LastMessage string
	LastTime    time.Time
	Peer        interface{} // holds tg.InputPeerClass for sending
}

type Message struct {
	ID         int
	ChatID     int64
	SenderName string
	SenderID   int64
	Text       string
	Timestamp  time.Time
	Out        bool // true if sent by us
}

type AuthState int

const (
	AuthStateNone AuthState = iota
	AuthStatePhone
	AuthStateCode
	AuthState2FA
	AuthStateAuthenticated
)
```

**Step 2: Verify it compiles**

```bash
go build ./internal/domain/
```

Expected: no errors.

**Step 3: Commit**

```bash
git add internal/domain/
git commit -m "feat: add domain types for chats, messages, and auth state"
```

---

### Task 4: State Store

**Files:**
- Create: `internal/state/store.go`
- Create: `internal/state/store_test.go`

**Step 1: Write failing tests**

Create `internal/state/store_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/state/
```

Expected: compilation failure.

**Step 3: Write implementation**

Create `internal/state/store.go`:

```go
package state

import (
	"sort"
	"sync"

	"github.com/danhigham/tg-tui/internal/domain"
)

const maxMessages = 500

type Store struct {
	mu           sync.RWMutex
	chatList     []domain.ChatInfo
	messages     map[int64][]domain.Message
	activeChat   int64
	authState    domain.AuthState
	drawFunc     func()
}

func New(drawFunc func()) *Store {
	return &Store{
		messages: make(map[int64][]domain.Message),
		drawFunc: drawFunc,
	}
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
```

**Step 4: Run tests**

```bash
go test ./internal/state/ -v
```

Expected: all 5 tests PASS.

**Step 5: Commit**

```bash
git add internal/state/
git commit -m "feat: add state store with event handlers and tests"
```

---

### Task 5: Telegram Client Interface

**Files:**
- Create: `internal/telegram/client.go`

This defines the interface the rest of the app uses to talk to Telegram. The actual gotd/td implementation will be wired up later.

**Step 1: Create the interface and types**

Create `internal/telegram/client.go`:

```go
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
}

// Client is the interface for Telegram operations.
type Client interface {
	Run(ctx context.Context) error
	SendMessage(ctx context.Context, chatID int64, text string) error
	GetHistory(ctx context.Context, chatID int64, limit int) ([]domain.Message, error)
	GetDialogs(ctx context.Context) ([]domain.ChatInfo, error)
	MarkAsRead(ctx context.Context, chatID int64, maxID int) error
}
```

**Step 2: Verify it compiles**

```bash
go build ./internal/telegram/
```

Expected: no errors.

**Step 3: Commit**

```bash
git add internal/telegram/
git commit -m "feat: add Telegram client interface"
```

---

### Task 6: Telegram Client Implementation (gotd/td)

**Files:**
- Create: `internal/telegram/gotd.go`
- Create: `internal/telegram/auth.go`

**Step 1: Create the gotd client implementation**

Create `internal/telegram/gotd.go`:

```go
package telegram

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/query"
	"github.com/gotd/td/telegram/updates"
	updhook "github.com/gotd/td/telegram/updates/hook"
	"github.com/gotd/td/tg"

	"github.com/danhigham/tg-tui/internal/domain"
)

type GotdClient struct {
	apiID      int
	apiHash    string
	sessionDir string
	handler    EventHandler
	authFlow   *TUIAuth
	logger     *zap.Logger

	client *telegram.Client
	api    *tg.Client
	sender *message.Sender
	gaps   *updates.Manager
	self   *tg.User
}

func NewGotdClient(apiID int, apiHash, sessionDir string, handler EventHandler, authFlow *TUIAuth, logger *zap.Logger) *GotdClient {
	return &GotdClient{
		apiID:      apiID,
		apiHash:    apiHash,
		sessionDir: sessionDir,
		handler:    handler,
		authFlow:   authFlow,
		logger:     logger,
	}
}

func (c *GotdClient) Run(ctx context.Context) error {
	dispatcher := tg.NewUpdateDispatcher()

	c.gaps = updates.New(updates.Config{
		Handler: dispatcher,
		Logger:  c.logger.Named("gaps"),
	})

	c.client = telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
		Logger: c.logger,
		SessionStorage: &telegram.FileSessionStorage{
			Path: c.sessionDir + "/session.json",
		},
		UpdateHandler: c.gaps,
		Middlewares: []telegram.Middleware{
			updhook.UpdateHook(c.gaps.Handle),
		},
	})

	// Register update handlers
	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewMessage) error {
		msg, ok := u.Message.(*tg.Message)
		if !ok {
			return nil
		}
		c.handler.OnNewMessage(c.convertMessage(msg, e))
		return nil
	})

	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewChannelMessage) error {
		msg, ok := u.Message.(*tg.Message)
		if !ok {
			return nil
		}
		c.handler.OnNewMessage(c.convertMessage(msg, e))
		return nil
	})

	return c.client.Run(ctx, func(ctx context.Context) error {
		flow := auth.NewFlow(c.authFlow, auth.SendCodeOptions{})

		if err := c.client.Auth().IfNecessary(ctx, flow); err != nil {
			return fmt.Errorf("auth: %w", err)
		}

		self, err := c.client.Self(ctx)
		if err != nil {
			return fmt.Errorf("get self: %w", err)
		}
		c.self = self
		c.api = c.client.API()
		c.sender = message.NewSender(c.api)

		// Load initial chat list
		chats, err := c.GetDialogs(ctx)
		if err != nil {
			c.logger.Error("failed to load dialogs", zap.Error(err))
		} else {
			c.handler.OnChatListUpdate(chats)
		}

		// Run gap-aware update engine (blocks until ctx canceled)
		return c.gaps.Run(ctx, c.api, self.ID, updates.AuthOptions{
			OnStart: func(ctx context.Context) {
				c.logger.Info("listening for updates")
			},
		})
	})
}

func (c *GotdClient) SendMessage(ctx context.Context, chatID int64, text string) error {
	peer := c.findPeer(chatID)
	if peer == nil {
		return fmt.Errorf("unknown peer: %d", chatID)
	}
	_, err := c.sender.To(peer).Text(ctx, text)
	return err
}

func (c *GotdClient) GetHistory(ctx context.Context, chatID int64, limit int) ([]domain.Message, error) {
	peer := c.findPeer(chatID)
	if peer == nil {
		return nil, fmt.Errorf("unknown peer: %d", chatID)
	}

	result, err := c.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  peer,
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}

	return c.convertHistoryResult(result), nil
}

func (c *GotdClient) GetDialogs(ctx context.Context) ([]domain.ChatInfo, error) {
	var chats []domain.ChatInfo

	iter := query.GetDialogs(c.api).BatchSize(100).Iter()
	for iter.Next(ctx) {
		elem := iter.Value()
		if elem.Deleted() {
			continue
		}

		dlg, ok := elem.Dialog.(*tg.Dialog)
		if !ok {
			continue
		}

		info := domain.ChatInfo{
			UnreadCount: dlg.UnreadCount,
			Peer:        elem.Peer,
		}

		// Determine title from peer type
		switch p := elem.Peer.(type) {
		case *tg.InputPeerUser:
			if u, ok := elem.Entities.Users[p.UserID]; ok {
				info.ID = p.UserID
				info.Title = formatUserName(u)
			}
		case *tg.InputPeerChat:
			if ch, ok := elem.Entities.Chats[p.ChatID]; ok {
				info.ID = p.ChatID
				info.Title = ch.Title
			}
		case *tg.InputPeerChannel:
			if ch, ok := elem.Entities.Channels[p.ChannelID]; ok {
				info.ID = p.ChannelID
				info.Title = ch.Title
			}
		}

		if lastMsg, ok := elem.Last.(*tg.Message); ok {
			info.LastMessage = lastMsg.Message
			info.LastTime = time.Unix(int64(lastMsg.Date), 0)
		}

		if info.Title != "" {
			chats = append(chats, info)
		}
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return chats, nil
}

func (c *GotdClient) MarkAsRead(ctx context.Context, chatID int64, maxID int) error {
	peer := c.findPeer(chatID)
	if peer == nil {
		return fmt.Errorf("unknown peer: %d", chatID)
	}

	_, err := c.api.MessagesReadHistory(ctx, &tg.MessagesReadHistoryRequest{
		Peer:  peer.(*tg.InputPeerUser), // simplified - needs proper type handling
		MaxID: maxID,
	})
	return err
}

func (c *GotdClient) findPeer(chatID int64) tg.InputPeerClass {
	// Look up peer from cached chat list
	// This is a simplified version - in production we'd use a peer storage
	chats := c.handler.(*stateAdapter).store.GetChatList()
	for _, chat := range chats {
		if chat.ID == chatID {
			if p, ok := chat.Peer.(tg.InputPeerClass); ok {
				return p
			}
		}
	}
	return nil
}

func (c *GotdClient) convertMessage(msg *tg.Message, e tg.Entities) domain.Message {
	senderName := "Unknown"
	var senderID int64

	if msg.FromID != nil {
		if p, ok := msg.FromID.(*tg.PeerUser); ok {
			senderID = p.UserID
			if u, ok := e.Users[p.UserID]; ok {
				senderName = formatUserName(u)
			}
		}
	} else if c.self != nil && msg.Out {
		senderName = "You"
		senderID = c.self.ID
	}

	// Determine chatID from PeerID
	var chatID int64
	if peer := msg.GetPeerID(); peer != nil {
		switch p := peer.(type) {
		case *tg.PeerUser:
			chatID = p.UserID
		case *tg.PeerChat:
			chatID = p.ChatID
		case *tg.PeerChannel:
			chatID = p.ChannelID
		}
	}

	return domain.Message{
		ID:         msg.ID,
		ChatID:     chatID,
		SenderName: senderName,
		SenderID:   senderID,
		Text:       msg.Message,
		Timestamp:  time.Unix(int64(msg.Date), 0),
		Out:        msg.Out,
	}
}

func (c *GotdClient) convertHistoryResult(result tg.MessagesMessagesClass) []domain.Message {
	var rawMsgs []tg.MessageClass
	var users map[int64]*tg.User

	switch v := result.(type) {
	case *tg.MessagesMessages:
		rawMsgs = v.Messages
		users = usersToMap(v.Users)
	case *tg.MessagesMessagesSlice:
		rawMsgs = v.Messages
		users = usersToMap(v.Users)
	case *tg.MessagesChannelMessages:
		rawMsgs = v.Messages
		users = usersToMap(v.Users)
	default:
		return nil
	}

	entities := tg.Entities{Users: users}
	var msgs []domain.Message
	for i := len(rawMsgs) - 1; i >= 0; i-- { // reverse to chronological
		if msg, ok := rawMsgs[i].(*tg.Message); ok {
			msgs = append(msgs, c.convertMessage(msg, entities))
		}
	}
	return msgs
}

func usersToMap(users []tg.UserClass) map[int64]*tg.User {
	m := make(map[int64]*tg.User)
	for _, u := range users {
		if user, ok := u.(*tg.User); ok {
			m[user.ID] = user
		}
	}
	return m
}

func formatUserName(u *tg.User) string {
	if u.FirstName != "" && u.LastName != "" {
		return u.FirstName + " " + u.LastName
	}
	if u.FirstName != "" {
		return u.FirstName
	}
	if u.Username != "" {
		return u.Username
	}
	return "User"
}

// stateAdapter wraps the state store to expose it for peer lookup.
// This will be set by main when wiring things together.
type stateAdapter struct {
	store interface {
		GetChatList() []domain.ChatInfo
	}
}
```

**Step 2: Create the TUI auth flow**

Create `internal/telegram/auth.go`:

```go
package telegram

import (
	"context"
	"errors"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// TUIAuth implements gotd's auth.UserAuthenticator using channels
// so the TUI can provide input asynchronously.
type TUIAuth struct {
	PhoneCh    chan string
	CodeCh     chan string
	PasswordCh chan string
	ErrCh      chan error

	// Callbacks to notify TUI what's needed
	OnPhoneRequested    func()
	OnCodeRequested     func()
	OnPasswordRequested func()
}

func NewTUIAuth() *TUIAuth {
	return &TUIAuth{
		PhoneCh:    make(chan string, 1),
		CodeCh:     make(chan string, 1),
		PasswordCh: make(chan string, 1),
		ErrCh:      make(chan error, 1),
	}
}

func (a *TUIAuth) Phone(ctx context.Context) (string, error) {
	if a.OnPhoneRequested != nil {
		a.OnPhoneRequested()
	}
	select {
	case phone := <-a.PhoneCh:
		return phone, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (a *TUIAuth) Code(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
	if a.OnCodeRequested != nil {
		a.OnCodeRequested()
	}
	select {
	case code := <-a.CodeCh:
		return code, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (a *TUIAuth) Password(ctx context.Context) (string, error) {
	if a.OnPasswordRequested != nil {
		a.OnPasswordRequested()
	}
	select {
	case pw := <-a.PasswordCh:
		return pw, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (a *TUIAuth) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	return &auth.SignUpRequired{TermsOfService: tos}
}

func (a *TUIAuth) SignUp(ctx context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, errors.New("sign up not supported")
}
```

**Step 3: Verify it compiles**

```bash
go build ./internal/telegram/
```

Expected: may need small fixes for import issues. Fix and retry until it compiles.

**Step 4: Commit**

```bash
git add internal/telegram/
git commit -m "feat: add gotd/td Telegram client and TUI auth flow"
```

---

### Task 7: TUI - Chat List Component

**Files:**
- Create: `internal/ui/chatlist.go`

**Step 1: Create the chat list component**

Create `internal/ui/chatlist.go`:

```go
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
```

**Step 2: Verify it compiles**

```bash
go build ./internal/ui/
```

**Step 3: Commit**

```bash
git add internal/ui/chatlist.go
git commit -m "feat: add chat list TUI component"
```

---

### Task 8: TUI - Message View Component

**Files:**
- Create: `internal/ui/messageview.go`

**Step 1: Create the message view**

Create `internal/ui/messageview.go`:

```go
package ui

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"

	"github.com/danhigham/tg-tui/internal/domain"
)

type MessageView struct {
	*tview.TextView
}

func NewMessageView() *MessageView {
	mv := &MessageView{
		TextView: tview.NewTextView(),
	}
	mv.SetDynamicColors(true)
	mv.SetScrollable(true)
	mv.SetWordWrap(true)
	mv.SetBorder(true).SetTitle(" Messages ")
	mv.ScrollToEnd()
	return mv
}

func (mv *MessageView) SetChatTitle(title string) {
	mv.SetTitle(fmt.Sprintf(" %s ", title))
}

func (mv *MessageView) Update(messages []domain.Message) {
	var b strings.Builder

	for _, msg := range messages {
		ts := msg.Timestamp.Format("15:04")

		var nameColor string
		if msg.Out {
			nameColor = "green"
		} else {
			nameColor = "blue"
		}

		fmt.Fprintf(&b, "[gray]%s [%s]%s:[white] %s\n",
			ts, nameColor, tview.Escape(msg.SenderName), tview.Escape(msg.Text))
	}

	mv.SetText(b.String())
	mv.ScrollToEnd()
}
```

**Step 2: Verify it compiles**

```bash
go build ./internal/ui/
```

**Step 3: Commit**

```bash
git add internal/ui/messageview.go
git commit -m "feat: add message view TUI component"
```

---

### Task 9: TUI - Input Component

**Files:**
- Create: `internal/ui/input.go`

**Step 1: Create the input component**

Create `internal/ui/input.go`:

```go
package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Input struct {
	*tview.InputField
	onSend func(text string)
}

func NewInput() *Input {
	inp := &Input{
		InputField: tview.NewInputField(),
	}
	inp.SetLabel("> ")
	inp.SetFieldWidth(0)
	inp.SetPlaceholder("Type a message...")

	inp.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			text := inp.GetText()
			if text != "" && inp.onSend != nil {
				inp.onSend(text)
				inp.SetText("")
			}
		}
	})

	return inp
}

func (inp *Input) SetOnSend(fn func(text string)) {
	inp.onSend = fn
}
```

**Step 2: Verify it compiles**

```bash
go build ./internal/ui/
```

**Step 3: Commit**

```bash
git add internal/ui/input.go
git commit -m "feat: add message input TUI component"
```

---

### Task 10: TUI - Auth Modal Component

**Files:**
- Create: `internal/ui/auth.go`

**Step 1: Create the auth modal**

Create `internal/ui/auth.go`:

```go
package ui

import (
	"github.com/rivo/tview"
)

type AuthModal struct {
	pages *tview.Pages
	app   *tview.Application

	phoneForm    *tview.Form
	codeForm     *tview.Form
	passwordForm *tview.Form

	onPhone    func(phone string)
	onCode     func(code string)
	onPassword func(password string)
}

func NewAuthModal(app *tview.Application, pages *tview.Pages) *AuthModal {
	am := &AuthModal{
		pages: pages,
		app:   app,
	}
	am.buildForms()
	return am
}

func (am *AuthModal) buildForms() {
	// Phone form
	am.phoneForm = tview.NewForm().
		AddInputField("Phone Number", "", 20, nil, nil).
		AddButton("Submit", func() {
			phone := am.phoneForm.GetFormItemByLabel("Phone Number").(*tview.InputField).GetText()
			if phone != "" && am.onPhone != nil {
				am.onPhone(phone)
				am.pages.HidePage("auth")
			}
		})
	am.phoneForm.SetBorder(true).SetTitle(" Enter Phone Number ")

	// Code form
	am.codeForm = tview.NewForm().
		AddInputField("Verification Code", "", 10, nil, nil).
		AddButton("Submit", func() {
			code := am.codeForm.GetFormItemByLabel("Verification Code").(*tview.InputField).GetText()
			if code != "" && am.onCode != nil {
				am.onCode(code)
				am.pages.HidePage("auth")
			}
		})
	am.codeForm.SetBorder(true).SetTitle(" Enter Verification Code ")

	// Password form
	am.passwordForm = tview.NewForm().
		AddPasswordField("2FA Password", "", 20, '*', nil).
		AddButton("Submit", func() {
			pw := am.passwordForm.GetFormItemByLabel("2FA Password").(*tview.InputField).GetText()
			if pw != "" && am.onPassword != nil {
				am.onPassword(pw)
				am.pages.HidePage("auth")
			}
		})
	am.passwordForm.SetBorder(true).SetTitle(" Enter 2FA Password ")
}

func (am *AuthModal) SetCallbacks(onPhone func(string), onCode func(string), onPassword func(string)) {
	am.onPhone = onPhone
	am.onCode = onCode
	am.onPassword = onPassword
}

func (am *AuthModal) ShowPhone() {
	am.phoneForm.GetFormItemByLabel("Phone Number").(*tview.InputField).SetText("")
	centered := am.center(am.phoneForm, 50, 7)
	am.pages.AddAndSwitchToPage("auth", centered, true)
	am.app.SetFocus(am.phoneForm)
}

func (am *AuthModal) ShowCode() {
	am.codeForm.GetFormItemByLabel("Verification Code").(*tview.InputField).SetText("")
	centered := am.center(am.codeForm, 50, 7)
	am.pages.AddAndSwitchToPage("auth", centered, true)
	am.app.SetFocus(am.codeForm)
}

func (am *AuthModal) ShowPassword() {
	am.passwordForm.GetFormItemByLabel("2FA Password").(*tview.InputField).SetText("")
	centered := am.center(am.passwordForm, 50, 7)
	am.pages.AddAndSwitchToPage("auth", centered, true)
	am.app.SetFocus(am.passwordForm)
}

func (am *AuthModal) center(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
}
```

**Step 2: Verify it compiles**

```bash
go build ./internal/ui/
```

**Step 3: Commit**

```bash
git add internal/ui/auth.go
git commit -m "feat: add auth modal TUI component"
```

---

### Task 11: TUI - Main App Layout and Wiring

**Files:**
- Create: `internal/ui/app.go`

**Step 1: Create the app layout**

Create `internal/ui/app.go`:

```go
package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/danhigham/tg-tui/internal/domain"
	"github.com/danhigham/tg-tui/internal/state"
)

type App struct {
	Application *tview.Application
	Pages       *tview.Pages
	ChatList    *ChatList
	MessageView *MessageView
	Input       *Input
	AuthModal   *AuthModal
	StatusBar   *tview.TextView
	Store       *state.Store

	focusOrder []tview.Primitive
	focusIndex int
}

func NewApp(store *state.Store) *App {
	a := &App{
		Application: tview.NewApplication(),
		Store:       store,
	}

	a.ChatList = NewChatList()
	a.MessageView = NewMessageView()
	a.Input = NewInput()
	a.StatusBar = tview.NewTextView().SetDynamicColors(true)
	a.StatusBar.SetText("[yellow]Connecting...")

	a.Pages = tview.NewPages()
	a.AuthModal = NewAuthModal(a.Application, a.Pages)

	a.focusOrder = []tview.Primitive{a.ChatList, a.MessageView, a.Input}
	a.focusIndex = 0

	a.buildLayout()
	a.setupKeybindings()

	return a
}

func (a *App) buildLayout() {
	// Right pane: messages + input stacked vertically
	rightPane := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.MessageView, 0, 1, false).
		AddItem(a.Input, 1, 0, false)

	// Main layout: chat list | right pane
	mainLayout := tview.NewFlex().
		AddItem(a.ChatList, 30, 0, true).
		AddItem(rightPane, 0, 1, false)

	// Full layout: main + status bar
	fullLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(mainLayout, 0, 1, true).
		AddItem(a.StatusBar, 1, 0, false)

	a.Pages.AddPage("main", fullLayout, true, true)
	a.Application.SetRoot(a.Pages, true)
}

func (a *App) setupKeybindings() {
	a.Application.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Ctrl+C to quit
		if event.Key() == tcell.KeyCtrlC {
			a.Application.Stop()
			return nil
		}

		// Tab / Shift+Tab to cycle focus
		if event.Key() == tcell.KeyTab {
			a.focusIndex = (a.focusIndex + 1) % len(a.focusOrder)
			a.Application.SetFocus(a.focusOrder[a.focusIndex])
			return nil
		}
		if event.Key() == tcell.KeyBacktab {
			a.focusIndex = (a.focusIndex - 1 + len(a.focusOrder)) % len(a.focusOrder)
			a.Application.SetFocus(a.focusOrder[a.focusIndex])
			return nil
		}

		// Escape returns to chat list
		if event.Key() == tcell.KeyEscape {
			a.focusIndex = 0
			a.Application.SetFocus(a.focusOrder[0])
			return nil
		}

		// 'q' quits only when not in input
		if event.Rune() == 'q' && a.Application.GetFocus() != a.Input.InputField {
			a.Application.Stop()
			return nil
		}

		return event
	})
}

func (a *App) Refresh() {
	chats := a.Store.GetChatList()
	a.ChatList.Update(chats)

	activeChat := a.Store.GetActiveChat()
	if activeChat != 0 {
		msgs := a.Store.GetMessages(activeChat)
		a.MessageView.Update(msgs)
	}
}

func (a *App) SetStatus(text string) {
	a.StatusBar.SetText(text)
}

func (a *App) Run() error {
	return a.Application.Run()
}

func (a *App) QueueUpdateDraw(f func()) {
	a.Application.QueueUpdateDraw(f)
}

// DrawFunc returns a function suitable for state.New() that triggers a tview redraw.
func (a *App) DrawFunc() func() {
	return func() {
		a.Application.QueueUpdateDraw(func() {
			a.Refresh()
		})
	}
}
```

**Step 2: Verify it compiles**

```bash
go build ./internal/ui/
```

**Step 3: Commit**

```bash
git add internal/ui/app.go
git commit -m "feat: add main TUI app layout with keybindings"
```

---

### Task 12: Wire Everything Together in main.go

**Files:**
- Modify: `cmd/tg-tui/main.go`

**Step 1: Update main.go with full wiring**

Replace `cmd/tg-tui/main.go` with:

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/danhigham/tg-tui/internal/config"
	"github.com/danhigham/tg-tui/internal/state"
	"github.com/danhigham/tg-tui/internal/telegram"
	"github.com/danhigham/tg-tui/internal/ui"
)

func main() {
	// Load config
	cfgDir := config.Dir()
	cfgPath := filepath.Join(cfgDir, "config.yaml")

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config from %s: %v\n", cfgPath, err)
		fmt.Fprintf(os.Stderr, "\nCreate the config file with:\n")
		fmt.Fprintf(os.Stderr, "  mkdir -p %s\n", cfgDir)
		fmt.Fprintf(os.Stderr, "  cat > %s << 'EOF'\n", cfgPath)
		fmt.Fprintf(os.Stderr, "telegram:\n  api_id: YOUR_API_ID\n  api_hash: \"YOUR_API_HASH\"\nEOF\n")
		fmt.Fprintf(os.Stderr, "\nGet API credentials from https://my.telegram.org\n")
		os.Exit(1)
	}

	// Setup logging to file
	logPath := filepath.Join(cfgDir, "tg-tui.log")
	logCfg := zap.NewDevelopmentConfig()
	logCfg.OutputPaths = []string{logPath}
	logCfg.ErrorOutputPaths = []string{logPath}
	logger, err := logCfg.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Create store (drawFunc will be set after app is created)
	store := state.New(nil)

	// Create TUI
	app := ui.NewApp(store)

	// Now wire drawFunc
	store.SetDrawFunc(app.DrawFunc())

	// Create auth flow with TUI integration
	authFlow := telegram.NewTUIAuth()
	authFlow.OnPhoneRequested = func() {
		app.QueueUpdateDraw(func() {
			app.AuthModal.ShowPhone()
		})
	}
	authFlow.OnCodeRequested = func() {
		app.QueueUpdateDraw(func() {
			app.AuthModal.ShowCode()
		})
	}
	authFlow.OnPasswordRequested = func() {
		app.QueueUpdateDraw(func() {
			app.AuthModal.ShowPassword()
		})
	}

	// Wire auth modal callbacks to auth flow channels
	app.AuthModal.SetCallbacks(
		func(phone string) { authFlow.PhoneCh <- phone },
		func(code string) { authFlow.CodeCh <- code },
		func(password string) { authFlow.PasswordCh <- password },
	)

	// Ensure session directory exists
	sessionDir := cfgDir
	os.MkdirAll(sessionDir, 0700)

	// Create Telegram client
	tgClient := telegram.NewGotdClient(
		cfg.Telegram.APIID,
		cfg.Telegram.APIHash,
		sessionDir,
		store,
		authFlow,
		logger,
	)

	// Wire chat selection
	app.ChatList.SetOnSelect(func(chatID int64) {
		store.SetActiveChat(chatID)

		// Clear unread count
		chats := store.GetChatList()
		for _, c := range chats {
			if c.ID == chatID {
				// Update title
				app.MessageView.SetChatTitle(c.Title)
				break
			}
		}

		// Load history if not cached
		msgs := store.GetMessages(chatID)
		if len(msgs) == 0 {
			go func() {
				ctx := context.Background()
				history, err := tgClient.GetHistory(ctx, chatID, 50)
				if err != nil {
					logger.Error("failed to load history", zap.Error(err))
					return
				}
				store.SetMessages(chatID, history)
			}()
		} else {
			app.QueueUpdateDraw(func() {
				app.MessageView.Update(msgs)
			})
		}

		// Switch focus to input
		app.Application.SetFocus(app.Input.InputField)
	})

	// Wire message sending
	app.Input.SetOnSend(func(text string) {
		chatID := store.GetActiveChat()
		if chatID == 0 {
			return
		}
		go func() {
			ctx := context.Background()
			if err := tgClient.SendMessage(ctx, chatID, text); err != nil {
				logger.Error("failed to send message", zap.Error(err))
				app.QueueUpdateDraw(func() {
					app.SetStatus(fmt.Sprintf("[red]Send error: %v", err))
				})
			}
		}()
	})

	// Context for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Run Telegram client in background
	go func() {
		if err := tgClient.Run(ctx); err != nil {
			logger.Error("telegram client error", zap.Error(err))
			app.QueueUpdateDraw(func() {
				app.SetStatus(fmt.Sprintf("[red]Disconnected: %v", err))
			})
		}
	}()

	// Run TUI (blocks until quit)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cancel() // stop telegram client
}
```

**Step 2: Add SetDrawFunc to state store**

The state store needs a `SetDrawFunc` method. Add to `internal/state/store.go`:

```go
func (s *Store) SetDrawFunc(f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.drawFunc = f
}
```

**Step 3: Fix the findPeer method in gotd.go**

The `findPeer` method needs access to the store. Refactor to accept the store directly in the constructor. Update `internal/telegram/gotd.go`:

Replace the `findPeer` method and remove the `stateAdapter` type. Instead, add a `peerCache` map:

```go
// Add to GotdClient struct:
peerCache map[int64]tg.InputPeerClass
mu        sync.Mutex

// In NewGotdClient, initialize:
peerCache: make(map[int64]tg.InputPeerClass),

// Replace findPeer:
func (c *GotdClient) findPeer(chatID int64) tg.InputPeerClass {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.peerCache[chatID]
}

func (c *GotdClient) cachePeer(chatID int64, peer tg.InputPeerClass) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.peerCache[chatID] = peer
}
```

Update `GetDialogs` to cache peers:

```go
// After setting info.ID, add:
c.cachePeer(info.ID, elem.Peer)
```

**Step 4: Verify it compiles**

```bash
go build ./cmd/tg-tui/
```

Expected: compiles successfully.

**Step 5: Commit**

```bash
git add cmd/tg-tui/main.go internal/state/store.go internal/telegram/gotd.go
git commit -m "feat: wire all components together in main"
```

---

### Task 13: Fix Compilation Issues and Test

**Step 1: Run full build**

```bash
cd /home/danhigham/workspace/tg-tui
go build ./...
```

Fix any compilation errors that arise. Common issues:
- Import cycles
- Missing type assertions
- Unused imports

**Step 2: Run all tests**

```bash
go test ./...
```

Fix any test failures.

**Step 3: Run vet and basic checks**

```bash
go vet ./...
```

**Step 4: Commit any fixes**

```bash
git add -A
git commit -m "fix: resolve compilation and vet issues"
```

---

### Task 14: Manual Testing and Polish

**Step 1: Create a sample config for testing**

```bash
mkdir -p ~/.config/tg-tui
cat > ~/.config/tg-tui/config.yaml << 'EOF'
telegram:
  api_id: YOUR_API_ID
  api_hash: "YOUR_API_HASH"
log_level: debug
EOF
```

User must fill in real credentials from https://my.telegram.org.

**Step 2: Run the application**

```bash
cd /home/danhigham/workspace/tg-tui
go run ./cmd/tg-tui/
```

**Step 3: Verify auth flow**

- App should show phone number prompt
- Enter phone → receive code → enter code
- Session should be saved to `~/.config/tg-tui/session.json`
- Subsequent runs should skip auth

**Step 4: Verify chat list**

- After auth, chat list should populate
- Arrow keys / j/k should navigate
- Unread counts should show in bold

**Step 5: Verify messaging**

- Select a chat with Enter
- Message history should load
- Type a message and press Enter to send
- Incoming messages should appear in real-time

**Step 6: Verify keybindings**

- Tab cycles focus between chat list, message view, input
- Escape returns to chat list
- q quits (when not in input)
- Ctrl+C quits from anywhere

**Step 7: Fix any issues found and commit**

```bash
git add -A
git commit -m "fix: polish from manual testing"
```
