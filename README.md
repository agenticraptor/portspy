<div align="center">

# 🔌 portspy

### See — and kill — whatever's hogging your local ports. In one key.

`portspy` is a fast terminal UI that lists everything listening on your machine,
tells you **what it actually is** (the project, the dev server, the command, how
long it's been up), and lets you kill it with a single keypress. The *"why is
:3000 taken AGAIN?"* problem, solved.

[![CI](https://github.com/agenticraptor/portspy/actions/workflows/ci.yml/badge.svg)](https://github.com/agenticraptor/portspy/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/agenticraptor/portspy?sort=semver)](https://github.com/agenticraptor/portspy/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/agenticraptor/portspy.svg)](https://pkg.go.dev/github.com/agenticraptor/portspy)
[![Go Report Card](https://goreportcard.com/badge/github.com/agenticraptor/portspy)](https://goreportcard.com/report/github.com/agenticraptor/portspy)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

</div>

---

> **Try it in one line — no install, no signup, no config:**
>
> ```bash
> go run github.com/agenticraptor/portspy/cmd/portspy@latest
> ```

<!--
  📸 The hero GIF is the single biggest driver of stars. Render one in a single
  command with the included VHS tape, then uncomment the <img> below:

      go build -o /tmp/portspy ./cmd/portspy
      PATH=/tmp:$PATH vhs docs/demo.tape    # writes docs/demo.gif

  <p align="center"><img src="docs/demo.gif" alt="portspy demo" width="760"></p>
-->

```text
 🔌 portspy   7 listening  ·  sort: port                       updated 09:14:02

  PORT   PROTO  PID     PROCESS       UPTIME  ADDRESS    WHAT
  ────────────────────────────────────────────────────────────────────────────
  3000   tcp    44213   node          6m      localhost  Next.js · storefront
▌ 5173   tcp    44980   node          6m      localhost  Vite · admin-web      ▐
  5432   tcp    1188    postgres      3d      localhost  PostgreSQL
  6379   tcp    1190    redis-server  3d      localhost  Redis
  8080   tcp    52111   java          12m     *          Spring Boot · billing
  8765   tcp    61002   python3       2s      *          api-scratch
  9229   tcp    44980   node          6m      localhost  Vite · admin-web

  ▸ node /Users/me/code/admin-web/node_modules/.bin/vite --port 5173

  ↑/k up · ↓/j down · enter details · x kill · / filter · s sort · ? help · q quit
```

Press `enter` for the full story behind a port — including the process lineage —
or `x` to kill it:

```text
        ╭──────────────────────────────────────╮     ╭─────────────────────────────╮
        │  :5173/tcp                           │     │  Terminate this process?    │
        │                                      │     │                             │
        │  Process   node (pid 44980)          │     │  Port      :5173/tcp        │
        │  Service   Vite                      │     │  Process   node (pid 44980) │
        │  Project   admin-web (node)          │     │  Uptime    6m               │
        │  Root      /Users/me/code/admin-web  │     │  Project   admin-web        │
        │  Address   localhost                 │     │  Service   Vite             │
        │  Started   2026-06-15 09:08:14 (6m)  │     │  Parent    npm ← zsh        │
        │  Parent    npm ← zsh                 │     │                             │
        │                                      │     │  node …/.bin/vite --port…   │
        │  node …/.bin/vite --port 5173        │     │                             │
        │                                      │     │  [y] yes    [n] cancel      │
        │  [x] kill  [X] force  [esc] close    │     ╰─────────────────────────────╯
        ╰──────────────────────────────────────╯
```

## The :3000 problem

You run `npm run dev`. **"Port 3000 is already in use."** Again. So begins the
ritual: `lsof -i :3000`, squint at a PID, `kill -9`, hope it was the right one.
Was that stray process your *other* project's dev server? A zombie from last
week? A database? `lsof` won't tell you — it gives you a number, not a story.

portspy gives you the story: **which project, which dev server, which command,
running for how long** — for every port at once — and one key to end it.

## Why you'll like it

- **It tells you what it _is_.** Not just `node (44980)`, but **`Vite ·
  admin-web`**. portspy reads each process's working directory to find the
  owning project (Go, Node, Rust, Python, Ruby, PHP, Java…) and recognizes
  common dev servers and databases (Vite, Next.js, Postgres, Redis, Docker…).
- **Press `enter` for the full story.** A detail view shows the executable, the
  project root, the exact start time, and the **process lineage** (`npm ← zsh`)
  — so you know precisely what you're about to kill.
- **One-key kill, done safely.** `x` terminates gracefully (SIGTERM, escalating
  to SIGKILL); `X` force-kills. Both confirm first, and portspy never kills
  itself.
- **Live & searchable.** Auto-refreshing table you can filter (`/`) and sort
  (`s`) by port, PID, process, or uptime.
- **Spots the exposed ones.** Anything bound beyond loopback is flagged, so a
  service accidentally listening on `0.0.0.0` jumps out.
- **Scriptable too.** `portspy list --json` for your shell pipelines, and
  `portspy kill 3000 --yes` for your Makefiles.
- **One static binary.** No runtime, no daemon, no telemetry, no network. Works
  on macOS, Linux, and Windows.

## Install

### `go install`

```bash
go install github.com/agenticraptor/portspy/cmd/portspy@latest
```

### Pre-built binaries

Grab a binary for your OS/arch from the
[**Releases**](https://github.com/agenticraptor/portspy/releases) page.

### Homebrew (macOS / Linux)

```bash
brew install agenticraptor/tap/portspy
```

> Available once the Homebrew tap is published — see the note in
> [`.goreleaser.yaml`](.goreleaser.yaml) to enable it.

### From source

```bash
git clone https://github.com/agenticraptor/portspy
cd portspy
make install
```

## Quickstart

```bash
# 1. Open the interactive table (this is the whole product)
portspy

# 2. Free a port without the TUI
portspy kill 3000

# 3. See what's listening, as JSON, for scripts
portspy list --json | jq '.[] | select(.exposed)'

# 4. Check what portspy can see on your machine
portspy doctor
```

Full key bindings, flags, and the JSON schema live in
[**docs/usage.md**](docs/usage.md).

## How it works

```
listening sockets ──┐
 (TCP LISTEN /      │
  bound UDP)        ├─► map socket ➜ PID ──┐
                    │                       │
per-PID details ────┘                       ├─► enrich ──► render
 (name, command,                            │     • project from cwd
  exe, start time,                          │     • service from command
  parent lineage)                           │     • merge IPv4/IPv6
                                            │     • flag exposed binds
                                            └─►  TUI  ·  table  ·  JSON
                                                          │
                                              kill ◄──────┘  (SIGTERM ➜ SIGKILL,
                                                              confirmed)
```

Cross-platform socket and process discovery is handled by
[gopsutil](https://github.com/shirou/gopsutil); the TUI is built with
[Bubble Tea](https://github.com/charmbracelet/bubbletea). See
[**docs/platforms.md**](docs/platforms.md) for per-OS support and the (small)
privilege caveats.

## Privacy

portspy runs entirely on your machine and **makes no network connections** — no
telemetry, no update checks, nothing. It reads local socket and process
information and prints it. The only thing it changes about your system is the
processes you explicitly choose to kill.

## Contributing

Contributions are very welcome — see [CONTRIBUTING.md](CONTRIBUTING.md). Good
first issues include new service-detection patterns, a `--watch` mode for
`portspy list`, and richer Windows process detail. Please also read our
[Code of Conduct](CODE_OF_CONDUCT.md).

## License

[MIT](LICENSE) © portspy contributors.
