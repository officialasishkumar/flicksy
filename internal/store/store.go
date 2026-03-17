package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type State struct {
	LinkedAccounts map[string]LinkedAccount `json:"linked_accounts"`
	Follows        []FollowSubscription     `json:"follows"`
	UpdatedAt      time.Time                `json:"updated_at"`
}

type LinkedAccount struct {
	Username  string    `json:"username"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FollowSubscription struct {
	GuildID       string    `json:"guild_id"`
	ChannelID     string    `json:"channel_id"`
	Username      string    `json:"username"`
	AddedByUserID string    `json:"added_by_user_id"`
	LastSeenEntry string    `json:"last_seen_entry"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Store struct {
	mu    sync.RWMutex
	path  string
	state State
}

func New(path string) (*Store, error) {
	store := &Store{
		path: path,
		state: State{
			LinkedAccounts: make(map[string]LinkedAccount),
			Follows:        make([]FollowSubscription, 0),
		},
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create state directory: %w", err)
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if err := store.saveLocked(); err != nil {
				return nil, err
			}
			return store, nil
		}
		return nil, fmt.Errorf("stat state file: %w", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}
	if len(body) == 0 {
		return store, nil
	}

	if err := json.Unmarshal(body, &store.state); err != nil {
		return nil, fmt.Errorf("parse state file: %w", err)
	}

	if store.state.LinkedAccounts == nil {
		store.state.LinkedAccounts = make(map[string]LinkedAccount)
	}
	if store.state.Follows == nil {
		store.state.Follows = make([]FollowSubscription, 0)
	}

	return store, nil
}

func (s *Store) LinkedAccount(userID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	account, ok := s.state.LinkedAccounts[userID]
	return account.Username, ok
}

func (s *Store) SetLinkedAccount(userID, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.LinkedAccounts[userID] = LinkedAccount{
		Username:  normalizeUsername(username),
		UpdatedAt: time.Now().UTC(),
	}
	return s.saveLocked()
}

func (s *Store) RemoveLinkedAccount(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.state.LinkedAccounts, userID)
	return s.saveLocked()
}

func (s *Store) AddFollow(subscription FollowSubscription) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subscription.Username = normalizeUsername(subscription.Username)
	now := time.Now().UTC()
	subscription.CreatedAt = now
	subscription.UpdatedAt = now

	for _, existing := range s.state.Follows {
		if existing.ChannelID == subscription.ChannelID && existing.Username == subscription.Username {
			return false, nil
		}
	}

	s.state.Follows = append(s.state.Follows, subscription)
	sortSubscriptions(s.state.Follows)
	return true, s.saveLocked()
}

func (s *Store) RemoveFollow(channelID, username string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	username = normalizeUsername(username)
	filtered := s.state.Follows[:0]
	removed := false

	for _, existing := range s.state.Follows {
		if existing.ChannelID == channelID && existing.Username == username {
			removed = true
			continue
		}
		filtered = append(filtered, existing)
	}

	if !removed {
		return false, nil
	}

	s.state.Follows = append([]FollowSubscription(nil), filtered...)
	return true, s.saveLocked()
}

func (s *Store) ListFollows(channelID string) []FollowSubscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	follows := make([]FollowSubscription, 0)
	for _, existing := range s.state.Follows {
		if existing.ChannelID == channelID {
			follows = append(follows, existing)
		}
	}
	sortSubscriptions(follows)
	return follows
}

func (s *Store) SnapshotFollows() []FollowSubscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	follows := append([]FollowSubscription(nil), s.state.Follows...)
	sortSubscriptions(follows)
	return follows
}

func (s *Store) UpdateLastSeen(channelID, username, entryID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	username = normalizeUsername(username)
	updated := false
	for index := range s.state.Follows {
		if s.state.Follows[index].ChannelID == channelID && s.state.Follows[index].Username == username {
			s.state.Follows[index].LastSeenEntry = entryID
			s.state.Follows[index].UpdatedAt = time.Now().UTC()
			updated = true
		}
	}
	if !updated {
		return nil
	}
	return s.saveLocked()
}

func (s *Store) SeedFollow(channelID, username, entryID string) error {
	return s.UpdateLastSeen(channelID, username, entryID)
}

func (s *Store) saveLocked() error {
	s.state.UpdatedAt = time.Now().UTC()

	body, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, body, 0o644); err != nil {
		return fmt.Errorf("write temp state file: %w", err)
	}
	if err := os.Rename(tempPath, s.path); err != nil {
		return fmt.Errorf("replace state file: %w", err)
	}
	return nil
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.Trim(strings.TrimSpace(username), "/"))
}

func sortSubscriptions(follows []FollowSubscription) {
	sort.Slice(follows, func(left, right int) bool {
		if follows[left].ChannelID == follows[right].ChannelID {
			return follows[left].Username < follows[right].Username
		}
		return follows[left].ChannelID < follows[right].ChannelID
	})
}
