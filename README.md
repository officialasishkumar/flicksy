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

- `/help`
- `/connect username`
- `/disconnect`
- `/profile [username]`
- `/diary [username] [count]`
- `/film query`
- `/list query [username]`
- `/follow username [channel]`
- `/unfollow username [channel]`
- `/following [channel]`
- `/logged film [username]`
- `/refresh [username]`
- `/compare other [username]`
- `/taste other [username]`
- `/roulette [theme]`
- `/stats [username]` when official API credentials are configured
- `/watchlist [username] [genre] [count]` when official API credentials are configured
- `/watchpick [username] [genre]` when official API credentials are configured
- `/activity [username] [count]` when official API credentials are configured
- `/discover [genre] [service] [count]` when official API credentials are configured
- `/party user1 user2 [user3] [user4] [user5]` when official API credentials are configured
- `/rec [username]` when official API credentials are configured

## Setup

1. Create a Discord application and bot in the Discord developer portal.
2. Copy the bot token.
3. Fill out `.env` from `.env.example`.
4. Run `make test`.
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
