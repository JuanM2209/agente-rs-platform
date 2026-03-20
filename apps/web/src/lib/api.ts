// ============================================================
// Nucleus Portal — API Client
// ============================================================
import type {
  APIResponse,
  LoginRequest,
  LoginResponse,
  Device,
  DeviceInventory,
  Session,
  ExportHistory,
  CreateSessionRequest,
  CreateBridgeRequest,
  BridgeProfile,
  AdminStats,
  User,
} from "@/types";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

class APIError extends Error {
  constructor(
    message: string,
    public status: number,
    public code?: string,
  ) {
    super(message);
    this.name = "APIError";
  }
}

function getAuthToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("nucleus_token");
}

async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const token = getAuthToken();
  const headers: HeadersInit = {
    "Content-Type": "application/json",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...options.headers,
  };

  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers,
  });

  const data: APIResponse<T> = await res.json();

  if (!res.ok || !data.success) {
    throw new APIError(
      data.error || "An unexpected error occurred",
      res.status,
    );
  }

  return data.data as T;
}

// --- Auth ---

export async function login(req: LoginRequest): Promise<LoginResponse> {
  const res = await fetch(`${API_URL}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  const data: APIResponse<LoginResponse> = await res.json();
  if (!res.ok || !data.success) {
    throw new APIError(data.error || "Login failed", res.status);
  }
  return data.data!;
}

export async function logout(): Promise<void> {
  await request("/api/v1/auth/logout", { method: "POST" });
}

export async function getMe(): Promise<User> {
  return request<User>("/api/v1/me");
}

// --- Devices ---

export async function getDevice(deviceId: string): Promise<Device> {
  return request<Device>(`/api/v1/devices/${deviceId}`);
}

export async function getDeviceInventory(
  deviceId: string,
): Promise<DeviceInventory> {
  return request<DeviceInventory>(`/api/v1/devices/${deviceId}/inventory`);
}

export async function scanDevice(deviceId: string): Promise<void> {
  await request(`/api/v1/devices/${deviceId}/scan`, { method: "POST" });
}

// --- Sessions ---

export async function createSession(
  deviceId: string,
  req: CreateSessionRequest,
): Promise<Session> {
  return request<Session>(`/api/v1/devices/${deviceId}/sessions`, {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export async function stopSession(sessionId: string): Promise<void> {
  await request(`/api/v1/sessions/${sessionId}`, { method: "DELETE" });
}

export async function getActiveSessions(): Promise<Session[]> {
  return request<Session[]>("/api/v1/me/active-sessions");
}

export async function getMyExportHistory(): Promise<ExportHistory[]> {
  return request<ExportHistory[]>("/api/v1/me/export-history");
}

export async function getDeviceExportHistory(
  deviceId: string,
): Promise<ExportHistory[]> {
  return request<ExportHistory[]>(`/api/v1/devices/${deviceId}/export-history`);
}

// --- Bridges ---

export async function createModbusBridge(
  deviceId: string,
  req: CreateBridgeRequest,
): Promise<BridgeProfile> {
  return request<BridgeProfile>(
    `/api/v1/devices/${deviceId}/bridges/modbus-serial`,
    {
      method: "POST",
      body: JSON.stringify(req),
    },
  );
}

export async function stopBridge(bridgeId: string): Promise<void> {
  await request(`/api/v1/bridges/${bridgeId}`, { method: "DELETE" });
}

// --- Admin ---

export async function getAdminStats(): Promise<AdminStats> {
  return request<AdminStats>("/api/v1/admin/stats");
}

export { APIError };
