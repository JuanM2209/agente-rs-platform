# Public Deploy Runbook

Last updated: 2026-03-20

## Purpose

This runbook documents the current public deployment plan for the project on:

- GitHub
- GitHub Container Registry
- Cloudflare Tunnel
- Windows host runtime

## Current Public Targets

- GitHub repository: `JuanM2209/agente-rs-platform`
- Agent image name: `ghcr.io/juanm2209/agente-rs:latest`
- API image name: `ghcr.io/juanm2209/nucleus-api:latest`
- Portal hostname: `portal.datadesng.com`
- API hostname: `api.datadesng.com`
- Cloudflare tunnel: `agente-rs-public`
- Cloudflare tunnel ID: `8825530a-d505-4d9e-bd15-bbc1b85c1f15`
- Host runtime: Windows + Docker Desktop + cloudflared

## Deployment Model

### Local Windows host

- `web` container on `localhost:3000`
- `api` container on `localhost:8080`
- `postgres` on `localhost:5432`
- `redis` on `localhost:6379`

### Cloudflare ingress

- `portal.datadesng.com` -> `http://localhost:3000`
- `api.datadesng.com` -> `http://localhost:8080`

### External Nucleus agent

- Each Linux device runs the `agente-rs` container
- The container reads `Device ID` from `/data/nucleus/factory/nucleus_serial_number`
- The current Modbus serial device is `/dev/ttymxc5`
- The published ARMv7 image includes the provided `mbusd` binary at `/usr/local/bin/mbusd`
- The container connects outbound to:
  - `wss://api.datadesng.com/ws/agent`

## Windows Host Startup

### 1. Prepare `.env`

Create a local `.env` in the repo root with real values for:

- `JWT_SECRET`
- `AGENT_WS_SECRET`
- `NEXT_PUBLIC_API_URL=https://api.datadesng.com`
- `NEXT_PUBLIC_APP_URL=https://portal.datadesng.com`
- `API_CORS_ORIGINS=https://portal.datadesng.com,https://api.datadesng.com,http://localhost:3000`

### 2. Build and start the stack

```powershell
cd C:\Users\JML\nucleus-remote-access-portal
docker compose up -d --build
```

### 3. Run migrations and seeds

```powershell
docker compose exec postgres psql -U nucleus -d nucleus_portal -f /migrations/001_initial_schema.sql
docker compose exec postgres psql -U nucleus -d nucleus_portal -f /migrations/002_session_functions.sql
docker compose exec -e DATABASE_URL=postgresql://nucleus:nucleus_dev@localhost:5432/nucleus_portal postgres bash /seeds/run_seeds.sh
```

### 4. Run Cloudflare named tunnel

First create DNS routes for the dedicated project tunnel:

```powershell
cloudflared tunnel route dns agente-rs-public portal.datadesng.com
cloudflared tunnel route dns agente-rs-public api.datadesng.com
```

Then run the named tunnel with the repo config:

```powershell
cloudflared tunnel --config infra\cloudflare\tunnel-config.yml run agente-rs-public
```

### 5. Health checks

```powershell
Invoke-WebRequest -UseBasicParsing https://portal.datadesng.com/login | Select-Object -ExpandProperty StatusCode
Invoke-WebRequest -UseBasicParsing https://api.datadesng.com/health | Select-Object -ExpandProperty Content
```

## GitHub Publishing

### Repo visibility

- The repository is intended to be public for testing and external installation

### Package publishing

- The `deploy.yml` workflow publishes:
  - `ghcr.io/<owner>/nucleus-api`
  - `ghcr.io/<owner>/agente-rs`
- The workflow normalizes the owner name to lowercase for valid GHCR tags
- The workflow requires `packages: write` permission

### Notes

- External Linux devices should use `docker pull`, not `git clone`, for runtime installation
- The repo remains useful for docs, issues, and version tracking

## External Device Install

### Pull the latest agent image

```bash
docker pull ghcr.io/juanm2209/agente-rs:latest
```

### Or use the installer script from the public repo

```bash
curl -fsSL https://raw.githubusercontent.com/JuanM2209/agente-rs-platform/main/scripts/install-agente-rs.sh -o install-agente-rs.sh
chmod +x install-agente-rs.sh
CONTROL_PLANE_URL='wss://api.datadesng.com/ws/agent' \
AGENT_SECRET='replace-with-real-agent-secret' \
TENANT_ID='test-org' \
./install-agente-rs.sh
```

The installer defaults `SERIAL_DEVICE` to `/dev/ttymxc5`.

### Legacy ARMv7 Nucleus install

If the target device looks like the existing Node-RED appliance environment (`armv7l`, Docker 19.x), use the local-build installer instead of pulling GHCR directly:

```bash
curl -fsSL https://raw.githubusercontent.com/JuanM2209/agente-rs-platform/main/scripts/install-remote-s-nucleus.sh -o install-remote-s-nucleus.sh
chmod +x install-remote-s-nucleus.sh
sudo env CONTROL_PLANE_URL='wss://api.datadesng.com/ws/agent' AGENT_SECRET='replace-with-real-agent-secret' TENANT_ID='test-org' CONTAINER_NAME='Remote-S' SERIAL_DEVICE='/dev/ttymxc5' ./install-remote-s-nucleus.sh
```

### Run the agent

```bash
docker run -d \
  --name agente-rs \
  --restart unless-stopped \
  --network host \
  -v /data/nucleus/factory:/data/nucleus/factory:ro \
  -e CONTROL_PLANE_URL='wss://api.datadesng.com/ws/agent' \
  -e AGENT_SECRET='replace-with-real-agent-secret' \
  -e TENANT_ID='test-org' \
  -e LOG_LEVEL='info' \
  ghcr.io/juanm2209/agente-rs:latest
```

### With serial device

```bash
docker run -d \
  --name agente-rs \
  --restart unless-stopped \
  --network host \
  --device /dev/ttymxc5:/dev/ttymxc5 \
  -v /data/nucleus/factory:/data/nucleus/factory:ro \
  -e CONTROL_PLANE_URL='wss://api.datadesng.com/ws/agent' \
  -e AGENT_SECRET='replace-with-real-agent-secret' \
  -e TENANT_ID='test-org' \
  -e LOG_LEVEL='info' \
  ghcr.io/juanm2209/agente-rs:latest
```

### If the external device is not ARMv7

Mount a compatible binary explicitly:

```bash
docker run -d \
  --name agente-rs \
  --restart unless-stopped \
  --network host \
  --device /dev/ttymxc5:/dev/ttymxc5 \
  -v /data/nucleus/factory:/data/nucleus/factory:ro \
  -v /usr/local/bin/mbusd:/usr/local/bin/mbusd:ro \
  -e CONTROL_PLANE_URL='wss://api.datadesng.com/ws/agent' \
  -e AGENT_SECRET='replace-with-real-agent-secret' \
  -e TENANT_ID='test-org' \
  -e LOG_LEVEL='info' \
  ghcr.io/juanm2209/agente-rs:latest
```

## Operational Notes

- This first public phase is still using test-org / seeded data
- Cloudflare quick tunnels are only for visual preview, not the permanent deployment path
- The dedicated tunnel `agente-rs-public` exists specifically to avoid interfering with unrelated routes on the older `api-dbv` tunnel
- The external install flow should standardize on `agente-rs`
- Activating MBUSD on `/dev/ttymxc5` temporarily interrupts Node-RED Modbus serial communication on that same port
- The portal now shows that warning before allowing the serial export flow
