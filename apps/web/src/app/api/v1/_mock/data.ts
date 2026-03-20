// ============================================================
// Mock seed data — mirrors infra/seeds/ for standalone dev
// ============================================================
import type {
  User,
  Device,
  Endpoint,
  Session,
  ExportHistory,
  SessionTelemetry,
} from "@/types";

export const MOCK_PASSWORD = "DevPass123!";

export const MOCK_USERS: (User & { password: string })[] = [
  {
    id: "u-0001",
    tenant_id: "t-alpha",
    email: "admin@alpha.com",
    display_name: "Admin Console",
    role: "admin",
    is_active: true,
    last_login_at: new Date(Date.now() - 3600_000).toISOString(),
    password: MOCK_PASSWORD,
  },
  {
    id: "u-0002",
    tenant_id: "t-alpha",
    email: "operator@alpha.com",
    display_name: "Field Operator",
    role: "operator",
    is_active: true,
    last_login_at: new Date(Date.now() - 7200_000).toISOString(),
    password: MOCK_PASSWORD,
  },
  {
    id: "u-0003",
    tenant_id: "t-alpha",
    email: "viewer@alpha.com",
    display_name: "Read Only Viewer",
    role: "viewer",
    is_active: true,
    password: MOCK_PASSWORD,
  },
];

export const MOCK_DEVICES: Device[] = [
  {
    id: "d-1001",
    tenant_id: "t-alpha",
    device_id: "N-1001",
    display_name: "Pump Station Alpha",
    status: "online",
    last_seen: new Date(Date.now() - 12_000).toISOString(),
    firmware_version: "2.4.1",
    ip_address: "10.0.1.101",
    hardware_model: "Nucleus-X2",
    tags: { site: "Houston Plant", zone: "A" },
    inventory_updated_at: new Date(Date.now() - 60_000).toISOString(),
    site: { id: "s-001", tenant_id: "t-alpha", name: "Houston Plant", location: "Houston, TX", timezone: "America/Chicago" },
  },
  {
    id: "d-1002",
    tenant_id: "t-alpha",
    device_id: "N-1002",
    display_name: "Compressor Unit Beta",
    status: "online",
    last_seen: new Date(Date.now() - 8_000).toISOString(),
    firmware_version: "2.3.9",
    ip_address: "10.0.1.102",
    hardware_model: "Nucleus-X2",
    tags: { site: "Permian Basin", zone: "B" },
    inventory_updated_at: new Date(Date.now() - 120_000).toISOString(),
    site: { id: "s-002", tenant_id: "t-alpha", name: "Permian Basin", location: "Midland, TX", timezone: "America/Chicago" },
  },
  {
    id: "d-1003",
    tenant_id: "t-alpha",
    device_id: "N-1003",
    display_name: "Separator Unit Gamma",
    status: "offline",
    last_seen: new Date(Date.now() - 3_600_000).toISOString(),
    firmware_version: "2.3.7",
    ip_address: "10.0.1.103",
    hardware_model: "Nucleus-X1",
    tags: { site: "Midland Yard", zone: "C" },
    site: { id: "s-003", tenant_id: "t-alpha", name: "Midland Yard", location: "Midland, TX", timezone: "America/Chicago" },
  },
  {
    id: "d-1004",
    tenant_id: "t-alpha",
    device_id: "N-1004",
    display_name: "Control Node Delta",
    status: "online",
    last_seen: new Date(Date.now() - 5_000).toISOString(),
    firmware_version: "2.4.1",
    ip_address: "10.0.1.104",
    hardware_model: "Nucleus-X2",
    tags: { site: "Houston Plant", zone: "D" },
    inventory_updated_at: new Date(Date.now() - 90_000).toISOString(),
    site: { id: "s-001", tenant_id: "t-alpha", name: "Houston Plant", location: "Houston, TX", timezone: "America/Chicago" },
  },
];

