package ui

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"

	"github.com/danhigham/tg-tui/internal/domain"
)

type MessageView struct {
	*tview.TextView
}

func NewMessageView() *MessageView {
	mv := &MessageView{
		TextView: tview.NewTextView(),
	}
	mv.SetDynamicColors(true)
	mv.SetScrollable(true)
	mv.SetWordWrap(true)
	mv.SetBorder(true).SetTitle(" Messages ")
	mv.ScrollToEnd()
	return mv
}

func (mv *MessageView) SetChatTitle(title string) {
	mv.SetTitle(fmt.Sprintf(" %s ", title))
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

		fmt.Fprintf(&b, "[gray]%s [%s]%s:[white] %s\n",
			ts, nameColor, tview.Escape(msg.SenderName), tview.Escape(msg.Text))
	}

	mv.SetText(b.String())
	mv.ScrollToEnd()
}
