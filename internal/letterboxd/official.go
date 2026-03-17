package letterboxd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultDiscoverSort = "FilmPopularityThisWeek"
	defaultDiscoverSize = 6
	maxOfficialPageSize = 100
)

func (c *Client) FetchMemberStats(ctx context.Context, username string) (MemberStats, error) {
	if !c.HasOfficialAPI() {
		return MemberStats{}, fmt.Errorf("official Letterboxd API credentials are not configured")
	}

	username = strings.ToLower(strings.Trim(strings.TrimSpace(username), "/"))
	if username == "" {
		return MemberStats{}, fmt.Errorf("username is required")
	}

	memberID, err := c.resolveMemberID(ctx, username)
	if err != nil {
		return MemberStats{}, err
	}

	var response apiMemberStatisticsResponse
	if err := c.apiGetJSON(ctx, "stats:"+username, "/member/"+memberID+"/statistics", nil, &response); err != nil {
		return MemberStats{}, err
	}

	stats := MemberStats{
		Member:           response.Member.toSummary(),
		Counts:           response.Counts.toModel(),
		RatingsHistogram: make([]RatingsHistogramBar, 0, len(response.RatingsHistogram)),
		YearsInReview:    append([]int(nil), response.YearsInReview...),
	}
	for _, bar := range response.RatingsHistogram {
		stats.RatingsHistogram = append(stats.RatingsHistogram, RatingsHistogramBar(bar))
	}
	if stats.Member.Username == "" {
		stats.Member.Username = username
	}

	return stats, nil
}

func (c *Client) FetchWatchlist(ctx context.Context, username string, limit int, genreQuery string) ([]FilmSummary, error) {
	if !c.HasOfficialAPI() {
		return nil, fmt.Errorf("official Letterboxd API credentials are not configured")
	}

	username = strings.ToLower(strings.Trim(strings.TrimSpace(username), "/"))
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}

	memberID, err := c.resolveMemberID(ctx, username)
	if err != nil {
		return nil, err
	}

	limit = clampOfficialLimit(limit, defaultDiscoverSize)
	values := url.Values{}
	values.Set("perPage", strconv.Itoa(limit))
	if genreQuery != "" {
		genreID, _, err := c.resolveGenre(ctx, genreQuery)
		if err != nil {
			return nil, err
		}
		values.Set("genre", genreID)
	}

	var response apiFilmItemsResponse
	if err := c.apiGetJSON(ctx, "watchlist:"+username+":"+normalizeText(genreQuery)+":"+strconv.Itoa(limit), "/member/"+memberID+"/watchlist", values, &response); err != nil {
		return nil, err
	}

	return mapFilmSummaries(response.Items, limit), nil
}

func (c *Client) FetchActivity(ctx context.Context, username string, limit int) ([]ActivityItem, error) {
	if !c.HasOfficialAPI() {
		return nil, fmt.Errorf("official Letterboxd API credentials are not configured")
	}

	username = strings.ToLower(strings.Trim(strings.TrimSpace(username), "/"))
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}

	memberID, err := c.resolveMemberID(ctx, username)
	if err != nil {
		return nil, err
	}

	limit = clampOfficialLimit(limit, 5)
	values := url.Values{}
	values.Set("perPage", strconv.Itoa(limit))

	var response apiActivityResponse
	if err := c.apiGetJSON(ctx, "activity:"+username+":"+strconv.Itoa(limit), "/member/"+memberID+"/activity", values, &response); err != nil {
		return nil, err
	}

	items := make([]ActivityItem, 0, len(response.Items))
	for _, raw := range response.Items {
		item, ok := decodeActivityItem(raw)
		if ok {
			items = append(items, item)
		}
		if len(items) >= limit {
			break
		}
	}

	return items, nil
}

