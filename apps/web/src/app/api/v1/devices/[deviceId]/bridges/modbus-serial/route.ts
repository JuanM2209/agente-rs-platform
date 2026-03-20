import { decodeToken, MOCK_BRIDGES, MOCK_DEVICES, MOCK_ENDPOINTS, ok, err } from "@/app/api/v1/_mock/data";

export async function POST(req: Request, { params }: { params: { deviceId: string } }) {
  const auth = req.headers.get("Authorization") ?? "";
  const user = decodeToken(auth.replace("Bearer ", ""));
  if (!user) return err("Unauthorized", 401);

  const device = MOCK_DEVICES.find((d) => d.device_id === params.deviceId);
  if (!device) return err("Device not found", 404);
  if (device.status === "offline") return err("Device is offline", 409);

  const body = await req.json();
  const serialPort = String(body.serial_port ?? "").trim();
  const tcpPort = Number(body.tcp_port ?? 0);
  const baudRate = Number(body.baud_rate ?? 9600);
  const parity = String(body.parity ?? "N").trim().toUpperCase();
  const stopBits = Number(body.stop_bits ?? 1);
  const dataBits = Number(body.data_bits ?? 8);

  if (!serialPort) return err("serial_port is required", 400);
  if (!tcpPort || tcpPort < 1024 || tcpPort > 65535) {
    return err("tcp_port must be between 1024 and 65535", 400);
  }

  const now = new Date().toISOString();
  const endpointId = `e-${params.deviceId}-bridge-${tcpPort}`;
  const bridgeId = `bridge-${Date.now()}`;

  const existingEndpoints = MOCK_ENDPOINTS[params.deviceId] ?? [];
  const existingEndpoint = existingEndpoints.find((endpoint) => endpoint.port === tcpPort);

  if (existingEndpoint) {
    existingEndpoint.type = "BRIDGE";
    existingEndpoint.protocol = "mbusd";
    existingEndpoint.label = "Serial Modbus Bridge";
    existingEndpoint.description = `Ephemeral MBUSD bridge for ${serialPort}`;
    existingEndpoint.enabled = true;
  } else {
    existingEndpoints.push({
      id: endpointId,
      device_id: device.id,
      type: "BRIDGE",
      port: tcpPort,
      label: "Serial Modbus Bridge",
      protocol: "mbusd",
      description: `Ephemeral MBUSD bridge for ${serialPort}`,
      enabled: true,
      discovered_at: now,
    });
  }
  MOCK_ENDPOINTS[params.deviceId] = existingEndpoints;

  const bridge = {
    id: bridgeId,
    device_id: device.id,
    endpoint_id: existingEndpoint?.id ?? endpointId,
    serial_port: serialPort,
    baud_rate: baudRate,
    parity,
    stop_bits: stopBits,
    data_bits: dataBits,
    tcp_port: tcpPort,
    status: "active" as const,
  };

  MOCK_BRIDGES.push(bridge);
  return ok(bridge);
}
