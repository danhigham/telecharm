package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/danhigham/tg-tui/internal/config"
	"github.com/danhigham/tg-tui/internal/domain"
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

	// Create auth flow with TUI integration
	authFlow := telegram.NewTUIAuth()

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

	// Create TUI app with all dependencies
	app := ui.NewApp(store, tgClient, authFlow)

	// Wire drawFunc so store changes trigger re-render
	store.SetDrawFunc(app.DrawFunc())

	// Wire auth callbacks to send messages into Bubble Tea
	authFlow.OnPhoneRequested = func() {
		app.Send(ui.AuthRequestMsg{Stage: domain.AuthStatePhone})
	}
	authFlow.OnCodeRequested = func() {
		app.Send(ui.AuthRequestMsg{Stage: domain.AuthStateCode})
	}
	authFlow.OnPasswordRequested = func() {
		app.Send(ui.AuthRequestMsg{Stage: domain.AuthState2FA})
	}

	// Update status when connected
	tgClient.SetOnReady(func() {
		app.Send(ui.StatusMsg{Text: "Connected", Connected: true})
	})

	// Context for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Run Telegram client in background
	go func() {
		if err := tgClient.Run(ctx); err != nil {
			logger.Error("telegram client error", zap.Error(err))
			app.Send(ui.StatusMsg{Text: "Disconnected", Connected: false})
		}
	}()

	// Run TUI (blocks until quit)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cancel() // stop telegram client
}
