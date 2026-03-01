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

## Running Tests

Run all tests:

```bash
go test ./...
```
