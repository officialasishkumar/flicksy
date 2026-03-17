package store

import (
	"path/filepath"
	"testing"
)

func TestStorePersistsLinksAndFollows(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")

	store, err := New(statePath)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if err := store.SetLinkedAccount("user-1", "ProtoLexus"); err != nil {
		t.Fatalf("SetLinkedAccount returned error: %v", err)
	}

	added, err := store.AddFollow(FollowSubscription{
		GuildID:       "guild-1",
		ChannelID:     "channel-1",
		Username:      "ProtoLexus",
		AddedByUserID: "user-1",
	})
	if err != nil {
		t.Fatalf("AddFollow returned error: %v", err)
	}
	if !added {
		t.Fatal("AddFollow reported false, want true")
	}

	if err := store.UpdateLastSeen("channel-1", "protolexus", "entry-123"); err != nil {
		t.Fatalf("UpdateLastSeen returned error: %v", err)
	}

	reloaded, err := New(statePath)
	if err != nil {
		t.Fatalf("reloading store returned error: %v", err)
	}

	username, ok := reloaded.LinkedAccount("user-1")
	if !ok || username != "protolexus" {
		t.Fatalf("LinkedAccount = (%q, %v), want (protolexus, true)", username, ok)
	}

	follows := reloaded.ListFollows("channel-1")
	if len(follows) != 1 {
		t.Fatalf("len(follows) = %d, want 1", len(follows))
	}
	if follows[0].LastSeenEntry != "entry-123" {
		t.Fatalf("LastSeenEntry = %q, want entry-123", follows[0].LastSeenEntry)
	}
}
