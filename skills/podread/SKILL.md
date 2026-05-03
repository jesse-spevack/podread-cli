---
name: podread
description: Convert articles, URLs, or text to podcast episodes using the podread CLI. Use when the user asks to create a podcast, convert an article to audio, turn text into a listenable episode, or mentions podread.
allowed-tools: [Bash, Read]
---

# podread — Article-to-Podcast CLI

Use the `podread` CLI to convert articles, URLs, or text into podcast episodes via podread.app. Episodes are processed server-side and delivered through an RSS feed the user subscribes to in any podcast app.

## Prerequisites

The `podread` CLI must be installed. Check with:

```bash
which podread
```

If not installed:

```bash
# Homebrew
brew install jesse-spevack/tap/podread

# Or curl
curl -fsSL https://raw.githubusercontent.com/jesse-spevack/podread-cli/main/install.sh | sh
```

## Authentication

Always check auth status first. If not logged in, start the device auth flow:

```bash
# Check auth (prints email, tier, and — for credit-tier accounts — credits remaining + character limit)
podread auth status

# Login (opens browser for device code confirmation)
podread auth login

# Logout
podread auth logout
```

## Creating Episodes

### From a URL

```bash
podread episode create --url https://example.com/article
```

### From inline text

```bash
podread episode create --text "Your long-form text here..." --title "Episode Title"
```

### From piped input (stdin)

```bash
echo "Long text content" | podread episode create --stdin --title "Episode Title"
cat article.txt | podread episode create --stdin --title "From File"
```

### Options

| Flag | Purpose |
|------|---------|
| `--url <url>` | Source URL to convert |
| `--text "..."` | Inline text to convert |
| `--stdin` | Read text from stdin |
| `--title "..."` | Episode title |
| `--author "..."` | Author name (optional) |
| `--voice <name>` | Voice to use (see `podread voices`) |
| `--no-wait` | Return immediately without waiting for processing |
| `--timeout 300` | Custom timeout in seconds (default: 600) |
| `--json` | Output as JSON |

By default, `episode create` waits for processing to complete (~1-5 minutes). Use `--no-wait` to fire-and-forget.

## Managing Episodes

```bash
# List recent episodes (ep is an alias for episode)
podread ep list
podread ep list --limit 20
podread ep list --limit 100 --page 2   # paginate older episodes (max limit 100)
podread ep list --json

# Check processing status
podread episode status <id>

# Delete an episode
podread episode delete <id>
```

## Voices and Feed

```bash
# List available voices
podread voices

# Show RSS feed URL
podread feed
```

## Workflow Examples

### First-time setup

```bash
podread auth login
# Follow browser auth flow, then:
podread feed
# Subscribe to the RSS URL in any podcast app
```

### Convert an article

```bash
podread episode create --url https://example.com/long-article
```

### Pipe markdown into an episode

```bash
cat notes.md | podread episode create --stdin --title "Meeting Notes"
```

### Fire-and-forget with later status check

```bash
podread episode create --url https://example.com/article --no-wait --json
# Returns episode ID immediately, check later:
podread episode status <id>
```

## Notes

- `ep` is a shorthand alias for `episode` in all subcommands
- `--json` is available on most commands for scripting or piping to `jq`
- The RSS feed URL is stable — subscribe once, new episodes appear automatically
- Episode processing typically takes 1-5 minutes depending on source length
