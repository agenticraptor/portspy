# Security Policy

## Supported versions

The latest released minor version receives security fixes. Please upgrade to the
most recent release before reporting an issue.

## Reporting a vulnerability

Please **do not** open a public issue for security problems.

Instead, report it privately through GitHub's built-in vulnerability reporting:
[**Report a vulnerability**](https://github.com/agenticraptor/portspy/security/advisories/new).
This keeps the report confidential between you and the maintainers until a fix
is released.

Please include:

- A description of the issue and its impact.
- Steps to reproduce (a minimal proof of concept is ideal).
- Affected version(s) and platform.

We aim to acknowledge reports within **72 hours** and to provide a remediation
timeline after triage. We will credit reporters in the release notes unless you
prefer to remain anonymous.

## Scope & data handling notes

portspy is a local tool. A few things worth knowing for your own threat model:

- **No network, no telemetry.** portspy never makes outbound network
  connections. It reads local socket and process information and prints it; it
  sends nothing anywhere.
- **It can terminate processes.** `portspy kill` and the TUI's kill action send
  real signals (SIGTERM, escalating to SIGKILL; `TerminateProcess` on Windows).
  Both paths require explicit confirmation and never target portspy itself, but
  treat the tool with the same care as `kill(1)`.
- **Privileges.** Reading full command lines and working directories for
  processes owned by **other** users may require elevated privileges. portspy
  degrades gracefully when it cannot read a process rather than failing, so run
  it with only the privileges you need.
