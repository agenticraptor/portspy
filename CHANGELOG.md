# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-06-15

### Added

- Initial release. 🎉
- Interactive TUI (Bubble Tea) listing every listening TCP/UDP port with the
  process, project, service, start time, and bind address behind it.
- Project-aware enrichment: detects the owning project from the process working
  directory (Go, Node, Rust, Python, Ruby, PHP, Java, and more) and guesses the
  service (Vite, Next.js, Postgres, Redis, Docker, …) from the command line.
- One-key kill from the TUI with a confirmation dialog: graceful `x`
  (SIGTERM → SIGKILL) and immediate force-kill `X`.
- Detail view (`enter`) showing the executable, project root, exact start time,
  and the process lineage (`npm ← zsh`) for the selected port.
- Live auto-refresh, fuzzy filtering (`/`), and sorting by port, PID, process,
  or uptime (`s`).
- Scriptable CLI: `portspy list` (table or `--json`), `portspy kill <port…>`
  (with `--force` and `--yes`), and `portspy doctor`.
- Single static binary with no runtime dependencies; cross-platform for macOS,
  Linux, and Windows (amd64 + arm64).

[Unreleased]: https://github.com/agenticraptor/portspy/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/agenticraptor/portspy/releases/tag/v0.1.0
