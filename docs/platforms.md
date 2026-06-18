# Platform support

portspy is a single static binary (no runtime, no cgo) and runs on macOS,
Linux, and Windows. Socket and process discovery is powered by
[gopsutil](https://github.com/shirou/gopsutil).

| Platform | Listening ports | Process & command | Project (cwd) | Start time | Kill |
|----------|:---------------:|:-----------------:|:-------------:|:----------:|:----:|
| **Linux**   | ✅ | ✅ | ✅ | ✅ | ✅ |
| **macOS**   | ✅ | ✅ | ✅¹ | ✅ | ✅ |
| **Windows** | ✅ | ✅ | ✅¹ | ✅ | ✅² |

¹ The process **working directory** (used to detect the owning project) is read
on a best-effort basis. For processes owned by another user it may be empty
without elevated privileges; portspy falls back to the executable's directory
and then to the command line.

² On Windows there is no SIGTERM; both the graceful and force paths use
`TerminateProcess`. The graceful/force distinction is therefore cosmetic there.

## Privileges

You can always see **which ports are listening**. Reading the full **command
line and working directory** for processes owned by *other* users can require
elevated privileges:

- **macOS / Linux:** run with `sudo` to attribute every socket. Without it,
  some rows may show `unknown` for the process or omit the project.
- **Windows:** run the terminal **as Administrator** for complete detail.

portspy degrades gracefully: it always shows the port and whatever it could
resolve, and never fails the whole scan because one process was unreadable. Run
`portspy doctor` to see how many sockets it could fully attribute on your
machine.

## Notes

- **UDP "listeners":** a UDP socket has no `LISTEN` state, so portspy reports a
  UDP socket as a listener when it is bound and has no connected remote peer.
  Use `--proto tcp` if you only care about TCP servers.
- **IPv4 + IPv6 merge:** a server bound to both `0.0.0.0` and `[::]` on the same
  port is shown as a single row to keep the table readable.
