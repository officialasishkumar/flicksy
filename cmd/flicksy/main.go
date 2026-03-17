package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/asish/flicksy/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	application, err := app.New(ctx, logger)
	if err != nil {
		logger.Error("failed to initialize application", "error", err)
		os.Exit(1)
	}

	if err := application.Run(ctx); err != nil {
		logger.Error("application stopped with error", "error", err)
		os.Exit(1)
	}
}
