"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { clsx } from "clsx";
import { StatusOrb } from "@/components/ui/StatusOrb";
import {
  createModbusBridge,
  createSession,
  getDeviceInventory,
  scanDevice,
  stopBridge,
} from "@/lib/api";
import {
  formatHoursLabel,
  getPortalPreferences,
  sessionHoursToSeconds,
} from "@/lib/portal-settings";
import type { CreateSessionRequest, DeviceInventory, Endpoint } from "@/types";

const endpointMeta: Record<number, { icon: string; tone: string; family: string }> = {
  22: { icon: "terminal", tone: "text-slate-300", family: "SSH" },
  80: { icon: "language", tone: "text-sky-300", family: "HTTP" },
  443: { icon: "lock", tone: "text-emerald-300", family: "HTTPS" },
  502: { icon: "electrical_services", tone: "text-orange-300", family: "Modbus TCP" },
  1880: { icon: "account_tree", tone: "text-cyan-300", family: "Node-RED" },
  9090: { icon: "monitoring", tone: "text-indigo-300", family: "Web Console" },
  44818: { icon: "settings_ethernet", tone: "text-violet-300", family: "EtherNet/IP" },
};

const defaultBridgeTCPPort = 5020;
const fallbackModbusSerialPort = "/dev/ttymxc5";

type SessionModalState = {
  endpoint: Endpoint;
  localPort: number;
  ttlHours: number;
};

type BridgeFormState = {
  serial_port: string;
  baud_rate: number;
  parity: "N" | "E" | "O";
  stop_bits: 1 | 2;
  data_bits: 7 | 8;
  tcp_port: number;
  export_local_port: number;
  acknowledge_warning: boolean;
};

function formatProtocol(endpoint: Endpoint) {
  return endpoint.protocol.replace(/_/g, " ").toUpperCase();
}

function getEndpointMeta(port: number) {
  return endpointMeta[port] || {
    icon: "device_hub",
    tone: "text-on-surface-variant",
    family: "TCP Service",
  };
}

function SectionHeader({
  title,
  caption,
  count,
}: {
  title: string;
  caption: string;
  count: number;
}) {
  return (
    <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
      <div>
        <h2 className="font-headline text-2xl font-bold text-on-surface">{title}</h2>
        <p className="text-sm text-on-surface-variant">{caption}</p>
      </div>
      <div className="rounded-xl bg-surface-container-highest px-4 py-2 text-xs font-technical uppercase tracking-[0.18em] text-on-surface-variant">
        {count} available
      </div>
    </div>
  );
}

