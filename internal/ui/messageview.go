package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/glamour"
	"charm.land/lipgloss/v2"

	"github.com/danhigham/telecharm/internal/domain"
)

// MessageViewModel displays messages using a viewport and glamour for markdown.
type MessageViewModel struct {
	viewport   viewport.Model
	renderer   *glamour.TermRenderer
	focused    bool
	width      int
	height     int
	typingUser string
	messages   []domain.Message
	loading    bool // true while fetching older history
	hasMore    bool // false when history is exhausted
}

func NewMessageViewModel() MessageViewModel {
	vp := viewport.New()
	return MessageViewModel{viewport: vp}
}

func (m MessageViewModel) Update(msg tea.Msg) (MessageViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j":
			m.viewport.ScrollDown(1)
			return m, nil
		case "k":
			m.viewport.ScrollUp(1)
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
	if m.viewport.YOffset() == 0 && !m.loading && m.hasMore && len(m.messages) > 0 {
		chatID := m.messages[0].ChatID
		return func() tea.Msg {
			return LoadOlderHistoryMsg{ChatID: chatID}
		}
	}
	return nil
}

func (m MessageViewModel) View() string {
	contentH := m.height - 2
	if contentH < 0 {
		contentH = 0
	}

	content := truncateHeight(m.viewport.View(), contentH)

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(m.width).
		Height(m.height)
	style = applyBorderColor(style, m.focused)

	return style.Render(content)
}

func (m MessageViewModel) SetSize(w, h int) MessageViewModel {
	m.width = w
	m.height = h
	// Viewport inner: subtract border (2)
	vpW := w - 2
	vpH := h - 2
	if vpW < 1 {
		vpW = 1
	}
	if vpH < 1 {
		vpH = 1
	}
	m.viewport.SetWidth(vpW)
	m.viewport.SetHeight(vpH)
	m = m.recreateRenderer()
	m = m.renderContent()
	return m
}

func (m MessageViewModel) SetFocused(f bool) MessageViewModel {
	m.focused = f
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

func (m MessageViewModel) recreateRenderer() MessageViewModel {
	wordWrap := m.viewport.Width() - 2
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
		multiLine := strings.Contains(text, "\n")
		if msg.HasMarkdown {
			rendered := m.renderMessageText(text)
			fmt.Fprintf(&b, "%s %s\n%s\n", ts, name, rendered)
			b.WriteString("\n")
		} else if multiLine {
			fmt.Fprintf(&b, "%s %s\n%s\n", ts, name, text)
			b.WriteString("\n")
		} else {
			fmt.Fprintf(&b, "%s %s %s\n", ts, name, text)
		}
	}

	if m.typingUser != "" {
		b.WriteString("\n")
		b.WriteString(typingStyle.Render(fmt.Sprintf("%s is typing...", m.typingUser)))
	}

	// Wrap content to viewport width so long lines don't overflow
	wrapped := lipgloss.NewStyle().Width(m.viewport.Width()).Render(b.String())
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

	// Glamour collapses single newlines (standard markdown paragraph
	// continuation). To preserve line breaks from Telegram while still
	// supporting multi-line markdown constructs (tables, code blocks),
	// split text into blocks by blank lines. Blocks that are multi-line
	// markdown (tables, fenced code) are rendered as a whole; regular
	// text blocks are rendered line-by-line to preserve line breaks.
	blocks := strings.Split(text, "\n\n")
	renderedBlocks := make([]string, len(blocks))

	for i, block := range blocks {
		if block == "" {
			renderedBlocks[i] = ""
			continue
		}

		if isMultiLineMarkdown(block) {
			r := m.renderBlock(block)
			renderedBlocks[i] = r
		} else {
			// Render each line individually to preserve line breaks.
			lines := strings.Split(block, "\n")
			renderedLines := make([]string, len(lines))
			for j, line := range lines {
				if line == "" {
					renderedLines[j] = ""
				} else {
					renderedLines[j] = m.renderBlock(line)
				}
			}
			renderedBlocks[i] = strings.Join(renderedLines, "\n")
		}
	}

	return strings.Join(renderedBlocks, "\n")
}

// renderBlock renders a single text block through glamour, trimming whitespace.
func (m MessageViewModel) renderBlock(text string) string {
	r, err := m.renderer.Render(text)
	if err != nil {
		return text
	}
	r = strings.TrimRight(r, "\n ")
	r = strings.TrimLeft(r, "\n")
	return r
}

// isMultiLineMarkdown returns true if the block is a multi-line markdown
// construct that must be rendered as a whole (tables, fenced code blocks).
func isMultiLineMarkdown(block string) bool {
	if !strings.Contains(block, "\n") {
		return false
	}
	trimmed := strings.TrimSpace(block)
	// Fenced code blocks.
	if strings.HasPrefix(trimmed, "```") {
		return true
	}
	// Tables: all lines contain pipes.
	lines := strings.Split(trimmed, "\n")
	allPipes := true
	for _, line := range lines {
		if !strings.Contains(line, "|") {
			allPipes = false
			break
		}
	}
	return allPipes
}
