package telegram

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/query/dialogs"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/tg"

	"github.com/danhigham/tg-tui/internal/domain"
)

// GotdClient implements the Client interface using gotd/td.
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

	peerCache map[int64]tg.InputPeerClass
	nameCache map[int64]string
	mu        sync.Mutex

	onReady func()
}

// NewGotdClient creates a new GotdClient.
func (c *GotdClient) SetOnReady(fn func()) {
	c.onReady = fn
}

func NewGotdClient(apiID int, apiHash, sessionDir string, handler EventHandler, authFlow *TUIAuth, logger *zap.Logger) *GotdClient {
	return &GotdClient{
		apiID:      apiID,
		apiHash:    apiHash,
		sessionDir: sessionDir,
		handler:    handler,
		authFlow:   authFlow,
		logger:     logger,
		peerCache:  make(map[int64]tg.InputPeerClass),
		nameCache:  make(map[int64]string),
	}
}

// Run starts the Telegram client and blocks until ctx is cancelled.
func (c *GotdClient) Run(ctx context.Context) error {
	dispatcher := tg.NewUpdateDispatcher()

	// Register message handlers on the dispatcher.
	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
		msg, ok := update.Message.(*tg.Message)
		if !ok {
			return nil
		}
		users := e.Users
		domainMsg := c.convertMessage(msg, users)
		c.handler.OnNewMessage(domainMsg)
		return nil
	})

	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewChannelMessage) error {
		msg, ok := update.Message.(*tg.Message)
		if !ok {
			return nil
		}
		users := e.Users
		domainMsg := c.convertMessage(msg, users)
		c.handler.OnNewMessage(domainMsg)
		return nil
	})

	// Register typing event handlers.
	dispatcher.OnUserTyping(func(ctx context.Context, e tg.Entities, update *tg.UpdateUserTyping) error {
		switch update.Action.(type) {
		case *tg.SendMessageTypingAction:
			userName := c.findUserName(update.UserID)
			if userName == "" {
				if u, ok := e.Users[update.UserID]; ok {
					userName = formatUserName(u)
					c.cacheUserName(update.UserID, userName)
				} else {
					userName = "Someone"
				}
			}
			c.handler.OnUserTyping(update.UserID, userName)
		case *tg.SendMessageCancelAction:
			c.handler.OnUserTypingStop(update.UserID)
		}
		return nil
	})

	dispatcher.OnChatUserTyping(func(ctx context.Context, e tg.Entities, update *tg.UpdateChatUserTyping) error {
		switch update.Action.(type) {
		case *tg.SendMessageTypingAction:
			userName := "Someone"
			if p, ok := update.FromID.(*tg.PeerUser); ok {
				userName = c.findUserName(p.UserID)
				if userName == "" {
					if u, ok := e.Users[p.UserID]; ok {
						userName = formatUserName(u)
						c.cacheUserName(p.UserID, userName)
					} else {
						userName = "Someone"
					}
				}
			}
			c.handler.OnUserTyping(update.ChatID, userName)
		case *tg.SendMessageCancelAction:
			c.handler.OnUserTypingStop(update.ChatID)
		}
		return nil
	})

	// Create gap-aware update manager.
	c.gaps = updates.New(updates.Config{
		Handler: dispatcher,
		Logger:  c.logger.Named("gaps"),
	})

	// Create the telegram client.
	c.client = telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
		Logger:         c.logger,
		UpdateHandler:  c.gaps,
		SessionStorage: &session.FileStorage{Path: filepath.Join(c.sessionDir, "session.json")},
	})

	return c.client.Run(ctx, func(ctx context.Context) error {
		// Authenticate if necessary.
		flow := auth.NewFlow(c.authFlow, auth.SendCodeOptions{})
		if err := c.client.Auth().IfNecessary(ctx, flow); err != nil {
			return fmt.Errorf("auth: %w", err)
		}

		// Get self user.
		self, err := c.client.Self(ctx)
		if err != nil {
			return fmt.Errorf("get self: %w", err)
		}
		c.self = self

		// Set up API and sender.
		c.api = c.client.API()
		c.sender = message.NewSender(c.api)

		// Load initial dialogs to populate handler and peer cache.
		chatInfos, err := c.GetDialogs(ctx)
		if err != nil {
			c.logger.Warn("Failed to load initial dialogs", zap.Error(err))
		} else {
			c.handler.OnChatListUpdate(chatInfos)
		}

		// Notify that connection is ready.
		if c.onReady != nil {
			c.onReady()
		}

		// Run gap manager to process updates.
		return c.gaps.Run(ctx, c.api, self.ID, updates.AuthOptions{})
	})
}

