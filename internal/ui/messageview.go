package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/danhigham/tg-tui/internal/domain"
)

type MessageView struct {
	*tview.TextView
	typingUser string
}

func NewMessageView() *MessageView {
	mv := &MessageView{
		TextView: tview.NewTextView(),
	}
	mv.SetDynamicColors(true)
	mv.SetScrollable(true)
	mv.SetWordWrap(true)
	mv.SetBorder(true).SetTitle(" Messages ")
	mv.SetBackgroundColor(tcell.ColorDefault)
	mv.ScrollToEnd()
	return mv
}

func (mv *MessageView) SetChatTitle(title string) {
	mv.SetTitle(fmt.Sprintf(" %s ", title))
}

func (mv *MessageView) SetTypingUser(name string) {
	mv.typingUser = name
}

func (mv *MessageView) Update(messages []domain.Message) {
	var b strings.Builder

	for _, msg := range messages {
		ts := msg.Timestamp.Format("15:04")

		var nameColor string
		if msg.Out {
			nameColor = "green"
		} else {
			nameColor = "blue"
		}

		text := formatMessageText(msg.Text)

		fmt.Fprintf(&b, "[gray]%s [%s]%s:[white] %s\n",
			ts, nameColor, tview.Escape(msg.SenderName), text)
	}

	if mv.typingUser != "" {
		fmt.Fprintf(&b, "\n[::i]%s is typing...[::-]", tview.Escape(mv.typingUser))
	}

	mv.SetText(b.String())
	mv.ScrollToEnd()
}

// formatMessageText detects markdown tables in the text and reformats them
// with uniform column widths. Non-table text is escaped and returned as-is.
func formatMessageText(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	i := 0

	for i < len(lines) {
		// Try to detect a table starting at this line.
		tableEnd := detectTable(lines, i)
		if tableEnd > i {
			formatted := formatTable(lines[i:tableEnd])
			result = append(result, formatted...)
			i = tableEnd
		} else {
			result = append(result, tview.Escape(lines[i]))
			i++
		}
	}

	return strings.Join(result, "\n")
}

// detectTable checks if lines starting at idx form a markdown table.
// Returns the end index (exclusive) of the table, or idx if no table found.
func detectTable(lines []string, idx int) int {
	if idx+2 >= len(lines) {
		return idx
	}

	// First row must have pipes.
	if !isTableRow(lines[idx]) {
		return idx
	}

	// Second row must be a separator (e.g., |---|---|).
	if !isSeparatorRow(lines[idx+1]) {
		return idx
	}

	// Consume all subsequent table rows.
	end := idx + 2
	for end < len(lines) && isTableRow(lines[end]) {
		end++
	}

	return end
}

func isTableRow(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.Contains(trimmed, "|") && len(splitTableCells(trimmed)) >= 2
}

func isSeparatorRow(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.Contains(trimmed, "|") {
		return false
	}
	cells := splitTableCells(trimmed)
	for _, cell := range cells {
		cleaned := strings.TrimSpace(cell)
		cleaned = strings.Trim(cleaned, ":")
		if len(cleaned) == 0 {
			continue
		}
		for _, ch := range cleaned {
			if ch != '-' {
				return false
			}
		}
	}
	return true
}

func splitTableCells(line string) []string {
	// Strip leading/trailing pipes.
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "|") {
		trimmed = trimmed[1:]
	}
	if strings.HasSuffix(trimmed, "|") {
		trimmed = trimmed[:len(trimmed)-1]
	}
	return strings.Split(trimmed, "|")
}

// formatTable reformats a set of markdown table lines with uniform column widths.
func formatTable(lines []string) []string {
	// Parse all rows into cells.
	var rows [][]string
	var separatorIdx int = -1
	for i, line := range lines {
		if isSeparatorRow(line) {
			separatorIdx = i
			continue
		}
		cells := splitTableCells(line)
		trimmed := make([]string, len(cells))
		for j, c := range cells {
			trimmed[j] = strings.TrimSpace(c)
		}
		rows = append(rows, trimmed)
	}

	if len(rows) == 0 {
		// Fallback: return escaped lines.
		result := make([]string, len(lines))
		for i, l := range lines {
			result[i] = tview.Escape(l)
		}
		return result
	}

	// Determine the maximum number of columns.
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	// Calculate max width per column.
	colWidths := make([]int, maxCols)
	for _, row := range rows {
		for j := 0; j < len(row) && j < maxCols; j++ {
			w := utf8.RuneCountInString(row[j])
			if w > colWidths[j] {
				colWidths[j] = w
			}
		}
	}

	// Render rows.
	var result []string
	for i, row := range rows {
		var b strings.Builder
		b.WriteString("  ") // indent table slightly
		for j := 0; j < maxCols; j++ {
			cell := ""
			if j < len(row) {
				cell = row[j]
			}
			padded := cell + strings.Repeat(" ", colWidths[j]-utf8.RuneCountInString(cell))
			if i == 0 && separatorIdx >= 0 {
				// Header row: bold.
				b.WriteString("[::b]")
				b.WriteString(tview.Escape(padded))
				b.WriteString("[::-]")
			} else {
				b.WriteString(tview.Escape(padded))
			}
			if j < maxCols-1 {
				b.WriteString(" │ ")
			}
		}
		result = append(result, b.String())
	}

	// Add a separator line after the header.
	if separatorIdx >= 0 && len(rows) > 0 {
		var sep strings.Builder
		sep.WriteString("  ")
		for j := 0; j < maxCols; j++ {
			sep.WriteString(strings.Repeat("─", colWidths[j]))
			if j < maxCols-1 {
				sep.WriteString("─┼─")
			}
		}
		// Insert separator after first row.
		newResult := make([]string, 0, len(result)+1)
		newResult = append(newResult, result[0])
		newResult = append(newResult, sep.String())
		newResult = append(newResult, result[1:]...)
		result = newResult
	}

	return result
}
