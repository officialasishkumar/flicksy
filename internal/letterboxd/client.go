package letterboxd

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
)

var (
	ErrCloudflareChallenge = errors.New("letterboxd blocked the request with a challenge page")
	jsonLDPattern          = regexp.MustCompile(`(?s)<script type="application/ld\+json">\s*(?:/\*.*?\*/\s*)?(\{.*?\})(?:\s*/\*.*?\*/)?\s*</script>`)
)

type Client struct {
	httpClient  *http.Client
	userAgent   string
	cache       *cache
	official    OfficialAPIConfig
	officialMu  sync.Mutex
	token       string
	tokenExpiry time.Time
	baseURL     string
	tokenURL    string
	webBaseURL  string
}

func NewClient(httpTimeout time.Duration, userAgent string, official ...OfficialAPIConfig) *Client {
	cfg := OfficialAPIConfig{}
	if len(official) > 0 {
		cfg = official[0]
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.letterboxd.com/api/v0"
	}
	tokenURL := strings.TrimSpace(cfg.TokenURL)
	if tokenURL == "" {
		tokenURL = baseURL + "/auth/token"
	}
	webBaseURL := strings.TrimRight(cfg.WebBaseURL, "/")
	if webBaseURL == "" {
		webBaseURL = "https://letterboxd.com"
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
		userAgent:  userAgent,
		cache:      newCache(10 * time.Minute),
		official:   cfg,
		baseURL:    baseURL,
		tokenURL:   tokenURL,
		webBaseURL: webBaseURL,
	}
}

func (c *Client) HasOfficialAPI() bool {
	return strings.TrimSpace(c.official.ClientID) != "" && strings.TrimSpace(c.official.ClientSecret) != ""
}

func (c *Client) RefreshProfile(username string) {
	username = strings.ToLower(strings.Trim(strings.TrimSpace(username), "/"))
	c.cache.clearPrefix("profile:" + username)
	c.cache.clearPrefix("rss:" + username)
	c.cache.clearPrefix("member-id:" + username)
	c.cache.clearPrefix("stats:" + username)
	c.cache.clearPrefix("watchlist:" + username)
	c.cache.clearPrefix("activity:" + username)
}

func (c *Client) ClearAll() {
	c.cache.clearAll()
}

func (c *Client) FetchProfile(ctx context.Context, username string) (Profile, error) {
	username = strings.Trim(strings.TrimSpace(username), "/")
	if username == "" {
		return Profile{}, fmt.Errorf("username is required")
	}

	body, err := c.fetchCached(ctx, "profile:"+strings.ToLower(username), "https://letterboxd.com/"+username+"/")
	if err != nil {
		return Profile{}, err
	}

	document, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return Profile{}, fmt.Errorf("parse profile html: %w", err)
	}

	profile := Profile{
		Username: username,
		URL:      "https://letterboxd.com/" + username + "/",
	}

	profile.DisplayName = strings.TrimSpace(document.Find(".person-display-name .label").First().Text())
	if profile.DisplayName == "" {
		profile.DisplayName = username
	}

	if avatarURL, ok := document.Find(".profile-avatar img").First().Attr("src"); ok {
		profile.AvatarURL = strings.TrimSpace(avatarURL)
	}

	profile.Bio = strings.TrimSpace(document.Find(".js-bio-content").First().Text())
	if websiteText := strings.TrimSpace(document.Find(".profile-metadata .label").First().Text()); websiteText != "" {
		profile.Website = websiteText
	}

	document.Find(".profile-statistic").Each(func(_ int, selection *goquery.Selection) {
		label := normalizeText(selection.Find(".definition").Text())
		value := numberFromString(selection.Find(".value").Text())

		switch label {
		case "films":
			profile.FilmCount = value
		case "this year":
			profile.ThisYearCount = value
		case "lists":
			profile.ListCount = value
		case "following":
			profile.FollowingCount = value
		case "followers":
			profile.FollowersCount = value
		}
	})

	document.Find("#favourites .griditem .react-component").Each(func(_ int, selection *goquery.Selection) {
		title := strings.TrimSpace(selection.AttrOr("data-item-name", ""))
		fullTitle := strings.TrimSpace(selection.AttrOr("data-item-full-display-name", title))
		link := selection.AttrOr("data-item-link", "")
		if title == "" || link == "" {
			return
		}

		profile.Favorites = append(profile.Favorites, FavoriteFilm{
			Title: titleWithoutYear(title),
			Year:  yearFromTitle(fullTitle),
			URL:   absoluteLetterboxdURL(link),
			Slug:  strings.TrimSpace(selection.AttrOr("data-item-slug", slugFromURL(link))),
		})
	})

	return profile, nil
}

