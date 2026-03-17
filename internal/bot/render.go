package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/asish/flicksy/internal/letterboxd"
	"github.com/asish/flicksy/internal/social"
	"github.com/asish/flicksy/internal/store"
)

const (
	accentColor = 0x00B4D8
	alertColor  = 0xF77F00
)

func helpEmbed(includeOfficialAPI bool) *discordgo.MessageEmbed {
	fields := []*discordgo.MessageEmbedField{
		{Name: "Get started", Value: "`/connect username` links your default profile so most commands work without repeating it.", Inline: false},
		{Name: "Profiles", Value: "`/profile`, `/diary`, `/film`, `/list`, `/logged`", Inline: false},
		{Name: "Channel feeds", Value: "`/follow`, `/unfollow`, `/following`", Inline: false},
		{Name: "Social features", Value: "`/compare`, `/taste`, `/roulette`", Inline: false},
		{Name: "Cache", Value: "`/refresh` clears cached Letterboxd data if something looks stale.", Inline: false},
	}
	if includeOfficialAPI {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Official API",
			Value: "`/stats`, `/watchlist`, `/watchpick`, `/activity`, `/discover`",
		})
	}

	return &discordgo.MessageEmbed{
		Title:       "Flicksy",
		Description: "A Letterboxd Discord bot built for easy profile lookups, channel follows, taste comparisons, and API-backed discovery.",
		Color:       accentColor,
		Fields:      fields,
	}
}

func profileEmbed(profile letterboxd.Profile) *discordgo.MessageEmbed {
	description := profile.Bio
	if description == "" {
		description = "No public bio."
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s (@%s)", profile.DisplayName, profile.Username),
		URL:         profile.URL,
		Description: description,
		Color:       accentColor,
		Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: profile.AvatarURL},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Stats", Value: fmt.Sprintf("Films: `%s`\nThis year: `%s`\nLists: `%s`\nFollowing: `%s`\nFollowers: `%s`", comma(profile.FilmCount), comma(profile.ThisYearCount), comma(profile.ListCount), comma(profile.FollowingCount), comma(profile.FollowersCount)), Inline: true},
			{Name: "Favorites", Value: favoriteList(profile.Favorites), Inline: true},
		},
	}
}

func statsEmbed(username string, stats letterboxd.MemberStats) *discordgo.MessageEmbed {
	memberLabel := statsMemberLabel(stats)
	if memberLabel == "" {
		memberLabel = "@" + username
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: "Counts", Value: statsCountSummary(stats), Inline: false},
	}
	if years := statsYearsInReview(stats); len(years) > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Years in review",
			Value:  strings.Join(years, ", "),
			Inline: false,
		})
	}
	if histogram := statsRatingHistogram(stats); histogram != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Ratings spread",
			Value:  histogram,
			Inline: false,
		})
	}

	return &discordgo.MessageEmbed{
		Title:     fmt.Sprintf("Letterboxd stats for %s", memberLabel),
		Color:     accentColor,
		Thumbnail: &discordgo.MessageEmbedThumbnail{URL: stats.Member.AvatarURL},
		Fields:    fields,
	}
}

func watchlistEmbed(username, genre string, films []letterboxd.FilmSummary) *discordgo.MessageEmbed {
	fields := []*discordgo.MessageEmbedField{
		{Name: "Films", Value: filmSummaryList(films), Inline: false},
	}
	if genre != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Filter", Value: genre, Inline: true})
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s's watchlist", displayUsername(username)),
		Description: fmt.Sprintf("Showing %d film%s.", len(films), pluralize(len(films))),
		Color:       accentColor,
		Thumbnail:   filmSummaryThumbnail(films),
		Fields:      fields,
	}
}

func watchpickEmbed(username, genre string, film letterboxd.FilmSummary) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s's watchpick", displayUsername(username)),
		Description: fmt.Sprintf("Random pick from the watchlist%s.", filterSuffix(genre)),
		Color:       alertColor,
		URL:         filmSummaryURL(film),
		Thumbnail:   filmSummaryThumbnail([]letterboxd.FilmSummary{film}),
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Film", Value: filmSummaryLine(film), Inline: false},
		},
	}
}

