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
	ID          int
	ChatID      int64
	SenderName  string
	SenderID    int64
	Text        string
	HasMarkdown bool // true if Text contains markdown from Telegram entities
	Timestamp   time.Time
	Out         bool // true if sent by us
}

type AuthState int

const (
	AuthStateNone AuthState = iota
	AuthStatePhone
	AuthStateCode
	AuthState2FA
	AuthStateAuthenticated
)
