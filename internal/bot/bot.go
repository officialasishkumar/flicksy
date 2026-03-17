package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/asish/filmpal/internal/config"
	"github.com/asish/filmpal/internal/follows"
	"github.com/asish/filmpal/internal/letterboxd"
	"github.com/asish/filmpal/internal/social"
	"github.com/asish/filmpal/internal/store"
)

type Bot struct {
	config   config.Config
	logger   *slog.Logger
	client   *letterboxd.Client
	store    *store.Store
	session  *discordgo.Session
	poller   *follows.Poller
	random   *rand.Rand
	commands []*discordgo.ApplicationCommand
}

type commandResponse struct {
	content string
	embeds  []*discordgo.MessageEmbed
}

func New(cfg config.Config, logger *slog.Logger, client *letterboxd.Client, stateStore *store.Store) (*Bot, error) {
	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, fmt.Errorf("create discord session: %w", err)
	}
	session.Identify.Intents = discordgo.IntentsGuilds

	bot := &Bot{
		config:   cfg,
		logger:   logger,
		client:   client,
		store:    stateStore,
		session:  session,
		random:   rand.New(rand.NewSource(time.Now().UnixNano())),
		commands: slashCommands(),
	}

	session.AddHandler(bot.onInteractionCreate)
	bot.poller = follows.NewPoller(logger, client, stateStore, cfg.PollInterval, bot.publishFollowEntry)

	return bot, nil
}

func (b *Bot) Run(ctx context.Context) error {
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("open discord session: %w", err)
	}
	defer b.session.Close()

	if err := b.registerCommands(); err != nil {
		return err
	}

	go b.poller.Run(ctx)

	<-ctx.Done()
	return nil
}

func (b *Bot) registerCommands() error {
	appID := b.session.State.User.ID
	if appID == "" {
		return errors.New("discord application id unavailable")
	}

	_, err := b.session.ApplicationCommandBulkOverwrite(appID, b.config.GuildID, b.commands)
	if err != nil {
		return fmt.Errorf("register slash commands: %w", err)
	}

	scope := "global"
	if b.config.GuildID != "" {
		scope = "guild"
	}
	b.logger.Info("registered slash commands", "scope", scope, "count", len(b.commands))
	return nil
}

