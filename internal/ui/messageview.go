package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/danhigham/tg-tui/internal/domain"
)

// MessageViewModel displays messages using a viewport and glamour for markdown.
type MessageViewModel struct {
	viewport    viewport.Model
	renderer    *glamour.TermRenderer
	focused     bool
	width       int
	height      int
	chatTitle   string
	typingUser  string
	messages    []domain.Message
	statusPill  string // rendered status pill to show in header
	loading     bool   // true while fetching older history
	hasMore     bool   // false when history is exhausted
}

func NewMessageViewModel() MessageViewModel {
	vp := viewport.New(0, 0)
	return MessageViewModel{viewport: vp}
}

func (m MessageViewModel) Update(msg tea.Msg) (MessageViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j":
			m.viewport.LineDown(1)
			return m, nil
		case "k":
			m.viewport.LineUp(1)
			return m, m.checkScrollTop()
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)

	var cmds []tea.Cmd
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	if scrollCmd := m.checkScrollTop(); scrollCmd != nil {
		cmds = append(cmds, scrollCmd)
	}
	return m, tea.Batch(cmds...)
}

// checkScrollTop returns a command to load older history if scrolled to top.
func (m MessageViewModel) checkScrollTop() tea.Cmd {
	if m.viewport.YOffset == 0 && !m.loading && m.hasMore && len(m.messages) > 0 {
		chatID := m.messages[0].ChatID
		return func() tea.Msg {
			return LoadOlderHistoryMsg{ChatID: chatID}
		}
	}
	return nil
}

func (m MessageViewModel) View() string {
	innerW := m.width - 2
	innerH := m.height - 2
	if innerW < 0 {
		innerW = 0
	}
	if innerH < 0 {
		innerH = 0
	}

	title := " Messages "
	if m.chatTitle != "" {
		title = fmt.Sprintf(" %s ", m.chatTitle)
	}

	// Header: title on left, status pill on right
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(borderColor(m.focused))
	titleStr := headerStyle.Render(title)

	header := titleStr
	if m.statusPill != "" {
		gap := innerW - lipgloss.Width(titleStr) - lipgloss.Width(m.statusPill)
		if gap < 1 {
			gap = 1
		}
		header = titleStr + strings.Repeat(" ", gap) + m.statusPill
	}

	content := header + "\n" + m.viewport.View()

	// Truncate content to innerH lines to prevent border overflow
	content = truncateHeight(content, innerH)

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor(m.focused)).
		Width(innerW).
		Height(innerH)

	return style.Render(content)
}

func (m MessageViewModel) SetSize(w, h int) MessageViewModel {
	m.width = w
	m.height = h
	// Viewport inner: border (2) + header line (1)
	vpW := w - 2
	vpH := h - 2 - 1
	if vpW < 1 {
		vpW = 1
	}
	if vpH < 1 {
		vpH = 1
	}
	m.viewport.Width = vpW
	m.viewport.Height = vpH
	m = m.recreateRenderer()
	m = m.renderContent()
	return m
}

func (m MessageViewModel) SetFocused(f bool) MessageViewModel {
	m.focused = f
	return m
}

func (m MessageViewModel) SetChatTitle(title string) MessageViewModel {
	m.chatTitle = title
	return m
}

func (m MessageViewModel) SetTypingUser(name string) MessageViewModel {
	m.typingUser = name
	return m
}

func (m MessageViewModel) SetMessages(msgs []domain.Message) MessageViewModel {
	m.messages = msgs
	m.hasMore = true
	m.loading = false
	m = m.renderContent()
	return m
}

