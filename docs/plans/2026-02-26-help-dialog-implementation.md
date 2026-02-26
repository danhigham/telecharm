# Help Dialog Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a toggleable help dialog overlay that lists all keyboard shortcuts, triggered by `h` (context-aware) or `F1`.

**Architecture:** New `HelpModel` in `internal/ui/help.go` following the existing `SplashModel` layer-overlay pattern. Composited on top of the main UI using lipgloss layers. Integrated into the root `Model` in `app.go` with priority key handling.

**Tech Stack:** Go, Bubble Tea v2, Lipgloss v2

---

### Task 1: Create `HelpModel` struct and constructor

**Files:**
- Create: `internal/ui/help.go`

**Step 1: Create the help model file**

```go
package ui

import "charm.land/lipgloss/v2"

// HelpModel renders a centered help overlay listing keyboard shortcuts.
type HelpModel struct {
	visible       bool
	width, height int
}

// NewHelpModel creates a hidden help model.
func NewHelpModel() HelpModel {
	return HelpModel{}
}

// IsVisible reports whether the help overlay is showing.
func (h HelpModel) IsVisible() bool {
	return h.visible
}

// Toggle flips the help overlay visibility.
func (h HelpModel) Toggle() HelpModel {
	h.visible = !h.visible
	return h
}

// SetSize updates the terminal dimensions for centering.
func (h HelpModel) SetSize(w, ht int) HelpModel {
	h.width = w
	h.height = ht
	return h
}

const helpText = ` Keyboard Shortcuts

 General
   Ctrl+C        Quit
   h / F1        Toggle this help
   Tab           Next pane
   Shift+Tab     Previous pane
   Esc           Back to chat list
   Ctrl+B        Toggle sidebar
   [ / ]         Resize sidebar

 Chat List
   j/k / ↑/↓     Navigate chats
   Enter         Select chat
   /             Filter chats

 Messages
   j / k         Scroll down / up
   PgUp / PgDn   Page scroll

 Input
   Enter         Send message

 Press h, F1, or Esc to close`

// View renders the help box (without full-screen placement).
// Use BoxOffset to get the X/Y for centering via the Layer API.
func (h HelpModel) View() string {
	if !h.visible || h.width == 0 || h.height == 0 {
		return ""
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 3).
		BorderForegroundBlend(rainbowBlend...)

	return style.Render(helpText)
}

// BoxOffset returns the (x, y) needed to center the help box
// within the terminal dimensions.
func (h HelpModel) BoxOffset() (int, int) {
	box := h.View()
	bw := lipgloss.Width(box)
	bh := lipgloss.Height(box)
	x := (h.width - bw) / 2
	y := (h.height - bh) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}
```

**Step 2: Verify it compiles**

Run: `cd /home/danhigham/workspace/telecharm && go build ./internal/ui/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/ui/help.go
git commit -m "feat: add HelpModel for keyboard shortcuts overlay"
```

---

### Task 2: Integrate `HelpModel` into root `Model`

**Files:**
- Modify: `internal/ui/app.go`

**Step 1: Add `help` field to `Model` struct**

In `app.go:35-52`, add `help HelpModel` after `splash SplashModel`:

```go
type Model struct {
	chatList    ChatListModel
	messageView MessageViewModel
	input       InputModel
	auth        AuthModel
	status      statusModel
	splash      SplashModel
	help        HelpModel       // <-- add this line

	// ... rest unchanged
}
```

**Step 2: Initialize `help` in `NewModel`**

In `app.go:56-69`, add `help: NewHelpModel(),` in the struct literal:

```go
m := Model{
	chatList:    NewChatListModel(),
	messageView: NewMessageViewModel(),
	input:       NewInputModel(),
	auth:        NewAuthModel(),
	status:      newStatusModel(),
	splash:      NewSplashModel(),
	help:        NewHelpModel(),        // <-- add this line
	// ... rest unchanged
}
```

**Step 3: Pass size to help in `distributeSize`**

In `app.go:380-418`, add `m.help = m.help.SetSize(m.width, m.height)` after the splash line (line 415):

