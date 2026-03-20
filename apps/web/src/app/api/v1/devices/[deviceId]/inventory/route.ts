import { decodeToken, MOCK_DEVICES, MOCK_ENDPOINTS, ok, err } from "@/app/api/v1/_mock/data";

const SERIAL_PORTS: Record<string, string[]> = {
  "N-1002": ["/dev/ttyUSB0"],
  "N-1004": ["/dev/ttyS0"],
};

export async function GET(req: Request, { params }: { params: { deviceId: string } }) {
  const auth = req.headers.get("Authorization") ?? "";
  if (!decodeToken(auth.replace("Bearer ", ""))) return err("Unauthorized", 401);

  const device = MOCK_DEVICES.find((d) => d.device_id === params.deviceId);
  if (!device) return err("Device not found", 404);

  const all = MOCK_ENDPOINTS[params.deviceId] ?? [];
  const serialPorts = SERIAL_PORTS[params.deviceId] ?? [];

  return ok({
    device,
    endpoints: {
      web:     all.filter((e) => e.type === "WEB"),
      program: all.filter((e) => e.type === "PROGRAM"),
      bridge:  all.filter((e) => e.type === "BRIDGE"),
    },
    capabilities: { has_serial: serialPorts.length > 0, serial_ports: serialPorts },
    freshness: { last_scan: device.inventory_updated_at ?? new Date().toISOString(), is_stale: false },
  });
}
