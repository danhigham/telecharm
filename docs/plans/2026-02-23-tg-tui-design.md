# tg-tui Design Document

A curses-based Telegram client written in Go, using the same TUI stack as k9s.

## Dependencies

- **TUI:** `rivo/tview` + `gdamore/tcell/v2` (upstream versions of k9s's TUI libraries)
- **Telegram:** `gotd/td` (pure Go MTProto 2.0 implementation)
- **Logging:** `go.uber.org/zap` (structured logging to file)

## Architecture

Three-layer event-driven architecture:

```
Telegram layer (gotd/td wrapper)
        |  events via EventHandler interface
    State layer (app state, mutex-protected)
        |  QueueUpdateDraw()
    TUI layer (tview components)
```

### Telegram Layer (`internal/telegram/`)

Wraps gotd/td and exposes a clean interface.

**Files:**
- `client.go` — gotd client lifecycle (connect, reconnect, disconnect)
- `auth.go` — Multi-step auth: phone → code → optional 2FA. Session stored to disk.
- `updates.go` — Translates MTProto updates into domain events

**Event interface:**
```go
type EventHandler interface {
    OnNewMessage(msg Message)
    OnChatListUpdate(chats []ChatInfo)
    OnMessageRead(chatID int64, maxID int)
    OnUserStatus(userID int64, online bool)
}
```

**Outbound API:**
- `SendMessage(chatID int64, text string) error`
- `MarkAsRead(chatID int64, maxID int) error`
- `GetHistory(chatID int64, limit int) ([]Message, error)`

### State Layer (`internal/state/`)

Single source of truth. Implements `EventHandler`.

**State fields:**
- `ChatList []ChatInfo` — sorted by last message time
- `Messages map[int64][]Message` — per-chat message history (ring buffer, ~500 msgs)
- `SelectedChatIndex int` — highlighted chat in list
- `ActiveChatID int64` — chat being viewed
- `AuthState` — current auth step

Receives Telegram events, updates state, triggers `tview.Application.QueueUpdateDraw()`.
Lazy-loads message history when a chat is selected.

### TUI Layer (`internal/ui/`)

Two-pane layout:

```
┌──────────────────────────────────────────────────┐
│  tg-tui                                     [?]  │
├──────────────┬───────────────────────────────────┤
│ Chats        │ Chat: Alice                       │
│              │                                   │
│ > Alice    2 │ [14:30] Alice: Hey!               │
│   Bob      1 │ [14:31] You: Hi there             │
│   Work grp   │ [14:32] Alice: How's it going?    │
│   News ch    │                                   │
│              │                                   │
├──────────────┼───────────────────────────────────┤
│              │ > Type a message...            [⏎] │
└──────────────┴───────────────────────────────────┘
```

**Components:**
- `app.go` — tview.Application, layout grid, global keybindings
- `chatlist.go` — tview.List with unread counts
- `messageview.go` — tview.TextView, scrollable, auto-scroll on new messages
- `input.go` — tview.InputField, Enter to send
- `auth.go` — Modal forms for auth flow

**Key bindings:**
- `Tab` / `Shift+Tab` — cycle focus
- `j`/`k` or arrows — navigate chat list
- `PgUp`/`PgDn` — scroll messages
- `Enter` — select chat or send message
- `Esc` — return to chat list
- `q` / `Ctrl+C` — quit

### Configuration (`internal/config/`)

Config file: `~/.config/tg-tui/config.yaml`

```yaml
telegram:
  api_id: 12345
  api_hash: "abcdef..."
log_level: info
```

Session: `~/.config/tg-tui/session.json`
Logs: `~/.config/tg-tui/tg-tui.log`

## Error Handling

- **Network disconnects:** Status bar indicator, gotd auto-reconnects
- **Auth failures:** Re-prompt auth flow
- **API errors:** Non-blocking status bar display

## Testing

- **Telegram layer:** Interface-based mocking, unit tests
- **State layer:** Pure state transitions, unit tests
- **TUI layer:** Manual integration testing
- **E2E:** Manual with real Telegram account

## Project Structure

```
tg-tui/
├── cmd/tg-tui/main.go
├── internal/
│   ├── telegram/
│   │   ├── client.go
│   │   ├── auth.go
│   │   └── updates.go
│   ├── state/
│   │   └── store.go
│   ├── ui/
│   │   ├── app.go
│   │   ├── chatlist.go
│   │   ├── messageview.go
│   │   ├── input.go
│   │   └── auth.go
│   └── config/
│       └── config.go
├── go.mod
├── go.sum
└── docs/plans/
```

## MVP Scope

- View chat list with unread counts
- Select a chat and view message history
- Send and receive text messages
- Interactive TUI authentication flow
- Persistent session (no re-auth on restart)