func (c *Client) FetchRecentDiary(ctx context.Context, username string, limit int) ([]DiaryEntry, error) {
	items, err := c.fetchRSS(ctx, username)
	if err != nil {
		return nil, err
	}

	entries := make([]DiaryEntry, 0, len(items))
	for _, item := range items {
		if item.EntryType != "review" && item.EntryType != "watch" {
			continue
		}
		entries = append(entries, item.DiaryEntry)
	}

	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	return entries, nil
}

func (c *Client) FetchRecentLists(ctx context.Context, username string, limit int) ([]ListSummary, error) {
	items, err := c.fetchRSS(ctx, username)
	if err != nil {
		return nil, err
	}

	lists := make([]ListSummary, 0, len(items))
	for _, item := range items {
		if item.EntryType != "list" {
			continue
		}
		lists = append(lists, item.ListSummary())
	}

	if limit > 0 && len(lists) > limit {
		lists = lists[:limit]
	}

	return lists, nil
}

func (c *Client) FindList(ctx context.Context, username, query string) (*ListSummary, error) {
	lists, err := c.FetchRecentLists(ctx, username, 100)
	if err != nil {
		return nil, err
	}

	normalizedQuery := normalizeText(query)
	if normalizedQuery == "" {
		return nil, fmt.Errorf("list query is required")
	}

	for i := range lists {
		if strings.Contains(normalizeText(lists[i].Title), normalizedQuery) {
			return &lists[i], nil
		}
	}

	return nil, nil
}

func (c *Client) FindRecentLogs(ctx context.Context, username, filmQuery string, limit int) ([]DiaryEntry, error) {
	entries, err := c.FetchRecentDiary(ctx, username, 100)
	if err != nil {
		return nil, err
	}

	targetTitle := normalizeText(filmQuery)
	film, filmErr := c.FetchFilm(ctx, filmQuery)
	if filmErr == nil {
		targetTitle = normalizeText(film.Title)
	}

	matches := make([]DiaryEntry, 0)
	for _, entry := range entries {
		if sameTitle(entry.FilmTitle, targetTitle) || strings.Contains(normalizeText(entry.FilmTitle), targetTitle) {
			matches = append(matches, entry)
		}
	}

	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}

	return matches, nil
}

func (c *Client) SearchFilm(ctx context.Context, query string, limit int) ([]FilmSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("film query is required")
	}

	if looksLikeFilmURL(query) {
		return []FilmSearchResult{{
			Title: filmTitleFromURL(query),
			URL:   normalizeFilmURL(query),
			Slug:  slugFromURL(query),
		}}, nil
	}

	if c.HasOfficialAPI() {
		if results, err := c.searchFilmOfficial(ctx, query, limit); err == nil && len(results) > 0 {
			return results, nil
		}
	}

	searchURL := "https://duckduckgo.com/html/?q=" + url.QueryEscape(`site:letterboxd.com/film/ "`+query+`"`)
	body, err := c.fetchCached(ctx, "search:"+normalizeText(query), searchURL)
	if err != nil {
		return nil, err
	}

	document, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse search html: %w", err)
	}

	results := make([]FilmSearchResult, 0, limit)
	seen := make(map[string]struct{})

	document.Find(".result__a").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		href, ok := selection.Attr("href")
		if !ok {
			return true
		}

		resolved := resolveDuckDuckGoResult(href)
		if !strings.Contains(resolved, "letterboxd.com/film/") {
			return true
		}

		normalized := normalizeFilmURL(resolved)
		if _, exists := seen[normalized]; exists {
			return true
		}
		seen[normalized] = struct{}{}

		title := strings.TrimSpace(strings.TrimSuffix(selection.Text(), " - Letterboxd"))
		results = append(results, FilmSearchResult{
			Title: title,
			URL:   normalized,
			Slug:  slugFromURL(normalized),
		})

		return limit <= 0 || len(results) < limit
	})

	return results, nil
}

