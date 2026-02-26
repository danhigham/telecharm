# Help Dialog Design

## Overview

Add a toggleable help dialog overlay that displays all keyboard shortcuts. Triggered by `h` (context-aware — only when focus is not on the text input) or `F1` (always works). Dismissed by `h`, `F1`, `Esc`, or `q`.

## Approach

**Layer overlay** — follows the existing `SplashModel` pattern. The help dialog is composited as a centered lipgloss layer on top of the main UI. The main content remains visible underneath.

## Component: `HelpModel`

New file: `internal/ui/help.go`

```go
type HelpModel struct {
    visible bool
    width   int
    height  int
}
```

Methods:
- `IsVisible() bool`
- `Toggle()`
- `SetSize(w, h int)`
- `View() string` — renders centered box with keybinding table

### Visual Style

- Rounded border with `rainbowBlend` foreground gradient (matching splash/auth)
- Box sized to fit content (~55 chars wide, ~22 rows tall)
- Centered via `lipgloss.Place`

### Keybinding Table Content

```
 Keyboard Shortcuts
 ──────────────────────────────────
 General
   Ctrl+C       Quit
   h / F1       Toggle this help
   Tab          Next pane
   Shift+Tab    Previous pane
   Esc          Back to chat list
   Ctrl+B       Toggle sidebar
   [ / ]        Resize sidebar

 Chat List
   j/k / ↑/↓    Navigate chats
   Enter        Select chat
   /            Filter chats

 Messages
   j / k        Scroll down / up
   PgUp/PgDn    Page scroll

 Input
   Enter        Send message
```

Footer: `Press h, F1, or Esc to close`

## Integration in `app.go`

1. Add `help HelpModel` field to root `Model`
2. In `Update()`:
   - Check `m.help.IsVisible()` after splash/auth checks
   - When visible: `h`, `F1`, `esc`, `q` dismiss it; all other keys swallowed
   - When not visible: `h` toggles on (only if focus != input), `F1` toggles on (always)
3. In `View()`:
   - When `help.IsVisible()`, composite as a layer on top using `lipgloss.NewLayer`/`NewCompositor` (same as splash pattern)
4. In `SetSize()`: pass dimensions to `help.SetSize()`
