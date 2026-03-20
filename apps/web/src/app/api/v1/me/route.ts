import { decodeToken, MOCK_USERS, ok, err } from "@/app/api/v1/_mock/data";

function getUser(req: Request) {
  const auth = req.headers.get("Authorization") ?? "";
  const token = auth.replace("Bearer ", "");
  return decodeToken(token);
}

export async function GET(req: Request) {
  const user = getUser(req);
  if (!user) return err("Unauthorized", 401);
  const { password: _, ...safeUser } = user;
  return ok(safeUser);
}
