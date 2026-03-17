package social

import (
	"testing"

	"github.com/asish/cinebuddy/internal/letterboxd"
)

func TestCompatibilityScoreRewardsOverlap(t *testing.T) {
	comparison := Comparison{
		Left:  structProfile("left", 1000, 100),
		Right: structProfile("right", 900, 90),
		SharedFavorites: []letterboxd.FavoriteFilm{
			{Title: "Heat"},
			{Title: "The Matrix"},
		},
		SharedRecent: []SharedRecentEntry{
			{Title: "Heat", LeftRating: 5, RightRating: 4.5},
			{Title: "Thief", LeftRating: 4.5, RightRating: 4},
		},
	}

	score := compatibilityScore(comparison)
	if score < 60 {
		t.Fatalf("compatibilityScore = %d, want >= 60", score)
	}
}

func TestCompatibilityScoreFallsWithoutOverlap(t *testing.T) {
	comparison := Comparison{
		Left:  structProfile("left", 1200, 120),
		Right: structProfile("right", 50, 3),
	}

	score := compatibilityScore(comparison)
	if score >= 40 {
		t.Fatalf("compatibilityScore = %d, want < 40", score)
	}
}

func structProfile(username string, filmCount, thisYear int) letterboxd.Profile {
	return letterboxd.Profile{
		Username:      username,
		FilmCount:     filmCount,
		ThisYearCount: thisYear,
	}
}