func (c *Client) FetchFilm(ctx context.Context, query string) (Film, error) {
	if c.HasOfficialAPI() {
		if film, err := c.fetchFilmOfficial(ctx, query); err == nil {
			return film, nil
		}
	}

	filmURL := normalizeFilmURL(query)
	if !looksLikeFilmURL(filmURL) {
		results, err := c.SearchFilm(ctx, query, 1)
		if err != nil {
			return Film{}, err
		}
		if len(results) == 0 {
			return Film{}, fmt.Errorf("no Letterboxd film match found for %q", query)
		}
		filmURL = results[0].URL
	}

	body, err := c.fetchCached(ctx, "film:"+filmURL, filmURL)
	if err != nil {
		return Film{}, err
	}

	document, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return Film{}, fmt.Errorf("parse film html: %w", err)
	}

	film := Film{
		URL:         filmURL,
		Description: strings.TrimSpace(document.Find(`meta[property="og:description"]`).AttrOr("content", "")),
		Director:    strings.TrimSpace(document.Find(`meta[name="twitter:data1"]`).AttrOr("content", "")),
		PosterURL:   strings.TrimSpace(document.Find(`meta[property="og:image"]`).AttrOr("content", "")),
	}

	if shortURL := strings.TrimSpace(document.Find(`input[id^="url-field-film-"]`).AttrOr("value", "")); shortURL != "" {
		film.ShortURL = shortURL
	}

	jsonLD, err := extractJSONLD(body)
	if err != nil {
		return Film{}, err
	}

	var raw filmJSONLD
	if err := json.Unmarshal(jsonLD, &raw); err != nil {
		return Film{}, fmt.Errorf("parse film json-ld: %w", err)
	}
	film = mergeFilmFromJSONLD(film, raw)

	if film.RuntimeMinutes == 0 {
		runtimeText := runtimePattern.FindStringSubmatch(string(body))
		if len(runtimeText) == 2 {
			film.RuntimeMinutes = numberFromString(runtimeText[1])
		}
	}

	if len(film.Cast) == 0 {
		document.Find("#tab-cast a.text-slug.tooltip").Each(func(index int, selection *goquery.Selection) {
			if index >= 10 {
				return
			}
			castMember := strings.TrimSpace(selection.Text())
			if castMember != "" {
				film.Cast = append(film.Cast, castMember)
			}
		})
	}

	return film, nil
}

func (c *Client) fetchRSS(ctx context.Context, username string) ([]rssItem, error) {
	username = strings.Trim(strings.TrimSpace(username), "/")
	body, err := c.fetchCached(ctx, "rss:"+strings.ToLower(username), "https://letterboxd.com/"+username+"/rss/")
	if err != nil {
		return nil, err
	}

	var document rssDocument
	if err := xml.Unmarshal(body, &document); err != nil {
		return nil, fmt.Errorf("parse rss feed: %w", err)
	}

	items := make([]rssItem, 0, len(document.Channel.Items))
	for _, item := range document.Channel.Items {
		items = append(items, parseRSSItem(item))
	}

	slices.SortFunc(items, func(left, right rssItem) int {
		switch {
		case left.PublishedAt.After(right.PublishedAt):
			return -1
		case left.PublishedAt.Before(right.PublishedAt):
			return 1
		default:
			return 0
		}
	})

	return items, nil
}