func (c *Client) DiscoverFilms(ctx context.Context, options DiscoveryOptions) ([]FilmSummary, error) {
	if !c.HasOfficialAPI() {
		return nil, fmt.Errorf("official Letterboxd API credentials are not configured")
	}

	limit := clampOfficialLimit(options.Limit, defaultDiscoverSize)
	values := url.Values{}
	values.Set("perPage", strconv.Itoa(limit))
	values.Set("sort", emptyDefault(strings.TrimSpace(options.Sort), defaultDiscoverSort))
	values.Add("where", "Released")
	values.Add("where", "FeatureLength")
	values.Set("adult", strconv.FormatBool(options.Adult))

	if options.Genre != "" {
		genreID, _, err := c.resolveGenre(ctx, options.Genre)
		if err != nil {
			return nil, err
		}
		values.Set("genre", genreID)
	}

	if options.Service != "" {
		serviceID, _, err := c.resolveService(ctx, options.Service)
		if err != nil {
			return nil, err
		}
		values.Set("service", serviceID)
	}

	var response apiFilmItemsResponse
	cacheKey := "discover:" + normalizeText(options.Genre) + ":" + normalizeText(options.Service) + ":" + values.Get("sort") + ":" + strconv.Itoa(limit)
	if err := c.apiGetJSON(ctx, cacheKey, "/films", values, &response); err != nil {
		return nil, err
	}

	return mapFilmSummaries(response.Items, limit), nil
}

func (c *Client) searchFilmOfficial(ctx context.Context, query string, limit int) ([]FilmSearchResult, error) {
	items, err := c.searchFilmsOfficial(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	results := make([]FilmSearchResult, 0, len(items))
	for _, item := range items {
		summary := item.toSummary()
		results = append(results, FilmSearchResult{
			Title: summary.Title,
			URL:   summary.URL,
			Slug:  slugFromURL(summary.URL),
		})
	}
	return results, nil
}

func (c *Client) fetchFilmOfficial(ctx context.Context, query string) (Film, error) {
	filmID, seed, err := c.resolveFilm(ctx, query)
	if err != nil {
		return Film{}, err
	}

	var raw apiFilm
	if err := c.apiGetJSON(ctx, "film-id:"+filmID, "/film/"+filmID, nil, &raw); err != nil {
		if seed.ID != "" {
			return seed.toFilm(), nil
		}
		return Film{}, err
	}

	film := raw.toFilm()
	if film.Title == "" && seed.ID != "" {
		return seed.toFilm(), nil
	}
	return film, nil
}

func (c *Client) resolveFilm(ctx context.Context, query string) (string, apiFilm, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", apiFilm{}, fmt.Errorf("film query is required")
	}

	if looksLikeFilmURL(query) {
		filmURL := normalizeFilmURL(query)
		filmID, err := c.resolveLetterboxdIdentifier(ctx, "film-id:"+filmURL, filmURL)
		return filmID, apiFilm{}, err
	}

	results, err := c.searchFilmsOfficial(ctx, query, 1)
	if err != nil {
		return "", apiFilm{}, err
	}
	if len(results) == 0 {
		return "", apiFilm{}, fmt.Errorf("no Letterboxd film match found for %q", query)
	}

	return results[0].ID, results[0], nil
}

func (c *Client) searchFilmsOfficial(ctx context.Context, query string, limit int) ([]apiFilm, error) {
	if !c.HasOfficialAPI() {
		return nil, fmt.Errorf("official Letterboxd API credentials are not configured")
	}

	limit = clampOfficialLimit(limit, 5)
	values := url.Values{}
	values.Set("input", query)
	values.Set("include", "FilmSearchItem")
	values.Set("perPage", strconv.Itoa(limit))
	values.Set("searchMethod", "FullText")

	var response apiSearchResponse
	if err := c.apiGetJSON(ctx, "official-search:"+normalizeText(query)+":"+strconv.Itoa(limit), "/search", values, &response); err != nil {
		return nil, err
	}

	items := make([]apiFilm, 0, len(response.Items))
	for _, raw := range response.Items {
		var item struct {
			Type string  `json:"type"`
			Film apiFilm `json:"film"`
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			continue
		}
		if item.Type != "FilmSearchItem" || item.Film.ID == "" {
			continue
		}
		items = append(items, item.Film)
	}

	return items, nil
}

