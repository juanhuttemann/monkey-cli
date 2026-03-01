# mogger

A Go CLI application that outputs a greeting message from an LLM API.

## Setup

No external dependencies required. Ensure you have Go 1.25 or later installed.

Set the required environment variables:

```bash
export ANTHROPIC_API_KEY="your-api-key"
export ANTHROPIC_BASE_URL="https://api.anthropic.com"
export ANTHROPIC_MODEL="claude-3-5-sonnet-20241022"
```

## Usage

Run the application:

```bash
go run main.go
```

The application makes an HTTP POST request to the configured LLM API and outputs the returned greeting message.

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
