"use client";

import { useState, useEffect } from "react";
import { getActiveSessions, stopSession } from "@/lib/api";
import { StatusOrb } from "@/components/ui/StatusOrb";
import type { Session } from "@/types";
import { formatDistanceToNow, differenceInSeconds } from "date-fns";
import { clsx } from "clsx";

function TTLBadge({ expiresAt }: { expiresAt: string }) {
  const [secondsLeft, setSecondsLeft] = useState(
    differenceInSeconds(new Date(expiresAt), new Date()),
  );

  useEffect(() => {
    const interval = setInterval(() => {
      setSecondsLeft(differenceInSeconds(new Date(expiresAt), new Date()));
    }, 1000);
    return () => clearInterval(interval);
  }, [expiresAt]);

  if (secondsLeft <= 0) return (
    <span className="font-technical text-xs text-error bg-error/10 px-2 py-0.5 rounded">EXPIRED</span>
  );

  const minutes = Math.floor(secondsLeft / 60);
  const hours = Math.floor(minutes / 60);
  const isUrgent = secondsLeft < 300; // < 5 min

  const display = hours > 0
    ? `${hours}h ${minutes % 60}m`
    : `${minutes}m ${secondsLeft % 60}s`;

  return (
    <span className={clsx(
      "font-technical text-xs px-2 py-0.5 rounded",
      isUrgent
        ? "text-error bg-error/10 animate-pulse"
        : "text-on-surface-variant bg-surface-container-highest",
    )}>
      {display}
    </span>
  );
}

const telemetryMeta: Record<string, { label: string; dot: string; text: string; bg: string }> = {
  pending: {
    label: "Pending",
    dot: "bg-outline",
    text: "text-on-surface-variant",
    bg: "bg-surface-container-highest",
  },
  reachable: {
    label: "Reachable",
    dot: "bg-tertiary",
    text: "text-tertiary",
    bg: "bg-tertiary/10",
  },
  degraded: {
    label: "Degraded",
    dot: "bg-amber-400",
    text: "text-amber-400",
    bg: "bg-amber-400/10",
  },
  unreachable: {
    label: "Unreachable",
    dot: "bg-error",
    text: "text-error",
    bg: "bg-error/10",
  },
  stopped: {
    label: "Stopped",
    dot: "bg-outline",
    text: "text-outline",
    bg: "bg-surface-container-highest",
  },
};

function ExportTelemetry({ session }: { session: Session }) {
  if (session.delivery_mode !== "export") return null;

  const telemetry = session.telemetry;
  const meta = telemetryMeta[telemetry?.connection_status || "pending"] || telemetryMeta.pending;
  const checkedAt = telemetry?.last_checked_at ? new Date(telemetry.last_checked_at) : null;
  const stale = checkedAt ? Date.now() - checkedAt.getTime() > 60_000 : false;

  return (
    <div className="mt-3 grid grid-cols-1 md:grid-cols-3 gap-2">
      <div className="bg-surface-container-highest rounded-lg px-3 py-2">
        <p className="text-[10px] font-medium text-on-surface-variant uppercase tracking-wider mb-1">
          Port Status
        </p>
        <div className="flex items-center gap-2">
          <span className={clsx("w-2 h-2 rounded-full", meta.dot, stale && "animate-pulse")} />
          <span className={clsx("text-xs font-medium", meta.text)}>
            {stale ? "Stale Telemetry" : meta.label}
          </span>
        </div>
      </div>

      <div className="bg-surface-container-highest rounded-lg px-3 py-2">
        <p className="text-[10px] font-medium text-on-surface-variant uppercase tracking-wider mb-1">
          Latency
        </p>
        <p className="font-technical text-sm text-on-surface">
          {typeof telemetry?.latency_ms === "number" ? `${telemetry.latency_ms} ms` : "Waiting..."}
        </p>
      </div>

      <div className="bg-surface-container-highest rounded-lg px-3 py-2">
        <p className="text-[10px] font-medium text-on-surface-variant uppercase tracking-wider mb-1">
          Last Check
        </p>
        <p className="text-xs text-on-surface-variant">
          {checkedAt ? formatDistanceToNow(checkedAt, { addSuffix: true }) : "Not reported yet"}
        </p>
      </div>

      {(session.remote_host || telemetry?.last_error) && (
        <div className="md:col-span-3 flex flex-wrap items-center gap-3 text-xs text-on-surface-variant">
          {session.remote_host && (
            <span className="font-technical bg-surface-container-highest px-2.5 py-1 rounded">
              Remote {session.remote_host}:{session.remote_port || session.endpoint?.port || "?"}
            </span>
          )}
          {telemetry?.last_error && (
            <span className="text-error">
              Last error: {telemetry.last_error}
            </span>
          )}
        </div>
      )}
    </div>
  );
}

