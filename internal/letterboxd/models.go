package letterboxd

import "time"

type Profile struct {
	Username       string
	DisplayName    string
	URL            string
	AvatarURL      string
	Bio            string
	Website        string
	FilmCount      int
	ThisYearCount  int
	ListCount      int
	FollowingCount int
	FollowersCount int
	Favorites      []FavoriteFilm
}

type FavoriteFilm struct {
	Title string
	Year  int
	URL   string
	Slug  string
}

type Film struct {
	Title          string
	Year           int
	URL            string
	ShortURL       string
	PosterURL      string
	Description    string
	Director       string
	RuntimeMinutes int
	Rating         float64
	RatingCount    int
	ReviewCount    int
	Genres         []string
	Cast           []string
}

type FilmSearchResult struct {
	Title string
	URL   string
	Slug  string
}

type DiaryEntry struct {
	ID          string
	EntryType   string
	FilmTitle   string
	FilmYear    int
	FilmURL     string
	FilmSlug    string
	WatchedDate string
	PublishedAt time.Time
	Rewatch     bool
	Rating      float64
	Liked       bool
	Excerpt     string
	PosterURL   string
}

type ListSummary struct {
	ID          string
	Title       string
	URL         string
	PublishedAt time.Time
	Excerpt     string
	Films       []FavoriteFilm
}
