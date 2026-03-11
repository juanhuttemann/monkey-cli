# Changelog

All notable changes to monkey-cli will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-03-11
### Added
- Thread context through tool executors and fix cancelled-prompt history

## [0.3.1] - 2026-03-10
### Fixed
- Web search POST request handling and webfetch bare domain and HTML truncation

## [0.3.0] - 2026-03-10
### Added
- Add monkey update subcommand to self-update to latest release
- Add -v/--version flag to print app name and version
- Default to latest Claude models when none configured

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

[Unreleased]: https://github.com/juanhuttemann/monkey-cli/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/juanhuttemann/monkey-cli/compare/v0.3.1...v0.4.0
