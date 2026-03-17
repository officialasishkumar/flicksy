package letterboxd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSearchAndFetchFilmPreferOfficialAPI(t *testing.T) {
	var tokenRequests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/auth/token":
			tokenRequests.Add(1)
			writeJSON(t, response, map[string]any{
				"access_token": "official-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		case request.Method == http.MethodGet && request.URL.Path == "/api/v0/search":
			if got := request.URL.Query().Get("include"); got != "FilmSearchItem" {
				t.Fatalf("include query = %q, want FilmSearchItem", got)
			}
			writeJSON(t, response, map[string]any{
				"items": []map[string]any{
					{
						"type": "FilmSearchItem",
						"film": map[string]any{
							"id":          "film-spirited-away",
							"name":        "Spirited Away",
							"releaseYear": 2001,
							"links": []map[string]any{
								{"type": "letterboxd", "url": "https://letterboxd.com/film/spirited-away/"},
								{"type": "short", "url": "https://boxd.it/spirited"},
							},
							"poster": map[string]any{
								"sizes": map[string]string{"230": "https://img.test/spirited.jpg"},
							},
							"statistics": map[string]any{
								"rating":      4.5,
								"ratingCount": 1000,
								"reviewCount": 200,
							},
							"genres": []map[string]any{
								{"id": "genre-animation", "displayName": "Animation"},
							},
							"directors": []map[string]any{
								{"name": "Hayao Miyazaki"},
							},
						},
					},
				},
			})
		case request.Method == http.MethodGet && request.URL.Path == "/api/v0/film/film-spirited-away":
			writeJSON(t, response, map[string]any{
				"id":          "film-spirited-away",
				"name":        "Spirited Away",
				"description": "A bathhouse adventure.",
				"releaseYear": 2001,
				"runTime":     125,
				"links": []map[string]any{
					{"type": "letterboxd", "url": "https://letterboxd.com/film/spirited-away/"},
					{"type": "short", "url": "https://boxd.it/spirited"},
				},
				"poster": map[string]any{
					"sizes": map[string]string{"230": "https://img.test/spirited.jpg"},
				},
				"statistics": map[string]any{
					"rating":      4.5,
					"ratingCount": 1000,
					"reviewCount": 200,
				},
				"genres": []map[string]any{
					{"id": "genre-animation", "displayName": "Animation"},
					{"id": "genre-fantasy", "displayName": "Fantasy"},
				},
				"directors": []map[string]any{
					{"name": "Hayao Miyazaki"},
				},
				"cast": []map[string]any{
					{"name": "Rumi Hiiragi"},
					{"name": "Miyu Irino"},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.String())
		}
	}))
	defer server.Close()

	client := NewClient(2*time.Second, "test-agent", OfficialAPIConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		BaseURL:      server.URL + "/api/v0",
		TokenURL:     server.URL + "/auth/token",
		WebBaseURL:   server.URL,
	})

	results, err := client.SearchFilm(context.Background(), "spirited away", 5)
	if err != nil {
		t.Fatalf("SearchFilm returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].URL != "https://letterboxd.com/film/spirited-away/" {
		t.Fatalf("results[0].URL = %q", results[0].URL)
	}

	film, err := client.FetchFilm(context.Background(), "spirited away")
	if err != nil {
		t.Fatalf("FetchFilm returned error: %v", err)
	}
	if film.Title != "Spirited Away" {
		t.Fatalf("film.Title = %q, want Spirited Away", film.Title)
	}
	if film.RuntimeMinutes != 125 {
		t.Fatalf("film.RuntimeMinutes = %d, want 125", film.RuntimeMinutes)
	}
	if film.Director != "Hayao Miyazaki" {
		t.Fatalf("film.Director = %q, want Hayao Miyazaki", film.Director)
	}
	if tokenRequests.Load() != 1 {
		t.Fatalf("token requests = %d, want 1", tokenRequests.Load())
	}
}

