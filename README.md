# 🐒 monkey CLI

[![CI](https://github.com/juanhuttemann/monkey-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/juanhuttemann/monkey-cli/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/juanhuttemann/monkey-cli)](https://goreportcard.com/report/github.com/juanhuttemann/monkey-cli)
[![Go Version](https://img.shields.io/github/go-mod/go-version/juanhuttemann/monkey-cli)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A Go CLI/TUI for chatting with Claude (Anthropic's LLM API). Run it without arguments for a full interactive terminal interface, or pass a prompt for quick one-shot output.

![demo](demo.gif)

## Features

- **Interactive TUI** — full-screen chat built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), with markdown rendering and conversation history
- **One-shot CLI** — pipe-friendly: `monkey -p "summarise this" < file.txt`
- **Agentic tool use** — the model can read, write, edit, search, and run bash commands on your behalf, with per-call approval prompts
- **Session persistence** — resume your last conversation with `--continue`
- **Model switching** — cycle between Opus, Sonnet, and Haiku at runtime with `/model`
- **Conversation compaction** — compress long histories with `/compact` to save tokens
- **Custom system prompts** — drop a `MONKEY.md` in your project root or `~/.config/monkey/MONKEY.md`

## Installation

### One-liner

```bash
curl -fsSL https://monkeycli.com/install.sh | sh
```

### From source

```bash
git clone https://github.com/juanhuttemann/monkey-cli.git
cd monkey-cli
go build -o monkey .
```

### go install

```bash
go install github.com/juanhuttemann/monkey-cli@latest
```

> **Note:** Requires Go 1.21 or later.

## Setup

Set your Anthropic API key and at least one model:

```bash
export ANTHROPIC_API_KEY="your-api-key"
export ANTHROPIC_DEFAULT_OPUS_MODEL="claude-opus-4-5"
```

All environment variables:

| Variable | Required | Description |
|---|---|---|
| `ANTHROPIC_API_KEY` | ✅ | Your Anthropic API key |
| `ANTHROPIC_BASE_URL` | no | Override the API base URL (default: `https://api.anthropic.com`) |
| `ANTHROPIC_DEFAULT_OPUS_MODEL` | one required | Opus model ID |
| `ANTHROPIC_DEFAULT_SONNET_MODEL` | one required | Sonnet model ID |
| `ANTHROPIC_DEFAULT_HAIKU_MODEL` | one required | Haiku model ID |

The default active model is the first one set, in order: Opus → Sonnet → Haiku.

## Usage

### Interactive TUI

```bash
monkey
```

| Key | Action |
|---|---|
| `Ctrl+Enter` | Send message |
| `Esc` / `Ctrl+C` | Quit |
| `/model` + `Ctrl+Enter` | Open model picker |
| `/clear` | Start a new session |
| `/compact` | Summarise and compress history |
| `/copy` | Copy last response to clipboard |
| `/ape` | Toggle auto-approve mode (skip tool confirmations) |
| `/exit` | Quit |

### One-shot CLI

```bash
monkey -p "Why do monkeys make the best programmers?"

# Unquoted multi-word prompts also work
monkey -p Write a haiku about bananas

# Resume the last saved session
monkey --continue
```

### System prompt

Create a `MONKEY.md` in your working directory (or `~/.config/monkey/MONKEY.md` for a global default) and monkey will include it as the system prompt for every conversation.

## Agentic tools

When the model needs to perform actions, it calls built-in tools. Each call shows a confirmation dialog unless `/ape` mode is active.

| Tool | Description |
|---|---|
| `bash` | Run a shell command |
| `read` | Read a file |
| `write` | Create or overwrite a file |
| `edit` | Make targeted edits to a file |
| `glob` | Find files by pattern |
| `grep` | Search file contents |
| `web_search` | Search the web via DuckDuckGo |
| `web_fetch` | Fetch and extract text from a URL |

## Running tests

```bash
go test ./...
```

## Contributing

Contributions are welcome! Please open an issue first to discuss what you'd like to change. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT](LICENSE) © Juan Huttemann
