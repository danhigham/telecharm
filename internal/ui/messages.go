package ui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/danhigham/tg-tui/internal/domain"
)

// StoreUpdatedMsg signals that the store state has changed.
type StoreUpdatedMsg struct{}

// AuthRequestMsg asks the UI to show the auth modal for a given stage.
type AuthRequestMsg struct {
	Stage domain.AuthState
}

// ChatSelectedMsg is emitted when the user picks a chat.
type ChatSelectedMsg struct {
	ChatID int64
}

// HistoryLoadedMsg delivers fetched history for a chat.
type HistoryLoadedMsg struct {
	ChatID   int64
	Messages []domain.Message
}

// sendMessageMsg is emitted when the user presses Enter in the input.
type sendMessageMsg struct {
	text string
}

// StatusMsg updates the status bar.
type StatusMsg struct {
	Text      string
	Connected bool
}

// SendErrorMsg reports a failed send attempt.
type SendErrorMsg struct {
	Err error
}

// LoadOlderHistoryMsg is emitted when the user scrolls to the top of messages.
type LoadOlderHistoryMsg struct {
	ChatID int64
}

// OlderHistoryLoadedMsg delivers older history fetched asynchronously.
type OlderHistoryLoadedMsg struct {
	ChatID   int64
	Messages []domain.Message
}

// SplashDoneMsg signals that the splash screen timeout has elapsed.
type SplashDoneMsg struct{}

// clockTickMsg triggers a status bar time refresh.
type clockTickMsg struct{}

// StoreUpdatedCmd returns a command that emits StoreUpdatedMsg.
func StoreUpdatedCmd() tea.Msg {
	return StoreUpdatedMsg{}
}