func (c *Client) resolveMemberID(ctx context.Context, username string) (string, error) {
	username = strings.ToLower(strings.Trim(strings.TrimSpace(username), "/"))
	if username == "" {
		return "", fmt.Errorf("username is required")
	}

	return c.resolveLetterboxdIdentifier(ctx, "member-id:"+username, c.webBaseURL+"/"+username+"/")
}

func (c *Client) resolveLetterboxdIdentifier(ctx context.Context, cacheKey, requestURL string) (string, error) {
	if cached, ok := c.cache.get(cacheKey); ok {
		if identifier := strings.TrimSpace(string(cached)); identifier != "" {
			return identifier, nil
		}
	}

	for _, method := range []string{http.MethodHead, http.MethodGet} {
		request, err := http.NewRequestWithContext(ctx, method, requestURL, nil)
		if err != nil {
			return "", fmt.Errorf("create %s request: %w", strings.ToLower(method), err)
		}
		request.Header.Set("User-Agent", c.userAgent)

		response, err := c.httpClient.Do(request)
		if err != nil {
			return "", fmt.Errorf("send %s request: %w", strings.ToLower(method), err)
		}

		identifier := strings.TrimSpace(response.Header.Get("x-letterboxd-identifier"))
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()

		if response.StatusCode >= 400 {
			if method == http.MethodHead {
				continue
			}
			return "", fmt.Errorf("resolve Letterboxd id failed with status %s", response.Status)
		}
		if identifier == "" {
			if method == http.MethodHead {
				continue
			}
			return "", fmt.Errorf("letterboxd identifier header missing for %s", requestURL)
		}

		c.cache.set(cacheKey, []byte(identifier))
		return identifier, nil
	}

	return "", fmt.Errorf("could not resolve Letterboxd identifier for %s", requestURL)
}

func (c *Client) resolveGenre(ctx context.Context, query string) (string, string, error) {
	return c.resolveNamedID(ctx, query, "/films/genres", "official genres")
}

func (c *Client) resolveService(ctx context.Context, query string) (string, string, error) {
	return c.resolveNamedID(ctx, query, "/films/film-services", "official services")
}

func (c *Client) resolveNamedID(ctx context.Context, query, path, label string) (string, string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", "", fmt.Errorf("%s query is required", label)
	}

	var response apiNamedItemsResponse
	if err := c.apiGetJSON(ctx, path, path, nil, &response); err != nil {
		return "", "", err
	}

	target := normalizeText(query)
	best := apiNamedItem{}
	for _, item := range response.Items {
		name := item.displayName()
		normalized := normalizeText(name)
		switch {
		case normalized == target:
			return item.ID, name, nil
		case best.ID == "" && (strings.Contains(normalized, target) || strings.Contains(target, normalized)):
			best = item
		}
	}

	if best.ID == "" {
		return "", "", fmt.Errorf("no %s match found for %q", label, query)
	}

	return best.ID, best.displayName(), nil
}

func (c *Client) apiGetJSON(ctx context.Context, cacheKey, path string, values url.Values, target any) error {
	body, err := c.apiGet(ctx, cacheKey, path, values)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode official api response: %w", err)
	}
	return nil
}

func (c *Client) apiGet(ctx context.Context, cacheKey, path string, values url.Values) ([]byte, error) {
	if body, ok := c.cache.get(cacheKey); ok {
		return body, nil
	}

	body, err := c.apiGetWithRetry(ctx, path, values)
	if err != nil {
		return nil, err
	}

	c.cache.set(cacheKey, body)
	return body, nil
}

func (c *Client) apiGetWithRetry(ctx context.Context, path string, values url.Values) ([]byte, error) {
	token, err := c.accessToken(ctx)
	if err != nil {
		return nil, err
	}

	body, status, err := c.apiRequest(ctx, path, values, token)
	if err != nil {
		return nil, err
	}
	if status != http.StatusUnauthorized {
		return body, nil
	}

	c.invalidateAccessToken()
	token, err = c.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	body, _, err = c.apiRequest(ctx, path, values, token)
	return body, err
}

