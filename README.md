# Nucleus Remote Access Portal

> Production-grade industrial remote support platform for managing thousands of Linux edge devices (Nucleus) at scale.

---

## What Is This?

The Nucleus Remote Access Portal gives field engineers and support teams a **search-first, session-based** interface for securely accessing industrial edge devices remotely.

Each Nucleus device may expose:
- **Web UIs** (Node-RED on 1880, device UI on 9090, HTTP/HTTPS on 80/443)
- **Industrial program ports** (Modbus TCP on 502, EtherNet/IP on 44818, SSH on 22)
- **Serial bridge capability** (MBUSD Modbus serial-to-TCP bridges)

All access is **on-demand**, **audited**, and **tenant-scoped**.

---

## Architecture

```
Browser (User)
  └─ Next.js Portal (apps/web)
       Login → Search by Device ID → Device Detail
       Active Sessions → Export History → Admin Overview
            │ REST API + JWT
Go Control Plane (apps/api)
  REST API + WebSocket hub for agent connections
  PostgreSQL (data) + Redis (sessions/locks)
       │ WebSocket outbound from edge          │ TCP proxy
nucleus-port-agent (Go)                Windows Helper (Go CLI)
  Runs on Nucleus edge device           Local TCP port mapper
  - Inventory scanning                  127.0.0.1:<port> to remote
  - Session tunnel worker
  - MBUSD wrapper
```

---

## Monorepo Structure

```
nucleus-remote-access-portal/
├── apps/
│   ├── web/                    # Next.js 14 + TypeScript + Tailwind portal
│   ├── api/                    # Go REST API + WebSocket control plane
│   └── agent/                  # Go edge agent (nucleus-port-agent)
├── tools/
│   └── windows-helper/         # Go CLI for local TCP port mapping on Windows
├── infra/
│   ├── migrations/             # PostgreSQL schema migrations
│   ├── seeds/                  # Development seed data
│   ├── docker/                 # Docker init files
│   └── cloudflare/             # Cloudflare tunnel + Zero Trust config
├── .github/workflows/          # GitHub Actions CI/CD
├── docker-compose.yml          # Full local dev environment
├── Makefile                    # Task runner
├── .env.example                # Environment variable template
└── README.md
```

---

## Quick Start (Docker Desktop + WSL)

### Prerequisites

- Docker Desktop with WSL2 integration enabled
- Git
- Node.js 20+ (for frontend development only)
- Go 1.22+ (for backend development only)

### 1. Clone and configure

```bash
git clone https://github.com/your-org/nucleus-remote-access-portal.git
cd nucleus-remote-access-portal

cp .env.example .env
# Edit .env — change JWT_SECRET to a strong random value
```

### 2. Start all services

```bash
make up
```

| Service | URL |
|---------|-----|
| Web Portal | http://localhost:3000 |
| API | http://localhost:8080 |
| PostgreSQL | localhost:5432 |
| Redis | localhost:6379 |
| Agent N-1001 | connected to API |
| Agent N-1002 | connected to API |
| Agent N-1003 | offline (simulated) |
| Agent N-1004 | connected to API |

### 3. Run migrations and seeds

```bash
make migrate
make seed
```

### 4. Open the portal

Navigate to http://localhost:3000

**Demo credentials:**

| Email | Password | Role |
|-------|----------|------|
| admin@alpha.com | DevPass123! | Admin |
| operator@alpha.com | DevPass123! | Operator |
| viewer@alpha.com | DevPass123! | Viewer |

---

## Testing Core Flows

### Search for a device

1. Log in as `operator@alpha.com`
2. On the home page, type `N-1001` in the search box
3. Click **CONNECT** to see the device detail page

### View endpoints

On the device detail page you see endpoints grouped by:
- **Web** - HTTP/HTTPS/Node-RED/Device UI ports
- **Program** - Modbus TCP, SSH, EtherNet/IP
- **Bridge** - Serial bridge capability

### Create a session

1. On any web endpoint, click **Open Web** to create a web access session
2. Click **Export** to create a local port mapping session
3. Sessions appear immediately in **Active Sessions** (/sessions)

