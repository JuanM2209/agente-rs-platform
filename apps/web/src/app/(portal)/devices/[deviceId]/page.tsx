"use client";

import { useState, useEffect, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { StatusOrb } from "@/components/ui/StatusOrb";
import { getDeviceInventory, createSession, createModbusBridge, scanDevice, stopBridge } from "@/lib/api";
import type { DeviceInventory, Endpoint, CreateSessionRequest } from "@/types";
import { clsx } from "clsx";

const endpointMeta: Record<number, { icon: string; color: string }> = {
  80: { icon: "language", color: "text-primary" },
  443: { icon: "https", color: "text-tertiary" },
  1880: { icon: "account_tree", color: "text-amber-400" },
  9090: { icon: "monitor", color: "text-primary" },
  502: { icon: "electrical_services", color: "text-orange-400" },
  22: { icon: "terminal", color: "text-on-surface-variant" },
  44818: { icon: "settings_ethernet", color: "text-purple-400" },
};

const defaultBridgeTTLSeconds = 3600;
const defaultBridgeTCPPort = 5020;
const fallbackModbusSerialPort = "/dev/ttymxc5";

type BridgeFormState = {
  serial_port: string;
  baud_rate: number;
  parity: "N" | "E" | "O";
  stop_bits: 1 | 2;
  data_bits: 7 | 8;
  tcp_port: number;
  acknowledge_warning: boolean;
};

function EndpointCard({
  endpoint,
  onOpen,
  onExport,
}: {
  endpoint: Endpoint;
  onOpen: (ep: Endpoint) => void;
  onExport: (ep: Endpoint) => void;
}) {
  const meta = endpointMeta[endpoint.port] || { icon: "device_hub", color: "text-outline" };

  return (
    <div className="bg-surface-container-high rounded-xl p-5 hover:bg-surface-bright transition-colors group">
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div className={`p-2.5 bg-surface-container-highest rounded-lg ${meta.color}`}>
            <span className="material-symbols-outlined text-lg">{meta.icon}</span>
          </div>
          <div>
            <p className="font-semibold text-on-surface text-sm">{endpoint.label}</p>
            <p className="font-technical text-xs text-outline mt-0.5">
              :{endpoint.port} / {endpoint.protocol.toUpperCase()}
            </p>
          </div>
        </div>
        <span className="w-1.5 h-1.5 rounded-full bg-tertiary flex-shrink-0 mt-1" />
      </div>

      {endpoint.description && (
        <p className="text-xs text-on-surface-variant mb-4 leading-relaxed">
          {endpoint.description}
        </p>
      )}

      <div className="flex gap-2">
        {endpoint.type === "WEB" && (
          <button
            onClick={() => onOpen(endpoint)}
            className="flex-1 flex items-center justify-center gap-2 bg-primary/10 hover:bg-primary/20 text-primary text-xs font-bold py-2 rounded-lg transition-colors"
          >
            <span className="material-symbols-outlined text-base">open_in_new</span>
            Open Web
          </button>
        )}
        <button
          onClick={() => onExport(endpoint)}
          className="flex-1 flex items-center justify-center gap-2 bg-surface-container-highest hover:bg-outline-variant text-on-surface-variant hover:text-on-surface text-xs font-bold py-2 rounded-lg transition-colors"
        >
          <span className="material-symbols-outlined text-base">output</span>
          Export
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
  const [sessionModal, setSessionModal] = useState<{
    endpoint: Endpoint;
    mode: "web" | "export";
  } | null>(null);
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
    acknowledge_warning: false,
  });
  const [activeTab, setActiveTab] = useState<"endpoints" | "sessions" | "history">("endpoints");

  const loadInventory = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const data = await getDeviceInventory(deviceId);
      setInventory(data);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Device not found";
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, [deviceId]);

  useEffect(() => {
    void loadInventory();
  }, [loadInventory]);

  const handleScan = async () => {
    setScanning(true);
    try {
      await scanDevice(deviceId);
      await loadInventory();
    } catch {
      // ignore
    } finally {
      setScanning(false);
    }
  };

  const handleOpenSession = (ep: Endpoint) => {
    setSessionModal({ endpoint: ep, mode: "web" });
  };

  const handleExport = (ep: Endpoint) => {
    setSessionModal({ endpoint: ep, mode: "export" });
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
      acknowledge_warning: false,
    });
    setBridgeError("");
    setBridgeModal(true);
  };

  const handleCreateBridge = async () => {
    if (!inventory) return;
    if (!bridgeForm.serial_port) {
      setBridgeError("Select a serial port before starting the Modbus bridge.");
      return;
    }
    if (!bridgeForm.acknowledge_warning) {
      setBridgeError("Confirm the Node-RED interruption warning before continuing.");
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
      });

      if (!createdBridge.endpoint_id) {
        throw new Error("Bridge endpoint was not created by the control plane.");
      }

      await createSession(deviceId, {
        endpoint_id: createdBridge.endpoint_id,
        delivery_mode: "export",
        ttl_seconds: defaultBridgeTTLSeconds,
      });

      await loadInventory();
      setBridgeModal(false);
      router.push("/sessions");
    } catch (err: unknown) {
      if (createdBridge?.id) {
        try {
          await stopBridge(createdBridge.id);
        } catch {
          // Best effort cleanup if session creation failed after the bridge started.
        }
      }

      setBridgeError(err instanceof Error ? err.message : "Failed to start MBUSD export bridge.");
    } finally {
      setCreatingBridge(false);
    }
  };

  const handleCreateSession = async () => {
    if (!sessionModal) return;
    const req: CreateSessionRequest = {
      endpoint_id: sessionModal.endpoint.id,
      delivery_mode: sessionModal.mode,
      ttl_seconds: 3600,
    };
    try {
      const session = await createSession(deviceId, req);
      if (sessionModal.mode === "web" && session.tunnel_url) {
        window.open(session.tunnel_url, "_blank", "noopener,noreferrer");
      }
      setSessionModal(null);
      router.push("/sessions");
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to create session");
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="text-center">
          <div className="w-12 h-12 rounded-full border-2 border-primary border-t-transparent animate-spin mx-auto mb-4" />
          <p className="text-on-surface-variant text-sm">Loading device profile...</p>
        </div>
      </div>
    );
  }

  if (error || !inventory) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="text-center max-w-sm">
          <div className="w-16 h-16 rounded-full bg-error/10 flex items-center justify-center mx-auto mb-4">
            <span className="material-symbols-outlined text-error text-2xl">device_unknown</span>
          </div>
          <h3 className="font-headline font-bold text-on-surface text-xl mb-2">Device Not Found</h3>
          <p className="text-on-surface-variant text-sm mb-6">
            {error || `No device with ID "${deviceId}" was found in your tenant.`}
          </p>
          <button
            onClick={() => router.push("/dashboard")}
            className="gradient-primary text-on-primary font-bold px-6 py-3 rounded-xl text-sm"
          >
            Back to Search
          </button>
        </div>
      </div>
    );
  }

  const { device, endpoints, capabilities, freshness } = inventory;

  return (
    <div className="max-w-7xl mx-auto px-6 py-8 animate-fade-in">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-sm text-on-surface-variant mb-6">
        <button onClick={() => router.push("/dashboard")} className="hover:text-on-surface transition-colors">
          Home
        </button>
        <span className="material-symbols-outlined text-base">chevron_right</span>
        <span className="font-technical text-primary">{device.device_id}</span>
      </div>

      {/* Device header */}
      <div className="bg-surface-container-low rounded-xl p-8 mb-6">
        <div className="flex flex-col lg:flex-row lg:items-start gap-6 justify-between">
          <div className="flex items-start gap-5">
            <div className="w-16 h-16 rounded-xl bg-surface-container-high flex items-center justify-center flex-shrink-0">
              <span className="material-symbols-outlined text-primary text-3xl">memory</span>
            </div>
            <div>
              <div className="flex items-center gap-3 mb-1">
                <h1 className="font-headline text-2xl font-bold text-on-surface">
                  {device.display_name || device.device_id}
                </h1>
                <StatusOrb status={device.status} showLabel animate />
              </div>
              <p className="font-technical text-sm text-on-surface-variant">{device.device_id}</p>
              {device.site && (
                <p className="text-sm text-on-surface-variant mt-1 flex items-center gap-1">
                  <span className="material-symbols-outlined text-base">location_on</span>
                  {device.site.name}
                  {device.site.location && ` — ${device.site.location}`}
                </p>
              )}
            </div>
          </div>

          <div className="flex flex-wrap gap-3">
            <button
              onClick={handleScan}
              disabled={scanning}
              className="flex items-center gap-2 bg-surface-container-high hover:bg-surface-bright px-4 py-2.5 rounded-xl text-sm font-medium text-on-surface-variant hover:text-on-surface transition-colors"
            >
              <span className={clsx("material-symbols-outlined text-base", scanning && "animate-spin")}>
                refresh
              </span>
              {scanning ? "Scanning..." : "Refresh Inventory"}
            </button>
            <button className="flex items-center gap-2 bg-surface-container-high hover:bg-surface-bright px-4 py-2.5 rounded-xl text-sm font-medium text-on-surface-variant hover:text-on-surface transition-colors">
              <span className="material-symbols-outlined text-base">history</span>
              Session History
            </button>
          </div>
        </div>

        {/* Metadata chips */}
        <div className="mt-6 flex flex-wrap gap-3">
          {device.firmware_version && (
            <div className="flex items-center gap-2 bg-surface-container-high px-3 py-1.5 rounded-lg">
              <span className="material-symbols-outlined text-outline text-base">system_update</span>
              <span className="text-xs font-technical text-on-surface-variant">
                FW {device.firmware_version}
              </span>
            </div>
          )}
          {device.ip_address && (
            <div className="flex items-center gap-2 bg-surface-container-high px-3 py-1.5 rounded-lg">
              <span className="material-symbols-outlined text-outline text-base">dns</span>
              <span className="text-xs font-technical text-on-surface-variant">{device.ip_address}</span>
            </div>
          )}
          {freshness && (
            <div className="flex items-center gap-2 bg-surface-container-high px-3 py-1.5 rounded-lg">
              <span className={clsx("material-symbols-outlined text-base", freshness.is_stale ? "text-error" : "text-tertiary")}>
                {freshness.is_stale ? "warning" : "check_circle"}
              </span>
              <span className="text-xs font-technical text-on-surface-variant">
                {freshness.is_stale ? "Stale inventory" : "Inventory fresh"}
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 bg-surface-container-low rounded-xl p-1 mb-6 w-fit">
        {(["endpoints", "sessions", "history"] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={clsx(
              "px-5 py-2.5 rounded-lg text-sm font-medium transition-colors capitalize",
              activeTab === tab
                ? "bg-surface-container-high text-on-surface"
                : "text-on-surface-variant hover:text-on-surface",
            )}
          >
            {tab}
          </button>
        ))}
      </div>

      {activeTab === "endpoints" && (
        <div className="space-y-8">
          {/* WEB endpoints */}
          {endpoints.web.length > 0 && (
            <div>
              <div className="flex items-center gap-3 mb-4">
                <div className="p-1.5 bg-primary/10 rounded-lg">
                  <span className="material-symbols-outlined text-primary text-base">language</span>
                </div>
                <h2 className="font-headline font-bold text-on-surface">Web Endpoints</h2>
                <span className="text-xs font-technical text-on-surface-variant bg-surface-container-high px-2 py-0.5 rounded">
                  {endpoints.web.length} available
                </span>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {endpoints.web.map((ep) => (
                  <EndpointCard
                    key={ep.id}
                    endpoint={ep}
                    onOpen={handleOpenSession}
                    onExport={handleExport}
                  />
                ))}
              </div>
            </div>
          )}

          {/* PROGRAM endpoints */}
          {endpoints.program.length > 0 && (
            <div>
              <div className="flex items-center gap-3 mb-4">
                <div className="p-1.5 bg-orange-400/10 rounded-lg">
                  <span className="material-symbols-outlined text-orange-400 text-base">
                    electrical_services
                  </span>
                </div>
                <h2 className="font-headline font-bold text-on-surface">Program Endpoints</h2>
                <span className="text-xs font-technical text-on-surface-variant bg-surface-container-high px-2 py-0.5 rounded">
                  {endpoints.program.length} available
                </span>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {endpoints.program.map((ep) => (
                  <EndpointCard
                    key={ep.id}
                    endpoint={ep}
                    onOpen={handleOpenSession}
                    onExport={handleExport}
                  />
                ))}
              </div>
            </div>
          )}

          {/* BRIDGE endpoints */}
          {(endpoints.bridge.length > 0 || capabilities.has_serial) && (
            <div>
              <div className="flex items-center gap-3 mb-4">
                <div className="p-1.5 bg-purple-400/10 rounded-lg">
                  <span className="material-symbols-outlined text-purple-400 text-base">
                    settings_ethernet
                  </span>
                </div>
                <h2 className="font-headline font-bold text-on-surface">Serial Bridges</h2>
                {capabilities.has_serial && (
                  <span className="text-xs font-technical text-tertiary bg-tertiary/10 px-2 py-0.5 rounded">
                    Serial capable
                  </span>
                )}
              </div>

              {capabilities.has_serial && (
                <div className="bg-surface-container-high rounded-xl p-6 border border-outline-variant/10">
                  {capabilities.activation_warning && (
                    <div className="mb-5 rounded-xl border border-amber-400/20 bg-amber-400/10 px-4 py-3">
                      <div className="flex items-start gap-3">
                        <span className="material-symbols-outlined text-amber-300 text-base mt-0.5">
                          warning
                        </span>
                        <div>
                          <p className="text-sm font-semibold text-amber-200">
                            Shared Serial Port Warning
                          </p>
                          <p className="text-xs text-amber-100/90 mt-1 leading-relaxed">
                            {capabilities.activation_warning}
                          </p>
                        </div>
                      </div>
                    </div>
                  )}

                  <div className="flex items-start justify-between">
                    <div>
                      <p className="font-semibold text-on-surface mb-1">Modbus Serial Bridge</p>
                      <p className="text-sm text-on-surface-variant">
                        Create an ephemeral TCP bridge over a serial port (MBUSD) and export it to the helper.
                      </p>
                      <div className="flex flex-wrap gap-2 mt-3">
                        {capabilities.serial_ports.map((port) => (
                          <span
                            key={port}
                            className="font-technical text-xs bg-surface-container-highest px-2 py-1 rounded text-on-surface-variant"
                          >
                            {port}
                          </span>
                        ))}
                        {capabilities.bundled_bridge_binary && (
                          <span className="font-technical text-xs bg-primary/10 px-2 py-1 rounded text-primary">
                            {capabilities.bundled_bridge_binary}
                          </span>
                        )}
                      </div>
                    </div>
                    <button
                      onClick={openBridgeModal}
                      className="flex items-center gap-2 gradient-primary text-on-primary font-bold px-5 py-2.5 rounded-xl text-sm hover:shadow-primary transition-all active:scale-95 flex-shrink-0 ml-4"
                    >
                      <span className="material-symbols-outlined text-base">add_link</span>
                      Start MBUSD + Export
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}

          {endpoints.web.length === 0 &&
            endpoints.program.length === 0 &&
            !capabilities.has_serial && (
              <div className="text-center py-16">
                <div className="w-16 h-16 rounded-full bg-surface-container-high flex items-center justify-center mx-auto mb-4">
                  <span className="material-symbols-outlined text-outline text-2xl">sensors_off</span>
                </div>
                <p className="font-headline font-bold text-on-surface mb-2">No Endpoints Found</p>
                <p className="text-sm text-on-surface-variant">
                  Run a scan to discover endpoints on this device.
                </p>
                <button
                  onClick={handleScan}
                  className="mt-4 gradient-primary text-on-primary font-bold px-6 py-3 rounded-xl text-sm"
                >
                  Scan Now
                </button>
              </div>
            )}
        </div>
      )}

      {activeTab === "sessions" && (
        <div className="bg-surface-container-low rounded-xl p-8 text-center">
          <span className="material-symbols-outlined text-outline text-4xl">sensors</span>
          <p className="text-on-surface-variant mt-2">Active sessions view — coming soon</p>
          <a href="/sessions" className="text-primary text-sm mt-2 inline-block">
            View all active sessions →
          </a>
        </div>
      )}

      {activeTab === "history" && (
        <div className="bg-surface-container-low rounded-xl p-8 text-center">
          <span className="material-symbols-outlined text-outline text-4xl">history</span>
          <p className="text-on-surface-variant mt-2">Export history for this device</p>
          <a href="/history" className="text-primary text-sm mt-2 inline-block">
            View full history →
          </a>
        </div>
      )}

      {/* Session Modal */}
      {sessionModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-surface-bright rounded-xl p-8 w-full max-w-md mx-4 surface-shadow">
            <h3 className="font-headline font-bold text-on-surface text-xl mb-2">
              {sessionModal.mode === "web" ? "Open Web Session" : "Export Port"}
            </h3>
            <p className="text-sm text-on-surface-variant mb-6">
              {sessionModal.mode === "web"
                ? "This will create a temporary web session and open the device UI in a new tab."
                : "This will create a local TCP mapping via the Nucleus Windows Helper."}
            </p>

            <div className="bg-surface-container-high rounded-xl p-4 mb-6">
              <div className="flex items-center justify-between">
                <span className="text-sm text-on-surface-variant">Endpoint</span>
                <span className="font-technical text-sm text-on-surface">
                  {sessionModal.endpoint.label} :{sessionModal.endpoint.port}
                </span>
              </div>
              <div className="flex items-center justify-between mt-2">
                <span className="text-sm text-on-surface-variant">TTL</span>
                <span className="font-technical text-sm text-on-surface">1 hour</span>
              </div>
            </div>

            <div className="flex gap-3">
              <button
                onClick={() => setSessionModal(null)}
                className="flex-1 py-3 rounded-xl text-sm font-medium text-on-surface-variant hover:bg-surface-container-high transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateSession}
                className="flex-1 gradient-primary text-on-primary font-bold py-3 rounded-xl text-sm hover:shadow-primary transition-all"
              >
                {sessionModal.mode === "web" ? "Open Session" : "Export"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Bridge Modal */}
      {bridgeModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-surface-bright rounded-xl p-8 w-full max-w-md mx-4 surface-shadow">
            <h3 className="font-headline font-bold text-on-surface text-xl mb-2">
              Create Modbus Serial Bridge
            </h3>
            <p className="text-sm text-on-surface-variant mb-6">
              Start an MBUSD bridge on the Nucleus and immediately export that serial Modbus channel to the helper.
            </p>
            <div className="space-y-4 mb-6">
              <div>
                <label className="text-sm text-on-surface-variant block mb-1">Serial Port</label>
                <select
                  value={bridgeForm.serial_port}
                  onChange={(e) => setBridgeForm((current) => ({ ...current, serial_port: e.target.value }))}
                  className="w-full bg-surface-container-high rounded-xl px-4 py-3 text-sm text-on-surface border-b-2 border-transparent focus:border-primary outline-none"
                >
                  {capabilities.serial_ports.map((p) => (
                    <option key={p} value={p}>{p}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-sm text-on-surface-variant block mb-1">Baud Rate</label>
                <select
                  value={bridgeForm.baud_rate}
                  onChange={(e) => setBridgeForm((current) => ({ ...current, baud_rate: Number(e.target.value) }))}
                  className="w-full bg-surface-container-high rounded-xl px-4 py-3 text-sm text-on-surface border-b-2 border-transparent focus:border-primary outline-none"
                >
                  {[9600, 19200, 38400, 57600, 115200].map((b) => (
                    <option key={b} value={b}>{b}</option>
                  ))}
                </select>
              </div>
              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className="text-sm text-on-surface-variant block mb-1">Parity</label>
                  <select
                    value={bridgeForm.parity}
                    onChange={(e) => setBridgeForm((current) => ({ ...current, parity: e.target.value as "N" | "E" | "O" }))}
                    className="w-full bg-surface-container-high rounded-xl px-4 py-3 text-sm text-on-surface border-b-2 border-transparent focus:border-primary outline-none"
                  >
                    <option value="N">None</option>
                    <option value="E">Even</option>
                    <option value="O">Odd</option>
                  </select>
                </div>
                <div>
                  <label className="text-sm text-on-surface-variant block mb-1">Data Bits</label>
                  <select
                    value={bridgeForm.data_bits}
                    onChange={(e) => setBridgeForm((current) => ({ ...current, data_bits: Number(e.target.value) as 7 | 8 }))}
                    className="w-full bg-surface-container-high rounded-xl px-4 py-3 text-sm text-on-surface border-b-2 border-transparent focus:border-primary outline-none"
                  >
                    <option value={7}>7</option>
                    <option value={8}>8</option>
                  </select>
                </div>
                <div>
                  <label className="text-sm text-on-surface-variant block mb-1">Stop Bits</label>
                  <select
                    value={bridgeForm.stop_bits}
                    onChange={(e) => setBridgeForm((current) => ({ ...current, stop_bits: Number(e.target.value) as 1 | 2 }))}
                    className="w-full bg-surface-container-high rounded-xl px-4 py-3 text-sm text-on-surface border-b-2 border-transparent focus:border-primary outline-none"
                  >
                    <option value={1}>1</option>
                    <option value={2}>2</option>
                  </select>
                </div>
              </div>
              <div>
                <label className="text-sm text-on-surface-variant block mb-1">Bridge TCP Port</label>
                <input
                  type="number"
                  min={1024}
                  max={65535}
                  value={bridgeForm.tcp_port}
                  onChange={(e) => setBridgeForm((current) => ({ ...current, tcp_port: Number(e.target.value) }))}
                  className="w-full bg-surface-container-high rounded-xl px-4 py-3 text-sm text-on-surface border-b-2 border-transparent focus:border-primary outline-none"
                />
                <p className="text-xs text-on-surface-variant mt-2">
                  This temporary TCP port is created on the Nucleus before being exported to the laptop helper.
                </p>
              </div>
              <div className="rounded-xl border border-amber-400/20 bg-amber-400/10 px-4 py-3">
                <div className="flex items-start gap-3">
                  <span className="material-symbols-outlined text-amber-300 text-base mt-0.5">
                    warning
                  </span>
                  <div>
                    <p className="text-sm font-semibold text-amber-200">
                      Node-RED serial communication will be interrupted
                    </p>
                    <p className="text-xs text-amber-100/90 mt-1 leading-relaxed">
                      MBUSD uses <span className="font-technical">{bridgeForm.serial_port || fallbackModbusSerialPort}</span>.
                      While this serial bridge is active, Node-RED Modbus communication that shares the same serial port will be temporarily unavailable.
                    </p>
                  </div>
                </div>
              </div>
              <label className="flex items-start gap-3 rounded-xl bg-surface-container-high px-4 py-3 cursor-pointer">
                <input
                  type="checkbox"
                  checked={bridgeForm.acknowledge_warning}
                  onChange={(e) => setBridgeForm((current) => ({ ...current, acknowledge_warning: e.target.checked }))}
                  className="mt-1"
                />
                <span className="text-sm text-on-surface-variant">
                  I understand that enabling MBUSD on <span className="font-technical">{bridgeForm.serial_port || fallbackModbusSerialPort}</span> temporarily interrupts Node-RED Modbus serial communication.
                </span>
              </label>
              {bridgeError && (
                <div className="rounded-xl border border-error/30 bg-error/10 px-4 py-3 text-sm text-error">
                  {bridgeError}
                </div>
              )}
            </div>
            <div className="flex gap-3">
              <button
                onClick={() => setBridgeModal(false)}
                disabled={creatingBridge}
                className="flex-1 py-3 rounded-xl text-sm font-medium text-on-surface-variant hover:bg-surface-container-high transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateBridge}
                disabled={creatingBridge}
                className="flex-1 gradient-primary text-on-primary font-bold py-3 rounded-xl text-sm hover:shadow-primary transition-all disabled:opacity-60"
              >
                {creatingBridge ? "Starting..." : "Start Bridge + Export"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
