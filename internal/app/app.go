package app

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/asish/cinebuddy/internal/config"
	"github.com/asish/cinebuddy/internal/letterboxd"
	"github.com/asish/cinebuddy/internal/store"
)

type App struct {
	config           config.Config
	logger           *slog.Logger
	letterboxdClient *letterboxd.Client
	store            *store.Store
}

func New(_ context.Context, logger *slog.Logger) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	stateStore, err := store.New(filepath.Join(cfg.DataDir, "state.json"))
	if err != nil {
		return nil, fmt.Errorf("initialize state store: %w", err)
	}

	return &App{
		config:           cfg,
		logger:           logger,
		letterboxdClient: letterboxd.NewClient(cfg.HTTPTimeout, cfg.UserAgent),
		store:            stateStore,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Info("cinebuddy initialized", "app_name", a.config.AppName, "data_dir", a.config.DataDir)
	<-ctx.Done()
	a.logger.Info("cinebuddy shutting down")
	return nil
}