function EndpointListRow({
  endpoint,
  deviceIP,
  canOpenWeb,
  openingWeb,
  onOpenWeb,
  onExport,
}: {
  endpoint: Endpoint;
  deviceIP?: string;
  canOpenWeb: boolean;
  openingWeb: boolean;
  onOpenWeb: (endpoint: Endpoint) => void;
  onExport: (endpoint: Endpoint) => void;
}) {
  const meta = getEndpointMeta(endpoint.port);

  return (
    <div className="grid gap-4 rounded-2xl border border-outline-variant/10 bg-surface-container-high px-5 py-4 transition-colors hover:bg-surface-container-highest lg:grid-cols-[minmax(0,1.35fr)_minmax(0,0.95fr)_auto]">
      <div className="flex items-start gap-4">
        <div className={clsx("rounded-xl bg-surface-container-highest p-3", meta.tone)}>
          <span className="material-symbols-outlined text-lg">{meta.icon}</span>
        </div>
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-3">
            <p className="font-semibold text-on-surface">{endpoint.label}</p>
            <span className="rounded-lg bg-surface-container-low px-2.5 py-1 text-[11px] font-technical uppercase tracking-[0.18em] text-on-surface-variant">
              {meta.family}
            </span>
          </div>
          <p className="mt-1 text-xs text-on-surface-variant">
            {endpoint.description || "Ready for remote diagnostics, browser access, or laptop export."}
          </p>
        </div>
      </div>

      <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-1">
        <div className="rounded-xl bg-surface-container-low px-3 py-2">
          <p className="text-[10px] uppercase tracking-[0.18em] text-outline">Remote Endpoint</p>
          <p className="mt-1 font-technical text-sm text-on-surface">
            {deviceIP || "Pending IP"}:{endpoint.port}
          </p>
        </div>
        <div className="rounded-xl bg-surface-container-low px-3 py-2">
          <p className="text-[10px] uppercase tracking-[0.18em] text-outline">Protocol</p>
          <p className="mt-1 font-technical text-sm text-on-surface">{formatProtocol(endpoint)}</p>
        </div>
      </div>

      <div className="flex flex-col gap-2 sm:flex-row lg:flex-col lg:items-end">
        {endpoint.type === "WEB" && (
          canOpenWeb ? (
            <button
              onClick={() => onOpenWeb(endpoint)}
              disabled={openingWeb}
              className="inline-flex min-w-[176px] items-center justify-center gap-2 rounded-xl bg-primary/10 px-4 py-3 text-sm font-semibold text-primary transition-colors hover:bg-primary/20"
            >
              <span className={clsx("material-symbols-outlined text-base", openingWeb && "animate-spin")}>{openingWeb ? "progress_activity" : "open_in_new"}</span>
              {openingWeb ? "Opening..." : "Open Web Port"}
            </button>
          ) : (
            <div className="inline-flex min-w-[176px] items-center justify-center gap-2 rounded-xl border border-outline-variant/10 bg-surface-container-low px-4 py-3 text-sm font-semibold text-outline">
              <span className="material-symbols-outlined text-base">schedule</span>
              Awaiting Device IP
            </div>
          )
        )}
        <button
          onClick={() => onExport(endpoint)}
          className="inline-flex min-w-[176px] items-center justify-center gap-2 rounded-xl gradient-primary px-4 py-3 text-sm font-bold text-on-primary transition-all hover:shadow-primary"
        >
          <span className="material-symbols-outlined text-base">laptop_windows</span>
          Export to Your Laptop
        </button>
      </div>
    </div>
  );
}