func activityEmbed(username string, items []letterboxd.ActivityItem) *discordgo.MessageEmbed {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, activityItemLine(item))
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Recent activity for %s", displayUsername(username)),
		Description: strings.Join(lines, "\n"),
		Color:       accentColor,
	}
}

func discoverEmbed(genre, service string, films []letterboxd.FilmSummary) *discordgo.MessageEmbed {
	description := "Trending recommendations from the official Letterboxd catalog."
	if genre != "" || service != "" {
		description = fmt.Sprintf("Filtered%s%s.", filterSuffix(genre), serviceSuffix(service))
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: "Films", Value: filmSummaryList(films), Inline: false},
	}
	if genre != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Genre", Value: genre, Inline: true})
	}
	if service != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Service", Value: service, Inline: true})
	}
	fields = append(fields, &discordgo.MessageEmbedField{Name: "Sort", Value: "Popularity this week", Inline: true})

	return &discordgo.MessageEmbed{
		Title:       "Discover films",
		Description: description,
		Color:       alertColor,
		Thumbnail:   filmSummaryThumbnail(films),
		Fields:      fields,
	}
}

func diaryEmbed(username string, entries []letterboxd.DiaryEntry) *discordgo.MessageEmbed {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("• [%s (%d)](%s) — %s%s%s",
			entry.FilmTitle,
			entry.FilmYear,
			entry.FilmURL,
			ratingLabel(entry.Rating),
			boolLabel(entry.Liked, " liked", ""),
			boolLabel(entry.Rewatch, " rewatch", ""),
		))
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Recent diary for @%s", username),
		Description: strings.Join(lines, "\n"),
		Color:       accentColor,
	}
}

func filmEmbed(film letterboxd.Film, contextLabel string) *discordgo.MessageEmbed {
	url := film.URL
	if film.ShortURL != "" {
		url = film.ShortURL
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s (%d)", film.Title, film.Year),
		URL:         url,
		Description: truncate(film.Description, 400),
		Color:       alertColor,
		Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: film.PosterURL},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Director", Value: emptyFallback(film.Director, "Unknown"), Inline: true},
			{Name: "Rating", Value: fmt.Sprintf("%.2f/5 from %s ratings", film.Rating, comma(film.RatingCount)), Inline: true},
			{Name: "Runtime", Value: emptyFallback(strconv.Itoa(film.RuntimeMinutes)+" mins", "Unknown"), Inline: true},
			{Name: "Genres", Value: emptyFallback(strings.Join(film.Genres, ", "), "Unknown"), Inline: false},
			{Name: "Cast", Value: emptyFallback(strings.Join(limitStrings(film.Cast, 8), ", "), "Unknown"), Inline: false},
			{Name: "Source", Value: contextLabel, Inline: false},
		},
	}
}

func listEmbed(username string, listSummary letterboxd.ListSummary) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s — @%s", listSummary.Title, username),
		URL:         listSummary.URL,
		Description: truncate(listSummary.Excerpt, 350),
		Color:       accentColor,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Films", Value: favoriteList(listSummary.Films), Inline: false},
			{Name: "Published", Value: formatTime(listSummary.PublishedAt), Inline: true},
		},
	}
}

func followingEmbed(channelID string, subscriptions []store.FollowSubscription) *discordgo.MessageEmbed {
	lines := make([]string, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		lines = append(lines, "• `"+subscription.Username+"`")
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Following in #%s", channelID),
		Description: strings.Join(lines, "\n"),
		Color:       accentColor,
	}
}

func loggedEmbed(username, filmQuery string, entries []letterboxd.DiaryEntry) *discordgo.MessageEmbed {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("• [%s](%s) — watched `%s`, rated `%s`%s",
			entry.FilmTitle,
			entry.FilmURL,
			entry.WatchedDate,
			ratingLabel(entry.Rating),
			boolLabel(entry.Liked, ", liked it", ""),
		))
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Recent `%s` logs for @%s", filmQuery, username),
		Description: strings.Join(lines, "\n"),
		Color:       accentColor,
	}
}

