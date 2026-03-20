import { decodeToken, MOCK_DEVICES, ok, err } from "@/app/api/v1/_mock/data";

export async function GET(req: Request, { params }: { params: { deviceId: string } }) {
  const auth = req.headers.get("Authorization") ?? "";
  if (!decodeToken(auth.replace("Bearer ", ""))) return err("Unauthorized", 401);

  const device = MOCK_DEVICES.find((d) => d.device_id === params.deviceId);
  if (!device) return err("Device not found", 404);
  return ok(device);
}