// PrependMessages adds older messages to the top and preserves scroll position.
func (m MessageViewModel) PrependMessages(msgs []domain.Message) MessageViewModel {
	m.loading = false
	m.hasMore = len(msgs) > 0

	if len(msgs) == 0 {
		return m
	}

	// Remember old content height.
	oldTotalLines := m.viewport.TotalLineCount()

	m.messages = append(msgs, m.messages...)
	m = m.renderContentNoScroll()

	// Calculate how many new lines were added and adjust offset.
	newTotalLines := m.viewport.TotalLineCount()
	delta := newTotalLines - oldTotalLines
	if delta < 0 {
		delta = 0
	}
	m.viewport.SetYOffset(delta)
	return m
}

// SetLoading marks the view as loading older history.
func (m MessageViewModel) SetLoading(v bool) MessageViewModel {
	m.loading = v
	return m
}

func (m MessageViewModel) SetStatusPill(pill string) MessageViewModel {
	m.statusPill = pill
	return m
}

func (m MessageViewModel) recreateRenderer() MessageViewModel {
	wordWrap := m.viewport.Width - 2
	if wordWrap < 10 {
		wordWrap = 10
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(wordWrap),
	)
	if err == nil {
		m.renderer = r
	}
	return m
}

func (m MessageViewModel) renderContentNoScroll() MessageViewModel {
	return m.renderContentInner(false)
}

func (m MessageViewModel) renderContent() MessageViewModel {
	return m.renderContentInner(true)
}

func (m MessageViewModel) renderContentInner(gotoBottom bool) MessageViewModel {
	var b strings.Builder
	var currentDate string

	for _, msg := range m.messages {
		msgDate := msg.Timestamp.Format("January 2, 2006")
		if msgDate != currentDate {
			if currentDate != "" {
				b.WriteString("\n")
			}
			sep := daySeparatorStyle.Render(fmt.Sprintf("───── %s ─────", msgDate))
			b.WriteString(sep + "\n")
			currentDate = msgDate
		}

		ts := timeStyle.Render(msg.Timestamp.Format("15:04"))

		var name string
		if msg.Out {
			name = outNameStyle.Render(msg.SenderName + ":")
		} else {
			name = inNameStyle.Render(msg.SenderName + ":")
		}

		text := msg.Text
		isMarkdown := containsMarkdown(text)
		multiLine := strings.Contains(text, "\n")
		if isMarkdown {
			rendered := m.renderMessageText(text)
			fmt.Fprintf(&b, "%s %s\n%s\n", ts, name, rendered)
		} else if multiLine {
			fmt.Fprintf(&b, "%s %s\n%s\n", ts, name, text)
		} else {
			fmt.Fprintf(&b, "%s %s %s\n", ts, name, text)
		}
		if multiLine || isMarkdown {
			b.WriteString("\n")
		}
	}

	if m.typingUser != "" {
		b.WriteString("\n")
		b.WriteString(typingStyle.Render(fmt.Sprintf("%s is typing...", m.typingUser)))
	}

	// Wrap content to viewport width so long lines don't overflow
	wrapped := lipgloss.NewStyle().Width(m.viewport.Width).Render(b.String())
	m.viewport.SetContent(wrapped)
	if gotoBottom {
		m.viewport.GotoBottom()
	}
	return m
}

func (m MessageViewModel) renderMessageText(text string) string {
	if m.renderer == nil {
		return text
	}

	if containsMarkdown(text) {
		rendered, err := m.renderer.Render(text)
		if err == nil {
			rendered = strings.TrimRight(rendered, "\n ")
			return rendered
		}
	}

	return text
}

func containsMarkdown(text string) bool {
	if strings.HasPrefix(text, "#") {
		return true
	}
	if strings.Contains(text, "\n#") {
		return true
	}
	if strings.Contains(text, "|") && strings.Contains(text, "---") {
		return true
	}
	if strings.Contains(text, "```") {
		return true
	}
	if strings.Contains(text, "**") || strings.Contains(text, "__") {
		return true
	}
	if strings.Contains(text, "* ") || strings.Contains(text, "- ") {
		return true
	}
	if strings.Contains(text, "](") {
		return true
	}
	return false
}
