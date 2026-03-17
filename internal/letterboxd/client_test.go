package letterboxd

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFetchProfileParsesCoreFields(t *testing.T) {
	client := newFixtureClient(t, map[string]string{
		"letterboxd.com/protolexus/": fixturePath("profile_protolexus.html"),
	})

	profile, err := client.FetchProfile(context.Background(), "protolexus")
	if err != nil {
		t.Fatalf("FetchProfile returned error: %v", err)
	}

	if profile.DisplayName != "protolexus" {
		t.Fatalf("DisplayName = %q, want %q", profile.DisplayName, "protolexus")
	}
	if profile.FilmCount != 2010 {
		t.Fatalf("FilmCount = %d, want 2010", profile.FilmCount)
	}
	if profile.ThisYearCount != 53 {
		t.Fatalf("ThisYearCount = %d, want 53", profile.ThisYearCount)
	}
	if profile.ListCount != 38 {
		t.Fatalf("ListCount = %d, want 38", profile.ListCount)
	}
	if profile.FollowersCount != 2227 {
		t.Fatalf("FollowersCount = %d, want 2227", profile.FollowersCount)
	}
	if !strings.Contains(profile.Bio, "Keeping it meta") {
		t.Fatalf("Bio = %q, want snippet", profile.Bio)
	}
	if len(profile.Favorites) != 4 {
		t.Fatalf("len(Favorites) = %d, want 4", len(profile.Favorites))
	}
	if profile.Favorites[0].Title != "Interstellar" {
		t.Fatalf("first favorite = %q, want Interstellar", profile.Favorites[0].Title)
	}
}

func TestFetchRecentDiaryAndListsFromRSS(t *testing.T) {
	client := newFixtureClient(t, map[string]string{
		"letterboxd.com/protolexus/rss/": fixturePath("feed_protolexus.xml"),
	})

	entries, err := client.FetchRecentDiary(context.Background(), "protolexus", 3)
	if err != nil {
		t.Fatalf("FetchRecentDiary returned error: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(entries))
	}
	if entries[0].FilmTitle != "Lessons of Darkness" {
		t.Fatalf("entries[0].FilmTitle = %q, want Lessons of Darkness", entries[0].FilmTitle)
	}
	if entries[0].Rating != 5 {
		t.Fatalf("entries[0].Rating = %v, want 5", entries[0].Rating)
	}

	listSummary, err := client.FindList(context.Background(), "protolexus", "ghibli")
	if err != nil {
		t.Fatalf("FindList returned error: %v", err)
	}
	if listSummary == nil {
		t.Fatal("FindList returned nil, want a match")
	}
	if listSummary.Title != "Ghibli Ranked" {
		t.Fatalf("listSummary.Title = %q, want Ghibli Ranked", listSummary.Title)
	}
	if len(listSummary.Films) == 0 {
		t.Fatal("listSummary.Films was empty")
	}
}

func TestFetchFilmParsesCoreFields(t *testing.T) {
	client := newFixtureClient(t, map[string]string{
		"letterboxd.com/film/spirited-away/": fixturePath("film_spirited_away.html"),
	})

	film, err := client.FetchFilm(context.Background(), "https://letterboxd.com/film/spirited-away/")
	if err != nil {
		t.Fatalf("FetchFilm returned error: %v", err)
	}

	if film.Title != "Spirited Away" {
		t.Fatalf("Title = %q, want Spirited Away", film.Title)
	}
	if film.Year != 2001 {
		t.Fatalf("Year = %d, want 2001", film.Year)
	}
	if film.Director != "Hayao Miyazaki" {
		t.Fatalf("Director = %q, want Hayao Miyazaki", film.Director)
	}
	if film.RuntimeMinutes != 125 {
		t.Fatalf("RuntimeMinutes = %d, want 125", film.RuntimeMinutes)
	}
	if film.Rating < 4.4 {
		t.Fatalf("Rating = %v, want >= 4.4", film.Rating)
	}
	if len(film.Genres) < 3 {
		t.Fatalf("Genres = %v, want at least 3", film.Genres)
	}
	if len(film.Cast) == 0 {
		t.Fatal("Cast was empty")
	}
}

func TestSearchFilmUsesDuckDuckGoResults(t *testing.T) {
	client := newFixtureClient(t, map[string]string{
		"duckduckgo.com/html/": fixturePath("search_spirited_away.html"),
	})

	results, err := client.SearchFilm(context.Background(), "spirited away", 5)
	if err != nil {
		t.Fatalf("SearchFilm returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].URL != "https://letterboxd.com/film/spirited-away/" {
		t.Fatalf("results[0].URL = %q, want direct film url", results[0].URL)
	}
}

func newFixtureClient(t *testing.T, responses map[string]string) *Client {
	t.Helper()

	client := NewClient(2*time.Second, "test-agent")
	client.httpClient = &http.Client{
		Timeout: 2 * time.Second,
		Transport: fixtureRoundTripper{
			t:         t,
			responses: responses,
		},
	}
	return client
}

type fixtureRoundTripper struct {
	t         *testing.T
	responses map[string]string
}

func (f fixtureRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	f.t.Helper()

	key := request.URL.Host + request.URL.Path
	fixture, ok := f.responses[key]
	if !ok {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Status:     "404 Not Found",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("missing fixture")),
			Request:    request,
		}, nil
	}

	body, err := os.ReadFile(fixture)
	if err != nil {
		return nil, err
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(body))),
		Request:    request,
	}, nil
}

func fixturePath(name string) string {
	return filepath.Join("testdata", name)
}