export const MOCK_ENDPOINTS: Record<string, Endpoint[]> = {
  "N-1001": [
    { id: "e-1001-80",   device_id: "d-1001", type: "WEB",     port: 80,   label: "HTTP Web UI",  protocol: "http",   description: "Main device web interface", enabled: true },
    { id: "e-1001-1880", device_id: "d-1001", type: "WEB",     port: 1880, label: "Node-RED",     protocol: "http",   description: "Node-RED flow editor",       enabled: true },
    { id: "e-1001-502",  device_id: "d-1001", type: "PROGRAM", port: 502,  label: "Modbus TCP",   protocol: "modbus", description: "Modbus TCP register access", enabled: true },
  ],
  "N-1002": [
    { id: "e-1002-443",  device_id: "d-1002", type: "WEB",     port: 443,  label: "HTTPS Web UI", protocol: "https",  description: "Secure device web interface", enabled: true },
    { id: "e-1002-9090", device_id: "d-1002", type: "WEB",     port: 9090, label: "Device UI",    protocol: "http",   description: "Manufacturer device UI",     enabled: true },
    { id: "e-1002-502",  device_id: "d-1002", type: "PROGRAM", port: 502,  label: "Modbus TCP",   protocol: "modbus", description: "Modbus TCP register access", enabled: true },
    { id: "e-1002-br",   device_id: "d-1002", type: "BRIDGE",  port: 0,    label: "Serial Bridge", protocol: "mbusd", description: "/dev/ttyUSB0 serial bridge", enabled: true },
  ],
  "N-1003": [
    { id: "e-1003-80",   device_id: "d-1003", type: "WEB",     port: 80,   label: "HTTP Web UI",  protocol: "http",   enabled: true },
    { id: "e-1003-22",   device_id: "d-1003", type: "PROGRAM", port: 22,   label: "SSH",          protocol: "ssh",    description: "Secure shell access",       enabled: true },
    { id: "e-1003-502",  device_id: "d-1003", type: "PROGRAM", port: 502,  label: "Modbus TCP",   protocol: "modbus", enabled: true },
  ],
  "N-1004": [
    { id: "e-1004-1880",  device_id: "d-1004", type: "WEB",     port: 1880,  label: "Node-RED",      protocol: "http",       description: "Node-RED flow editor",     enabled: true },
    { id: "e-1004-9090",  device_id: "d-1004", type: "WEB",     port: 9090,  label: "Device UI",     protocol: "http",       description: "Manufacturer device UI",   enabled: true },
    { id: "e-1004-44818", device_id: "d-1004", type: "PROGRAM", port: 44818, label: "EtherNet/IP",   protocol: "ethernet-ip", description: "EtherNet/IP CIP adapter", enabled: true },
    { id: "e-1004-br",    device_id: "d-1004", type: "BRIDGE",  port: 0,     label: "Serial Bridge",  protocol: "mbusd",      description: "/dev/ttyS0 serial bridge", enabled: true },
  ],
};

const WEB_TELEMETRY: SessionTelemetry = {
  connection_status: "reachable",
  latency_ms: 36,
  last_checked_at: new Date(Date.now() - 15_000).toISOString(),
  probe_source: "system",
};

const EXPORT_TELEMETRY: SessionTelemetry = {
  connection_status: "reachable",
  latency_ms: 22,
  last_checked_at: new Date(Date.now() - 10_000).toISOString(),
  probe_source: "helper",
};

export const MOCK_SESSIONS: Session[] = [
  {
    id: "sess-0001",
    device_id: "d-1001",
    endpoint_id: "e-1001-1880",
    user_id: "u-0002",
    tenant_id: "t-alpha",
    status: "active",
    delivery_mode: "web",
    ttl_seconds: 3600,
    started_at: new Date(Date.now() - 900_000).toISOString(),
    expires_at: new Date(Date.now() + 2700_000).toISOString(),
    tunnel_url: "https://session-abc123.nucleus.example.com",
    remote_host: "10.0.1.101",
    telemetry: WEB_TELEMETRY,
    device: MOCK_DEVICES[0],
    endpoint: MOCK_ENDPOINTS["N-1001"][1],
  },
  {
    id: "sess-0002",
    device_id: "d-1002",
    endpoint_id: "e-1002-502",
    user_id: "u-0002",
    tenant_id: "t-alpha",
    status: "active",
    delivery_mode: "export",
    ttl_seconds: 1800,
    local_port: 5020,
    remote_host: "10.0.1.102",
    remote_port: 502,
    started_at: new Date(Date.now() - 300_000).toISOString(),
    expires_at: new Date(Date.now() + 1500_000).toISOString(),
    telemetry: EXPORT_TELEMETRY,
    device: MOCK_DEVICES[1],
    endpoint: MOCK_ENDPOINTS["N-1002"][2],
  },
];

