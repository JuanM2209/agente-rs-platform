# Cloudflare Integration — Nucleus Remote Access Portal

## Overview

This directory contains Cloudflare configuration templates for the Nucleus portal. The platform uses Cloudflare for:

1. **Cloudflare Tunnel** — Expose the portal and API publicly without opening inbound ports on the server
2. **Cloudflare Zero Trust** — (Optional) Add identity-aware access policy in front of the portal
3. **Cloudflare Workers** — (Phase 2) Signed/temporary URL generation for web sessions

---

## Local Development

For local dev, Cloudflare is **not required**. The services run directly on localhost.

```bash
# Start all services locally (no Cloudflare)
make up
```

---

## Production Setup

### Step 1: Install cloudflared

```bash
# Windows
winget install Cloudflare.cloudflared

# Linux/WSL
curl -L --output cloudflared.deb https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
sudo dpkg -i cloudflared.deb

# macOS
brew install cloudflare/cloudflare/cloudflared
```

### Step 2: Authenticate and create tunnel

```bash
cloudflared login
cloudflared tunnel create nucleus-portal

# Note the tunnel ID and credentials path
```

### Step 3: Configure DNS routes

```bash
cloudflared tunnel route dns nucleus-portal portal.yourdomain.com
cloudflared tunnel route dns nucleus-portal api.yourdomain.com
cloudflared tunnel route dns nucleus-portal agents.yourdomain.com
```

### Step 4: Update tunnel config

Edit `infra/cloudflare/tunnel-config.yml`:
- Replace `<TUNNEL_ID>` with your actual tunnel ID
- Replace `yourdomain.com` with your actual domain
- Update credentials-file path

### Step 5: Run the tunnel

```bash
# Manual run
cloudflared tunnel --config infra/cloudflare/tunnel-config.yml run

# Or as a service
sudo cloudflared service install
```

---

## Cloudflare Zero Trust Access (Optional)

To add identity-gating in front of the portal (SSO, email OTP, etc.):

1. Go to Cloudflare Zero Trust dashboard → Access → Applications
2. Add application for `portal.yourdomain.com`
3. Set policy: Allow `your-org-email@domain.com` or email list
4. Configure CF_ACCESS_TEAM_DOMAIN and CF_ACCESS_AUD in .env

The API has middleware to validate CF_Authorization JWT headers when `CF_ACCESS_AUD` is set.

---

## Environment Variables (Production)

```bash
CLOUDFLARE_ACCOUNT_ID=your-account-id
CLOUDFLARE_API_TOKEN=your-api-token
CLOUDFLARE_ZONE_ID=your-zone-id
CLOUDFLARE_TUNNEL_TOKEN=your-tunnel-token
CLOUDFLARE_TUNNEL_ID=your-tunnel-id
CF_ACCESS_TEAM_DOMAIN=yourteam.cloudflareaccess.com  # optional
CF_ACCESS_AUD=your-access-aud                         # optional
```

---

## Signed Web Session URLs (Phase 2)

For secure web session URLs, the planned implementation uses Cloudflare Workers to:
1. Generate signed URLs with HMAC signatures and expiry timestamps
2. Validate signatures at the edge before routing to backend
3. Short-circuit expired or tampered URLs before they reach the origin

Worker template: `infra/cloudflare/workers/session-url-signer.js` (Phase 2)

---

## Agent WebSocket Connections

Nucleus agents connect outbound via WebSocket to the control plane. Through Cloudflare Tunnel:

- Agents on-premise connect to `wss://agents.yourdomain.com/ws/agent`
- Cloudflare tunnels this to `http://localhost:8080`
- The tunnel handles TLS termination

No inbound ports need to be opened on the server side.

**Important**: The Cloudflare Tunnel must be configured with WebSocket support enabled. This is the default behavior for HTTP ingress rules.