### View history

Go to **Audit History** (/history) to see all completed sessions.

### Simulate offline device

```bash
make simulate-offline-n1003   # Stop N-1003 agent
make simulate-online-n1003    # Bring it back
```

---

## Development (Without Docker)

### API (Go)

```bash
cd apps/api
go mod download
# Install air: go install github.com/air-verse/air@latest
air
```

### Frontend (Next.js)

```bash
cd apps/web
npm install
npm run dev
```

### Agent (Go)

```bash
cd apps/agent
DEVICE_ID=N-LOCAL-DEV \
CONTROL_PLANE_URL=ws://localhost:8080/ws/agent \
AGENT_SECRET=dev-agent-secret \
TENANT_ID=tenant-alpha \
go run ./cmd
```

---

## API Reference

All endpoints require `Authorization: Bearer <token>` except login.

```
POST   /api/v1/auth/login
POST   /api/v1/auth/logout
GET    /api/v1/me
GET    /api/v1/me/active-sessions
GET    /api/v1/me/export-history
GET    /api/v1/devices/:deviceId
GET    /api/v1/devices/:deviceId/inventory
POST   /api/v1/devices/:deviceId/scan
POST   /api/v1/devices/:deviceId/sessions
DELETE /api/v1/sessions/:sessionId
GET    /api/v1/devices/:deviceId/export-history
POST   /api/v1/devices/:deviceId/bridges/modbus-serial
DELETE /api/v1/bridges/:bridgeId
```

---

## Multi-Agent Simulation

| Device ID | Endpoints | Serial Port |
|-----------|-----------|-------------|
| N-1001 | :80 (WEB), :1880 (Node-RED), :502 (Modbus) | No |
| N-1002 | :443 (HTTPS), :9090 (DevUI), :502 (Modbus) | /dev/ttyUSB0 |
| N-1003 | :80 (HTTP), :22 (SSH), :502 (Modbus) | No (offline) |
| N-1004 | :1880 (Node-RED), :9090 (DevUI), :44818 (EIP) | /dev/ttyS0 |

---

## Windows Helper CLI

```bash
# Build
cd tools/windows-helper
make build-windows

# Login
nucleus-helper.exe login --api-url http://api.nucleus.company.com --email user@company.com

# List active sessions
nucleus-helper.exe sessions

# Map a session port locally
nucleus-helper.exe map --session-id <id> --local-port 5020

# Show status with TTL countdown
nucleus-helper.exe status

# Stop a mapping
nucleus-helper.exe unmap --session-id <id>
```

---

## Cloudflare Integration

See `infra/cloudflare/README.md` for full setup.

**Summary:**
1. Create tunnel: `cloudflared tunnel create nucleus-portal`
2. Route DNS: `portal.yourdomain.com -> localhost:3000`, `api.yourdomain.com -> localhost:8080`
3. Agents connect to: `wss://agents.yourdomain.com/ws/agent`
4. Optionally add Zero Trust Access policy for SSO gating

---

## GitHub Repository Bootstrap

```bash
git init
git add .
git commit -m "feat: initial scaffold for Nucleus Remote Access Portal"

gh repo create your-org/nucleus-remote-access-portal \
  --private \
  --description "Industrial remote support platform for Nucleus edge devices"

git remote add origin https://github.com/your-org/nucleus-remote-access-portal.git
git branch -M main
git push -u origin main
```

---

## Branch Strategy

```
main       Production-ready, protected
develop    Integration branch, auto-deploys to staging
feature/*  Feature branches, PR to develop
fix/*      Bug fixes, PR to develop
hotfix/*   Critical fixes, PR to main + develop
```

---

## Phase 2 Roadmap

- SSH tunnel sessions (port 22)
- EtherNet/IP export (port 44818)
- Windows systray GUI for the helper app
- Signed/temporary web session URLs via Cloudflare Workers
- Real-time session monitoring via WebSocket push to browser
- Fleet-wide firmware version reporting
- Bulk device operations
