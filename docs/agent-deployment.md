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
- The current Nucleus Modbus serial path is `/dev/ttymxc5`
- The published `agente-rs` image bundles `mbusd` for ARMv7 Nucleus devices
- If a user activates MBUSD on `/dev/ttymxc5`, Node-RED Modbus serial communication on that same port is interrupted until the bridge stops
- On non-ARMv7 hosts, provide your own `mbusd` binary through `MBUSD_HOST_PATH` or a custom image

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

The installer defaults `SERIAL_DEVICE` to `/dev/ttymxc5`, which matches the current Modbus serial configuration on the Nucleus devices.

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

### If the device has the Nucleus Modbus serial adapter

Example with `/dev/ttymxc5`:

```bash
docker run -d \
  --name agente-rs \
  --restart unless-stopped \
  --network host \
  --device /dev/ttymxc5:/dev/ttymxc5 \
  -v /data/nucleus/factory:/data/nucleus/factory:ro \
  -e CONTROL_PLANE_URL='wss://api.datadesng.com/ws/agent' \
  -e AGENT_SECRET='YOUR_AGENT_SECRET' \
  -e TENANT_ID='test-org' \
  -e LOG_LEVEL='info' \
  ghcr.io/juanm2209/agente-rs:latest
```

### If the device is not ARMv7

Provide the working host binary explicitly:

```bash
docker run -d \
  --name agente-rs \
  --restart unless-stopped \
  --network host \
  --device /dev/ttymxc5:/dev/ttymxc5 \
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

### Legacy Nucleus / Docker 19.x compatibility

Some real Nucleus devices run an older Docker engine similar to the environment used by the existing Node-RED image:

- `Architecture: armv7l`
- `DockerVersion: 19.03.2`
- Debian-based runtime

For those devices, use the dedicated legacy installer for `Remote-S`. It first tries a local classic-Docker build. If that still fails on the target Nucleus, it automatically falls back to a prebuilt ARMv7 image hosted in GitHub Releases.

That installer also disables Docker Content Trust during the local build path, because some older Nucleus Docker engines fail with `missing signature key` while resolving modern base images from Docker Hub.

```bash
curl -fsSL https://raw.githubusercontent.com/JuanM2209/agente-rs-platform/main/scripts/install-remote-s-nucleus.sh -o install-remote-s-nucleus.sh
chmod +x install-remote-s-nucleus.sh
sudo env CONTROL_PLANE_URL='wss://api.datadesng.com/ws/agent' AGENT_SECRET='YOUR_AGENT_SECRET' TENANT_ID='test-org' CONTAINER_NAME='Remote-S' SERIAL_DEVICE='/dev/ttymxc5' ./install-remote-s-nucleus.sh
```

Default local image name for that path is `remote-s-local`.

### If the device has the Nucleus Modbus serial adapter

Example with `/dev/ttymxc5`:

```bash
docker run -d \
  --name agente-rs \
  --restart unless-stopped \
  --network host \
  --device /dev/ttymxc5:/dev/ttymxc5 \
  -v /data/nucleus/factory:/data/nucleus/factory:ro \
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
ls -l /dev/ttymxc5 /dev/ttyUSB* /dev/ttyACM* /dev/ttyS*
```

Then make sure the matching `--device` flag is present in `docker run`.
For ARMv7 Nucleus devices, `mbusd` is already bundled in the published image.
For other architectures, also mount or bake an `mbusd` binary into the image.

### Node-RED loses serial Modbus while MBUSD is active

This is expected in the current design.

- MBUSD and Node-RED cannot safely use `/dev/ttymxc5` at the same time
- The portal warns the operator before enabling the serial bridge
- Stop the export session or stop the bridge from the portal to return serial ownership to Node-RED

### Agent connects but inventory looks empty

The current scanner reads local listening ports from `/proc/net/tcp` and serial ports from `/dev`.
For host-level discovery on Linux, run the container with `--network host`.