func TestOfficialMemberFeatures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/auth/token":
			writeJSON(t, response, map[string]any{
				"access_token": "official-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		case request.Method == http.MethodHead && request.URL.Path == "/protolexus/":
			response.Header().Set("x-letterboxd-identifier", "member-protolexus")
			response.WriteHeader(http.StatusOK)
		case request.Method == http.MethodGet && request.URL.Path == "/api/v0/member/member-protolexus/statistics":
			writeJSON(t, response, map[string]any{
				"member": map[string]any{
					"id":          "member-protolexus",
					"username":    "protolexus",
					"displayName": "Proto",
				},
				"counts": map[string]any{
					"watches":              1200,
					"ratings":              1100,
					"reviews":              200,
					"diaryEntries":         900,
					"diaryEntriesThisYear": 40,
					"watchlist":            75,
					"followers":            250,
					"following":            180,
				},
				"yearsInReview": []int{2023, 2024},
				"ratingsHistogram": []map[string]any{
					{"rating": 4.5, "count": 80},
					{"rating": 4.0, "count": 120},
				},
			})
		case request.Method == http.MethodGet && request.URL.Path == "/api/v0/films/genres":
			writeJSON(t, response, map[string]any{
				"items": []map[string]any{
					{"id": "genre-animation", "displayName": "Animation"},
					{"id": "genre-horror", "displayName": "Horror"},
				},
			})
		case request.Method == http.MethodGet && request.URL.Path == "/api/v0/member/member-protolexus/watchlist":
			if got := request.URL.Query().Get("genre"); got != "genre-animation" {
				t.Fatalf("watchlist genre query = %q, want genre-animation", got)
			}
			writeJSON(t, response, map[string]any{
				"items": []map[string]any{
					{
						"id":          "film-perfect-blue",
						"name":        "Perfect Blue",
						"releaseYear": 1997,
						"links": []map[string]any{
							{"type": "letterboxd", "url": "https://letterboxd.com/film/perfect-blue/"},
						},
						"statistics": map[string]any{
							"rating":      4.3,
							"ratingCount": 500,
							"reviewCount": 100,
						},
					},
				},
			})
		case request.Method == http.MethodGet && request.URL.Path == "/api/v0/member/member-protolexus/activity":
			writeJSON(t, response, map[string]any{
				"items": []map[string]any{
					{
						"type":        "FilmWatchActivity",
						"whenCreated": "2025-02-01T12:00:00Z",
						"member": map[string]any{
							"id":          "member-protolexus",
							"username":    "protolexus",
							"displayName": "Proto",
						},
						"film": map[string]any{
							"id":          "film-spirited-away",
							"name":        "Spirited Away",
							"releaseYear": 2001,
							"links": []map[string]any{
								{"type": "letterboxd", "url": "https://letterboxd.com/film/spirited-away/"},
							},
						},
						"rating": 4.5,
					},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.String())
		}
	}))
	defer server.Close()

	client := NewClient(2*time.Second, "test-agent", OfficialAPIConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		BaseURL:      server.URL + "/api/v0",
		TokenURL:     server.URL + "/auth/token",
		WebBaseURL:   server.URL,
	})

	stats, err := client.FetchMemberStats(context.Background(), "protolexus")
	if err != nil {
		t.Fatalf("FetchMemberStats returned error: %v", err)
	}
	if stats.Member.Username != "protolexus" {
		t.Fatalf("stats.Member.Username = %q", stats.Member.Username)
	}
	if stats.Counts.Watchlist != 75 {
		t.Fatalf("stats.Counts.Watchlist = %d, want 75", stats.Counts.Watchlist)
	}

	watchlist, err := client.FetchWatchlist(context.Background(), "protolexus", 5, "animation")
	if err != nil {
		t.Fatalf("FetchWatchlist returned error: %v", err)
	}
	if len(watchlist) != 1 || watchlist[0].Title != "Perfect Blue" {
		t.Fatalf("watchlist = %#v, want Perfect Blue", watchlist)
	}

	activity, err := client.FetchActivity(context.Background(), "protolexus", 5)
	if err != nil {
		t.Fatalf("FetchActivity returned error: %v", err)
	}
	if len(activity) != 1 || activity[0].Type != "FilmWatchActivity" {
		t.Fatalf("activity = %#v, want one FilmWatchActivity item", activity)
	}
}

func TestDiscoverFilmsResolvesGenreAndService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/auth/token":
			writeJSON(t, response, map[string]any{
				"access_token": "official-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		case request.Method == http.MethodGet && request.URL.Path == "/api/v0/films/genres":
			writeJSON(t, response, map[string]any{
				"items": []map[string]any{
					{"id": "genre-animation", "displayName": "Animation"},
				},
			})
		case request.Method == http.MethodGet && request.URL.Path == "/api/v0/films/film-services":
			writeJSON(t, response, map[string]any{
				"items": []map[string]any{
					{"id": "service-max", "displayName": "Max"},
				},
			})
		case request.Method == http.MethodGet && request.URL.Path == "/api/v0/films":
			values := request.URL.Query()
			if values.Get("genre") != "genre-animation" {
				t.Fatalf("discover genre = %q", values.Get("genre"))
			}
			if values.Get("service") != "service-max" {
				t.Fatalf("discover service = %q", values.Get("service"))
			}
			if values.Get("sort") != defaultDiscoverSort {
				t.Fatalf("discover sort = %q", values.Get("sort"))
			}
			if !strings.Contains(values.Encode(), "where=Released") || !strings.Contains(values.Encode(), "where=FeatureLength") {
				t.Fatalf("discover where values missing: %s", values.Encode())
			}
			writeJSON(t, response, map[string]any{
				"items": []map[string]any{
					{
						"id":          "film-robot-dreams",
						"name":        "Robot Dreams",
						"releaseYear": 2023,
						"links": []map[string]any{
							{"type": "letterboxd", "url": "https://letterboxd.com/film/robot-dreams/"},
						},
						"statistics": map[string]any{
							"rating":      4.1,
							"ratingCount": 300,
							"reviewCount": 50,
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.String())
		}
	}))
	defer server.Close()

	client := NewClient(2*time.Second, "test-agent", OfficialAPIConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		BaseURL:      server.URL + "/api/v0",
		TokenURL:     server.URL + "/auth/token",
		WebBaseURL:   server.URL,
	})

	films, err := client.DiscoverFilms(context.Background(), DiscoveryOptions{
		Genre:   "animation",
		Service: "max",
		Limit:   4,
	})
	if err != nil {
		t.Fatalf("DiscoverFilms returned error: %v", err)
	}
	if len(films) != 1 || films[0].Title != "Robot Dreams" {
		t.Fatalf("films = %#v, want Robot Dreams", films)
	}
}

func writeJSON(t *testing.T, response http.ResponseWriter, payload any) {
	t.Helper()

	response.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(response).Encode(payload); err != nil {
		t.Fatalf("encode json response: %v", err)
	}
}
