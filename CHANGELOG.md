# Changelog

All notable changes to monkey-cli will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-03-10
### Added
- Web search and web fetch tools with user-agent rotation

### Fixed
- Handle resp.Body.Close error to satisfy errcheck
- Replace strings.Builder with []byte in streamBuf to prevent panic
- Track cursor position in multiline input view
- Multiline-aware Up/Down arrow history navigation

### Improved
- Avoid re-rendering pickers in syncViewportHeight to measure height
- Cache prior message renders during streaming
- Add install.sh and curl one-liner to README
- Improve demo resolution

## [0.1.0] - 2025-03-07
### Added
- Initial public release
- Go module path updated to `github.com/juanhuttemann/monkey-cli`
- CI/CD pipeline with GoReleaser for cross-platform builds
- Code coverage reporting with Codecov
