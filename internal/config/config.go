package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	AppName      string
	DiscordToken string
	GuildID      string
	DataDir      string
	HTTPTimeout  time.Duration
	PollInterval time.Duration
	UserAgent    string
}

func Load() (Config, error) {
	dataDir := getenv("CINEBUDDY_DATA_DIR", filepath.Join(".", "data"))

	httpTimeout, err := durationFromEnv("CINEBUDDY_HTTP_TIMEOUT", 15*time.Second)
	if err != nil {
		return Config{}, err
	}

	pollInterval, err := durationFromEnv("CINEBUDDY_POLL_INTERVAL", 5*time.Minute)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppName:      "CineBuddy",
		DiscordToken: os.Getenv("DISCORD_TOKEN"),
		GuildID:      os.Getenv("DISCORD_GUILD_ID"),
		DataDir:      dataDir,
		HTTPTimeout:  httpTimeout,
		PollInterval: pollInterval,
		UserAgent: getenv("CINEBUDDY_USER_AGENT",
			"Mozilla/5.0 (compatible; CineBuddy/1.0; +https://github.com/asish/cinebuddy)",
		),
	}

	if cfg.DiscordToken == "" {
		return Config{}, fmt.Errorf("DISCORD_TOKEN is required")
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err == nil {
		return duration, nil
	}

	seconds, convErr := strconv.Atoi(value)
	if convErr != nil {
		return 0, fmt.Errorf("%s must be a Go duration or integer seconds", key)
	}

	return time.Duration(seconds) * time.Second, nil
}