func (c *Client) fetchCached(ctx context.Context, cacheKey, requestURL string) ([]byte, error) {
	if body, ok := c.cache.get(cacheKey); ok {
		return body, nil
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	request.Header.Set("User-Agent", c.userAgent)
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %s", response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if looksLikeChallengePage(body) {
		return nil, ErrCloudflareChallenge
	}

	c.cache.set(cacheKey, body)
	return body, nil
}

func looksLikeChallengePage(body []byte) bool {
	document := strings.ToLower(string(body))
	return strings.Contains(document, "just a moment") &&
		(strings.Contains(document, "challenge-error-text") || strings.Contains(document, "__cf_chl_opt"))
}

func extractJSONLD(body []byte) ([]byte, error) {
	matches := jsonLDPattern.FindSubmatch(body)
	if len(matches) != 2 {
		return nil, fmt.Errorf("film json-ld not found")
	}
	return matches[1], nil
}

func absoluteLetterboxdURL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return "https://letterboxd.com" + path
}

func normalizeFilmURL(raw string) string {
	if raw == "" {
		return raw
	}
	if !strings.Contains(raw, "://") {
		raw = absoluteLetterboxdURL(raw)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	if strings.EqualFold(parsed.Host, "boxd.it") {
		return raw
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "film" {
			return "https://letterboxd.com/film/" + parts[i+1] + "/"
		}
	}

	return raw
}

func looksLikeFilmURL(value string) bool {
	return strings.Contains(value, "letterboxd.com/film/") ||
		strings.Contains(value, "/film/") ||
		strings.Contains(value, "boxd.it/")
}

func resolveDuckDuckGoResult(raw string) string {
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	if parsed.Host != "duckduckgo.com" {
		return raw
	}

	target := parsed.Query().Get("uddg")
	if target == "" {
		return raw
	}

	unescaped, err := url.QueryUnescape(target)
	if err != nil {
		return target
	}
	return unescaped
}

func filmTitleFromURL(raw string) string {
	slug := slugFromURL(raw)
	slug = strings.ReplaceAll(slug, "-", " ")
	words := strings.Fields(strings.ToLower(slug))
	for i, word := range words {
		firstRune, size := utf8.DecodeRuneInString(word)
		words[i] = string(unicode.ToTitle(firstRune)) + word[size:]
	}
	return strings.Join(words, " ")
}

type filmJSONLD struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Image  string `json:"image"`
	Genre  []string
	Actors []struct {
		Name string `json:"name"`
	} `json:"actors"`
	Director []struct {
		Name string `json:"name"`
	} `json:"director"`
	ReleasedEvent []struct {
		StartDate string `json:"startDate"`
	} `json:"releasedEvent"`
	AggregateRating struct {
		RatingValue float64 `json:"ratingValue"`
		RatingCount int     `json:"ratingCount"`
		ReviewCount int     `json:"reviewCount"`
	} `json:"aggregateRating"`
}

func mergeFilmFromJSONLD(base Film, raw filmJSONLD) Film {
	base.Title = raw.Name
	base.URL = strings.TrimSpace(raw.URL)
	base.PosterURL = strings.TrimSpace(raw.Image)
	base.Genres = append([]string(nil), raw.Genre...)
	base.Rating = raw.AggregateRating.RatingValue
	base.RatingCount = raw.AggregateRating.RatingCount
	base.ReviewCount = raw.AggregateRating.ReviewCount

	if len(raw.ReleasedEvent) > 0 {
		base.Year = numberFromString(raw.ReleasedEvent[0].StartDate)
	}
	if len(raw.Director) > 0 {
		base.Director = raw.Director[0].Name
	}
	for _, actor := range raw.Actors {
		if actor.Name == "" {
			continue
		}
		base.Cast = append(base.Cast, actor.Name)
		if len(base.Cast) >= 10 {
			break
		}
	}

	return base
}

