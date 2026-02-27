# Speech Bubble Message Rendering

## Overview

Replace the current flat `timestamp name: text` message format with chat-app-style speech bubbles. Sent and received messages are visually distinguished by alignment, tail direction, and border color.

## Layout

- **Received messages:** left-aligned bubble, tail pointing left below bottom-left
- **Sent messages:** right-aligned bubble, tail pointing right below bottom-right
- **Timestamp:** displayed above each bubble in dim gray
- **No name labels** — alignment and color distinguish the sender
- **Bubble max width:** 75% of the pane width
- **1 char padding** inside the bubble on left and right

## Visual

```
   10:42
   ╭───────────────────────╮
   │ Hey, how are you?     │
   ╰──┬────────────────────╯
   ◀──╯

                        10:43
        ╭───────────────────────╮
        │ Doing great!          │
        ╰────────────────────┬──╯
                             ╰─▶
```

## Colors

- Received messages: purple border (`99`)
- Sent messages: deeper purple border (`63`)
- Timestamps: dim gray (existing style)

## Implementation

All changes in `internal/ui/messageview.go`:
- Replace message formatting in `renderContentInner` with a `renderBubble` function
- Standard `lipgloss.RoundedBorder()` for the bubble box with 1 char horizontal padding
- Tail rendered as a separate line below the bubble
  - Left tail: `◀──╯` with `┬` in the bottom border
  - Right tail: `╰─▶` with `┬` in the bottom border
- Sent bubbles right-aligned using `lipgloss.Place` with the pane width
- Markdown content still rendered through glamour inside bubbles
- Day separators remain unchanged