func (c *Client) apiRequest(ctx context.Context, path string, values url.Values, token string) ([]byte, int, error) {
	requestURL := c.baseURL + path
	if len(values) > 0 {
		requestURL += "?" + values.Encode()
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create official api request: %w", err)
	}
	request.Header.Set("User-Agent", c.userAgent)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, 0, fmt.Errorf("send official api request: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, response.StatusCode, fmt.Errorf("read official api response: %w", err)
	}
	if response.StatusCode >= 400 && response.StatusCode != http.StatusUnauthorized {
		return nil, response.StatusCode, fmt.Errorf("official api request failed with status %s", response.Status)
	}
	return body, response.StatusCode, nil
}

func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.officialMu.Lock()
	defer c.officialMu.Unlock()

	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return c.token, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("create official api token request: %w", err)
	}
	request.SetBasicAuth(c.official.ClientID, c.official.ClientSecret)
	request.Header.Set("User-Agent", c.userAgent)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("send official api token request: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode >= 400 {
		return "", fmt.Errorf("official api token request failed with status %s", response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("read official api token response: %w", err)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("decode official api token response: %w", err)
	}
	if payload.AccessToken == "" {
		return "", fmt.Errorf("official api token response did not include an access token")
	}

	expiry := time.Now().Add(55 * time.Minute)
	if payload.ExpiresIn > 0 {
		expiry = time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)
	}
	if time.Until(expiry) > time.Minute {
		expiry = expiry.Add(-time.Minute)
	}

	c.token = payload.AccessToken
	c.tokenExpiry = expiry
	return c.token, nil
}

func (c *Client) invalidateAccessToken() {
	c.officialMu.Lock()
	defer c.officialMu.Unlock()
	c.token = ""
	c.tokenExpiry = time.Time{}
}

type apiSearchResponse struct {
	Items []json.RawMessage `json:"items"`
}

type apiActivityResponse struct {
	Items []json.RawMessage `json:"items"`
}

type apiFilmItemsResponse struct {
	Items []apiFilm `json:"items"`
}

type apiNamedItemsResponse struct {
	Items []apiNamedItem `json:"items"`
}

type apiNamedItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

func (n apiNamedItem) displayName() string {
	if strings.TrimSpace(n.DisplayName) != "" {
		return strings.TrimSpace(n.DisplayName)
	}
	return strings.TrimSpace(n.Name)
}

type apiMemberStatisticsResponse struct {
	Member           apiMember         `json:"member"`
	Counts           apiCounts         `json:"counts"`
	YearsInReview    []int             `json:"yearsInReview"`
	RatingsHistogram []apiHistogramBar `json:"ratingsHistogram"`
}

type apiCounts struct {
	FilmLikes            int `json:"filmLikes"`
	ListLikes            int `json:"listLikes"`
	ReviewLikes          int `json:"reviewLikes"`
	StoryLikes           int `json:"storyLikes"`
	Watches              int `json:"watches"`
	Ratings              int `json:"ratings"`
	Reviews              int `json:"reviews"`
	DiaryEntries         int `json:"diaryEntries"`
	DiaryEntriesThisYear int `json:"diaryEntriesThisYear"`
	FilmsInDiaryThisYear int `json:"filmsInDiaryThisYear"`
	FilmsInDiaryLastYear int `json:"filmsInDiaryLastYear"`
	Watchlist            int `json:"watchlist"`
	Lists                int `json:"lists"`
	UnpublishedLists     int `json:"unpublishedLists"`
	AccessedSharedLists  int `json:"accessedSharedLists"`
	Followers            int `json:"followers"`
	Following            int `json:"following"`
	ListTags             int `json:"listTags"`
	FilmTags             int `json:"filmTags"`
}

func (c apiCounts) toModel() MemberStatisticsCounts {
	return MemberStatisticsCounts(c)
}

type apiHistogramBar struct {
	Rating float64 `json:"rating"`
	Count  int     `json:"count"`
}

