package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/asish/cinebuddy/internal/config"
)

type App struct {
	config config.Config
	logger *slog.Logger
}

func New(_ context.Context, logger *slog.Logger) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return &App{
		config: cfg,
		logger: logger,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Info("cinebuddy initialized", "app_name", a.config.AppName)
	<-ctx.Done()
	a.logger.Info("cinebuddy shutting down")
	return nil
}
