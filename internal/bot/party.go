package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/asish/flicksy/internal/letterboxd"
)

func (b *Bot) handleParty(ctx context.Context, event *discordgo.InteractionCreate) (commandResponse, error) {
	if err := b.requireOfficialAPI(); err != nil {
		return commandResponse{}, err
	}

	var usernames []string
	for i := 1; i <= 5; i++ {
		name := optionString(event, fmt.Sprintf("user%d", i))
		if name != "" {
			resolved, err := b.resolveUsername(event, name)
			if err == nil && resolved != "" {
				usernames = append(usernames, resolved)
			}
		}
	}

	if len(usernames) < 2 {
		return commandResponse{}, fmt.Errorf("at least two valid Letterboxd usernames are required for a watch party")
	}

	var allWatchlists [][]letterboxd.FilmSummary
	for _, username := range usernames {
		films, err := b.client.FetchWatchlist(ctx, username, 100, "")
		if err != nil {
			return commandResponse{}, fmt.Errorf("could not fetch watchlist for %q: %w", username, err)
		}
		allWatchlists = append(allWatchlists, films)
	}

	var commonFilms []letterboxd.FilmSummary
	if len(allWatchlists) > 0 {
		firstList := allWatchlists[0]
		for _, film := range firstList {
			inAll := true
			for i := 1; i < len(allWatchlists); i++ {
				found := false
				for _, otherFilm := range allWatchlists[i] {
					if film.ID == otherFilm.ID {
						found = true
						break
					}
				}
				if !found {
					inAll = false
					break
				}
			}
			if inAll {
				commonFilms = append(commonFilms, film)
			}
		}
	}

	if len(commonFilms) == 0 {
		return commandResponse{
			content: fmt.Sprintf("No common films found in the recent watchlists of %d users.", len(usernames)),
		}, nil
	}

	pick := commonFilms[b.random.Intn(len(commonFilms))]
	return commandResponse{embeds: []*discordgo.MessageEmbed{partyEmbed(usernames, pick, len(commonFilms))}}, nil
}

func partyEmbed(usernames []string, film letterboxd.FilmSummary, totalCommon int) *discordgo.MessageEmbed {
	var userList []string
	for _, u := range usernames {
		userList = append(userList, displayUsername(u))
	}

	desc := fmt.Sprintf("Found %d film%s in common! Randomly picked:", totalCommon, pluralize(totalCommon))

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Watch Party: %s", strings.Join(userList, ", ")),
		Description: desc,
		Color:       accentColor,
		URL:         filmSummaryURL(film),
		Thumbnail:   filmSummaryThumbnail([]letterboxd.FilmSummary{film}),
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Film", Value: filmSummaryLine(film), Inline: false},
		},
	}
}
