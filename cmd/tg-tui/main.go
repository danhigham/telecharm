package main

import (
	"fmt"
	"os"

	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()

	textView := tview.NewTextView().
		SetText("tg-tui - Telegram Client\n\nPress Ctrl+C to quit.")
	textView.SetBorder(true).SetTitle(" tg-tui ")

	if err := app.SetRoot(textView, true).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
