# Agent Deployment Guide

Last updated: 2026-03-20

## Purpose

This guide explains how to:

- expose the portal through a temporary public Cloudflare URL for visual testing
- install the `Agente-RS` container on an external Linux device
- pull the public agent image directly from GitHub Container Registry

## Current Public Targets

- Portal: `https://portal.datadesng.com/login`
- API / control plane: `https://api.datadesng.com`
- Agent WebSocket: `wss://api.datadesng.com/ws/agent`
- Public repo: `https://github.com/JuanM2209/agente-rs-platform`
- Public image: `ghcr.io/juanm2209/agente-rs:latest`

## Public Preview For Visual Testing

If you only need the portal UI for preview or visual QA, you can run the Next.js app directly and expose it with a temporary Cloudflare quick tunnel.

### Start the portal

From `apps/web`:

```powershell
npm install
npm run build
npm run start -- --hostname 0.0.0.0 --port 3000
```

### Expose the portal publicly

In another terminal:

```powershell
cloudflared tunnel --url http://127.0.0.1:3000 --no-autoupdate
```

Cloudflare will print a temporary `https://<random>.trycloudflare.com` URL.

### Notes

- This is a temporary preview URL, not a production deployment
- The URL stays alive only while the local `cloudflared` process is running
- In the current mock mode, demo login credentials are still:
  - `admin@alpha.com`
  - `operator@alpha.com`
  - `viewer@alpha.com`
  - password: `DevPass123!`

## External Nucleus Device Install

The agent container now reads the device identity from:

`/data/nucleus/factory/nucleus_serial_number`

That file must be mounted into the container as read-only.

### Required environment values

- `CONTROL_PLANE_URL`
- `AGENT_SECRET`
- `TENANT_ID`
- optional: `LOG_LEVEL`
- optional: `INVENTORY_SCAN_INTERVAL`
- optional: `HEARTBEAT_INTERVAL`
- optional: `MAX_CONCURRENT_SESSIONS`

### Important runtime notes

- Use `--network host` on Linux if you want inventory discovery to reflect host listening ports
- Mount `/data/nucleus/factory` read-only so the agent can read `nucleus_serial_number`
- Add `--device` for each serial adapter you want MBUSD to use
- If serial bridge support is needed, make sure `mbusd` is available inside the container PATH

## Option A: Pull The Public Image Directly (Recommended)

### One-command installer from the public repo

```bash
curl -fsSL https://raw.githubusercontent.com/JuanM2209/agente-rs-platform/main/scripts/install-agente-rs.sh -o install-agente-rs.sh
chmod +x install-agente-rs.sh
CONTROL_PLANE_URL='wss://api.datadesng.com/ws/agent' \
AGENT_SECRET='YOUR_AGENT_SECRET' \
TENANT_ID='test-org' \
./install-agente-rs.sh
```

### Direct docker commands

```bash
docker pull ghcr.io/juanm2209/agente-rs:latest
```

Then start the agent:

```bash
docker run -d \
  --name agente-rs \
  --restart unless-stopped \
  --network host \
  -v /data/nucleus/factory:/data/nucleus/factory:ro \
  -e CONTROL_PLANE_URL='wss://api.datadesng.com/ws/agent' \
  -e AGENT_SECRET='YOUR_AGENT_SECRET' \
  -e TENANT_ID='test-org' \
  -e LOG_LEVEL='info' \
  ghcr.io/juanm2209/agente-rs:latest
```

### If the device has a serial adapter

Example with `/dev/ttyUSB0`:

```bash
docker run -d \
  --name agente-rs \
  --restart unless-stopped \
  --network host \
  --device /dev/ttyUSB0:/dev/ttyUSB0 \
  -v /data/nucleus/factory:/data/nucleus/factory:ro \
  -v /usr/local/bin/mbusd:/usr/local/bin/mbusd:ro \
  -e CONTROL_PLANE_URL='wss://api.datadesng.com/ws/agent' \
  -e AGENT_SECRET='YOUR_AGENT_SECRET' \
  -e TENANT_ID='test-org' \
  -e LOG_LEVEL='info' \
  ghcr.io/juanm2209/agente-rs:latest
```

## Option B: Build On The External Device

Copy the repository to the Nucleus device and run:

```bash
cd /opt/nucleus-remote-access-portal/apps/agent
docker build -t agente-rs:latest .
```

Then start the agent:

```bash
docker run -d \
  --name agente-rs \
  --restart unless-stopped \
  --network host \
  -v /data/nucleus/factory:/data/nucleus/factory:ro \
  -e CONTROL_PLANE_URL='wss://YOUR-API-DOMAIN/ws/agent' \
  -e AGENT_SECRET='YOUR_AGENT_SECRET' \
  -e TENANT_ID='test-org' \
  -e LOG_LEVEL='info' \
  agente-rs:latest
```

### If the device has a serial adapter

Example with `/dev/ttyUSB0`:

```bash
docker run -d \
  --name agente-rs \
  --restart unless-stopped \
  --network host \
  --device /dev/ttyUSB0:/dev/ttyUSB0 \
  -v /data/nucleus/factory:/data/nucleus/factory:ro \
  -v /usr/local/bin/mbusd:/usr/local/bin/mbusd:ro \
  -e CONTROL_PLANE_URL='wss://YOUR-API-DOMAIN/ws/agent' \
  -e AGENT_SECRET='YOUR_AGENT_SECRET' \
  -e TENANT_ID='test-org' \
  -e LOG_LEVEL='info' \
  agente-rs:latest
```

## Option C: Build Once, Then Load On The External Device

If you do not want to compile on each Nucleus:

### Build on a workstation

```bash
docker buildx build \
  --platform linux/amd64,linux/arm/v7 \
  -t agente-rs:portable \
  --load \
  /path/to/nucleus-remote-access-portal/apps/agent
```

### Export the image

```bash
docker save agente-rs:portable -o agente-rs-portable.tar
```

### Copy to the Nucleus and load it

```bash
docker load -i agente-rs-portable.tar
```

Then run the same `docker run` command shown above.

## Verify The Agent

### Check container status

```bash
docker ps
docker logs -f agente-rs
```

### Expected startup behavior

The logs should show:

- the resolved `device_id`
- `device_id_source=file` if it came from `nucleus_serial_number`
- successful connection to the control plane
- inventory scan startup

## Update The Agent

```bash
docker stop agente-rs
docker rm agente-rs
docker image rm agente-rs:latest || true
docker image rm ghcr.io/juanm2209/agente-rs:latest || true
```

Then pull or rebuild the new image and run the same `docker run` command again.

## Troubleshooting

### Device ID not found

Check that this file exists on the host:

```bash
cat /data/nucleus/factory/nucleus_serial_number
```

### Serial bridge not available

Check that the serial device exists on the host:

```bash
ls -l /dev/ttyUSB* /dev/ttyACM* /dev/ttyS*
```

Then make sure the matching `--device` flag is present in `docker run`.
If you are using serial bridges, also mount or bake an `mbusd` binary into the image.

### Agent connects but inventory looks empty

The current scanner reads local listening ports from `/proc/net/tcp` and serial ports from `/dev`.
For host-level discovery on Linux, run the container with `--network host`.