type apiMember struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	ShortName   string   `json:"shortName"`
	Avatar      apiImage `json:"avatar"`
}

func (m apiMember) toSummary() MemberSummary {
	return MemberSummary{
		ID:          m.ID,
		Username:    strings.ToLower(strings.TrimSpace(m.Username)),
		DisplayName: strings.TrimSpace(m.DisplayName),
		ShortName:   strings.TrimSpace(m.ShortName),
		AvatarURL:   m.Avatar.bestURL(),
	}
}

type apiFilm struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	ReleaseYear int            `json:"releaseYear"`
	RunTime     int            `json:"runTime"`
	Poster      apiImage       `json:"poster"`
	Links       []apiLink      `json:"links"`
	Statistics  apiFilmStats   `json:"statistics"`
	Genres      []apiNamedItem `json:"genres"`
	Directors   []apiPerson    `json:"directors"`
	Cast        []apiPerson    `json:"cast"`
}

type apiFilmStats struct {
	Rating      float64 `json:"rating"`
	RatingCount int     `json:"ratingCount"`
	ReviewCount int     `json:"reviewCount"`
}

type apiPerson struct {
	Name string `json:"name"`
}

func (f apiFilm) toSummary() FilmSummary {
	return FilmSummary{
		ID:             f.ID,
		Title:          strings.TrimSpace(f.Name),
		Year:           f.ReleaseYear,
		URL:            preferredLink(f.Links, "letterboxd.com"),
		ShortURL:       preferredLink(f.Links, "boxd.it"),
		PosterURL:      f.Poster.bestURL(),
		Rating:         f.Statistics.Rating,
		RatingCount:    f.Statistics.RatingCount,
		ReviewCount:    f.Statistics.ReviewCount,
		RuntimeMinutes: f.RunTime,
		Genres:         namedItemsToStrings(f.Genres),
		Directors:      peopleToStrings(f.Directors, 3),
	}
}

func (f apiFilm) toFilm() Film {
	summary := f.toSummary()
	return Film{
		Title:          summary.Title,
		Year:           summary.Year,
		URL:            summary.URL,
		ShortURL:       summary.ShortURL,
		PosterURL:      summary.PosterURL,
		Description:    strings.TrimSpace(f.Description),
		Director:       strings.Join(summary.Directors, ", "),
		RuntimeMinutes: summary.RuntimeMinutes,
		Rating:         summary.Rating,
		RatingCount:    summary.RatingCount,
		ReviewCount:    summary.ReviewCount,
		Genres:         append([]string(nil), summary.Genres...),
		Cast:           peopleToStrings(f.Cast, 10),
	}
}

type apiList struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Links    []apiLink `json:"links"`
	WhenMade string    `json:"whenMade"`
}

func (l apiList) toSummary() *ListSummary {
	if l.ID == "" && l.Name == "" {
		return nil
	}
	return &ListSummary{
		ID:          l.ID,
		Title:       strings.TrimSpace(l.Name),
		URL:         preferredLink(l.Links, "letterboxd.com"),
		PublishedAt: parseAPITime(l.WhenMade),
	}
}

type apiLogEntry struct {
	ID          string    `json:"id"`
	Links       []apiLink `json:"links"`
	Film        apiFilm   `json:"film"`
	WhenCreated string    `json:"whenCreated"`
	WatchedDate string    `json:"watchedDate"`
	Rating      float64   `json:"rating"`
	Rewatch     bool      `json:"rewatch"`
	Liked       bool      `json:"liked"`
	Review      string    `json:"review"`
}

func (l apiLogEntry) toSummary() *LogEntrySummary {
	if l.ID == "" && l.Film.ID == "" {
		return nil
	}
	return &LogEntrySummary{
		ID:          l.ID,
		URL:         preferredLink(l.Links, "letterboxd.com"),
		Film:        l.Film.toSummary(),
		WatchedAt:   parseAPITime(l.WhenCreated),
		WatchedDate: strings.TrimSpace(l.WatchedDate),
		Rating:      l.Rating,
		Rewatch:     l.Rewatch,
		Liked:       l.Liked,
		Excerpt:     strings.TrimSpace(l.Review),
	}
}

