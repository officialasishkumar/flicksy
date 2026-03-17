package social

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/asish/cinebuddy/internal/letterboxd"
)

type LetterboxdSource interface {
	FetchProfile(ctx context.Context, username string) (letterboxd.Profile, error)
	FetchRecentDiary(ctx context.Context, username string, limit int) ([]letterboxd.DiaryEntry, error)
}

type Comparison struct {
	Left                letterboxd.Profile
	Right               letterboxd.Profile
	SharedFavorites     []letterboxd.FavoriteFilm
	SharedRecent        []SharedRecentEntry
	BiggestDisagreement *SharedRecentEntry
	Score               int
}

type SharedRecentEntry struct {
	Title       string
	Year        int
	LeftRating  float64
	RightRating float64
	LeftLiked   bool
	RightLiked  bool
}

func Analyze(ctx context.Context, source LetterboxdSource, leftUsername, rightUsername string) (Comparison, error) {
	leftProfile, err := source.FetchProfile(ctx, leftUsername)
	if err != nil {
		return Comparison{}, fmt.Errorf("load %s profile: %w", leftUsername, err)
	}
	rightProfile, err := source.FetchProfile(ctx, rightUsername)
	if err != nil {
		return Comparison{}, fmt.Errorf("load %s profile: %w", rightUsername, err)
	}

	leftDiary, err := source.FetchRecentDiary(ctx, leftUsername, 25)
	if err != nil {
		return Comparison{}, fmt.Errorf("load %s diary: %w", leftUsername, err)
	}
	rightDiary, err := source.FetchRecentDiary(ctx, rightUsername, 25)
	if err != nil {
		return Comparison{}, fmt.Errorf("load %s diary: %w", rightUsername, err)
	}

	comparison := Comparison{
		Left:            leftProfile,
		Right:           rightProfile,
		SharedFavorites: sharedFavorites(leftProfile.Favorites, rightProfile.Favorites),
		SharedRecent:    sharedRecentEntries(leftDiary, rightDiary),
	}

	if len(comparison.SharedRecent) > 0 {
		sort.Slice(comparison.SharedRecent, func(i, j int) bool {
			return comparison.SharedRecent[i].Title < comparison.SharedRecent[j].Title
		})
	}
	comparison.BiggestDisagreement = biggestDisagreement(comparison.SharedRecent)
	comparison.Score = compatibilityScore(comparison)

	return comparison, nil
}

func sharedFavorites(left, right []letterboxd.FavoriteFilm) []letterboxd.FavoriteFilm {
	index := make(map[string]letterboxd.FavoriteFilm, len(left))
	for _, film := range left {
		index[normalize(film.Title)] = film
	}

	shared := make([]letterboxd.FavoriteFilm, 0)
	for _, film := range right {
		if leftFilm, ok := index[normalize(film.Title)]; ok {
			shared = append(shared, leftFilm)
		}
	}

	sort.Slice(shared, func(i, j int) bool {
		return shared[i].Title < shared[j].Title
	})
	return shared
}

func sharedRecentEntries(left, right []letterboxd.DiaryEntry) []SharedRecentEntry {
	index := make(map[string]letterboxd.DiaryEntry, len(left))
	for _, entry := range left {
		index[normalize(entry.FilmTitle)] = entry
	}

	shared := make([]SharedRecentEntry, 0)
	for _, entry := range right {
		match, ok := index[normalize(entry.FilmTitle)]
		if !ok {
			continue
		}
		shared = append(shared, SharedRecentEntry{
			Title:       match.FilmTitle,
			Year:        pickYear(match.FilmYear, entry.FilmYear),
			LeftRating:  match.Rating,
			RightRating: entry.Rating,
			LeftLiked:   match.Liked,
			RightLiked:  entry.Liked,
		})
	}

	return shared
}

func biggestDisagreement(entries []SharedRecentEntry) *SharedRecentEntry {
	if len(entries) == 0 {
		return nil
	}

	var choice *SharedRecentEntry
	var largest float64
	for index := range entries {
		diff := math.Abs(entries[index].LeftRating - entries[index].RightRating)
		if diff >= largest {
			largest = diff
			entry := entries[index]
			choice = &entry
		}
	}

	return choice
}

func compatibilityScore(comparison Comparison) int {
	favoritesScore := 0
	if len(comparison.SharedFavorites) > 0 {
		favoritesScore = minInt(25, len(comparison.SharedFavorites)*8)
	}

	recentScore := minInt(30, len(comparison.SharedRecent)*10)
	alignmentScore := 10
	if len(comparison.SharedRecent) > 0 {
		var totalDiff float64
		var ratedCount int
		for _, entry := range comparison.SharedRecent {
			if entry.LeftRating == 0 || entry.RightRating == 0 {
				continue
			}
			totalDiff += math.Abs(entry.LeftRating - entry.RightRating)
			ratedCount++
		}
		if ratedCount > 0 {
			averageDiff := totalDiff / float64(ratedCount)
			alignmentScore = maxInt(0, 20-int(math.Round(averageDiff*6)))
		} else {
			alignmentScore = 14
		}
	}

	activityScore := similarityScore(comparison.Left.ThisYearCount, comparison.Right.ThisYearCount, 10)
	scaleScore := similarityScore(comparison.Left.FilmCount, comparison.Right.FilmCount, 15)

	score := favoritesScore + recentScore + alignmentScore + activityScore + scaleScore
	if score > 100 {
		return 100
	}
	return score
}

func similarityScore(left, right, maxScore int) int {
	if left == 0 && right == 0 {
		return maxScore
	}
	if left == 0 || right == 0 {
		return 0
	}

	smaller := minInt(left, right)
	larger := maxInt(left, right)
	return int(math.Round((float64(smaller) / float64(larger)) * float64(maxScore)))
}

func normalize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(":", " ", "-", " ", "'", "", "(", " ", ")", " ", ".", " ", ",", " ")
	value = replacer.Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

func pickYear(left, right int) int {
	if left != 0 {
		return left
	}
	return right
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
