package bot

import "github.com/bwmarrin/discordgo"

func slashCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "help",
			Description: "Show CineBuddy commands and shortcuts",
		},
		{
			Name:        "connect",
			Description: "Link your default Letterboxd username",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "username",
					Description: "Your Letterboxd username",
					Required:    true,
				},
			},
		},
		{
			Name:        "disconnect",
			Description: "Remove your default Letterboxd username",
		},
		{
			Name:        "profile",
			Description: "Show a Letterboxd profile card",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "username",
					Description: "Letterboxd username, or leave blank to use your linked account",
				},
			},
		},
		{
			Name:        "diary",
			Description: "Show recent public diary entries",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "username",
					Description: "Letterboxd username, or leave blank to use your linked account",
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "count",
					Description: "How many entries to show (1-10)",
				},
			},
		},
		{
			Name:        "film",
			Description: "Search Letterboxd and show a film card",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "query",
					Description: "Film title or Letterboxd film URL",
					Required:    true,
				},
			},
		},
		{
			Name:        "list",
			Description: "Find a user's recent public list by title",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "query",
					Description: "List title search",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "username",
					Description: "Letterboxd username, or leave blank to use your linked account",
				},
			},
		},
		{
			Name:        "follow",
			Description: "Post new diary entries from a Letterboxd account into a channel",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "username",
					Description: "Letterboxd username to follow",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "Target channel, or leave blank to use the current channel",
				},
			},
		},
		{
			Name:        "unfollow",
			Description: "Stop posting entries for a followed Letterboxd account",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "username",
					Description: "Letterboxd username to unfollow",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "Target channel, or leave blank to use the current channel",
				},
			},
		},
		{
			Name:        "following",
			Description: "List followed Letterboxd accounts for a channel",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "Channel to inspect, or leave blank to use the current channel",
				},
			},
		},
		{
			Name:        "logged",
			Description: "Find recent logs for a film in a user's RSS feed",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "film",
					Description: "Film title or Letterboxd film URL",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "username",
					Description: "Letterboxd username, or leave blank to use your linked account",
				},
			},
		},
		{
			Name:        "refresh",
			Description: "Clear cached Letterboxd data",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "username",
					Description: "Clear cache for one username, or leave blank to clear your linked/default cache",
				},
			},
		},
		{
			Name:        "compare",
			Description: "Compare your profile with another Letterboxd account",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "other",
					Description: "The other Letterboxd username",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "username",
					Description: "Use this account instead of your linked default",
				},
			},
		},
		{
			Name:        "taste",
			Description: "Score the compatibility between two Letterboxd accounts",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "other",
					Description: "The other Letterboxd username",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "username",
					Description: "Use this account instead of your linked default",
				},
			},
		},
		{
			Name:        "roulette",
			Description: "Pick a random film from a discovery theme",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "theme",
					Description: "Discovery theme like horror, animation, or heist",
				},
			},
		},
	}
}
