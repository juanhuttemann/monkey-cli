# Contributing to monkey

Thanks for your interest in contributing!

## Getting started

1. Fork the repository and clone your fork
2. Install Go 1.21+
3. Run `go test ./...` to confirm everything passes

## Workflow

1. **Open an issue first** for significant changes — discuss the approach before writing code
2. Create a branch: `git checkout -b feature/my-thing`
3. Make your changes, keeping commits small and focused
4. Add or update tests for any changed behaviour
5. Run the full test suite: `go test ./...`
6. Open a pull request against `main`

## Code style

- `gofmt` / `goimports` formatted (enforced by CI lint)
- Follow standard Go idioms and keep packages coherent
- Keep the `tools/`, `api/`, `config/`, and `tui/` package boundaries clean

## Reporting bugs

Use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md) and include your OS, Go version, and monkey version.