// SendMessage sends a text message to the given chat.
func (c *GotdClient) SendMessage(ctx context.Context, chatID int64, text string) error {
	peer := c.findPeer(chatID)
	if peer == nil {
		return fmt.Errorf("unknown peer: %d", chatID)
	}
	_, err := c.sender.To(peer).Text(ctx, text)
	return err
}

// GetHistory retrieves message history for a chat.
func (c *GotdClient) GetHistory(ctx context.Context, chatID int64, limit int, offsetID int) ([]domain.Message, error) {
	peer := c.findPeer(chatID)
	if peer == nil {
		return nil, fmt.Errorf("unknown peer: %d", chatID)
	}

	result, err := c.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:     peer,
		Limit:    limit,
		OffsetID: offsetID,
	})
	if err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}

	return c.convertHistoryResult(result)
}

// GetDialogs retrieves the list of dialogs (chats).
func (c *GotdClient) GetDialogs(ctx context.Context) ([]domain.ChatInfo, error) {
	queryBuilder := dialogs.NewQueryBuilder(c.api)
	iter := queryBuilder.GetDialogs().BatchSize(100).Iter()

	var result []domain.ChatInfo
	for iter.Next(ctx) {
		elem := iter.Value()

		// Cache the peer for later use.
		peerID := c.peerIDFromInputPeer(elem.Peer)
		if peerID != 0 {
			c.cachePeer(peerID, elem.Peer)
		}

		// Determine chat title from entities.
		title := c.titleFromEntities(elem)

		// Get dialog details.
		var unreadCount int
		var lastMsg string
		var lastTime time.Time

		if dlg, ok := elem.Dialog.(*tg.Dialog); ok {
			unreadCount = dlg.UnreadCount
		}
		if elem.Last != nil {
			if msg, ok := elem.Last.(*tg.Message); ok {
				lastMsg = msg.Message
				lastTime = time.Unix(int64(msg.Date), 0)
			}
		}

		result = append(result, domain.ChatInfo{
			ID:          peerID,
			Title:       title,
			UnreadCount: unreadCount,
			LastMessage: lastMsg,
			LastTime:    lastTime,
			Peer:        elem.Peer,
		})
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("iterate dialogs: %w", err)
	}

	return result, nil
}

// MarkAsRead marks messages in a chat as read up to the given message ID.
func (c *GotdClient) MarkAsRead(ctx context.Context, chatID int64, maxID int) error {
	peer := c.findPeer(chatID)
	if peer == nil {
		return fmt.Errorf("unknown peer: %d", chatID)
	}

	switch p := peer.(type) {
	case *tg.InputPeerUser:
		_, err := c.api.MessagesReadHistory(ctx, &tg.MessagesReadHistoryRequest{
			Peer:  p,
			MaxID: maxID,
		})
		return err
	case *tg.InputPeerChat:
		_, err := c.api.MessagesReadHistory(ctx, &tg.MessagesReadHistoryRequest{
			Peer:  p,
			MaxID: maxID,
		})
		return err
	case *tg.InputPeerChannel:
		_, err := c.api.ChannelsReadHistory(ctx, &tg.ChannelsReadHistoryRequest{
			Channel: &tg.InputChannel{ChannelID: p.ChannelID, AccessHash: p.AccessHash},
			MaxID:   maxID,
		})
		return err
	default:
		return fmt.Errorf("unsupported peer type for mark as read: %T", peer)
	}
}

// findPeer looks up a cached peer by chat ID.
func (c *GotdClient) findPeer(chatID int64) tg.InputPeerClass {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.peerCache[chatID]
}

// cachePeer stores a peer in the cache.
func (c *GotdClient) cachePeer(chatID int64, peer tg.InputPeerClass) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.peerCache[chatID] = peer
}

// cacheUserName stores a user's display name for later lookup (e.g. typing indicators).
func (c *GotdClient) cacheUserName(userID int64, name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nameCache[userID] = name
}

// findUserName looks up a cached user display name.
func (c *GotdClient) findUserName(userID int64) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.nameCache[userID]
}

