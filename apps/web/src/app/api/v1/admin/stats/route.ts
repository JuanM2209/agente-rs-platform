import { decodeToken, MOCK_DEVICES, MOCK_SESSIONS, ok, err } from "@/app/api/v1/_mock/data";

export async function GET(req: Request) {
  const auth = req.headers.get("Authorization") ?? "";
  const user = decodeToken(auth.replace("Bearer ", ""));
  if (!user) return err("Unauthorized", 401);
  if (user.role !== "admin") return err("Forbidden", 403);

  const online = MOCK_DEVICES.filter((d) => d.status === "online").length;
  const offline = MOCK_DEVICES.filter((d) => d.status === "offline").length;
  const active = MOCK_SESSIONS.filter((s) => s.status === "active").length;
  const expiring = MOCK_SESSIONS.filter((s) => {
    if (s.status !== "active") return false;
    return new Date(s.expires_at).getTime() - Date.now() < 300_000;
  }).length;

  return ok({
    active_sessions_count: active,
    expiring_sessions_count: expiring,
    online_devices_count: online,
    offline_devices_count: offline,
    total_devices_count: MOCK_DEVICES.length,
    system_health: { auth_gateway: "stable", relay_latency_ms: 12, tunnel_capacity_percent: 18 },
  });
}
