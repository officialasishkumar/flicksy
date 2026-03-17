# Flicksy

Flicksy is a self-hosted Discord bot for Letterboxd communities, built in Go and designed to be simple to run, easy to use, and fun to share in a server.

It focuses on public Letterboxd pages and RSS feeds, so it works without private API access or GCP infrastructure.

If you add official Letterboxd API credentials, Flicksy unlocks richer search and discovery without changing how people use the bot in Discord.

## What it does

- Link a Discord user to a default Letterboxd account with `/connect`
- Show profile cards with `/profile`
- Show recent diary activity with `/diary`
- Search films with `/film`
- Search recent public lists with `/list`
- Track recent logs for a film with `/logged`
- Follow public diary activity into a Discord channel with `/follow`
- Compare two profiles with `/compare`
- Score compatibility with `/taste`
- Do film discovery with `/roulette`
- Show richer member stats with `/stats`
- Browse a public watchlist with `/watchlist`
- Pick a random movie from a public watchlist with `/watchpick`
- Show recent all-up activity with `/activity`
- Discover popular films with `/discover`
- Match watchlists between multiple users with `/party`
- Get personalized recommendations with `/rec`
- Clear in-memory cache with `/refresh`

## Why the bot is easier to use

- Most commands can use your linked account automatically after `/connect`
- API-powered member commands also use your linked account by default
- The bot is a single Go binary
- State is stored in one JSON file under `data/state.json`
- No database, queue, or cloud service is required

## Commands

If a command accepts `[username]` and you leave it blank, Flicksy uses the account linked with `/connect`.

Commands with `[count]` clamp the result size to `1-10`.

`/connect`, `/disconnect`, and `/refresh` reply ephemerally so they do not spam the channel.

### Account and help

- `/help`
  Shows the in-Discord help card with the main command groups.
- `/connect username`
  Verifies the Letterboxd account exists and saves it as your default profile for future commands.
- `/disconnect`
  Removes your saved default profile.
- `/refresh [username]`
  Clears Flicksy's in-memory cache. With a username it refreshes that profile and feed data; without one it clears your linked account cache or, if nothing is linked, the full cache.

### Public profile and diary commands

- `/profile [username]`
  Shows a profile card with bio, counts, favorites, and profile avatar.
- `/diary [username] [count]`
  Shows the most recent public diary entries for a member. If no count is given, it shows `5`.
- `/film query`
  Looks up a film by title or Letterboxd URL and returns a card with rating, runtime, genres, cast, and source context.
- `/list query [username]`
  Searches a member's recent public list history by title and shows the first matching list. This uses public RSS history, so older or private lists may not appear.
- `/logged film [username]`
  Finds recent public diary logs for a film in the member's RSS feed window. This is a recent-feed lookup, not a full historical search.

### Channel follow commands

- `/follow username [channel]`
  Starts posting new public diary entries from that Letterboxd account into the current channel or a selected channel.
- `/unfollow username [channel]`
  Stops posting diary entries for that account in the chosen channel.
- `/following [channel]`
  Lists all Letterboxd accounts currently being followed in that channel.

### Social and discovery commands

- `/compare other [username]`
  Compares two members and shows shared favorites, shared recent watches, activity pace, and biggest disagreement.
- `/taste other [username]`
  Computes the same comparison data as `/compare` but presents it as a compact compatibility score.
- `/roulette [theme]`
  Picks a random film from a discovery theme such as `horror`, `animation`, `heist`, or a custom search phrase. If no theme is supplied, Flicksy chooses one at random.

### Official API commands

These commands only appear when `FLICKSY_LETTERBOXD_CLIENT_ID` and `FLICKSY_LETTERBOXD_CLIENT_SECRET` are configured.

- `/stats [username]`
  Shows official Letterboxd member stats, including counts, year summaries, and rating distribution.
- `/watchlist [username] [genre] [count]`
  Shows public watchlist titles for a member, optionally filtered by genre or keyword.
- `/watchpick [username] [genre]`
  Pulls up to `50` watchlist candidates, optionally filters them, and returns one random pick.
- `/activity [username] [count]`
  Shows recent official activity items for a member.
- `/discover [genre] [service] [count]`
  Returns currently popular films, optionally filtered by genre and streaming service.
- `/party user1 user2 [user3] [user4] [user5]`
  Loads up to `100` watchlist films for each user, finds the overlap, and randomly picks one common movie for the group.
- `/rec [username]`
  Builds a lightweight recommendation from one of the member's favorites, or a highly rated recent diary entry if no favorites exist, then picks a popular film from one of that movie's genres.

## Setup

1. Create a Discord application and bot in the Discord developer portal.
2. Copy the bot token.
3. Fill out `.env` from `.env.example`.
4. Run `make ci`.
5. Run `make build` or `make run`.
6. Invite the bot with the `applications.commands` and `bot` scopes.

## Environment variables

- `DISCORD_TOKEN`
  Required. Discord bot token.
- `DISCORD_GUILD_ID`
  Optional. If set, commands are registered to one guild for faster iteration. If unset, commands are registered globally.
- `FLICKSY_DATA_DIR`
  Optional. Defaults to `./data`.
- `FLICKSY_HTTP_TIMEOUT`
  Optional. Defaults to `15s`.
- `FLICKSY_POLL_INTERVAL`
  Optional. Defaults to `5m`.
- `FLICKSY_USER_AGENT`
  Optional. Override the default HTTP user agent.
- `FLICKSY_LETTERBOXD_CLIENT_ID`
  Optional. Enables official API-backed commands and search/discovery upgrades.
- `FLICKSY_LETTERBOXD_CLIENT_SECRET`
  Optional. Must be set together with `FLICKSY_LETTERBOXD_CLIENT_ID`.

## Development

```bash
make ci
make test
make build
make run
make release VERSION=v0.1.0
```

## GitHub Actions

- CI runs on every push, pull request, and manual dispatch.
- CD publishes GitHub Release assets when you push a version tag like `v0.1.0`.
- Release artifacts are written to `dist/` and include checksums plus binaries for Linux, macOS, and Windows.

```bash
git tag v0.1.0
git push origin v0.1.0
```

## Notes and limitations

- Flicksy intentionally uses public Letterboxd surfaces instead of private API access.
- Official API support is optional and uses the documented Letterboxd API with client credentials.
- Profile, film, diary, and follow features are based on public profile pages, film pages, and RSS feeds.
- List search is limited to the public RSS list history available for a user.
- `/logged` works against the recent public RSS window, not a complete historical archive.
- Film search falls back to DuckDuckGo site search when the official API is not configured.
