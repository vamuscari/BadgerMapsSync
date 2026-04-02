# Cross-Platform OS/Server Review

## Findings (ordered by severity)

### [P0] Windows `server status`/`server stop` are Unix-signal based
Uses `process.Signal(syscall.Signal(0))` for liveness and `SIGTERM` for stop in shared code:
- `app/server/server_manager.go:69`
- `app/server/server_manager.go:94`

On Windows this is unreliable/not supported behavior, so running servers can be misreported as stopped.

### [P0] Windows service runtime is a stub and does not run the HTTP server
The service `Execute` loop explicitly says it is placeholder logic and only waits for control events:
- `cli/server/service_windows.go:33`
- `cli/server/service_windows.go:39`

Result: service can be “running” without serving webhook endpoints.

### [P1] Background server launch does not propagate active config context
Both launchers run `exec.Command(executable, "server")` only:
- `app/server/server_starter_unix.go:26`
- `app/server/server_starter_windows.go:26`

No explicit `--config` pass-through means alternate profiles can drift between foreground/background/server-service executions.

### [P1] Scheduler job file location depends on `state.ConfigFile` (flag), not resolved config path
It falls back to `.` when the flag is not set:
- `app/server/scheduler.go:516`
- `app/server/scheduler.go:593`

But config discovery sets `a.ConfigFile` (resolved path), not `state.ConfigFile`:
- `app/app.go:188`
- `app/app.go:430`

This can place `scheduled_jobs.json` in CWD (especially problematic for services).

### [P1] PID file path handling is not anchored to active config and parent dir creation is missing at write time
PID path is initialized globally once:
- `app/app.go:137`

PID writes do not ensure directory exists:
- `app/server/server_starter_unix.go:39`
- `app/server/server_starter_windows.go:37`

This can fail on fresh profiles/service accounts and collide across multiple config profiles.

### [P2] Host/address handling is not IPv6-safe
Host validation only accepts hostname/`localhost`/IPv4 regex:
- `gui/welcome.go:1085`

Bind address is formed with `fmt.Sprintf("%s:%d", host, port)`:
- `cli/server/presenter.go:112`

This blocks or breaks common IPv6 host forms.

### [P3] OS support messaging drifts from implementation
CLI help says default config is `$HOME/.badgermaps.yaml`:
- `main.go:87`

But code uses OS-specific config dir + `config.yaml`.

Build script header/usage suggests Linux support, but selectable targets are only `darwin, windows`:
- `build.sh:3`
- `build.sh:16`

Linux block is commented out:
- `build.sh:118`

## What looks good
Most runtime path construction is already using `filepath.Join`, so there is no broad “Linux slash path” issue in core Go code.

## Assumptions locked
1. Target **CLI server parity first** across Windows/macOS/Linux; defer full Windows service parity to follow-up.
2. Anchor server state files (**PID, scheduled jobs, default server logs**) **next to the active config file**.