func compareEmbed(comparison social.Comparison) *discordgo.MessageEmbed {
	fields := []*discordgo.MessageEmbedField{
		{Name: "Taste match", Value: fmt.Sprintf("`%d/100`", comparison.Score), Inline: true},
		{Name: "Shared favorites", Value: favoriteList(comparison.SharedFavorites), Inline: true},
		{Name: "Shared recent watches", Value: sharedRecentList(comparison.SharedRecent), Inline: false},
		{Name: "Activity pace", Value: fmt.Sprintf("@%s: `%s` films this year\n@%s: `%s` films this year", comparison.Left.Username, comma(comparison.Left.ThisYearCount), comparison.Right.Username, comma(comparison.Right.ThisYearCount)), Inline: true},
		{Name: "Profile scale", Value: fmt.Sprintf("@%s: `%s` films logged\n@%s: `%s` films logged", comparison.Left.Username, comma(comparison.Left.FilmCount), comparison.Right.Username, comma(comparison.Right.FilmCount)), Inline: true},
	}

	if comparison.BiggestDisagreement != nil {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "Biggest disagreement",
			Value: fmt.Sprintf("%s (%d)\n@%s: `%s` | @%s: `%s`",
				comparison.BiggestDisagreement.Title,
				comparison.BiggestDisagreement.Year,
				comparison.Left.Username,
				ratingLabel(comparison.BiggestDisagreement.LeftRating),
				comparison.Right.Username,
				ratingLabel(comparison.BiggestDisagreement.RightRating),
			),
			Inline: false,
		})
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("@%s vs @%s", comparison.Left.Username, comparison.Right.Username),
		Description: "Comparison is based on public profile stats, favorites, and overlap in recent public diary activity.",
		Color:       accentColor,
		Fields:      fields,
	}
}

func tasteEmbed(comparison social.Comparison) *discordgo.MessageEmbed {
	verdict := "Different lanes"
	switch {
	case comparison.Score >= 80:
		verdict = "Same wavelength"
	case comparison.Score >= 60:
		verdict = "Strong overlap"
	case comparison.Score >= 40:
		verdict = "Some overlap"
	}

	description := verdict
	if len(comparison.SharedFavorites) > 0 {
		description += "\nShared favorites: " + favoriteList(comparison.SharedFavorites)
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Taste score: @%s + @%s", comparison.Left.Username, comparison.Right.Username),
		Description: description,
		Color:       alertColor,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Compatibility", Value: fmt.Sprintf("`%d/100`", comparison.Score), Inline: true},
			{Name: "Recent overlap", Value: fmt.Sprintf("`%d` shared recent diary titles", len(comparison.SharedRecent)), Inline: true},
		},
	}
}

func followEntryEmbed(username string, entry letterboxd.DiaryEntry) *discordgo.MessageEmbed {
	title := fmt.Sprintf("@%s watched %s", username, entry.FilmTitle)
	if entry.Rewatch {
		title = fmt.Sprintf("@%s rewatched %s", username, entry.FilmTitle)
	}

	description := truncate(entry.Excerpt, 350)
	if description == "" {
		description = "New public diary entry."
	}

	return &discordgo.MessageEmbed{
		Title:       title,
		URL:         entry.FilmURL,
		Description: description,
		Color:       alertColor,
		Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: entry.PosterURL},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Watched %s • %s%s%s",
				entry.WatchedDate,
				ratingLabel(entry.Rating),
				boolLabel(entry.Liked, " • liked", ""),
				boolLabel(entry.Rewatch, " • rewatch", ""),
			),
		},
	}
}

func ratingLabel(value float64) string {
	if value == 0 {
		return "unrated"
	}
	return fmt.Sprintf("%.1f/5", value)
}

func favoriteList(favorites []letterboxd.FavoriteFilm) string {
	if len(favorites) == 0 {
		return "No overlap yet."
	}

	items := make([]string, 0, len(favorites))
	for _, favorite := range favorites {
		label := favorite.Title
		if favorite.Year != 0 {
			label = fmt.Sprintf("%s (%d)", favorite.Title, favorite.Year)
		}
		if favorite.URL != "" {
			label = fmt.Sprintf("[%s](%s)", label, favorite.URL)
		}
		items = append(items, "• "+label)
	}
	return strings.Join(items, "\n")
}