type rssDocument struct {
	Channel struct {
		Items []rssXMLItem `xml:"item"`
	} `xml:"channel"`
}

type rssXMLItem struct {
	Title        string `xml:"title"`
	Link         string `xml:"link"`
	GUID         string `xml:"guid"`
	PubDate      string `xml:"pubDate"`
	WatchedDate  string `xml:"watchedDate"`
	Rewatch      string `xml:"rewatch"`
	FilmTitle    string `xml:"filmTitle"`
	FilmYear     string `xml:"filmYear"`
	MemberRating string `xml:"memberRating"`
	MemberLike   string `xml:"memberLike"`
	Description  string `xml:"description"`
}

type rssItem struct {
	DiaryEntry
	Title string
	List  ListSummary
}

func parseRSSItem(item rssXMLItem) rssItem {
	publishedAt, _ := time.Parse(time.RFC1123Z, strings.TrimSpace(item.PubDate))
	description := html.UnescapeString(strings.TrimSpace(item.Description))
	excerpt, posterURL := parseDescription(description)
	entryType := "review"
	switch {
	case strings.Contains(item.GUID, "letterboxd-list"):
		entryType = "list"
	case strings.Contains(item.GUID, "letterboxd-watch"):
		entryType = "watch"
	}

	parsed := rssItem{
		Title: item.Title,
		DiaryEntry: DiaryEntry{
			ID:          item.GUID,
			EntryType:   entryType,
			FilmTitle:   strings.TrimSpace(item.FilmTitle),
			FilmYear:    numberFromString(item.FilmYear),
			FilmURL:     strings.TrimSpace(item.Link),
			FilmSlug:    slugFromURL(item.Link),
			WatchedDate: strings.TrimSpace(item.WatchedDate),
			PublishedAt: publishedAt,
			Rewatch:     strings.EqualFold(strings.TrimSpace(item.Rewatch), "yes"),
			Rating:      parseFloat(item.MemberRating),
			Liked:       strings.EqualFold(strings.TrimSpace(item.MemberLike), "yes"),
			Excerpt:     excerpt,
			PosterURL:   posterURL,
		},
	}

	if entryType == "list" {
		parsed.List = ListSummary{
			ID:          item.GUID,
			Title:       strings.TrimSpace(item.Title),
			URL:         strings.TrimSpace(item.Link),
			PublishedAt: publishedAt,
			Excerpt:     excerpt,
			Films:       parseListFilms(description),
		}
	}

	return parsed
}

func (r rssItem) ListSummary() ListSummary {
	return r.List
}

func parseDescription(rawHTML string) (string, string) {
	document, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return strings.TrimSpace(rawHTML), ""
	}

	posterURL := strings.TrimSpace(document.Find("img").First().AttrOr("src", ""))
	document.Find("img").Remove()
	text := strings.TrimSpace(document.Text())
	text = strings.Join(strings.Fields(text), " ")
	return text, posterURL
}

func parseListFilms(rawHTML string) []FavoriteFilm {
	document, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return nil
	}

	films := make([]FavoriteFilm, 0, 5)
	document.Find("a[href*=\"/film/\"]").Each(func(index int, selection *goquery.Selection) {
		if index >= 5 {
			return
		}
		title := strings.TrimSpace(selection.Text())
		link, ok := selection.Attr("href")
		if !ok || title == "" {
			return
		}
		films = append(films, FavoriteFilm{
			Title: title,
			URL:   strings.TrimSpace(link),
			Slug:  slugFromURL(link),
		})
	})

	return films
}

func yearFromTitle(value string) int {
	pattern := regexp.MustCompile(`\(([0-9]{4})\)`)
	matches := pattern.FindStringSubmatch(value)
	if len(matches) != 2 {
		return 0
	}
	return numberFromString(matches[1])
}

func parseFloat(value string) float64 {
	number, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return number
}
