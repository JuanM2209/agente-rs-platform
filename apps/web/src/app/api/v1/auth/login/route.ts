import { MOCK_USERS, encodeToken, ok, err } from "@/app/api/v1/_mock/data";

export async function POST(req: Request) {
  const { email, password } = await req.json();
  const user = MOCK_USERS.find((u) => u.email === email && u.password === password);
  if (!user) return err("Invalid email or password", 401);

  const { password: _, ...safeUser } = user;
  const token = encodeToken(safeUser);

  return ok({
    token,
    refresh_token: `refresh_${token}`,
    user: safeUser,
    expires_at: new Date(Date.now() + 8 * 3600_000).toISOString(),
  });
}
