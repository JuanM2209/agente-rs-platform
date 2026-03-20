import { decodeToken, MOCK_HISTORY, MOCK_SESSIONS, ok, err } from "@/app/api/v1/_mock/data";

export async function DELETE(req: Request, { params }: { params: { sessionId: string } }) {
  const auth = req.headers.get("Authorization") ?? "";
  const user = decodeToken(auth.replace("Bearer ", ""));
  if (!user) return err("Unauthorized", 401);

  const idx = MOCK_SESSIONS.findIndex((s) => s.id === params.sessionId);
  if (idx === -1) return err("Session not found", 404);

  const stoppedAt = new Date().toISOString();
  const current = MOCK_SESSIONS[idx];
  MOCK_SESSIONS[idx] = {
    ...current,
    status: "stopped",
    stopped_at: stoppedAt,
    stop_reason: "user_stopped",
    telemetry: {
      ...current.telemetry,
      connection_status: "stopped",
      last_checked_at: stoppedAt,
    },
  };

  MOCK_HISTORY.unshift({
    id: `hist-${current.id}`,
    session_id: current.id,
    user_id: current.user_id,
    device_id: current.device_id,
    endpoint_id: current.endpoint_id,
    tenant_id: current.tenant_id,
    started_at: current.started_at,
    stopped_at: stoppedAt,
    stop_reason: "user_stopped",
    local_bind_port: current.local_port,
    delivery_mode: current.delivery_mode,
    duration_seconds: Math.max(
      0,
      Math.round((Date.parse(stoppedAt) - Date.parse(current.started_at)) / 1000),
    ),
    bytes_transferred: 0,
    metadata: {
      telemetry: {
        ...current.telemetry,
        connection_status: "stopped",
        last_checked_at: stoppedAt,
      },
    },
    telemetry: {
      ...current.telemetry,
      connection_status: "stopped",
      last_checked_at: stoppedAt,
    },
    device: current.device,
    endpoint: current.endpoint,
    user,
  });

  return ok(null);
}