func sharedRecentList(entries []social.SharedRecentEntry) string {
	if len(entries) == 0 {
		return "No shared recent public diary titles."
	}

	lines := make([]string, 0, minInt(len(entries), 6))
	for _, entry := range entries[:minInt(len(entries), 6)] {
		lines = append(lines, fmt.Sprintf("• %s (%d) — `%s` vs `%s`",
			entry.Title,
			entry.Year,
			ratingLabel(entry.LeftRating),
			ratingLabel(entry.RightRating),
		))
	}
	return strings.Join(lines, "\n")
}

func comma(value int) string {
	raw := strconv.Itoa(value)
	if len(raw) <= 3 {
		return raw
	}

	var builder strings.Builder
	leading := len(raw) % 3
	if leading == 0 {
		leading = 3
	}
	builder.WriteString(raw[:leading])
	for index := leading; index < len(raw); index += 3 {
		builder.WriteByte(',')
		builder.WriteString(raw[index : index+3])
	}
	return builder.String()
}

func truncate(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return strings.TrimSpace(value[:limit-1]) + "…"
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "Unknown"
	}
	return value.Format("02 Jan 2006")
}

func boolLabel(flag bool, whenTrue, whenFalse string) string {
	if flag {
		return whenTrue
	}
	return whenFalse
}

func emptyFallback(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "0 mins" {
		return fallback
	}
	return value
}

