# Nucleus Remote Access Portal: Current State

Last updated: 2026-03-20

## Purpose

The project provides a search-first remote access portal for Nucleus Linux devices. A user looks up a `Device ID`, inspects discovered endpoints, and starts either:

- a browser session for web ports such as `80`, `443`, `1880`, `9090`
- a local exported TCP mapping for program ports such as `502`
- a temporary serial bridge flow for MBUSD-backed Modbus RTU to TCP

## Current Runtime Modes

There are currently two runtime modes in the repository:

1. `Portal preview / mock mode`
   - Implemented in `apps/web/src/app/api/v1/*`
   - Used by the current preview deployment when `NEXT_PUBLIC_API_URL` is not set
   - Good for UI work and controlled demos

2. `Real backend mode`
   - Implemented in `apps/api`
   - Intended to be used when the portal points to the Go API
   - Not every frontend contract was originally aligned with the Go API; this repo now includes compatibility work, but future engineers should still treat mock and real backend parity as an active concern

## Current Public Deployment

The current externally reachable deployment is:

- Portal: `https://portal.datadesng.com/login`
- API: `https://api.datadesng.com`
- GitHub repo: `https://github.com/JuanM2209/agente-rs-platform`
- Runtime host: Windows + Docker Desktop
- Cloudflare named tunnel: `agente-rs-public`
- Cloudflare tunnel ID: `8825530a-d505-4d9e-bd15-bbc1b85c1f15`

### Important note

- The previous `trycloudflare.com` preview URLs are no longer the source of truth
- Future engineers should treat `portal.datadesng.com` and `api.datadesng.com` as the active external test endpoints unless the runbook changes

## Temporary Product Assumptions

These assumptions are intentional and should be preserved until the product owner provides the real ORG/device mapping:

- Use the existing credentials as-is for now
- Multi-ORG provisioning is deferred
- Test devices remain the working dataset
- Future ORG/device entitlements will be added later without changing the current login flow yet

## Current Test Credentials

For the current seeded/test deployment:

- `admin@alpha.com`
- `operator@alpha.com`
- `viewer@alpha.com`
- password: `DevPass123!`

## Monorepo Layout

- `apps/web`
  - Next.js portal UI
  - Also contains mock API routes for preview mode
- `apps/api`
  - Go control-plane HTTP API
  - Handles auth, devices, sessions, bridges, audit history
- `apps/agent`
  - Go edge agent that runs on the Nucleus device
  - Manages inventory scanning, sessions, and MBUSD process lifecycle
- `tools/windows-helper`
  - Go CLI that maps exported sessions to `127.0.0.1:<port>` on the engineer laptop
- `infra/migrations`
  - PostgreSQL schema and helper functions
- `infra/seeds`
  - Development seed data

## Device Identity Resolution

The agent now resolves the device identifier using this order:

1. `DEVICE_ID` environment variable
2. `/data/nucleus/factory/nucleus_serial_number`

This matches the current operational assumption that every Nucleus device has a file named `nucleus_serial_number` under `/data/nucleus/factory/`.

### Notes

- The file is read at agent startup
- The path can be overridden with `NUCLEUS_SERIAL_NUMBER_FILE`
- The goal is to let the container auto-identify the Nucleus without forcing manual configuration per device

## MBUSD Serial Export Support

This repo now includes a first end-to-end path for Modbus RTU export over the Nucleus serial port.

### Current assumptions

- The default Modbus serial device is `/dev/ttymxc5`
- The published `agente-rs` image bundles the provided `mbusd` binary for ARMv7 Nucleus devices
- The agent inventory scanner now looks for `/dev/ttymxc*` in addition to the older serial device families
- The existing Nucleus Node-RED image was inspected as a compatibility reference and confirmed an older target environment:
  - `Architecture: armv7l`
  - `DockerVersion: 19.03.2`
  - Debian-based runtime

### Operational impact

- Starting MBUSD on `/dev/ttymxc5` temporarily interrupts Node-RED Modbus serial communication if Node-RED is using the same serial port
- The portal now warns the operator before activating the serial bridge and requires explicit acknowledgement
- Stopping the export session also stops the MBUSD bridge and disables the temporary bridge endpoint