// convertMessage converts a tg.Message to a domain.Message.
func (c *GotdClient) convertMessage(msg *tg.Message, users map[int64]*tg.User) domain.Message {
	var senderName string
	var senderID int64

	if fromID := msg.FromID; fromID != nil {
		switch p := fromID.(type) {
		case *tg.PeerUser:
			senderID = p.UserID
			if u, ok := users[p.UserID]; ok {
				senderName = formatUserName(u)
			}
		case *tg.PeerChat:
			senderID = p.ChatID
		case *tg.PeerChannel:
			senderID = p.ChannelID
		}
	}

	// Determine chatID from PeerID.
	var chatID int64
	if peerID := msg.PeerID; peerID != nil {
		switch p := peerID.(type) {
		case *tg.PeerUser:
			chatID = p.UserID
		case *tg.PeerChat:
			chatID = p.ChatID
		case *tg.PeerChannel:
			chatID = p.ChannelID
		}
	}

	// In DMs, FromID is often nil. Derive sender from PeerID and Out flag.
	if senderName == "" && !msg.Out {
		if p, ok := msg.PeerID.(*tg.PeerUser); ok {
			senderID = p.UserID
			if u, ok := users[p.UserID]; ok {
				senderName = formatUserName(u)
			}
		}
	}
	if senderName == "" && msg.Out && c.self != nil {
		senderID = c.self.ID
		senderName = formatUserName(c.self)
	}

	// Cache the resolved name for typing indicator lookups.
	if senderID != 0 && senderName != "" {
		c.cacheUserName(senderID, senderName)
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

// convertHistoryResult extracts messages from a MessagesMessagesClass response.
func (c *GotdClient) convertHistoryResult(result tg.MessagesMessagesClass) ([]domain.Message, error) {
	var messages []tg.MessageClass
	var users []tg.UserClass

	switch r := result.(type) {
	case *tg.MessagesMessages:
		messages = r.Messages
		users = r.Users
	case *tg.MessagesMessagesSlice:
		messages = r.Messages
		users = r.Users
	case *tg.MessagesChannelMessages:
		messages = r.Messages
		users = r.Users
	default:
		return nil, fmt.Errorf("unexpected messages type: %T", result)
	}

	userMap := usersToMap(users)

	// Cache all user names for typing indicator lookups.
	for id, u := range userMap {
		c.cacheUserName(id, formatUserName(u))
	}

	// Messages come in reverse chronological order from the API; reverse them.
	var domainMsgs []domain.Message
	for i := len(messages) - 1; i >= 0; i-- {
		msg, ok := messages[i].(*tg.Message)
		if !ok {
			continue
		}
		domainMsgs = append(domainMsgs, c.convertMessage(msg, userMap))
	}

	return domainMsgs, nil
}

// titleFromEntities extracts the chat title from dialog entities.
func (c *GotdClient) titleFromEntities(elem dialogs.Elem) string {
	if elem.Peer == nil {
		return "Unknown"
	}

	entities := elem.Entities

	switch p := elem.Dialog.GetPeer().(type) {
	case *tg.PeerUser:
		if u, ok := entities.User(p.UserID); ok {
			name := formatUserName(u)
			c.cacheUserName(p.UserID, name)
			return name
		}
	case *tg.PeerChat:
		if ch, ok := entities.Chat(p.ChatID); ok {
			return ch.Title
		}
	case *tg.PeerChannel:
		if ch, ok := entities.Channel(p.ChannelID); ok {
			return ch.Title
		}
	}

	return "Unknown"
}

// peerIDFromInputPeer extracts a numeric peer ID from an InputPeerClass.
func (c *GotdClient) peerIDFromInputPeer(peer tg.InputPeerClass) int64 {
	switch p := peer.(type) {
	case *tg.InputPeerUser:
		return p.UserID
	case *tg.InputPeerChat:
		return p.ChatID
	case *tg.InputPeerChannel:
		return p.ChannelID
	default:
		return 0
	}
}

// formatUserName returns a display name for a user.
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
	return "Unknown"
}

// usersToMap converts a UserClass slice to a map of User by ID.
func usersToMap(users []tg.UserClass) map[int64]*tg.User {
	m := make(map[int64]*tg.User, len(users))
	for _, u := range users {
		user, ok := u.(*tg.User)
		if !ok {
			continue
		}
		m[user.ID] = user
	}
	return m
}