```go
m.splash = m.splash.SetSize(m.width, m.height)
m.help = m.help.SetSize(m.width, m.height)
```

**Step 4: Verify it compiles**

Run: `cd /home/danhigham/workspace/telecharm && go build ./...`
Expected: No errors

**Step 5: Commit**

```bash
git add internal/ui/app.go
git commit -m "feat: wire HelpModel into root Model"
```

---

### Task 3: Add key handling for help toggle

**Files:**
- Modify: `internal/ui/app.go`

**Step 1: Add help visibility check in `Update` after auth check**

In `app.go`, after the auth visibility block (lines 230-234), add:

```go
if m.help.IsVisible() {
	switch msg.String() {
	case "h", "f1", "esc", "q":
		m.help = m.help.Toggle()
		return m, nil
	}
	// Swallow all other keys while help is visible
	return m, nil
}
```

**Step 2: Add help toggle keys in the global switch**

In `app.go`, in the `switch msg.String()` block (line 236-298), add a new case before the existing `"q"` case:

```go
case "f1":
	m.help = m.help.Toggle()
	return m, nil
```

And modify the existing `"q"` case to also handle `"h"`:

The `"q"` case currently (lines 239-242):
```go
case "q":
	if m.focus != focusInput {
		return m, tea.Quit
	}
```

Change to:
```go
case "q":
	if m.focus != focusInput {
		return m, tea.Quit
	}
case "h":
	if m.focus != focusInput {
		m.help = m.help.Toggle()
		return m, nil
	}
```

**Step 3: Verify it compiles**

Run: `cd /home/danhigham/workspace/telecharm && go build ./...`
Expected: No errors

**Step 4: Commit**

```bash
git add internal/ui/app.go
git commit -m "feat: add keybindings to toggle help overlay"
```

---

### Task 4: Add help overlay rendering in `View`

**Files:**
- Modify: `internal/ui/app.go`

**Step 1: Add help layer compositing in `View`**

In `app.go`, in the `View()` method (lines 320-367), modify the section that handles splash overlay. The current code at lines 357-365:

```go
if m.splash.IsVisible() {
	x, y := m.splash.BoxOffset()
	bg := lipgloss.NewLayer(mainContent)
	fg := lipgloss.NewLayer(m.splash.View()).X(x).Y(y).Z(1)
	comp := lipgloss.NewCompositor(bg, fg)
	v.SetContent(comp.Render())
} else {
	v.SetContent(mainContent)
}
```

Replace with:

```go
if m.splash.IsVisible() {
	x, y := m.splash.BoxOffset()
	bg := lipgloss.NewLayer(mainContent)
	fg := lipgloss.NewLayer(m.splash.View()).X(x).Y(y).Z(1)
	comp := lipgloss.NewCompositor(bg, fg)
	v.SetContent(comp.Render())
} else if m.help.IsVisible() {
	x, y := m.help.BoxOffset()
	bg := lipgloss.NewLayer(mainContent)
	fg := lipgloss.NewLayer(m.help.View()).X(x).Y(y).Z(1)
	comp := lipgloss.NewCompositor(bg, fg)
	v.SetContent(comp.Render())
} else {
	v.SetContent(mainContent)
}
```

**Step 2: Verify it compiles**

Run: `cd /home/danhigham/workspace/telecharm && go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/ui/app.go
git commit -m "feat: render help overlay as composited layer"
```

---

### Task 5: Manual smoke test

**Step 1: Build and run**

Run: `cd /home/danhigham/workspace/telecharm && go build -o telecharm ./cmd/telecharm && ./telecharm`

**Step 2: Verify behavior**

- Press `F1` → help dialog appears centered with rainbow border
- Press `F1` again → help dialog disappears
- Press `h` (with chat list focused) → help dialog appears
- Press `Esc` → help dialog disappears
- Press `h` → help dialog appears
- Press `q` → help dialog disappears (does not quit app)
- Focus input box (Tab), type `h` → no help dialog (key goes to input)
- Focus input box, press `F1` → help dialog appears

**Step 3: Final commit if any fixes needed**
