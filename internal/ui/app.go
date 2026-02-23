package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/danhigham/tg-tui/internal/state"
)

type App struct {
	Application *tview.Application
	Pages       *tview.Pages
	ChatList    *ChatList
	MessageView *MessageView
	Input       *Input
	AuthModal   *AuthModal
	StatusBar   *tview.TextView
	Store       *state.Store

	focusOrder []tview.Primitive
	focusIndex int
}

func NewApp(store *state.Store) *App {
	a := &App{
		Application: tview.NewApplication(),
		Store:       store,
	}

	a.ChatList = NewChatList()
	a.MessageView = NewMessageView()
	a.Input = NewInput()
	a.StatusBar = tview.NewTextView().SetDynamicColors(true)
	a.StatusBar.SetText("[yellow]Connecting...")

	a.Pages = tview.NewPages()
	a.AuthModal = NewAuthModal(a.Application, a.Pages)

	a.focusOrder = []tview.Primitive{a.ChatList, a.MessageView, a.Input}
	a.focusIndex = 0

	a.buildLayout()
	a.setupKeybindings()

	return a
}

func (a *App) buildLayout() {
	// Right pane: messages + input stacked vertically
	rightPane := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.MessageView, 0, 1, false).
		AddItem(a.Input, 1, 0, false)

	// Main layout: chat list | right pane
	mainLayout := tview.NewFlex().
		AddItem(a.ChatList, 30, 0, true).
		AddItem(rightPane, 0, 1, false)

	// Full layout: main + status bar
	fullLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(mainLayout, 0, 1, true).
		AddItem(a.StatusBar, 1, 0, false)

	a.Pages.AddPage("main", fullLayout, true, true)
	a.Application.SetRoot(a.Pages, true)
}

func (a *App) setupKeybindings() {
	a.Application.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Ctrl+C to quit
		if event.Key() == tcell.KeyCtrlC {
			a.Application.Stop()
			return nil
		}

		// Tab / Shift+Tab to cycle focus
		if event.Key() == tcell.KeyTab {
			a.focusIndex = (a.focusIndex + 1) % len(a.focusOrder)
			a.Application.SetFocus(a.focusOrder[a.focusIndex])
			return nil
		}
		if event.Key() == tcell.KeyBacktab {
			a.focusIndex = (a.focusIndex - 1 + len(a.focusOrder)) % len(a.focusOrder)
			a.Application.SetFocus(a.focusOrder[a.focusIndex])
			return nil
		}

		// Escape returns to chat list
		if event.Key() == tcell.KeyEscape {
			a.focusIndex = 0
			a.Application.SetFocus(a.focusOrder[0])
			return nil
		}

		// 'q' quits only when not in input
		if event.Rune() == 'q' && a.Application.GetFocus() != a.Input.InputField {
			a.Application.Stop()
			return nil
		}

		return event
	})
}

func (a *App) Refresh() {
	chats := a.Store.GetChatList()
	a.ChatList.Update(chats)

	activeChat := a.Store.GetActiveChat()
	if activeChat != 0 {
		msgs := a.Store.GetMessages(activeChat)
		a.MessageView.Update(msgs)
	}
}

func (a *App) SetStatus(text string) {
	a.StatusBar.SetText(text)
}

func (a *App) Run() error {
	return a.Application.Run()
}

func (a *App) QueueUpdateDraw(f func()) {
	a.Application.QueueUpdateDraw(f)
}

// DrawFunc returns a function suitable for state.New() that triggers a tview redraw.
func (a *App) DrawFunc() func() {
	return func() {
		a.Application.QueueUpdateDraw(func() {
			a.Refresh()
		})
	}
}
