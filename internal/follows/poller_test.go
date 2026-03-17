package follows

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/asish/bloop/internal/letterboxd"
	"github.com/asish/bloop/internal/store"
)

func TestEntriesBeforeLastSeenReturnsOldestFirst(t *testing.T) {
	entries := []letterboxd.DiaryEntry{
		{ID: "newest"},
		{ID: "middle"},
		{ID: "oldest"},
	}

	result := entriesBeforeLastSeen(entries, "oldest")
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	if result[0].ID != "middle" || result[1].ID != "newest" {
		t.Fatalf("result IDs = %q, %q; want middle, newest", result[0].ID, result[1].ID)
	}
}

func TestPollerPublishesAndAdvancesCursor(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	storeRef, err := store.New(statePath)
	if err != nil {
		t.Fatalf("store.New returned error: %v", err)
	}

	_, err = storeRef.AddFollow(store.FollowSubscription{
		GuildID:       "guild-1",
		ChannelID:     "channel-1",
		Username:      "protolexus",
		AddedByUserID: "user-1",
		LastSeenEntry: "entry-1",
	})
	if err != nil {
		t.Fatalf("AddFollow returned error: %v", err)
	}

	var published []string
	poller := NewPoller(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		fakeFeedSource{
			entries: []letterboxd.DiaryEntry{
				{ID: "entry-3", FilmTitle: "Third"},
				{ID: "entry-2", FilmTitle: "Second"},
				{ID: "entry-1", FilmTitle: "First"},
			},
		},
		storeRef,
		time.Hour,
		func(_ context.Context, _ store.FollowSubscription, entry letterboxd.DiaryEntry) error {
			published = append(published, entry.ID)
			return nil
		},
	)

	poller.pollOnce(context.Background())

	if len(published) != 2 || published[0] != "entry-2" || published[1] != "entry-3" {
		t.Fatalf("published = %v, want [entry-2 entry-3]", published)
	}

	follows := storeRef.ListFollows("channel-1")
	if len(follows) != 1 {
		t.Fatalf("len(follows) = %d, want 1", len(follows))
	}
	if follows[0].LastSeenEntry != "entry-3" {
		t.Fatalf("LastSeenEntry = %q, want entry-3", follows[0].LastSeenEntry)
	}
}

type fakeFeedSource struct {
	entries []letterboxd.DiaryEntry
}

func (f fakeFeedSource) FetchRecentDiary(context.Context, string, int) ([]letterboxd.DiaryEntry, error) {
	return f.entries, nil
}
