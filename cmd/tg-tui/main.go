package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/danhigham/tg-tui/internal/config"
	"github.com/danhigham/tg-tui/internal/state"
	"github.com/danhigham/tg-tui/internal/telegram"
	"github.com/danhigham/tg-tui/internal/ui"
)

func main() {
	// Load config
	cfgDir := config.Dir()
	cfgPath := filepath.Join(cfgDir, "config.yaml")

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config from %s: %v\n", cfgPath, err)
		fmt.Fprintf(os.Stderr, "\nCreate the config file with:\n")
		fmt.Fprintf(os.Stderr, "  mkdir -p %s\n", cfgDir)
		fmt.Fprintf(os.Stderr, "  cat > %s << 'EOF'\n", cfgPath)
		fmt.Fprintf(os.Stderr, "telegram:\n  api_id: YOUR_API_ID\n  api_hash: \"YOUR_API_HASH\"\nEOF\n")
		fmt.Fprintf(os.Stderr, "\nGet API credentials from https://my.telegram.org\n")
		os.Exit(1)
	}

	// Setup logging to file
	logPath := filepath.Join(cfgDir, "tg-tui.log")
	logCfg := zap.NewDevelopmentConfig()
	logCfg.OutputPaths = []string{logPath}
	logCfg.ErrorOutputPaths = []string{logPath}
	logger, err := logCfg.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Create store (drawFunc will be set after app is created)
	store := state.New(nil)

	// Create TUI
	app := ui.NewApp(store)

	// Now wire drawFunc
	store.SetDrawFunc(app.DrawFunc())

	// Create auth flow with TUI integration
	authFlow := telegram.NewTUIAuth()
	authFlow.OnPhoneRequested = func() {
		app.QueueUpdateDraw(func() {
			app.AuthModal.ShowPhone()
		})
	}
	authFlow.OnCodeRequested = func() {
		app.QueueUpdateDraw(func() {
			app.AuthModal.ShowCode()
		})
	}
	authFlow.OnPasswordRequested = func() {
		app.QueueUpdateDraw(func() {
			app.AuthModal.ShowPassword()
		})
	}

	// Wire auth modal callbacks to auth flow channels
	app.AuthModal.SetCallbacks(
		func(phone string) { authFlow.PhoneCh <- phone },
		func(code string) { authFlow.CodeCh <- code },
		func(password string) { authFlow.PasswordCh <- password },
	)

	// Ensure session directory exists
	sessionDir := cfgDir
	os.MkdirAll(sessionDir, 0700)

	// Create Telegram client
	tgClient := telegram.NewGotdClient(
		cfg.Telegram.APIID,
		cfg.Telegram.APIHash,
		sessionDir,
		store,
		authFlow,
		logger,
	)

	// Wire chat selection
	app.ChatList.SetOnSelect(func(chatID int64) {
		store.SetActiveChat(chatID)

		// Find chat title
		chats := store.GetChatList()
		for _, c := range chats {
			if c.ID == chatID {
				app.MessageView.SetChatTitle(c.Title)
				break
			}
		}

		// Load history if not cached
		msgs := store.GetMessages(chatID)
		if len(msgs) == 0 {
			go func() {
				ctx := context.Background()
				history, err := tgClient.GetHistory(ctx, chatID, 50)
				if err != nil {
					logger.Error("failed to load history", zap.Error(err))
					return
				}
				store.SetMessages(chatID, history)
			}()
		} else {
			app.QueueUpdateDraw(func() {
				app.MessageView.Update(msgs)
			})
		}

		// Switch focus to input
		app.Application.SetFocus(app.Input.InputField)
	})

	// Wire message sending
	app.Input.SetOnSend(func(text string) {
		chatID := store.GetActiveChat()
		if chatID == 0 {
			return
		}
		go func() {
			ctx := context.Background()
			if err := tgClient.SendMessage(ctx, chatID, text); err != nil {
				logger.Error("failed to send message", zap.Error(err))
				app.QueueUpdateDraw(func() {
					app.SetStatus(fmt.Sprintf("[red]Send error: %v", err))
				})
			}
		}()
	})

	// Context for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Run Telegram client in background
	go func() {
		if err := tgClient.Run(ctx); err != nil {
			logger.Error("telegram client error", zap.Error(err))
			app.QueueUpdateDraw(func() {
				app.SetStatus(fmt.Sprintf("[red]Disconnected: %v", err))
			})
		}
	}()

	// Run TUI (blocks until quit)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cancel() // stop telegram client
}
