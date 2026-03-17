package follows

import (
	"context"
	"log/slog"
	"time"

	"github.com/asish/filmpal/internal/letterboxd"
	"github.com/asish/filmpal/internal/store"
)

type FeedSource interface {
	FetchRecentDiary(ctx context.Context, username string, limit int) ([]letterboxd.DiaryEntry, error)
}

type Publisher func(ctx context.Context, subscription store.FollowSubscription, entry letterboxd.DiaryEntry) error

type Poller struct {
	logger    *slog.Logger
	source    FeedSource
	store     *store.Store
	interval  time.Duration
	publisher Publisher
}

func NewPoller(
	logger *slog.Logger,
	source FeedSource,
	store *store.Store,
	interval time.Duration,
	publisher Publisher,
) *Poller {
	return &Poller{
		logger:    logger,
		source:    source,
		store:     store,
		interval:  interval,
		publisher: publisher,
	}
}

func (p *Poller) Run(ctx context.Context) {
	p.pollOnce(ctx)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.pollOnce(ctx)
		}
	}
}

func (p *Poller) pollOnce(ctx context.Context) {
	for _, subscription := range p.store.SnapshotFollows() {
		entries, err := p.source.FetchRecentDiary(ctx, subscription.Username, 10)
		if err != nil {
			p.logger.Warn("failed to refresh followed user", "username", subscription.Username, "error", err)
			continue
		}
		if len(entries) == 0 {
			continue
		}

		if subscription.LastSeenEntry == "" {
			if err := p.store.SeedFollow(subscription.ChannelID, subscription.Username, entries[0].ID); err != nil {
				p.logger.Warn("failed to seed follow state", "username", subscription.Username, "error", err)
			}
			continue
		}

		newEntries := entriesBeforeLastSeen(entries, subscription.LastSeenEntry)
		for _, entry := range newEntries {
			if err := p.publisher(ctx, subscription, entry); err != nil {
				p.logger.Warn("failed to publish follow entry", "username", subscription.Username, "entry_id", entry.ID, "error", err)
				break
			}
			if err := p.store.UpdateLastSeen(subscription.ChannelID, subscription.Username, entry.ID); err != nil {
				p.logger.Warn("failed to update follow cursor", "username", subscription.Username, "entry_id", entry.ID, "error", err)
				break
			}
		}
	}
}

func entriesBeforeLastSeen(entries []letterboxd.DiaryEntry, lastSeen string) []letterboxd.DiaryEntry {
	index := -1
	for i := range entries {
		if entries[i].ID == lastSeen {
			index = i
			break
		}
	}

	var newEntries []letterboxd.DiaryEntry
	if index == -1 {
		newEntries = append(newEntries, entries...)
	} else {
		newEntries = append(newEntries, entries[:index]...)
	}

	for left, right := 0, len(newEntries)-1; left < right; left, right = left+1, right-1 {
		newEntries[left], newEntries[right] = newEntries[right], newEntries[left]
	}

	return newEntries
}