export const MOCK_HISTORY: ExportHistory[] = [
  {
    id: "h-0001", session_id: "sess-old-1", user_id: "u-0002", device_id: "d-1001", tenant_id: "t-alpha",
    started_at: new Date(Date.now() - 86400_000).toISOString(),
    stopped_at: new Date(Date.now() - 82800_000).toISOString(),
    stop_reason: "user_stopped", delivery_mode: "web", duration_seconds: 3600, bytes_transferred: 1245000,
    metadata: {
      telemetry: {
        connection_status: "reachable",
        latency_ms: 34,
        last_checked_at: new Date(Date.now() - 82830_000).toISOString(),
        probe_source: "system",
      },
    },
    telemetry: {
      connection_status: "reachable",
      latency_ms: 34,
      last_checked_at: new Date(Date.now() - 82830_000).toISOString(),
      probe_source: "system",
    },
    device: MOCK_DEVICES[0], endpoint: MOCK_ENDPOINTS["N-1001"][0],
  },
  {
    id: "h-0002", session_id: "sess-old-2", user_id: "u-0001", device_id: "d-1002", tenant_id: "t-alpha",
    started_at: new Date(Date.now() - 172800_000).toISOString(),
    stopped_at: new Date(Date.now() - 169200_000).toISOString(),
    stop_reason: "ttl_expired", delivery_mode: "export", local_bind_port: 5020, duration_seconds: 3600, bytes_transferred: 840000,
    metadata: {
      telemetry: {
        connection_status: "degraded",
        latency_ms: 248,
        last_checked_at: new Date(Date.now() - 169260_000).toISOString(),
        probe_source: "helper",
      },
    },
    telemetry: {
      connection_status: "degraded",
      latency_ms: 248,
      last_checked_at: new Date(Date.now() - 169260_000).toISOString(),
      probe_source: "helper",
    },
    device: MOCK_DEVICES[1], endpoint: MOCK_ENDPOINTS["N-1002"][2],
  },
  {
    id: "h-0003", session_id: "sess-old-3", user_id: "u-0002", device_id: "d-1004", tenant_id: "t-alpha",
    started_at: new Date(Date.now() - 259200_000).toISOString(),
    stopped_at: new Date(Date.now() - 255600_000).toISOString(),
    stop_reason: "user_stopped", delivery_mode: "web", duration_seconds: 3600, bytes_transferred: 560000,
    metadata: {
      telemetry: {
        connection_status: "reachable",
        latency_ms: 42,
        last_checked_at: new Date(Date.now() - 255660_000).toISOString(),
        probe_source: "system",
      },
    },
    telemetry: {
      connection_status: "reachable",
      latency_ms: 42,
      last_checked_at: new Date(Date.now() - 255660_000).toISOString(),
      probe_source: "system",
    },
    device: MOCK_DEVICES[3], endpoint: MOCK_ENDPOINTS["N-1004"][0],
  },
];

// Decode a mock token back to a user
export function decodeToken(token: string): (User & { password: string }) | null {
  if (!token.startsWith("mock_")) return null;
  try {
    const payload = JSON.parse(Buffer.from(token.slice(5), "base64").toString());
    return MOCK_USERS.find((u) => u.id === payload.id) ?? null;
  } catch {
    return null;
  }
}

// Create a mock JWT for a user
export function encodeToken(user: User): string {
  const payload = Buffer.from(JSON.stringify({ id: user.id, email: user.email })).toString("base64");
  return `mock_${payload}`;
}

export function ok<T>(data: T) {
  return Response.json({ success: true, data });
}

export function err(message: string, status = 400) {
  return Response.json({ success: false, error: message }, { status });
}
