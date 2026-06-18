# Contributing to portspy

Thanks for your interest in contributing! portspy aims to be a small, fast,
focused, dependency-light tool — contributions that keep it that way are
especially appreciated.

## Getting started

```bash
git clone https://github.com/agenticraptor/portspy
cd portspy
go mod tidy        # fetch dependencies & populate go.sum
make build         # build into ./bin/portspy
make test          # run the unit tests
make run           # launch the TUI
```

Requirements:

- Go 1.22 or newer
- (optional) [`golangci-lint`](https://golangci-lint.run/) for `make lint`
- (optional) [`goreleaser`](https://goreleaser.com/) for `make snapshot`

## Development workflow

1. Fork the repo and create a feature branch from `main`.
2. Make your change, with tests where it makes sense.
3. Run the full check suite locally:
   ```bash
   make fmt vet test
   ```
4. Open a pull request. Fill in the PR template and link any related issue.

CI runs `gofmt`, `go vet`, `golangci-lint`, and the test suite on Linux, macOS,
and Windows. All checks must pass before review.

## Architecture at a glance

portspy keeps the system-specific code small and isolates the pure logic so it
stays testable:

- `internal/ports` — the engine. Enumerates listening sockets, maps them to
  processes, and enriches each with start time, lineage, project, and service.
  The cross-platform glue lives in `system.go` (gopsutil); the merge/enrich
  logic in `scan.go` (`build`) is pure and unit-tested with fakes.
- `internal/killer` — graceful-then-forceful process termination, with the
  escalation policy isolated behind injectable hooks for testing.
- `internal/render` — table and JSON output for the non-interactive CLI.
- `internal/tui` — the Bubble Tea interactive UI.
- `internal/cli` — the cobra command wiring (`list`, `kill`, `doctor`, …).

If you add behavior, prefer to put the logic in a pure function in `ports`,
`killer`, or `render` (easy to test) and keep the system calls thin.

## Commit messages

We use [Conventional Commits](https://www.conventionalcommits.org/). This keeps
the generated changelog readable and drives semantic-version bumps.

```
feat: add a column for the bind interface
fix: handle UDP sockets with no owning process
docs: clarify macOS permission behavior
test: cover the IPv4/IPv6 merge path
chore: bump gopsutil to v3.24.6
```

## Coding guidelines

- **Keep dependencies minimal.** portspy intentionally ships with a small set of
  direct dependencies. Prefer the standard library; if a new dependency is truly
  needed, call it out in the PR description.
- **Tolerant enumeration.** Reading process details for other users can fail
  without elevated privileges. Always degrade gracefully — show the port and
  whatever is known, never crash the scan.
- **Killing is serious.** Anything that terminates a process must be explicit,
  confirmed (in the TUI and CLI), and never target portspy itself.
- **Format with `gofmt -s`** and keep `go vet` clean.

## Good first issues

- Add Cursor / `pnpm` / `bun` service detection patterns.
- Improve Windows process detail (some fields need elevation).
- Add a `--watch` mode to `portspy list`.
- Add a column or filter for the bind interface (loopback vs exposed).

## Reporting bugs & requesting features

Use the [issue templates](https://github.com/agenticraptor/portspy/issues/new/choose).
For anything security-related, please follow [SECURITY.md](SECURITY.md) instead
of opening a public issue.

## License

By contributing, you agree that your contributions will be licensed under the
[MIT License](LICENSE).
