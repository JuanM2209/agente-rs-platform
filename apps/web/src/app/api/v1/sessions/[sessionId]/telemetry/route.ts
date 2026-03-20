import { decodeToken, MOCK_SESSIONS, ok, err } from "@/app/api/v1/_mock/data";

export async function POST(req: Request, { params }: { params: { sessionId: string } }) {
  const auth = req.headers.get("Authorization") ?? "";
  const user = decodeToken(auth.replace("Bearer ", ""));
  if (!user) return err("Unauthorized", 401);

  const idx = MOCK_SESSIONS.findIndex((session) => session.id === params.sessionId);
  if (idx === -1) return err("Session not found", 404);

  const session = MOCK_SESSIONS[idx];
  if (session.user_id !== user.id) return err("Forbidden", 403);

  const body = await req.json();
  const checkedAt = body.last_checked_at || new Date().toISOString();

  MOCK_SESSIONS[idx] = {
    ...session,
    telemetry: {
      connection_status: body.connection_status || session.telemetry?.connection_status || "pending",
      latency_ms: typeof body.latency_ms === "number" ? body.latency_ms : session.telemetry?.latency_ms,
      last_checked_at: checkedAt,
      last_error: body.last_error || undefined,
      probe_source: body.probe_source || session.telemetry?.probe_source || "helper",
    },
  };

  return ok(MOCK_SESSIONS[idx].telemetry);
}
