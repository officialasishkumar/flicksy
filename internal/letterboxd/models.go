package letterboxd

import "time"

type OfficialAPIConfig struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
	TokenURL     string
	WebBaseURL   string
}

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

type MemberSummary struct {
	ID          string
	Username    string
	DisplayName string
	ShortName   string
	AvatarURL   string
}

type MemberStatisticsCounts struct {
	FilmLikes            int
	ListLikes            int
	ReviewLikes          int
	StoryLikes           int
	Watches              int
	Ratings              int
	Reviews              int
	DiaryEntries         int
	DiaryEntriesThisYear int
	FilmsInDiaryThisYear int
	FilmsInDiaryLastYear int
	Watchlist            int
	Lists                int
	UnpublishedLists     int
	AccessedSharedLists  int
	Followers            int
	Following            int
	ListTags             int
	FilmTags             int
}

type RatingsHistogramBar struct {
	Rating float64
	Count  int
}

type MemberStats struct {
	Member           MemberSummary
	Counts           MemberStatisticsCounts
	RatingsHistogram []RatingsHistogramBar
	YearsInReview    []int
}

type FilmSummary struct {
	ID             string
	Title          string
	Year           int
	URL            string
	ShortURL       string
	PosterURL      string
	Rating         float64
	RatingCount    int
	ReviewCount    int
	RuntimeMinutes int
	Genres         []string
	Directors      []string
}

type LogEntrySummary struct {
	ID          string
	URL         string
	Film        FilmSummary
	WatchedAt   time.Time
	WatchedDate string
	Rating      float64
	Rewatch     bool
	Liked       bool
	Excerpt     string
}

type ActivityItem struct {
	Type        string
	WhenCreated time.Time
	Member      MemberSummary
	Film        *FilmSummary
	List        *ListSummary
	Followed    *MemberSummary
	LogEntry    *LogEntrySummary
	Rating      float64
	Response    string
	Comment     string
}

type DiscoveryOptions struct {
	Genre       string
	Service     string
	Limit       int
	Sort        string
	Adult       bool
	IncludeTags []string
	ExcludeTags []string
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
