package ui

import (
	"fmt"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/glamour"

	"github.com/danhigham/telecharm/internal/domain"
)

// ansiTrailingSpaces matches ANSI-styled trailing spaces that glamour adds
// as padding. These look like real content to lipgloss width measurement
// but are visually just whitespace.
var ansiTrailingSpaces = regexp.MustCompile(`(\x1b\[[0-9;]*m[ ]*)+\s*$`)

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
		case "pgup":
			m.viewport.PageUp()
			return m, m.checkScrollTop()
		case "pgdown":
			m.viewport.PageDown()
			return m, nil
		}
	case tea.MouseWheelMsg:
		e := msg.Mouse()
		switch e.Button {
		case tea.MouseWheelUp:
			m.viewport.ScrollUp(3)
			return m, m.checkScrollTop()
		case tea.MouseWheelDown:
			m.viewport.ScrollDown(3)
			return m, nil
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

func (m MessageViewModel) bubbleWidth() int {
	w := m.viewport.Width() * 3 / 4
	if w < 20 {
		w = 20
	}
	return w
}

func (m MessageViewModel) recreateRenderer() MessageViewModel {
	// Word wrap fits inside bubble: bubble width minus border (2) and padding (2).
	wordWrap := m.bubbleWidth() - 4
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

		text := msg.Text
		if msg.HasMarkdown {
			text = m.renderMessageText(text)
		}

		bubble := m.renderBubble(text, msg.Out)
		ts := timeStyle.Render(msg.Timestamp.Format("15:04"))

		if msg.Out {
			// Right-align timestamp and bubble.
			tsLine := lipgloss.NewStyle().Width(m.viewport.Width()).Align(lipgloss.Right).Render(ts)
			bubbleLine := lipgloss.NewStyle().Width(m.viewport.Width()).Align(lipgloss.Right).Render(bubble)
			b.WriteString(tsLine + "\n" + bubbleLine + "\n")
		} else {
			// Left-align timestamp and bubble.
			b.WriteString(ts + "\n" + bubble + "\n")
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

		if isTable(block) {
			renderedBlocks[i] = renderTable(block)
		} else if isMultiLineMarkdown(block) {
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
	// Tables (also handled by isTable, but keep for isMultiLineMarkdown callers).
	return isTable(block)
}

// isTable returns true if the block looks like a markdown table.
func isTable(block string) bool {
	if !strings.Contains(block, "\n") {
		return false
	}
	trimmed := strings.TrimSpace(block)
	lines := strings.Split(trimmed, "\n")
	if len(lines) < 3 {
		return false
	}
	for _, line := range lines {
		if !strings.Contains(line, "|") {
			return false
		}
	}
	// Check that line 2 is a separator row (contains dashes).
	return isSeparatorRow(lines[1])
}

// isSeparatorRow returns true if the line is a markdown table separator (e.g. "| --- | --- |").
func isSeparatorRow(line string) bool {
	cells := splitTableRow(line)
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		cleaned := strings.TrimSpace(cell)
		cleaned = strings.Trim(cleaned, ":")
		if len(cleaned) == 0 || strings.Trim(cleaned, "-") != "" {
			return false
		}
	}
	return true
}

// splitTableRow splits a markdown table row by pipes and trims leading/trailing empty cells.
func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	cells := make([]string, len(parts))
	for i, p := range parts {
		cells[i] = strings.TrimSpace(p)
	}
	return cells
}

var (
	receivedBubbleColor = lipgloss.Color("99")
	sentBubbleColor     = lipgloss.Color("63")
)

// renderBubble wraps text in a speech bubble with a tail.
func (m MessageViewModel) renderBubble(text string, sent bool) string {
	maxW := m.bubbleWidth()
	borderColor := receivedBubbleColor
	if sent {
		borderColor = sentBubbleColor
	}

	sc := func(ch string) string {
		return lipgloss.NewStyle().Foreground(borderColor).Render(ch)
	}

	text = strings.TrimRight(text, "\n ")

	// Strip ANSI-styled trailing spaces from each line so lipgloss can
	// measure the true content width and auto-size the bubble.
	textLines := strings.Split(text, "\n")
	for i, line := range textLines {
		textLines[i] = ansiTrailingSpaces.ReplaceAllString(line, "\x1b[0m")
	}
	text = strings.Join(textLines, "\n")

	// Render the full box with all borders, then replace the bottom line
	// with our custom one that includes the tail connector.
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		MaxWidth(maxW).
		Padding(0, 1)

	box := boxStyle.Render(text)

	// Measure actual rendered width and remove the last line (bottom border).
	boxLines := strings.Split(box, "\n")
	actualW := 0
	for _, line := range boxLines {
		if w := lipgloss.Width(line); w > actualW {
			actualW = w
		}
	}
	if len(boxLines) > 1 {
		boxLines = boxLines[:len(boxLines)-1]
	}
	box = strings.Join(boxLines, "\n")

	if actualW < 4 {
		actualW = 4
	}

	// Inner dash count = total width minus the two corner chars.
	inner := actualW - 2

	var bottomLine string
	var tailLine string

	if sent {
		// Tail on right: ╰───────┬──╯, then ╰─▶ below.
		rightSegment := 2
		leftSegment := inner - rightSegment - 1 // -1 for the ┬
		if leftSegment < 0 {
			leftSegment = 0
		}
		bottomLine = sc("╰" + strings.Repeat("─", leftSegment) + "┬" + strings.Repeat("─", rightSegment) + "╯")
		// ┬ is at visual position (1 + leftSegment) from the left edge.
		tailLine = strings.Repeat(" ", 1+leftSegment) + sc("╰─▶")
	} else {
		// Tail on left: ╰──┬───────╯, then ◀──╯ below.
		leftSegment := 2
		rightSegment := inner - leftSegment - 1 // -1 for the ┬
		if rightSegment < 0 {
			rightSegment = 0
		}
		bottomLine = sc("╰" + strings.Repeat("─", leftSegment) + "┬" + strings.Repeat("─", rightSegment) + "╯")
		tailLine = sc("◀──╯")
	}

	return box + "\n" + bottomLine + "\n" + tailLine
}

var (
	tablePurple    = lipgloss.Color("99")
	tableGray      = lipgloss.Color("245")
	tableLightGray = lipgloss.Color("241")

	tableHeaderStyle  = lipgloss.NewStyle().Foreground(tablePurple).Bold(true).Align(lipgloss.Center)
	tableCellStyle    = lipgloss.NewStyle().Padding(0, 1)
	tableOddRowStyle  = tableCellStyle.Foreground(tableGray)
	tableEvenRowStyle = tableCellStyle.Foreground(tableLightGray)
)

// renderTable parses a markdown table block and renders it using lipgloss table.
func renderTable(block string) string {
	trimmed := strings.TrimSpace(block)
	lines := strings.Split(trimmed, "\n")

	headers := splitTableRow(lines[0])
	// lines[1] is the separator row — skip it.
	var rows [][]string
	for _, line := range lines[2:] {
		rows = append(rows, splitTableRow(line))
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(tablePurple)).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return tableHeaderStyle
			case row%2 == 0:
				return tableEvenRowStyle
			default:
				return tableOddRowStyle
			}
		}).
		Headers(headers...)

	for _, row := range rows {
		t.Row(row...)
	}

	// Indent the table by two spaces.
	rendered := t.String()
	outLines := strings.Split(rendered, "\n")
	for i, line := range outLines {
		if line != "" {
			outLines[i] = "  " + line
		}
	}
	return strings.Join(outLines, "\n")
}