export default function SessionsPage() {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [stopping, setStopping] = useState<string | null>(null);

  useEffect(() => {
    loadSessions();
    const interval = setInterval(loadSessions, 30000);
    return () => clearInterval(interval);
  }, []);

  const loadSessions = async () => {
    try {
      const data = await getActiveSessions();
      setSessions(data);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  };

  const handleStop = async (sessionId: string) => {
    setStopping(sessionId);
    try {
      await stopSession(sessionId);
      setSessions((prev) => prev.filter((s) => s.id !== sessionId));
    } catch {
      // ignore
    } finally {
      setStopping(null);
    }
  };

  return (
    <div className="max-w-7xl mx-auto px-6 py-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="font-headline text-3xl font-bold text-on-surface">Active Sessions</h1>
          <p className="text-on-surface-variant text-sm mt-1">
            All currently open remote sessions across your devices
          </p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2 bg-tertiary/10 px-3 py-1.5 rounded-lg">
            <span className="w-2 h-2 rounded-full bg-tertiary pulse-glow" />
            <span className="text-xs font-technical text-tertiary">
              {sessions.length} active
            </span>
          </div>
          <button
            onClick={loadSessions}
            className="p-2 hover:bg-surface-container-high rounded-xl transition-colors text-on-surface-variant"
          >
            <span className="material-symbols-outlined text-base">refresh</span>
          </button>
        </div>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-20">
          <div className="w-10 h-10 rounded-full border-2 border-primary border-t-transparent animate-spin" />
        </div>
      ) : sessions.length === 0 ? (
        <div className="text-center py-20 bg-surface-container-low rounded-xl">
          <span className="material-symbols-outlined text-outline text-4xl">sensors_off</span>
          <p className="font-headline font-bold text-on-surface mt-4 mb-2">No Active Sessions</p>
          <p className="text-sm text-on-surface-variant">
            Search for a device and start a session to see it here.
          </p>
          <a
            href="/dashboard"
            className="inline-block mt-4 gradient-primary text-on-primary font-bold px-6 py-3 rounded-xl text-sm"
          >
            Search Devices
          </a>
        </div>
      ) : (
        <div className="space-y-3">
          {sessions.map((session) => (
            <div
              key={session.id}
              className="bg-surface-container-high rounded-xl p-6 hover:bg-surface-bright transition-colors"
            >
              <div className="flex flex-col lg:flex-row lg:items-center gap-4 justify-between">
                <div className="flex items-start gap-4">
                  <div className="w-10 h-10 rounded-xl bg-surface-container-highest flex items-center justify-center flex-shrink-0">
                    <span className="material-symbols-outlined text-primary text-base">
                      {session.delivery_mode === "web" ? "language" : "output"}
                    </span>
                  </div>
                  <div>
                    <div className="flex items-center gap-3 mb-1">
                      <span className="font-technical text-sm text-primary bg-primary/10 px-2 py-0.5 rounded">
                        {session.device?.device_id || "Unknown Device"}
                      </span>
                      <StatusOrb status="active" showLabel />
                    </div>
                    <p className="text-sm text-on-surface font-medium">
                      {session.endpoint?.label || "Unknown Endpoint"} —{" "}
                      <span className="font-technical">:{session.endpoint?.port}</span>
                    </p>
                    <p className="text-xs text-outline font-technical mt-1">
                      Started {formatDistanceToNow(new Date(session.started_at))} ago •{" "}
                      {session.delivery_mode === "export"
                        ? `Local :${session.local_port}`
                        : "Web Access"}
                    </p>
                    <ExportTelemetry session={session} />
                  </div>
                </div>

                <div className="flex items-center gap-4 flex-shrink-0">
                  <div className="text-right">
                    <p className="text-xs text-on-surface-variant mb-1">TTL Remaining</p>
                    <TTLBadge expiresAt={session.expires_at} />
                  </div>

                  {session.tunnel_url && session.delivery_mode === "web" && (
                    <a
                      href={session.tunnel_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center gap-1.5 text-xs font-medium text-primary bg-primary/10 hover:bg-primary/20 px-3 py-2 rounded-lg transition-colors"
                    >
                      <span className="material-symbols-outlined text-base">open_in_new</span>
                      Open
                    </a>
                  )}

                  <button
                    onClick={() => handleStop(session.id)}
                    disabled={stopping === session.id}
                    className="flex items-center gap-1.5 text-xs font-medium text-error bg-error/10 hover:bg-error/20 px-3 py-2 rounded-lg transition-colors disabled:opacity-50"
                  >
                    <span className="material-symbols-outlined text-base">stop_circle</span>
                    {stopping === session.id ? "Stopping..." : "Stop"}
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
