import { decodeToken, MOCK_SESSIONS, ok, err } from "@/app/api/v1/_mock/data";

export async function GET(req: Request) {
  const auth = req.headers.get("Authorization") ?? "";
  const user = decodeToken(auth.replace("Bearer ", ""));
  if (!user) return err("Unauthorized", 401);
  const sessions = MOCK_SESSIONS.filter((s) => s.user_id === user.id && s.status === "active");
  return ok(sessions);
}
