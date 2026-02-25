# telecharm

A terminal-based Telegram client built with Go, using the [Charm](https://charm.sh) ecosystem for the UI and [gotd/td](https://github.com/gotd/td) for the Telegram MTProto protocol.

```
┌─────────────────────────────────────────────────┐
│ CONNECTED  General Chat     Dan Higham  05:42   │
├──────────────┬──────────────────────────────────┤
│ > Alice      │ 10:30 Alice: Hey, how's it going │
│   Bob        │ 10:31 Bob: Not bad, you?         │
│   Work Chat  │ 10:32 Alice: **Great!**          │
│              │                                  │
│              ├──────────────────────────────────┤
│              │ > Type a message...              │
└──────────────┴──────────────────────────────────┘
```

## Features

- Full chat list with unread counts and last message preview
- Markdown rendering in messages (tables, code blocks, bold, links, etc.)
- Typing indicators
- Infinite scroll to load older message history
- Resizable chat list / message pane split
- Rainbow gradient borders on the focused pane
- Full-width status bar with connection state, chat title, user name, and clock
- Splash screen with Telegram logo on startup
- Persistent sessions (authenticate once, stay logged in)
- Interactive authentication flow (phone, code, optional 2FA)

## Prerequisites

- Go 1.24 or later
- A Telegram account
- Telegram API credentials (see below)

## Getting API Credentials

1. Go to [https://my.telegram.org](https://my.telegram.org) and log in with your phone number
2. Navigate to **API development tools**
3. Create a new application (any name/description)
4. Note your **api_id** and **api_hash**

## Configuration

Create the config file at `~/.config/telecharm/config.yaml`:

```yaml
telegram:
  api_id: 12345
  api_hash: "your_api_hash_here"
log_level: info  # optional, defaults to "info"
```

The app stores its data in `~/.config/telecharm/`:

| File | Purpose |
|------|---------|
| `config.yaml` | API credentials and settings |
| `session.json` | Telegram session (auto-created after first login) |
| `telecharm.log` | Application logs |

## Installation

```bash
go install github.com/danhigham/telecharm/cmd/telecharm@latest
```

Or build from source:

```bash
git clone https://github.com/danhigham/telecharm.git
cd telecharm
go build -o telecharm ./cmd/telecharm
```

## Usage

```bash
telecharm
```

On first run, you'll be prompted to authenticate:

1. Enter your phone number (with country code, e.g. `+44...`)
2. Enter the verification code sent to your Telegram app
3. Enter your 2FA password if you have one enabled

After authenticating, your session is saved and you won't need to log in again.

## Keyboard Shortcuts

### Navigation

| Key | Action |
|-----|--------|
| `Tab` | Cycle focus: Chat List → Messages → Input |
| `Shift+Tab` | Reverse cycle focus |
| `Ctrl+B` | Toggle chat list sidebar |
| `Esc` | Return focus to Chat List |
| `Ctrl+C` | Quit |
| `q` | Quit (when not typing in input) |

### Chat List

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `Enter` | Select chat |
| `/` | Filter/search chats |

### Messages

| Key | Action |
|-----|--------|
| `j` | Scroll down |
| `k` | Scroll up |
| Scroll to top | Automatically loads older messages |

### Input

| Key | Action |
|-----|--------|
| `Enter` | Send message |

### Pane Resizing

| Key | Action |
|-----|--------|
| `[` | Shrink chat list (when not in input) |
| `]` | Expand chat list (when not in input) |

## Architecture

The app uses a three-layer event-driven architecture:

```
Telegram (gotd/td)  →  State Store  →  TUI (Bubble Tea)
   MTProto events      Thread-safe       Reactive rendering
                        store
```

- **Telegram layer** handles the MTProto connection, message sending/receiving, and authentication
- **State layer** is a thread-safe store that holds chat lists, messages, and typing indicators
- **UI layer** is a Bubble Tea application that reactively renders based on state changes

## License

MIT
