package app

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/asish/cinebuddy/internal/bot"
	"github.com/asish/cinebuddy/internal/config"
	"github.com/asish/cinebuddy/internal/letterboxd"
	"github.com/asish/cinebuddy/internal/store"
)

type App struct {
	config           config.Config
	logger           *slog.Logger
	bot              *bot.Bot
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

	client := letterboxd.NewClient(cfg.HTTPTimeout, cfg.UserAgent)

	discordBot, err := bot.New(cfg, logger, client, stateStore)
	if err != nil {
		return nil, fmt.Errorf("initialize bot: %w", err)
	}

	return &App{
		config:           cfg,
		logger:           logger,
		bot:              discordBot,
		letterboxdClient: client,
		store:            stateStore,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Info("cinebuddy initialized", "app_name", a.config.AppName, "data_dir", a.config.DataDir)
	if err := a.bot.Run(ctx); err != nil {
		return err
	}
	a.logger.Info("cinebuddy shutting down")
	return nil
}