export default function DeviceDetailPage() {
  const params = useParams();
  const router = useRouter();
  const deviceId = params.deviceId as string;
  const [inventory, setInventory] = useState<DeviceInventory | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [scanning, setScanning] = useState(false);
  const [defaultSessionHours, setDefaultSessionHours] = useState(8);
  const [sessionModal, setSessionModal] = useState<SessionModalState | null>(null);
  const [creatingWebSessionFor, setCreatingWebSessionFor] = useState<string | null>(null);
  const [bridgeModal, setBridgeModal] = useState(false);
  const [creatingBridge, setCreatingBridge] = useState(false);
  const [bridgeError, setBridgeError] = useState("");
  const [bridgeForm, setBridgeForm] = useState<BridgeFormState>({
    serial_port: fallbackModbusSerialPort,
    baud_rate: 9600,
    parity: "N",
    stop_bits: 1,
    data_bits: 8,
    tcp_port: defaultBridgeTCPPort,
    export_local_port: defaultBridgeTCPPort,
    acknowledge_warning: false,
  });

  const loadInventory = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const data = await getDeviceInventory(deviceId);
      setInventory(data);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Device not found");
    } finally {
      setLoading(false);
    }
  }, [deviceId]);

  useEffect(() => {
    setDefaultSessionHours(getPortalPreferences().defaultSessionHours);
    void loadInventory();
  }, [loadInventory]);

  const handleScan = async () => {
    setScanning(true);
    try {
      await scanDevice(deviceId);
      await loadInventory();
    } finally {
      setScanning(false);
    }
  };

  const openExportModal = (endpoint: Endpoint) => {
    setSessionModal({
      endpoint,
      localPort: endpoint.port,
      ttlHours: getPortalPreferences().defaultSessionHours,
    });
  };

  const handleOpenWebPort = async (endpoint: Endpoint) => {
    if (!inventory?.device.ip_address) {
      window.alert("This device is online, but its Nucleus IP has not been reported yet. Refresh inventory after updating the Remote-S agent image, or use Export to Your Laptop for now.");
      return;
    }

    setCreatingWebSessionFor(endpoint.id);
    try {
      const session = await createSession(deviceId, {
        endpoint_id: endpoint.id,
        delivery_mode: "web",
        ttl_seconds: sessionHoursToSeconds(getPortalPreferences().defaultSessionHours),
      });

      if (!session.tunnel_url) {
        throw new Error("This web port still needs a device IP or tunnel target. Refresh inventory and try again.");
      }

      window.open(session.tunnel_url, "_blank", "noopener,noreferrer");
      router.push("/sessions");
    } catch (err: unknown) {
      window.alert(err instanceof Error ? err.message : "Failed to open the web port.");
    } finally {
      setCreatingWebSessionFor(null);
    }
  };

  const handleCreateSession = async () => {
    if (!sessionModal) return;

    const req: CreateSessionRequest = {
      endpoint_id: sessionModal.endpoint.id,
      delivery_mode: "export",
      ttl_seconds: sessionHoursToSeconds(sessionModal.ttlHours),
      local_port: sessionModal.localPort,
    };

    try {
      const session = await createSession(deviceId, req);
      setSessionModal(null);
      router.push("/sessions");
    } catch (err: unknown) {
      window.alert(err instanceof Error ? err.message : "Failed to create session.");
    }
  };

  const openBridgeModal = () => {
    const serialPort =
      inventory?.capabilities.serial_ports[0]
      ?? inventory?.capabilities.modbus_serial_port
      ?? fallbackModbusSerialPort;

    setBridgeForm({
      serial_port: serialPort,
      baud_rate: 9600,
      parity: "N",
      stop_bits: 1,
      data_bits: 8,
      tcp_port: defaultBridgeTCPPort,
      export_local_port: defaultBridgeTCPPort,
      acknowledge_warning: false,
    });
    setBridgeError("");
    setBridgeModal(true);
  };

  const handleCreateBridge = async () => {
    if (!inventory) return;
    if (!bridgeForm.acknowledge_warning) {
      setBridgeError("Confirm the serial-port warning before creating the bridge.");
      return;
    }

    setCreatingBridge(true);
    setBridgeError("");
    let createdBridge: { id: string; endpoint_id?: string } | null = null;

    try {
      createdBridge = await createModbusBridge(deviceId, {
        serial_port: bridgeForm.serial_port,
        baud_rate: bridgeForm.baud_rate,
        parity: bridgeForm.parity,
        stop_bits: bridgeForm.stop_bits,
        data_bits: bridgeForm.data_bits,
        tcp_port: bridgeForm.tcp_port,
        ttl_seconds: sessionHoursToSeconds(defaultSessionHours),
      });

      if (!createdBridge.endpoint_id) {
        throw new Error("Bridge endpoint was not created by the control plane.");
      }

      await createSession(deviceId, {
        endpoint_id: createdBridge.endpoint_id,
        delivery_mode: "export",
        ttl_seconds: sessionHoursToSeconds(defaultSessionHours),
        local_port: bridgeForm.export_local_port,
      });

      setBridgeModal(false);
      await loadInventory();
      router.push("/sessions");
    } catch (err: unknown) {
      if (createdBridge?.id) {
        try {
          await stopBridge(createdBridge.id);
        } catch {
          // best effort cleanup
        }
      }
      setBridgeError(err instanceof Error ? err.message : "Failed to start the serial bridge.");
    } finally {
      setCreatingBridge(false);
    }
  };

  if (loading) {
    return <div className="flex min-h-[60vh] items-center justify-center text-on-surface-variant">Loading device profile...</div>;
  }

  if (error || !inventory) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center px-6">
        <div className="rounded-3xl border border-outline-variant/10 bg-surface-container-high p-10 text-center">
          <p className="font-headline text-3xl font-bold text-on-surface">Device Not Found</p>
          <p className="mt-3 text-sm text-on-surface-variant">{error || `No device with ID "${deviceId}" was found.`}</p>
          <button onClick={() => router.push("/dashboard")} className="mt-6 rounded-xl gradient-primary px-6 py-3 font-bold text-on-primary">Back to Search</button>
        </div>
      </div>
    );
  }

  const { device, endpoints, capabilities, freshness } = inventory;
  const allEndpointCount = endpoints.web.length + endpoints.program.length + endpoints.bridge.length;
  const canOpenWebPorts = Boolean(device.ip_address);
  const ipStatusLabel = device.ip_address || "Awaiting device IP";
  const ipStatusHelp = device.ip_address
    ? `Last inventory scan: ${new Date(freshness.last_scan).toLocaleString()}`
    : "Remote-S is connected, but this Nucleus has not reported its LAN IP yet. Web-open needs that IP. Laptop export still works now.";

  return (
    <div className="mx-auto max-w-7xl px-6 py-8">
      <div className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <div className="mb-3 flex items-center gap-3">
            <StatusOrb status={device.status} showLabel animate />
            <span className="rounded-lg bg-surface-container-high px-3 py-1 text-[11px] font-technical uppercase tracking-[0.18em] text-primary">
              {device.device_id}
            </span>
          </div>
          <h1 className="font-headline text-4xl font-bold text-on-surface">{device.display_name || device.device_id}</h1>
          <p className="mt-2 max-w-3xl text-sm text-on-surface-variant">
            Choose a discovered port from the list below, open browser-ready services instantly, or export any TCP port to a custom localhost port on your laptop.
          </p>
        </div>
        <div className="flex flex-wrap gap-3">
          <button
            onClick={handleScan}
            disabled={scanning}
            className="inline-flex items-center gap-2 rounded-xl bg-surface-container-high px-4 py-3 text-sm font-semibold text-on-surface transition-colors hover:bg-surface-container-highest"
          >
            <span className={clsx("material-symbols-outlined text-base", scanning && "animate-spin")}>refresh</span>
            {scanning ? "Refreshing..." : "Refresh Inventory"}
          </button>
          <Link href="/settings" className="inline-flex items-center gap-2 rounded-xl border border-outline-variant/10 bg-surface-container-low px-4 py-3 text-sm font-semibold text-on-surface-variant transition-colors hover:bg-surface-container-high">
            <span className="material-symbols-outlined text-base">settings</span>
            Session Settings
          </Link>
        </div>
      </div>

      <div className="mb-8 grid gap-4 lg:grid-cols-3">
        <div className="rounded-2xl border border-outline-variant/10 bg-surface-container-high p-5">
          <p className="text-[11px] uppercase tracking-[0.18em] text-outline">Default Session Window</p>
          <p className="mt-2 font-headline text-3xl text-on-surface">{defaultSessionHours}h</p>
          <p className="mt-2 text-xs text-on-surface-variant">New sessions start at {formatHoursLabel(defaultSessionHours)}. Admins can change the default in Settings.</p>
        </div>
        <div className="rounded-2xl border border-outline-variant/10 bg-surface-container-high p-5">
          <p className="text-[11px] uppercase tracking-[0.18em] text-outline">Available Endpoints</p>
          <p className="mt-2 font-headline text-3xl text-on-surface">{allEndpointCount}</p>
          <p className="mt-2 text-xs text-on-surface-variant">Web, program, and temporary bridge ports ready for remote support workflows.</p>
        </div>
        <div className="rounded-2xl border border-outline-variant/10 bg-surface-container-high p-5">
          <p className="text-[11px] uppercase tracking-[0.18em] text-outline">Current Device IP</p>
          <p className={clsx("mt-2 font-technical text-2xl", device.ip_address ? "text-on-surface" : "text-amber-200")}>{ipStatusLabel}</p>
          <p className="mt-2 text-xs text-on-surface-variant">{ipStatusHelp}</p>
        </div>
      </div>

      {freshness.is_stale && (
        <div className="mb-8 rounded-2xl border border-amber-400/20 bg-amber-400/10 px-5 py-4 text-sm text-amber-100">
          Inventory is stale. Refresh the device before opening or exporting ports so the customer sees the latest IP and service state.
        </div>
      )}

      {!device.ip_address && (
        <div className="mb-8 rounded-2xl border border-primary/20 bg-primary/10 px-5 py-4 text-sm text-on-surface">
          Web ports are waiting on the device IP from the external Remote-S agent. `Export to Your Laptop` is available now. Once the updated agent reports the Nucleus IP, `Open Web Port` becomes a true one-click action.
        </div>
      )}

      <div className="space-y-8">
        <section className="space-y-4">
          <SectionHeader title="Web Ports" caption="Browser-friendly services that can open directly in a new tab or be exported to your laptop." count={endpoints.web.length} />
          <div className="space-y-3">
            {endpoints.web.length > 0 ? endpoints.web.map((endpoint) => (
              <EndpointListRow key={endpoint.id} endpoint={endpoint} deviceIP={device.ip_address} canOpenWeb={canOpenWebPorts} openingWeb={creatingWebSessionFor === endpoint.id} onOpenWeb={handleOpenWebPort} onExport={openExportModal} />
            )) : <div className="rounded-2xl border border-dashed border-outline-variant/20 bg-surface-container-low px-5 py-8 text-sm text-on-surface-variant">No browser-ready services are currently available on this device.</div>}
          </div>
        </section>

        <section className="space-y-4">
          <SectionHeader title="Program Ports" caption="Industrial protocols and engineering ports that are typically exported into a local laptop tool." count={endpoints.program.length} />
          <div className="space-y-3">
            {endpoints.program.length > 0 ? endpoints.program.map((endpoint) => (
              <EndpointListRow key={endpoint.id} endpoint={endpoint} deviceIP={device.ip_address} canOpenWeb={false} openingWeb={false} onOpenWeb={handleOpenWebPort} onExport={openExportModal} />
            )) : <div className="rounded-2xl border border-dashed border-outline-variant/20 bg-surface-container-low px-5 py-8 text-sm text-on-surface-variant">No program ports were discovered in the latest scan.</div>}
          </div>
        </section>

        {capabilities.has_serial && (
          <section className="space-y-4">
            <SectionHeader title="Serial Modbus Bridge" caption="Temporarily convert the Nucleus serial port into Modbus TCP with MBUSD, then export it to your laptop." count={capabilities.serial_ports.length} />
            <div className="rounded-3xl border border-outline-variant/10 bg-surface-container-high p-6">
              <div className="grid gap-6 lg:grid-cols-[minmax(0,1.15fr)_minmax(0,0.85fr)]">
                <div>
                  <div className="mb-4 flex flex-wrap gap-2">
                    {capabilities.serial_ports.map((serialPort) => (
                      <span key={serialPort} className="rounded-lg bg-surface-container-highest px-3 py-1 text-xs font-technical text-on-surface-variant">{serialPort}</span>
                    ))}
                    {capabilities.bundled_bridge_binary && (
                      <span className="rounded-lg bg-primary/10 px-3 py-1 text-xs font-technical text-primary">{capabilities.bundled_bridge_binary}</span>
                    )}
                  </div>
                  <p className="text-sm text-on-surface-variant">
                    Use this when the customer needs Modbus Poll, Prolink, or another TCP-based tool to reach a serial-only meter or controller through the Nucleus.
                  </p>
                </div>
                <div className="rounded-2xl border border-amber-400/20 bg-amber-400/10 p-4">
                  <p className="font-semibold text-amber-200">Important warning</p>
                  <p className="mt-2 text-sm text-amber-100/90">
                    Activating MBUSD on <span className="font-technical">{capabilities.modbus_serial_port || fallbackModbusSerialPort}</span> temporarily interrupts Node-RED Modbus serial communication on the same port until the bridge stops.
                  </p>
                  <button onClick={openBridgeModal} className="mt-4 inline-flex items-center gap-2 rounded-xl gradient-primary px-4 py-3 text-sm font-bold text-on-primary">
                    <span className="material-symbols-outlined text-base">add_link</span>
                    Start Serial Export
                  </button>
                </div>
              </div>
            </div>
          </section>
        )}
      </div>

      {sessionModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-4 backdrop-blur-sm">
          <div className="w-full max-w-xl rounded-3xl border border-outline-variant/10 bg-surface-bright p-8 surface-shadow">
            <h3 className="font-headline text-3xl font-bold text-on-surface">Export to Your Laptop</h3>
            <p className="mt-3 text-sm text-on-surface-variant">
              Map this remote device port into your laptop helper and choose the localhost port you want to use.
            </p>
            <div className="mt-6 grid gap-4 rounded-2xl bg-surface-container-high p-5 md:grid-cols-2">
              <div><p className="text-[10px] uppercase tracking-[0.18em] text-outline">Remote Port</p><p className="mt-1 font-technical text-xl text-on-surface">{device.ip_address || "Pending"}:{sessionModal.endpoint.port}</p></div>
              <div><p className="text-[10px] uppercase tracking-[0.18em] text-outline">Session Window</p><p className="mt-1 font-technical text-xl text-on-surface">{sessionModal.ttlHours}h</p></div>
            </div>
            <div className="mt-5">
              <label className="mb-2 block text-sm text-on-surface-variant">Laptop localhost port</label>
              <input
                type="number"
                min={1}
                max={65535}
                value={sessionModal.localPort}
                onChange={(e) => setSessionModal((current) => current ? { ...current, localPort: Number(e.target.value) || current.endpoint.port } : current)}
                className="w-full rounded-xl border border-outline-variant/10 bg-surface-container-high px-4 py-3 text-sm text-on-surface outline-none focus:border-primary"
              />
              <p className="mt-2 text-xs text-on-surface-variant">Example: export device port {sessionModal.endpoint.port} to <span className="font-technical">127.0.0.1:1889</span> if your tool expects a custom local port.</p>
            </div>
            <div className="mt-8 flex gap-3">
              <button onClick={() => setSessionModal(null)} className="flex-1 rounded-xl px-4 py-3 text-sm font-semibold text-on-surface-variant transition-colors hover:bg-surface-container-high">Cancel</button>
              <button onClick={handleCreateSession} className="flex-1 rounded-xl gradient-primary px-4 py-3 text-sm font-bold text-on-primary">Export Port</button>
            </div>
          </div>
        </div>
      )}

      {bridgeModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-4 backdrop-blur-sm">
          <div className="w-full max-w-2xl rounded-3xl border border-outline-variant/10 bg-surface-bright p-8 surface-shadow">
            <h3 className="font-headline text-3xl font-bold text-on-surface">Create Serial Modbus Export</h3>
            <p className="mt-3 text-sm text-on-surface-variant">Build an MBUSD bridge on the Nucleus, then export that temporary TCP port to the localhost port of your choice.</p>
            <div className="mt-6 grid gap-4 md:grid-cols-2">
              <div><label className="mb-2 block text-sm text-on-surface-variant">Serial Port</label><select value={bridgeForm.serial_port} onChange={(e) => setBridgeForm((current) => ({ ...current, serial_port: e.target.value }))} className="w-full rounded-xl border border-outline-variant/10 bg-surface-container-high px-4 py-3 text-sm text-on-surface outline-none focus:border-primary">{capabilities.serial_ports.map((port) => <option key={port} value={port}>{port}</option>)}</select></div>
              <div><label className="mb-2 block text-sm text-on-surface-variant">Baud Rate</label><select value={bridgeForm.baud_rate} onChange={(e) => setBridgeForm((current) => ({ ...current, baud_rate: Number(e.target.value) }))} className="w-full rounded-xl border border-outline-variant/10 bg-surface-container-high px-4 py-3 text-sm text-on-surface outline-none focus:border-primary">{[9600, 19200, 38400, 57600, 115200].map((baud) => <option key={baud} value={baud}>{baud}</option>)}</select></div>
              <div><label className="mb-2 block text-sm text-on-surface-variant">Bridge TCP Port on Nucleus</label><input type="number" min={1024} max={65535} value={bridgeForm.tcp_port} onChange={(e) => setBridgeForm((current) => ({ ...current, tcp_port: Number(e.target.value) || defaultBridgeTCPPort }))} className="w-full rounded-xl border border-outline-variant/10 bg-surface-container-high px-4 py-3 text-sm text-on-surface outline-none focus:border-primary" /></div>
              <div><label className="mb-2 block text-sm text-on-surface-variant">Laptop localhost export port</label><input type="number" min={1} max={65535} value={bridgeForm.export_local_port} onChange={(e) => setBridgeForm((current) => ({ ...current, export_local_port: Number(e.target.value) || current.tcp_port }))} className="w-full rounded-xl border border-outline-variant/10 bg-surface-container-high px-4 py-3 text-sm text-on-surface outline-none focus:border-primary" /></div>
            </div>
            <div className="mt-4 grid gap-4 md:grid-cols-3">
              <div><label className="mb-2 block text-sm text-on-surface-variant">Parity</label><select value={bridgeForm.parity} onChange={(e) => setBridgeForm((current) => ({ ...current, parity: e.target.value as "N" | "E" | "O" }))} className="w-full rounded-xl border border-outline-variant/10 bg-surface-container-high px-4 py-3 text-sm text-on-surface outline-none focus:border-primary"><option value="N">None</option><option value="E">Even</option><option value="O">Odd</option></select></div>
              <div><label className="mb-2 block text-sm text-on-surface-variant">Data Bits</label><select value={bridgeForm.data_bits} onChange={(e) => setBridgeForm((current) => ({ ...current, data_bits: Number(e.target.value) as 7 | 8 }))} className="w-full rounded-xl border border-outline-variant/10 bg-surface-container-high px-4 py-3 text-sm text-on-surface outline-none focus:border-primary"><option value={7}>7</option><option value={8}>8</option></select></div>
              <div><label className="mb-2 block text-sm text-on-surface-variant">Stop Bits</label><select value={bridgeForm.stop_bits} onChange={(e) => setBridgeForm((current) => ({ ...current, stop_bits: Number(e.target.value) as 1 | 2 }))} className="w-full rounded-xl border border-outline-variant/10 bg-surface-container-high px-4 py-3 text-sm text-on-surface outline-none focus:border-primary"><option value={1}>1</option><option value={2}>2</option></select></div>
            </div>
            <div className="mt-5 rounded-2xl border border-amber-400/20 bg-amber-400/10 p-4">
              <p className="font-semibold text-amber-200">Customer warning</p>
              <p className="mt-2 text-sm text-amber-100/90">While this MBUSD bridge is active on <span className="font-technical">{bridgeForm.serial_port}</span>, Node-RED Modbus serial communication on the same port will be paused.</p>
            </div>
            <label className="mt-4 flex items-start gap-3 rounded-2xl bg-surface-container-high px-4 py-4">
              <input type="checkbox" checked={bridgeForm.acknowledge_warning} onChange={(e) => setBridgeForm((current) => ({ ...current, acknowledge_warning: e.target.checked }))} className="mt-1" />
              <span className="text-sm text-on-surface-variant">I understand the serial-port interruption and want to continue with the export.</span>
            </label>
            {bridgeError && <div className="mt-4 rounded-xl border border-error/30 bg-error/10 px-4 py-3 text-sm text-error">{bridgeError}</div>}
            <div className="mt-8 flex gap-3">
              <button onClick={() => setBridgeModal(false)} disabled={creatingBridge} className="flex-1 rounded-xl px-4 py-3 text-sm font-semibold text-on-surface-variant transition-colors hover:bg-surface-container-high">Cancel</button>
              <button onClick={handleCreateBridge} disabled={creatingBridge} className="flex-1 rounded-xl gradient-primary px-4 py-3 text-sm font-bold text-on-primary disabled:opacity-60">{creatingBridge ? "Starting..." : "Start Serial Export"}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
