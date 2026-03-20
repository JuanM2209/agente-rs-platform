# nucleus-helper — Windows TCP Port Mapper

A standalone Windows CLI that maps local TCP ports to remote Nucleus session endpoints. It is the MVP stepping stone toward a full Windows system-tray GUI; the TCP forwarding engine in `internal/mapper.go` is intentionally GUI-free so it can be reused directly by a future tray application.

---

## Installation

### Pre-built binary

Download `nucleus-helper.exe` from the project releases page and place it somewhere on your `%PATH%` (e.g. `C:\Tools\`).

### Build from source

Requirements: Go 1.22 or later.

```bat
git clone https://github.com/nucleus-portal/windows-helper.git
cd windows-helper
go mod download
build.bat
```

The resulting `nucleus-helper.exe` is a self-contained binary with no runtime dependencies.

---

## Quick Start

```bat
REM 1. Authenticate
nucleus-helper login --api-url https://api.nucleus.example.com --email you@example.com

REM 2. List your active sessions
nucleus-helper sessions

REM 3. Forward a session to a local port (e.g. RDP on 3389)
nucleus-helper map --session-id <SESSION_ID> --local-port 3389

REM 4. Open your RDP client and connect to 127.0.0.1:3389

REM 5. Stop the mapping manually (or press Ctrl+C in step 3)
nucleus-helper unmap --session-id <SESSION_ID>
```

---

## Commands

### `login`

Authenticates with the Nucleus API and saves credentials to `~\.nucleus\config.json`.

```
nucleus-helper login --api-url <URL> --email <EMAIL>
```

| Flag | Required | Description |
|------|----------|-------------|
| `--api-url` | Yes | Base URL of the Nucleus API |
| `--email` | Yes | Your account email address |

You will be prompted for your password. The password is never stored; only the JWT token is persisted.

---

### `sessions`

Lists your active Nucleus sessions fetched from the API.

```
nucleus-helper sessions
```

Output columns: Session ID, Device, Remote Host, Port, TTL, Status.

---

### `map`

Binds `127.0.0.1:<local-port>` and forwards all TCP connections to the session's remote endpoint. The process stays in the foreground until you press Ctrl+C or the session TTL expires.

```
nucleus-helper map --session-id <ID> --local-port <PORT>
```

| Flag | Required | Description |
|------|----------|-------------|
| `--session-id` | Yes | Session ID from `nucleus-helper sessions` |
| `--local-port` | Yes | Local TCP port to bind (1–65535) |

**Example — RDP:**
```bat
nucleus-helper map --session-id abc123 --local-port 3389
REM then connect mstsc to 127.0.0.1:3389
```

**Example — SSH:**
```bat
nucleus-helper map --session-id def456 --local-port 2222
ssh user@127.0.0.1 -p 2222
```

---

### `unmap`

Stops an active mapping without terminating the process (useful when running `nucleus-helper` as a background service).

```
nucleus-helper unmap --session-id <ID>
```

---

### `status`

Displays all mappings currently active in this process, including bytes forwarded and TTL countdown.

```
nucleus-helper status
```

Output columns: Session ID, Local Port, Remote Host, Remote Port, Bytes Fwd, Started, TTL, Status.

---

## Global Flags

| Flag | Description |
|------|-------------|
| `-v`, `--verbose` | Enable debug-level logging to stderr |

---

## Configuration

Credentials are stored in plain JSON at:

```
%USERPROFILE%\.nucleus\config.json
```

The file is created with permissions `0600` (owner read/write only). Do not commit this file to source control.

---

## Architecture

```
nucleus-helper/
├── cmd/
│   ├── main.go              # Cobra root command + shared Mapper wiring
│   └── commands/
│       ├── login.go         # `login` subcommand
│       ├── sessions.go      # `sessions` subcommand
│       ├── map.go           # `map` subcommand
│       ├── unmap.go         # `unmap` subcommand
│       └── status.go        # `status` subcommand
└── internal/
    ├── mapper.go            # TCP forwarding engine (GUI-ready)
    ├── auth.go              # Token persistence (~/.nucleus/config.json)
    └── api_client.go        # Nucleus REST API client
```

### TCP forwarding

`internal.Mapper` binds a local `net.Listener` per session, accepts connections in a goroutine, and bidirectionally proxies each connection to the remote host using `io.Copy` in two goroutines. Each mapping carries a `context.Context` with a deadline set to the session `ExpiresAt` timestamp; when the deadline fires, the listener is closed and all active connections are torn down automatically.

### GUI extension path

To convert this into a Windows tray application:

1. Add a dependency on a tray library (e.g. `github.com/getlantern/systray`).
2. Create a new `cmd/tray/main.go` that imports `internal.Mapper` and wires menu items to `StartMapping` / `StopMapping` / `ListMappings`.
3. The existing `cmd/main.go` CLI remains unchanged.

No changes to `internal/` are needed.

---

## Development

```bat
REM Run tests
go test ./...

REM Build for Windows (stripped binary)
make build-windows

REM Build for Linux (cross-compile)
make build-linux
```

---

## License

See the root repository LICENSE file.
