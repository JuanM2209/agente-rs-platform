// ============================================================
// Nucleus Portal — Shared TypeScript Types
// ============================================================

export type UserRole = "admin" | "operator" | "viewer" | "support";
export type DeviceStatus = "online" | "offline" | "unknown" | "maintenance";
export type EndpointType = "WEB" | "PROGRAM" | "BRIDGE";
export type SessionStatus = "active" | "expired" | "stopped" | "error";
export type DeliveryMode = "web" | "export";
export type StopReason = "user_stopped" | "ttl_expired" | "idle_timeout" | "error" | "agent_disconnect";
export type SessionConnectionStatus = "pending" | "reachable" | "degraded" | "unreachable" | "stopped";

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  created_at: string;
}

export interface Site {
  id: string;
  tenant_id: string;
  name: string;
  location?: string;
  timezone: string;
}

export interface User {
  id: string;
  tenant_id: string;
  email: string;
  display_name: string;
  role: UserRole;
  is_active: boolean;
  last_login_at?: string;
}

export interface Device {
  id: string;
  tenant_id: string;
  site_id?: string;
  device_id: string;
  display_name?: string;
  status: DeviceStatus;
  last_seen?: string;
  firmware_version?: string;
  ip_address?: string;
  hardware_model?: string;
  tags?: Record<string, string>;
  inventory_updated_at?: string;
  site?: Site;
}

export interface Endpoint {
  id: string;
  device_id: string;
  type: EndpointType;
  port: number;
  label: string;
  protocol: string;
  description?: string;
  enabled: boolean;
  discovered_at?: string;
}

export interface SessionTelemetry {
  connection_status: SessionConnectionStatus;
  latency_ms?: number;
  last_checked_at?: string;
  last_error?: string;
  probe_source?: "helper" | "agent" | "system";
}

export interface DeviceInventory {
  device: Device;
  endpoints: {
    web: Endpoint[];
    program: Endpoint[];
    bridge: Endpoint[];
  };
  capabilities: {
    has_serial: boolean;
    serial_ports: string[];
    modbus_serial_port?: string;
    activation_warning?: string;
    bundled_bridge_binary?: string;
  };
  freshness: {
    last_scan: string;
    is_stale: boolean;
  };
}

export interface Session {
  id: string;
  device_id: string;
  endpoint_id?: string;
  user_id: string;
  tenant_id: string;
  status: SessionStatus;
  local_port?: number;
  remote_port?: number;
  delivery_mode: DeliveryMode;
  ttl_seconds: number;
  idle_timeout_seconds?: number;
  remote_host?: string;
  started_at: string;
  expires_at: string;
  last_activity_at?: string;
  stopped_at?: string;
  stop_reason?: StopReason;
  tunnel_url?: string;
  telemetry?: SessionTelemetry;
  device?: Device;
  endpoint?: Endpoint;
  user?: User;
}

export interface ExportHistory {
  id: string;
  session_id?: string;
  user_id: string;
  device_id: string;
  endpoint_id?: string;
  tenant_id: string;
  site_id?: string;
  started_at: string;
  stopped_at?: string;
  stop_reason?: StopReason;
  local_bind_port?: number;
  delivery_mode?: DeliveryMode;
  duration_seconds?: number;
  bytes_transferred?: number;
  metadata?: Record<string, unknown>;
  telemetry?: SessionTelemetry;
  device?: Device;
  endpoint?: Endpoint;
  user?: User;
  site?: Site;
}

export interface BridgeProfile {
  id: string;
  device_id: string;
  endpoint_id?: string;
  serial_port: string;
  baud_rate: number;
  parity: string;
  stop_bits: number;
  data_bits: number;
  tcp_port?: number;
  status: "idle" | "active" | "error";
}

export interface AuditLog {
  id: string;
  tenant_id: string;
  user_id?: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  ip_address?: string;
  metadata?: Record<string, unknown>;
  created_at: string;
  user?: User;
}

// API Response shapes
export interface APIResponse<T = unknown> {
  success: boolean;
  data?: T;
  error?: string;
  meta?: {
    total: number;
    page: number;
    limit: number;
  };
}

// Auth
export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  refresh_token: string;
  user: User;
  expires_at: string;
}

// Session creation
export interface CreateSessionRequest {
  endpoint_id: string;
  delivery_mode: DeliveryMode;
  ttl_seconds?: number;
  local_port?: number;
}

// Bridge creation
export interface CreateBridgeRequest {
  serial_port: string;
  baud_rate: number;
  parity?: string;
  stop_bits?: number;
  data_bits?: number;
  tcp_port?: number;
  ttl_seconds?: number;
}

// Admin overview stats
export interface AdminStats {
  active_sessions_count: number;
  expiring_sessions_count: number;
  online_devices_count: number;
  offline_devices_count: number;
  total_devices_count: number;
  system_health: {
    auth_gateway: "stable" | "degraded" | "down";
    relay_latency_ms: number;
    tunnel_capacity_percent: number;
  };
}
