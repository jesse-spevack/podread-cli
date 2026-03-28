# podread

Command-line interface for [podread.app](https://podread.app) — turn articles and text into podcast episodes delivered to your personal RSS feed.

## Quick start

```sh
brew install jesse-spevack/tap/podread   # or: curl -fsSL https://raw.githubusercontent.com/jesse-spevack/podread-cli/main/install.sh | sh
podread auth login                       # opens browser for device code auth
podread feed                             # get your RSS feed URL — subscribe in any podcast app
podread episode create --url https://example.com/article
```

## Install

### Homebrew (macOS / Linux)

```sh
brew install jesse-spevack/tap/podread
```

### Shell script

```sh
curl -fsSL https://raw.githubusercontent.com/jesse-spevack/podread-cli/main/install.sh | sh
```

### Claude Code plugin

This repo is also a [Claude Code plugin](https://code.claude.com/docs/en/plugins). Install it so Claude can create podcast episodes for you:

```
/plugin > Add marketplace > jesse-spevack/podread-cli
/plugin install podread@podread-cli
```

Once installed, just ask Claude:

> "Turn this article into a podcast episode: https://example.com/article"

## Authentication

podread uses device code authorization. Run `login` and follow the browser prompt:

```sh
podread auth login
# Open this URL: https://podread.app/auth/device/XXXX
# Enter code: XXXX-XXXX
# Waiting for authorization...
# Logged in as you@example.com
```

Check your session or log out:

```sh
podread auth status
podread auth logout
```

Credentials are stored at `~/.config/podread/token`.

## Usage

### Create an episode

```sh
# From a URL
podread episode create --url https://example.com/article

# From inline text
podread episode create --text "Your text here" --title "My Episode"

# From stdin
cat article.txt | podread episode create --stdin --title "Article"
```

The command waits for processing by default (~1-5 minutes). Use `--no-wait` to return immediately.

### Choose a voice

```sh
podread voices                                                      # list available voices
podread episode create --url https://example.com/article --voice alloy
```

### Manage episodes

```sh
podread episode list              # recent episodes (ep is an alias for episode)
podread episode status <id>       # check processing status
podread episode delete <id>
```

### Subscribe to your feed

```sh
podread feed
```

Add this URL to any podcast app. New episodes appear automatically.

## JSON output

All commands support `--json` for structured output:

```sh
podread episode list --json
podread episode create --url https://example.com --json
```

## Environment variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PODREAD_API_URL` | Override the API base URL | `https://podread.app` |

## Building from source

```sh
git clone https://github.com/jesse-spevack/podread-cli.git
cd podread-cli
make build        # build for current platform
make build-all    # cross-compile for all platforms
```

Binaries are written to `dist/`.
