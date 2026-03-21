import { decodeToken, MOCK_DEVICES, MOCK_ENDPOINTS, MOCK_SESSIONS, ok, err } from "@/app/api/v1/_mock/data";
import type { Session } from "@/types";

const minTTLSeconds = 8 * 60 * 60;

export async function POST(req: Request, { params }: { params: { deviceId: string } }) {
  const auth = req.headers.get("Authorization") ?? "";
  const user = decodeToken(auth.replace("Bearer ", ""));
  if (!user) return err("Unauthorized", 401);

  const device = MOCK_DEVICES.find((d) => d.device_id === params.deviceId);
  if (!device) return err("Device not found", 404);
  if (device.status === "offline") return err("Device is offline", 409);

  const { endpoint_id, delivery_mode, ttl_seconds = minTTLSeconds, local_port } = await req.json();
  const allEndpoints = Object.values(MOCK_ENDPOINTS).flat();
  const endpoint = allEndpoints.find((e) => e.id === endpoint_id);
  const effectiveTTLSeconds = Math.max(Number(ttl_seconds) || minTTLSeconds, minTTLSeconds);
  const effectiveLocalPort = Number(local_port) || endpoint?.port;

  const now = Date.now();
  const session: Session = {
    id: `sess-${now}`,
    device_id: device.id,
    endpoint_id,
    user_id: user.id,
    tenant_id: user.tenant_id,
    status: "active",
    delivery_mode,
    ttl_seconds: effectiveTTLSeconds,
    local_port: effectiveLocalPort,
    remote_host: device.ip_address,
    remote_port: endpoint?.port,
    started_at: new Date(now).toISOString(),
    expires_at: new Date(now + effectiveTTLSeconds * 1000).toISOString(),
    tunnel_url: delivery_mode === "web" ? `https://session-${now}.nucleus.example.com` : undefined,
    telemetry: {
      connection_status: "pending",
      last_checked_at: new Date(now).toISOString(),
      probe_source: delivery_mode === "export" ? "helper" : "system",
    },
    device,
    endpoint: endpoint ?? undefined,
  };

  MOCK_SESSIONS.push(session);
  return ok(session);
}
