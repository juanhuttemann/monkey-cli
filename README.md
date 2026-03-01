# mogger

A Go CLI application that sends custom prompts to an LLM API and displays the response.

## Setup

No external dependencies required. Ensure you have Go 1.25 or later installed.

Set the required environment variables:

```bash
export ANTHROPIC_API_KEY="your-api-key"
export ANTHROPIC_BASE_URL="https://api.anthropic.com"
export ANTHROPIC_MODEL="claude-3-5-sonnet-20241022"
```

## Usage

The application requires a `-p` or `--prompt` flag to specify the prompt to send to the LLM.

```bash
go run main.go -p "What is the capital of France?"
```

The `-p` flag supports both quoted and unquoted multi-word prompts:

```bash
# Quoted prompt
./mogger -p "Write a haiku about coding"

# Unquoted multi-word prompt
./mogger -p Write a haiku about coding
```

If the `-p` flag is not provided, a usage message is displayed and the application exits with a non-zero status code.

## Architecture

The codebase is organized into three packages for clear separation of concerns:

- `api/` - HTTP client for communicating with the LLM API, including models for request/response structures
- `config/` - Configuration loading from environment variables with interface-based design for testability
- `main.go` - Thin orchestration layer that wires together the API and config packages

## Running Tests

Run all tests:

```bash
go test ./...
```
