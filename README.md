# mogger

A Go CLI/TUI application for chatting with an LLM API. Run it without arguments for an interactive terminal interface, or pass a prompt directly for one-shot CLI output.

## Setup

Ensure you have Go 1.25 or later installed, then set the required environment variables:

```bash
export ANTHROPIC_API_KEY="your-api-key"
export ANTHROPIC_BASE_URL="https://api.anthropic.com"
export ANTHROPIC_MODEL="claude-3-5-sonnet-20241022"
```

## Usage

### Interactive TUI

Run with no arguments to open the interactive chat interface:

```bash
./mogger
```

- Type your message in the input box at the bottom
- Press **Ctrl+Enter** to send
- Press **Esc** or **Ctrl+C** to quit

### One-shot CLI

Pass a prompt with `-p` or `--prompt` for a single response printed to stdout:

```bash
./mogger -p "What is the capital of France?"
```

The `-p` flag supports both quoted and unquoted multi-word prompts:

```bash
# Quoted
./mogger -p "Write a haiku about coding"

# Unquoted
./mogger -p Write a haiku about coding
```

## Architecture

The codebase is organised into four packages:

- `api/` — HTTP client for the LLM API; handles single-turn (`SendMessage`) and multi-turn (`SendMessageWithHistory`) requests
- `config/` — environment variable loading (`ANTHROPIC_API_KEY`, `ANTHROPIC_BASE_URL`, `ANTHROPIC_MODEL`)
- `tui/` — interactive terminal UI built with [Bubble Tea](https://github.com/charmbracelet/bubbletea); manages conversation history, input, and loading state
- `main.go` — entry point; routes to TUI or CLI mode based on whether `-p` is provided

## Running Tests

```bash
go test ./...
```
