import { decodeToken, MOCK_HISTORY, ok, err } from "@/app/api/v1/_mock/data";

export async function GET(req: Request) {
  const auth = req.headers.get("Authorization") ?? "";
  const user = decodeToken(auth.replace("Bearer ", ""));
  if (!user) return err("Unauthorized", 401);

  const history = user.role === "admin"
    ? MOCK_HISTORY
    : MOCK_HISTORY.filter((record) => record.user_id === user.id);

  return ok(history);
}
