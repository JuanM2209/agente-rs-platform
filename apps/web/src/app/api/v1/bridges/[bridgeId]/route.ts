import { decodeToken, MOCK_BRIDGES, MOCK_ENDPOINTS, ok, err } from "@/app/api/v1/_mock/data";

export async function DELETE(req: Request, { params }: { params: { bridgeId: string } }) {
  const auth = req.headers.get("Authorization") ?? "";
  const user = decodeToken(auth.replace("Bearer ", ""));
  if (!user) return err("Unauthorized", 401);

  const bridge = MOCK_BRIDGES.find((item) => item.id === params.bridgeId);
  if (!bridge) return err("Bridge not found", 404);

  bridge.status = "idle";

  Object.values(MOCK_ENDPOINTS).forEach((endpoints) => {
    endpoints.forEach((endpoint) => {
      if (endpoint.id === bridge.endpoint_id) {
        endpoint.enabled = false;
      }
    });
  });

  return ok({ bridge_id: bridge.id, message: "bridge stopped" });
}
