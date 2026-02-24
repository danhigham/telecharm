package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Input struct {
	*tview.InputField
	onSend func(text string)
}

func NewInput() *Input {
	inp := &Input{
		InputField: tview.NewInputField(),
	}
	inp.SetLabel("> ")
	inp.SetFieldWidth(0)
	inp.SetPlaceholder("Type a message...")
	inp.SetBorder(true).SetTitle(" Input ")
	inp.SetBackgroundColor(tcell.ColorDefault)
	inp.SetFieldBackgroundColor(tcell.ColorDefault)

	inp.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			text := inp.GetText()
			if text != "" && inp.onSend != nil {
				inp.onSend(text)
				inp.SetText("")
			}
		}
	})

	return inp
}

func (inp *Input) SetOnSend(fn func(text string)) {
	inp.onSend = fn
}