### UI behavior

- Device detail pages now expose a `Start MBUSD + Export` action when the device has serial capability
- The bridge modal lets the operator choose serial parameters and a temporary TCP bridge port
- After bridge creation, the portal immediately creates the export session so the helper can import the channel on the laptop

### Legacy install path

Because some real Nucleus devices use older Docker engines that do not handle the current GHCR multi-arch flow reliably, the repo now includes a legacy-safe installer:

- `scripts/install-remote-s-nucleus.sh`

That installer:

- clones or downloads the repo source on the Nucleus
- tries to build the agent image locally without requiring `FROM --platform=...`
- falls back to a prebuilt ARMv7 image from GitHub Releases if the local build fails on the target Docker engine
- starts the container as `Remote-S` by default

## Export Session Telemetry

This iteration introduces first-pass telemetry for exported TCP sessions.

### Telemetry fields

- `connection_status`
  - `pending`
  - `reachable`
  - `degraded`
  - `unreachable`
  - `stopped`
- `latency_ms`
- `last_checked_at`
- `last_error`
- `probe_source`

### Source of truth

- The Windows helper performs a periodic TCP probe to the exported remote endpoint
- Probe results are sent back to the API using `POST /api/v1/sessions/{sessionId}/telemetry`
- The portal surfaces this data in the active sessions UI
- Mock routes in `apps/web` also support the same telemetry shape so preview mode remains functional

### Important limitation

The current telemetry is transport-level TCP latency, not protocol-level application latency. For example, Modbus or PLC response times can still differ from the measured `latency_ms`.

## Export Session Flow

### Current intended flow

1. User logs into the portal
2. User searches by `Device ID`
3. User starts an export session for a program port
4. Portal creates a session record
5. Windows helper maps the remote endpoint to a local laptop port
6. Helper periodically reports connection status and latency
7. When the helper closes the mapping, it also requests remote session stop

## Session UX And Operator Defaults

The portal now enforces and presents a longer support window by default.

### Current session policy

- New sessions default to `8 hours`
- The Go API now clamps requested session TTL to a minimum of `8 hours`
- Export sessions can specify a custom local laptop port through `local_port`

### Settings behavior

- The Settings page now includes an admin-only control for the default session duration
- Allowed admin presets currently range from `8` to `24` hours
- The setting is currently stored in browser local storage for the signed-in operator profile, not yet in central backend configuration

### Device detail UX

- Device endpoint actions are now presented as a list instead of separate cards
- Each row gives clearer choices:
  - `Open Web Port`
  - `Export to Your Laptop`
- Export workflows now allow a custom local target such as mapping remote `1880` to laptop `127.0.0.1:1889`
- Serial bridge flows now ask for both the temporary bridge TCP port on the Nucleus and the laptop-side localhost export port

### History behavior

- The Go API now writes a minimal `export_history` row when a session is explicitly stopped
- Session telemetry is copied into history metadata when available

## Documentation of the Current UI State

The current UI direction includes:

- dark operations-console style
- device-centric search
- session monitoring page
- audit/history page
- serial bridge modal

The UI preview may still be backed by mock data even when the backend implementation exists in Go.

## Known Gaps

These are still open after this iteration:

- Agent identity/authentication is still not production-grade
- Full multi-ORG entitlements are not implemented yet
- Real backend and mock backend still need more contract convergence in some areas
- Inventory still needs a more scalable cached/on-demand strategy
- The helper is still a CLI MVP, not a persistent Windows tray app
- GHCR publishing depends on the GitHub Actions deploy workflow remaining healthy
- The bundled `mbusd` binary is currently only provided for ARMv7; non-ARMv7 environments still need an override binary

## Next Engineer Checklist

- Verify the agent container can read `/data/nucleus/factory/nucleus_serial_number` in the real deployment mount setup
- Validate the helper telemetry endpoint against the real Go API, not only preview mocks
- Decide whether `remote_host` should remain the device IP fallback or come from richer endpoint inventory data
- Keep documenting any change that affects session shape, auth flow, or export telemetry