func limitStrings(items []string, limit int) []string {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func statsMemberLabel(stats letterboxd.MemberStats) string {
	if stats.Member.DisplayName != "" && stats.Member.Username != "" {
		return fmt.Sprintf("%s (@%s)", stats.Member.DisplayName, stats.Member.Username)
	}
	if stats.Member.DisplayName != "" {
		return stats.Member.DisplayName
	}
	if stats.Member.Username != "" {
		return "@" + stats.Member.Username
	}
	return ""
}

func statsCountSummary(stats letterboxd.MemberStats) string {
	counts := stats.Counts
	lines := []string{
		fmt.Sprintf("Watches: `%s`", comma(counts.Watches)),
		fmt.Sprintf("Ratings: `%s`", comma(counts.Ratings)),
		fmt.Sprintf("Reviews: `%s`", comma(counts.Reviews)),
		fmt.Sprintf("Diary: `%s` total • `%s` this year", comma(counts.DiaryEntries), comma(counts.DiaryEntriesThisYear)),
		fmt.Sprintf("Watchlist: `%s`", comma(counts.Watchlist)),
		fmt.Sprintf("Followers: `%s` • Following: `%s`", comma(counts.Followers), comma(counts.Following)),
	}
	return strings.Join(lines, "\n")
}

func statsYearsInReview(stats letterboxd.MemberStats) []string {
	values := make([]string, 0, len(stats.YearsInReview))
	for _, year := range stats.YearsInReview {
		if year <= 0 {
			continue
		}
		values = append(values, strconv.Itoa(year))
	}
	return values
}

func statsRatingHistogram(stats letterboxd.MemberStats) string {
	if len(stats.RatingsHistogram) == 0 {
		return ""
	}

	lines := make([]string, 0, minInt(len(stats.RatingsHistogram), 6))
	for _, bar := range stats.RatingsHistogram[:minInt(len(stats.RatingsHistogram), 6)] {
		lines = append(lines, fmt.Sprintf("`%.1f` stars: `%s`", bar.Rating, comma(bar.Count)))
	}
	return strings.Join(lines, "\n")
}

func filmSummaryList(films []letterboxd.FilmSummary) string {
	if len(films) == 0 {
		return "No films found."
	}

	lines := make([]string, 0, len(films))
	for _, film := range films {
		lines = append(lines, filmSummaryLine(film))
	}
	return strings.Join(lines, "\n")
}

func filmSummaryLine(film letterboxd.FilmSummary) string {
	label := filmSummaryTitle(film)
	if url := filmSummaryURL(film); url != "" {
		label = fmt.Sprintf("[%s](%s)", label, url)
	}

	details := make([]string, 0, 3)
	if film.RuntimeMinutes > 0 {
		details = append(details, fmt.Sprintf("%d mins", film.RuntimeMinutes))
	}
	if film.Rating > 0 {
		details = append(details, fmt.Sprintf("%.1f/5", film.Rating))
	}
	if len(film.Directors) > 0 {
		details = append(details, strings.Join(limitStrings(film.Directors, 2), ", "))
	}

	if len(details) == 0 {
		return "• " + label
	}
	return fmt.Sprintf("• %s — %s", label, strings.Join(details, " • "))
}

func filmSummaryTitle(film letterboxd.FilmSummary) string {
	title := strings.TrimSpace(film.Title)
	if title == "" {
		title = "Unknown title"
	}
	if film.Year > 0 {
		return fmt.Sprintf("%s (%d)", title, film.Year)
	}
	return title
}

func filmSummaryURL(film letterboxd.FilmSummary) string {
	if strings.TrimSpace(film.ShortURL) != "" {
		return strings.TrimSpace(film.ShortURL)
	}
	return strings.TrimSpace(film.URL)
}

func filmSummaryThumbnail(films []letterboxd.FilmSummary) *discordgo.MessageEmbedThumbnail {
	for _, film := range films {
		if strings.TrimSpace(film.PosterURL) != "" {
			return &discordgo.MessageEmbedThumbnail{URL: film.PosterURL}
		}
	}
	return nil
}

func activityItemLine(item letterboxd.ActivityItem) string {
	title := ""
	if item.LogEntry != nil && item.LogEntry.Film.Title != "" {
		title = filmSummaryTitle(item.LogEntry.Film)
	}
	if title == "" && item.Film != nil {
		title = filmSummaryTitle(*item.Film)
	}

	switch item.Type {
	case "FilmWatchActivity":
		return fmt.Sprintf("• Watched %s%s", emptyFallback(title, "a film"), ratingSuffix(item.Rating))
	case "ReviewResponseActivity":
		return fmt.Sprintf("• Reviewed %s%s", emptyFallback(title, "a film"), ratingSuffix(item.Rating))
	case "DiaryEntryActivity":
		return fmt.Sprintf("• Logged %s%s", emptyFallback(title, "a film"), ratingSuffix(item.Rating))
	case "MemberFollowingActivity":
		if item.Followed != nil && item.Followed.Username != "" {
			return fmt.Sprintf("• Followed @%s", item.Followed.Username)
		}
	}

	if item.List != nil && item.List.Title != "" {
		if strings.TrimSpace(item.List.URL) != "" {
			return fmt.Sprintf("• %s — [%s](%s)", prettifyActivityType(item.Type), item.List.Title, item.List.URL)
		}
		return fmt.Sprintf("• %s — %s", prettifyActivityType(item.Type), item.List.Title)
	}
	if title != "" {
		return fmt.Sprintf("• %s %s%s", prettifyActivityType(item.Type), title, ratingSuffix(item.Rating))
	}
	return "• " + prettifyActivityType(item.Type)
}

func prettifyActivityType(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "Recent activity"
	}
	raw = strings.TrimSuffix(raw, "Activity")
	raw = strings.TrimSuffix(raw, "Response")
	raw = strings.ReplaceAll(raw, "Member", "")
	raw = strings.ReplaceAll(raw, "Film", "")
	raw = strings.ReplaceAll(raw, "LogEntry", "")
	parts := splitCamel(raw)
	if len(parts) == 0 {
		return "Recent activity"
	}
	return strings.Join(parts, " ")
}

func splitCamel(value string) []string {
	if value == "" {
		return nil
	}

	parts := make([]string, 0, 4)
	start := 0
	for index := 1; index < len(value); index++ {
		if value[index] >= 'A' && value[index] <= 'Z' {
			parts = append(parts, value[start:index])
			start = index
		}
	}
	parts = append(parts, value[start:])
	return parts
}

func displayUsername(username string) string {
	username = strings.TrimSpace(username)
	if username == "" {
		return "This member"
	}
	return "@" + username
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func filterSuffix(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return " for `" + value + "`"
}

func serviceSuffix(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return " on `" + value + "`"
}

func ratingSuffix(rating float64) string {
	if rating == 0 {
		return ""
	}
	return fmt.Sprintf(" • `%s`", ratingLabel(rating))
}
