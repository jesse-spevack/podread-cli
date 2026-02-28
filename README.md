# podread

Command-line interface for [podread.app](https://podread.app) -- turn text into podcast episodes delivered to your personal RSS feed.

## Install

### Homebrew (macOS / Linux)

```sh
brew install jesse-spevack/tap/podread
```

### Shell script

```sh
curl -fsSL https://raw.githubusercontent.com/jesse-spevack/podread-cli/main/install.sh | sh
```

### From source

```sh
go install github.com/jspevack/podread-cli@latest
```

## Authentication

podread uses device code authorization. Run `login` and follow the browser prompt:

```sh
podread auth login
# Open this URL: https://podread.app/auth/device/XXXX
# Enter code: XXXX-XXXX
# Waiting for authorization...
# Logged in as you@example.com
```

Check your session:

```sh
podread auth status
```

Log out:

```sh
podread auth logout
```

Credentials are stored at `~/.config/podread/token`.

## Usage

### Create an episode from a URL

```sh
podread episode create --url https://example.com/article
```

The command waits for processing by default and prints the result when done.

### Create an episode from text

```sh
podread episode create --text "Your text here" --title "My Episode"
```

Or pipe text from stdin:

```sh
cat article.txt | podread episode create --stdin --title "Article"
```

### Fire and forget

```sh
podread episode create --url https://example.com/article --no-wait
```

### Choose a voice

```sh
podread voices                           # list available voices
podread episode create --url https://example.com/article --voice alloy
```

### List episodes

```sh
podread episode list
podread episode list --limit 25
```

### Check episode status

```sh
podread episode status <id>
```

### Delete an episode

```sh
podread episode delete <id>
```

### Get your RSS feed URL

```sh
podread feed
```

Add this URL to any podcast app to receive your episodes.

## JSON output

All commands that produce output support a `--json` flag for structured output:

```sh
podread episode list --json
podread episode create --url https://example.com --json
podread voices --json
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
