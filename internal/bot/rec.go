package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/asish/flicksy/internal/letterboxd"
)

func (b *Bot) handleRec(ctx context.Context, event *discordgo.InteractionCreate) (commandResponse, error) {
	if err := b.requireOfficialAPI(); err != nil {
		return commandResponse{}, err
	}

	username, err := b.resolveUsername(event, optionString(event, "username"))
	if err != nil {
		return commandResponse{}, err
	}

	profile, err := b.client.FetchProfile(ctx, username)
	if err != nil {
		return commandResponse{}, fmt.Errorf("could not fetch profile for %q: %w", username, err)
	}

	var baseFilmURL string
	var baseFilmTitle string

	if len(profile.Favorites) > 0 {
		fav := profile.Favorites[b.random.Intn(len(profile.Favorites))]
		baseFilmURL = fav.URL
		baseFilmTitle = fav.Title
	} else {
		diary, err := b.client.FetchRecentDiary(ctx, username, 50)
		if err != nil {
			return commandResponse{}, fmt.Errorf("could not fetch recent diary for %q: %w", username, err)
		}

		var highlyRated []letterboxd.DiaryEntry
		for _, entry := range diary {
			if entry.Rating >= 4.0 {
				highlyRated = append(highlyRated, entry)
			}
		}

		if len(highlyRated) > 0 {
			fav := highlyRated[b.random.Intn(len(highlyRated))]
			baseFilmURL = fav.FilmURL
			baseFilmTitle = fav.FilmTitle
		} else {
			return commandResponse{}, fmt.Errorf("no favorites or highly rated recent films found to base a recommendation on for %q", username)
		}
	}

	film, err := b.client.FetchFilm(ctx, baseFilmURL)
	if err != nil {
		return commandResponse{}, fmt.Errorf("could not fetch film details for %q: %w", baseFilmTitle, err)
	}

	if len(film.Genres) == 0 {
		return commandResponse{}, fmt.Errorf("film %q has no genres listed to base a recommendation on", film.Title)
	}

	genre := film.Genres[b.random.Intn(len(film.Genres))]

	options := letterboxd.DiscoveryOptions{
		Genre: genre,
		Limit: 20,
	}

	recs, err := b.client.DiscoverFilms(ctx, options)
	if err != nil {
		return commandResponse{}, fmt.Errorf("could not discover films in genre %q: %w", genre, err)
	}

	if len(recs) == 0 {
		return commandResponse{}, fmt.Errorf("no popular films found in genre %q", genre)
	}

	pick := recs[b.random.Intn(len(recs))]

	return commandResponse{embeds: []*discordgo.MessageEmbed{recEmbed(username, baseFilmTitle, genre, pick)}}, nil
}

func recEmbed(username, baseFilmTitle, genre string, film letterboxd.FilmSummary) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s's Recommendation", displayUsername(username)),
		Description: fmt.Sprintf("Because you loved **%s**, here is a popular **%s** film:", baseFilmTitle, strings.ToLower(genre)),
		Color:       accentColor,
		URL:         filmSummaryURL(film),
		Thumbnail:   filmSummaryThumbnail([]letterboxd.FilmSummary{film}),
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Film", Value: filmSummaryLine(film), Inline: false},
		},
	}
}