func (b *Bot) onInteractionCreate(session *discordgo.Session, event *discordgo.InteractionCreate) {
	if event.Type != discordgo.InteractionApplicationCommand {
		return
	}
	if event.GuildID == "" {
		respondImmediate(session, event.Interaction, "FilmPal does not support direct messages.", true)
		return
	}

	name := event.ApplicationCommandData().Name
	ephemeral := isEphemeralCommand(name)
	if err := session.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: interactionFlags(ephemeral),
		},
	}); err != nil {
		b.logger.Warn("failed to defer interaction", "command", name, "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := b.dispatchCommand(ctx, session, event)
	if err != nil {
		b.logger.Warn("command failed", "command", name, "error", err)
		editResponse(session, event.Interaction, commandResponse{content: err.Error()})
		return
	}

	if err := editResponse(session, event.Interaction, response); err != nil {
		b.logger.Warn("failed to edit interaction response", "command", name, "error", err)
	}
}

func (b *Bot) dispatchCommand(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate) (commandResponse, error) {
	switch event.ApplicationCommandData().Name {
	case "help":
		return commandResponse{embeds: []*discordgo.MessageEmbed{helpEmbed()}}, nil
	case "connect":
		return b.handleConnect(ctx, event)
	case "disconnect":
		return b.handleDisconnect(event), nil
	case "profile":
		return b.handleProfile(ctx, event)
	case "diary":
		return b.handleDiary(ctx, event)
	case "film":
		return b.handleFilm(ctx, event)
	case "list":
		return b.handleList(ctx, event)
	case "follow":
		return b.handleFollow(ctx, event)
	case "unfollow":
		return b.handleUnfollow(event)
	case "following":
		return b.handleFollowing(event), nil
	case "logged":
		return b.handleLogged(ctx, event)
	case "refresh":
		return b.handleRefresh(event)
	case "compare":
		return b.handleCompare(ctx, event)
	case "taste":
		return b.handleTaste(ctx, event)
	case "roulette":
		return b.handleRoulette(ctx, event)
	default:
		return commandResponse{}, fmt.Errorf("no handler available for %s", event.ApplicationCommandData().Name)
	}
}

func (b *Bot) handleConnect(ctx context.Context, event *discordgo.InteractionCreate) (commandResponse, error) {
	username := optionString(event, "username")
	if username == "" {
		return commandResponse{}, fmt.Errorf("username is required")
	}

	profile, err := b.client.FetchProfile(ctx, username)
	if err != nil {
		return commandResponse{}, fmt.Errorf("could not verify %q on Letterboxd: %w", username, err)
	}

	if err := b.store.SetLinkedAccount(discordUserID(event), profile.Username); err != nil {
		return commandResponse{}, fmt.Errorf("save linked account: %w", err)
	}

	return commandResponse{
		content: fmt.Sprintf("Linked this Discord account to `%s`.", profile.Username),
	}, nil
}

func (b *Bot) handleDisconnect(event *discordgo.InteractionCreate) commandResponse {
	if err := b.store.RemoveLinkedAccount(discordUserID(event)); err != nil {
		return commandResponse{content: fmt.Sprintf("Failed to remove linked account: %v", err)}
	}
	return commandResponse{content: "Removed your default Letterboxd account."}
}

func (b *Bot) handleProfile(ctx context.Context, event *discordgo.InteractionCreate) (commandResponse, error) {
	username, err := b.resolveUsername(event, optionString(event, "username"))
	if err != nil {
		return commandResponse{}, err
	}

	profile, err := b.client.FetchProfile(ctx, username)
	if err != nil {
		return commandResponse{}, err
	}

	return commandResponse{embeds: []*discordgo.MessageEmbed{profileEmbed(profile)}}, nil
}

func (b *Bot) handleDiary(ctx context.Context, event *discordgo.InteractionCreate) (commandResponse, error) {
	username, err := b.resolveUsername(event, optionString(event, "username"))
	if err != nil {
		return commandResponse{}, err
	}

	limit := optionInt(event, "count", 5)
	if limit < 1 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	entries, err := b.client.FetchRecentDiary(ctx, username, limit)
	if err != nil {
		return commandResponse{}, err
	}
	if len(entries) == 0 {
		return commandResponse{content: fmt.Sprintf("No recent public diary entries found for `%s`.", username)}, nil
	}

	return commandResponse{embeds: []*discordgo.MessageEmbed{diaryEmbed(username, entries)}}, nil
}

func (b *Bot) handleFilm(ctx context.Context, event *discordgo.InteractionCreate) (commandResponse, error) {
	query := optionString(event, "query")
	film, err := b.client.FetchFilm(ctx, query)
	if err != nil {
		return commandResponse{}, err
	}
	return commandResponse{embeds: []*discordgo.MessageEmbed{filmEmbed(film, "Film lookup")}}, nil
}

func (b *Bot) handleList(ctx context.Context, event *discordgo.InteractionCreate) (commandResponse, error) {
	username, err := b.resolveUsername(event, optionString(event, "username"))
	if err != nil {
		return commandResponse{}, err
	}

	listSummary, err := b.client.FindList(ctx, username, optionString(event, "query"))
	if err != nil {
		return commandResponse{}, err
	}
	if listSummary == nil {
		return commandResponse{
			content: fmt.Sprintf("No recent public list matched that search for `%s`. FilmPal searches the public RSS list history, so older/private lists may not appear.", username),
		}, nil
	}

	return commandResponse{embeds: []*discordgo.MessageEmbed{listEmbed(username, *listSummary)}}, nil
}

func (b *Bot) handleFollow(ctx context.Context, event *discordgo.InteractionCreate) (commandResponse, error) {
	username := optionString(event, "username")
	if username == "" {
		return commandResponse{}, fmt.Errorf("username is required")
	}

	profile, err := b.client.FetchProfile(ctx, username)
	if err != nil {
		return commandResponse{}, fmt.Errorf("could not verify %q on Letterboxd: %w", username, err)
	}

	channelID := event.ChannelID
	if option := findOption(event, "channel"); option != nil && option.ChannelValue(b.session) != nil {
		channelID = option.ChannelValue(b.session).ID
	}

	subscription := store.FollowSubscription{
		GuildID:       event.GuildID,
		ChannelID:     channelID,
		Username:      profile.Username,
		AddedByUserID: discordUserID(event),
	}
	if entries, err := b.client.FetchRecentDiary(ctx, profile.Username, 1); err == nil && len(entries) > 0 {
		subscription.LastSeenEntry = entries[0].ID
	}

	added, err := b.store.AddFollow(subscription)
	if err != nil {
		return commandResponse{}, fmt.Errorf("save follow subscription: %w", err)
	}
	if !added {
		return commandResponse{content: fmt.Sprintf("`%s` is already followed in <#%s>.", profile.Username, channelID)}, nil
	}

	return commandResponse{content: fmt.Sprintf("Following `%s` in <#%s>.", profile.Username, channelID)}, nil
}

func (b *Bot) handleUnfollow(event *discordgo.InteractionCreate) (commandResponse, error) {
	username := optionString(event, "username")
	if username == "" {
		return commandResponse{}, fmt.Errorf("username is required")
	}

	channelID := event.ChannelID
	if option := findOption(event, "channel"); option != nil && option.ChannelValue(b.session) != nil {
		channelID = option.ChannelValue(b.session).ID
	}

	removed, err := b.store.RemoveFollow(channelID, username)
	if err != nil {
		return commandResponse{}, fmt.Errorf("remove follow subscription: %w", err)
	}
	if !removed {
		return commandResponse{content: fmt.Sprintf("`%s` is not currently followed in <#%s>.", strings.ToLower(username), channelID)}, nil
	}
	return commandResponse{content: fmt.Sprintf("Stopped following `%s` in <#%s>.", strings.ToLower(username), channelID)}, nil
}

func (b *Bot) handleFollowing(event *discordgo.InteractionCreate) commandResponse {
	channelID := event.ChannelID
	if option := findOption(event, "channel"); option != nil && option.ChannelValue(b.session) != nil {
		channelID = option.ChannelValue(b.session).ID
	}

	subscriptions := b.store.ListFollows(channelID)
	if len(subscriptions) == 0 {
		return commandResponse{content: fmt.Sprintf("No Letterboxd accounts are followed in <#%s>.", channelID)}
	}

	return commandResponse{embeds: []*discordgo.MessageEmbed{followingEmbed(channelID, subscriptions)}}
}

func (b *Bot) handleLogged(ctx context.Context, event *discordgo.InteractionCreate) (commandResponse, error) {
	username, err := b.resolveUsername(event, optionString(event, "username"))
	if err != nil {
		return commandResponse{}, err
	}

	filmQuery := optionString(event, "film")
	entries, err := b.client.FindRecentLogs(ctx, username, filmQuery, 8)
	if err != nil {
		return commandResponse{}, err
	}
	if len(entries) == 0 {
		return commandResponse{
			content: fmt.Sprintf("No recent public `%s` logs were found for `%s` in the RSS feed window.", filmQuery, username),
		}, nil
	}

	return commandResponse{embeds: []*discordgo.MessageEmbed{loggedEmbed(username, filmQuery, entries)}}, nil
}

func (b *Bot) handleRefresh(event *discordgo.InteractionCreate) (commandResponse, error) {
	username := optionString(event, "username")
	if username == "" {
		if linked, ok := b.store.LinkedAccount(discordUserID(event)); ok {
			username = linked
		}
	}

	if username == "" {
		b.client.ClearAll()
		return commandResponse{content: "Cleared the in-memory Letterboxd cache."}, nil
	}

	b.client.RefreshProfile(username)
	return commandResponse{content: fmt.Sprintf("Cleared cached profile and feed data for `%s`.", strings.ToLower(username))}, nil
}

func (b *Bot) handleCompare(ctx context.Context, event *discordgo.InteractionCreate) (commandResponse, error) {
	left, right, err := b.resolveComparisonUsers(event)
	if err != nil {
		return commandResponse{}, err
	}

	comparison, err := social.Analyze(ctx, b.client, left, right)
	if err != nil {
		return commandResponse{}, err
	}

	return commandResponse{embeds: []*discordgo.MessageEmbed{compareEmbed(comparison)}}, nil
}

func (b *Bot) handleTaste(ctx context.Context, event *discordgo.InteractionCreate) (commandResponse, error) {
	left, right, err := b.resolveComparisonUsers(event)
	if err != nil {
		return commandResponse{}, err
	}

	comparison, err := social.Analyze(ctx, b.client, left, right)
	if err != nil {
		return commandResponse{}, err
	}

	return commandResponse{embeds: []*discordgo.MessageEmbed{tasteEmbed(comparison)}}, nil
}

func (b *Bot) handleRoulette(ctx context.Context, event *discordgo.InteractionCreate) (commandResponse, error) {
	theme := optionString(event, "theme")
	if strings.TrimSpace(theme) == "" {
		theme = rouletteThemes[b.random.Intn(len(rouletteThemes))]
	}

	results, err := b.client.SearchFilm(ctx, theme, 8)
	if err != nil {
		return commandResponse{}, err
	}
	if len(results) == 0 {
		return commandResponse{}, fmt.Errorf("no film candidates found for theme %q", theme)
	}

	pick := results[b.random.Intn(len(results))]
	film, err := b.client.FetchFilm(ctx, pick.URL)
	if err != nil {
		return commandResponse{}, err
	}

	return commandResponse{embeds: []*discordgo.MessageEmbed{filmEmbed(film, "Roulette: "+theme)}}, nil
}

func (b *Bot) publishFollowEntry(ctx context.Context, subscription store.FollowSubscription, entry letterboxd.DiaryEntry) error {
	embed := followEntryEmbed(subscription.Username, entry)
	_, err := b.session.ChannelMessageSendEmbed(subscription.ChannelID, embed)
	return err
}

func (b *Bot) resolveUsername(event *discordgo.InteractionCreate, explicit string) (string, error) {
	if explicit = strings.ToLower(strings.Trim(strings.TrimSpace(explicit), "/")); explicit != "" {
		return explicit, nil
	}

	if linked, ok := b.store.LinkedAccount(discordUserID(event)); ok {
		return linked, nil
	}

	return "", fmt.Errorf("no default Letterboxd account linked. Use `/connect username:<your-account>` or pass `username` directly")
}

func (b *Bot) resolveComparisonUsers(event *discordgo.InteractionCreate) (string, string, error) {
	right := optionString(event, "other")
	if right == "" {
		return "", "", fmt.Errorf("the `other` username is required")
	}

	left, err := b.resolveUsername(event, optionString(event, "username"))
	if err != nil {
		return "", "", err
	}

	return left, strings.ToLower(strings.Trim(strings.TrimSpace(right), "/")), nil
}

func discordUserID(event *discordgo.InteractionCreate) string {
	if event.Member != nil && event.Member.User != nil {
		return event.Member.User.ID
	}
	if event.User != nil {
		return event.User.ID
	}
	return ""
}

func optionString(event *discordgo.InteractionCreate, name string) string {
	option := findOption(event, name)
	if option == nil {
		return ""
	}
	return option.StringValue()
}

func optionInt(event *discordgo.InteractionCreate, name string, fallback int) int {
	option := findOption(event, name)
	if option == nil {
		return fallback
	}
	return int(option.IntValue())
}

func findOption(event *discordgo.InteractionCreate, name string) *discordgo.ApplicationCommandInteractionDataOption {
	for _, option := range event.ApplicationCommandData().Options {
		if option.Name == name {
			return option
		}
	}
	return nil
}

func isEphemeralCommand(name string) bool {
	switch name {
	case "connect", "disconnect", "refresh":
		return true
	default:
		return false
	}
}

func interactionFlags(ephemeral bool) discordgo.MessageFlags {
	if ephemeral {
		return discordgo.MessageFlagsEphemeral
	}
	return 0
}

func editResponse(session *discordgo.Session, interaction *discordgo.Interaction, response commandResponse) error {
	content := response.content
	embeds := response.embeds
	_, err := session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
		Content: &content,
		Embeds:  &embeds,
	})
	return err
}

func respondImmediate(session *discordgo.Session, interaction *discordgo.Interaction, content string, ephemeral bool) {
	_ = session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   interactionFlags(ephemeral),
		},
	})
}

var rouletteThemes = []string{
	"criterion",
	"neo noir",
	"anime",
	"sci fi",
	"horror",
	"heist",
	"romance",
	"western",
	"animation",
	"thriller",
	"coming of age",
	"cult classic",
}