type apiActivityCommon struct {
	Type         string      `json:"type"`
	WhenCreated  string      `json:"whenCreated"`
	Member       apiMember   `json:"member"`
	Film         apiFilm     `json:"film"`
	List         apiList     `json:"list"`
	TargetMember apiMember   `json:"targetMember"`
	LogEntry     apiLogEntry `json:"logEntry"`
	Rating       float64     `json:"rating"`
	Response     string      `json:"response"`
	Comment      string      `json:"comment"`
}

func decodeActivityItem(raw json.RawMessage) (ActivityItem, bool) {
	var item apiActivityCommon
	if err := json.Unmarshal(raw, &item); err != nil {
		return ActivityItem{}, false
	}

	activity := ActivityItem{
		Type:        strings.TrimSpace(item.Type),
		WhenCreated: parseAPITime(item.WhenCreated),
		Member:      item.Member.toSummary(),
		Rating:      item.Rating,
		Response:    strings.TrimSpace(item.Response),
		Comment:     strings.TrimSpace(item.Comment),
	}
	if summary := item.List.toSummary(); summary != nil {
		activity.List = summary
	}
	if followed := item.TargetMember.toSummary(); followed.ID != "" || followed.Username != "" {
		activity.Followed = &followed
	}
	if item.Film.ID != "" || item.Film.Name != "" {
		film := item.Film.toSummary()
		activity.Film = &film
	}
	if logEntry := item.LogEntry.toSummary(); logEntry != nil {
		activity.LogEntry = logEntry
		if activity.Film == nil && (logEntry.Film.ID != "" || logEntry.Film.Title != "") {
			film := logEntry.Film
			activity.Film = &film
		}
		if activity.Rating == 0 {
			activity.Rating = logEntry.Rating
		}
	}

	return activity, true
}

type apiLink struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type apiImage struct {
	URL   string            `json:"url"`
	Sizes map[string]string `json:"sizes"`
}

func (i apiImage) bestURL() string {
	if strings.TrimSpace(i.URL) != "" {
		return strings.TrimSpace(i.URL)
	}

	best := ""
	bestSize := -1
	for key, value := range i.Sizes {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if size, err := strconv.Atoi(key); err == nil && size > bestSize {
			best = value
			bestSize = size
			continue
		}
		if best == "" {
			best = value
		}
	}
	return strings.TrimSpace(best)
}

func mapFilmSummaries(items []apiFilm, limit int) []FilmSummary {
	capacity := len(items)
	if limit < capacity {
		capacity = limit
	}
	films := make([]FilmSummary, 0, capacity)
	for _, item := range items {
		films = append(films, item.toSummary())
		if len(films) >= limit {
			break
		}
	}
	return films
}

func namedItemsToStrings(items []apiNamedItem) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		if name := item.displayName(); name != "" {
			values = append(values, name)
		}
	}
	return values
}

func peopleToStrings(items []apiPerson, limit int) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		if name := strings.TrimSpace(item.Name); name != "" {
			values = append(values, name)
		}
		if limit > 0 && len(values) >= limit {
			break
		}
	}
	return values
}

func preferredLink(links []apiLink, contains string) string {
	for _, link := range links {
		if strings.Contains(strings.ToLower(link.URL), strings.ToLower(contains)) {
			return strings.TrimSpace(link.URL)
		}
	}
	for _, link := range links {
		if strings.TrimSpace(link.URL) != "" {
			return strings.TrimSpace(link.URL)
		}
	}
	return ""
}

func parseAPITime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}

	layouts := []string{time.RFC3339, "2006-01-02"}
	for _, layout := range layouts {
		if value, err := time.Parse(layout, raw); err == nil {
			return value
		}
	}
	return time.Time{}
}

func clampOfficialLimit(value, fallback int) int {
	if value <= 0 {
		value = fallback
	}
	if value > maxOfficialPageSize {
		value = maxOfficialPageSize
	}
	return value
}

func emptyDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
