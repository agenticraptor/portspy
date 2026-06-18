# Usage

portspy has two faces: an interactive **TUI** (the default) and a small set of
**scriptable commands**.

## The TUI

```bash
portspy            # open the interactive table
portspy --proto tcp
```

### Keys

| Key | Action |
|-----|--------|
| `↑`/`k`, `↓`/`j` | Move the selection |
| `g` / `G` | Jump to top / bottom |
| `enter` / `i` | Show full details (executable, project root, start time, process lineage) |
| `/` | Filter by port, process, project, or service (Enter to apply, Esc to clear) |
| `s` | Cycle the sort order: port → pid → process → uptime |
| `r` | Refresh now (the table also auto-refreshes every few seconds) |
| `x` | Kill the selected process gracefully (SIGTERM → SIGKILL), with confirmation |
| `X` | Force-kill immediately (SIGKILL), with confirmation |
| `?` | Toggle the full help |
| `q` / `Ctrl-C` | Quit |

The status line shows the full command of the selected process, and rows bound
beyond loopback show their address (`*` for all interfaces) so you can spot
anything unexpectedly exposed.

## `portspy list`

Print the listeners as a table, or as JSON for scripting.

```bash
portspy list                     # aligned table
portspy list --proto tcp         # TCP only (tcp | udp | all)
portspy list --json              # stable JSON array
portspy list --no-color          # disable ANSI styling
```

Examples:

```bash
# What is on :3000 right now?
portspy list --json | jq '.[] | select(.port == 3000)'

# Everything exposed beyond localhost
portspy list --json | jq '.[] | select(.exposed)'

# Just the ports owned by node processes
portspy list --json | jq -r '.[] | select(.process=="node") | .port'
```

### JSON shape

Each entry looks like:

```json
{
  "proto": "tcp",
  "port": 3000,
  "addr": "127.0.0.1",
  "display_addr": "localhost",
  "exposed": false,
  "pid": 4242,
  "process": "node",
  "command": "node node_modules/.bin/vite",
  "exe": "/usr/local/bin/node",
  "service": "Vite",
  "project": { "name": "web", "type": "node", "root": "/Users/me/code/web" },
  "started": "2026-06-15T09:14:02Z",
  "uptime_seconds": 384,
  "label": "Vite · web"
}
```

Fields that can't be resolved (`project`, `service`, `started`, …) are omitted.

## `portspy kill`

Free one or more ports by terminating whatever is listening.

```bash
portspy kill 3000                # graceful; prompts for confirmation
portspy kill 3000 8080 5173      # several at once
portspy kill 3000 --force        # skip straight to SIGKILL
portspy kill 3000 --yes          # no prompt (required when not a terminal)
portspy kill 5432 --proto tcp    # only match the TCP listener
```

A graceful kill sends SIGTERM and waits a few seconds for the process to exit,
escalating to SIGKILL only if needed (`TerminateProcess` on Windows). portspy
will never kill itself, and de-duplicates when several ports map to one process.

## `portspy doctor`

Print platform, privilege, and a quick self-test scan so you can see how much
detail portspy can resolve on your machine.

```bash
portspy doctor
```

## `portspy version`

```bash
portspy version
# portspy 0.1.0 (commit abc1234, built 2026-06-15T...)
```
