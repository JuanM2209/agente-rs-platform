"use client";

import { useState, useEffect } from "react";
import { getMyExportHistory } from "@/lib/api";
import type { ExportHistory } from "@/types";
import { format, formatDistanceToNow } from "date-fns";
import { clsx } from "clsx";

const stopReasonLabel: Record<string, { label: string; color: string }> = {
  user_stopped: { label: "User Stopped", color: "text-on-surface-variant" },
  ttl_expired: { label: "TTL Expired", color: "text-amber-400" },
  idle_timeout: { label: "Idle Timeout", color: "text-outline" },
  error: { label: "Error", color: "text-error" },
  agent_disconnect: { label: "Agent Disconnect", color: "text-error" },
};

const deliveryModeIcon: Record<string, string> = {
  web: "language",
  export: "output",
};

export default function HistoryPage() {
  const [history, setHistory] = useState<ExportHistory[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<"all" | "web" | "export">("all");

  useEffect(() => {
    getMyExportHistory()
      .then(setHistory)
      .catch(() => setHistory([]))
      .finally(() => setLoading(false));
  }, []);

  const filtered = history.filter(
    (h) => filter === "all" || h.delivery_mode === filter,
  );

  return (
    <div className="max-w-7xl mx-auto px-6 py-8">
      {/* Header */}
      <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4 mb-8">
        <div>
          <h1 className="font-headline text-3xl font-bold text-on-surface">Export History</h1>
          <p className="text-on-surface-variant text-sm mt-1">
            Complete audit trail of all sessions and exports
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button className="flex items-center gap-2 bg-surface-container-high hover:bg-surface-bright px-4 py-2.5 rounded-xl text-sm font-medium text-on-surface-variant transition-colors">
            <span className="material-symbols-outlined text-base">download</span>
            Export CSV
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-2 mb-6 bg-surface-container-low rounded-xl p-1 w-fit">
        {(["all", "web", "export"] as const).map((f) => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={clsx(
              "px-5 py-2.5 rounded-lg text-sm font-medium capitalize transition-colors",
              filter === f
                ? "bg-surface-container-high text-on-surface"
                : "text-on-surface-variant hover:text-on-surface",
            )}
          >
            {f === "all" ? "All Sessions" : f === "web" ? "Web Access" : "Port Exports"}
          </button>
        ))}
      </div>

      {/* Table */}
      {loading ? (
        <div className="flex items-center justify-center py-20">
          <div className="w-10 h-10 rounded-full border-2 border-primary border-t-transparent animate-spin" />
        </div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-20 bg-surface-container-low rounded-xl">
          <span className="material-symbols-outlined text-outline text-4xl">history_toggle_off</span>
          <p className="font-headline font-bold text-on-surface mt-4 mb-2">No History Found</p>
          <p className="text-sm text-on-surface-variant">
            No sessions match the selected filter.
          </p>
        </div>
      ) : (
        <div className="bg-surface-container-low rounded-xl overflow-hidden">
          {/* Table header */}
          <div className="grid grid-cols-7 gap-4 px-6 py-4 border-b border-outline-variant/10 text-xs font-medium text-on-surface-variant uppercase tracking-wider">
            <div>Device</div>
            <div>Endpoint</div>
            <div>User</div>
            <div>Mode</div>
            <div>Started</div>
            <div>Duration</div>
            <div>Stop Reason</div>
          </div>

          {/* Table rows */}
          <div className="divide-y divide-outline-variant/5">
            {filtered.map((record) => {
              const reason = record.stop_reason
                ? stopReasonLabel[record.stop_reason] || { label: record.stop_reason, color: "text-outline" }
                : null;
              const durationMin = record.duration_seconds
                ? Math.round(record.duration_seconds / 60)
                : null;

              return (
                <div
                  key={record.id}
                  className="grid grid-cols-7 gap-4 px-6 py-4 hover:bg-surface-container transition-colors text-sm"
                >
                  <div>
                    <span className="font-technical text-primary text-xs bg-primary/10 px-1.5 py-0.5 rounded">
                      {record.device?.device_id || "—"}
                    </span>
                    {record.device?.display_name && (
                      <p className="text-xs text-on-surface-variant mt-1 truncate">
                        {record.device.display_name}
                      </p>
                    )}
                  </div>

                  <div>
                    <p className="text-on-surface">
                      {record.endpoint?.label || "—"}
                    </p>
                    {record.endpoint?.port && (
                      <p className="font-technical text-xs text-outline mt-0.5">
                        :{record.endpoint.port}
                      </p>
                    )}
                  </div>

                  <div className="text-on-surface-variant truncate">
                    {record.user?.display_name || "—"}
                  </div>

                  <div>
                    <div className="flex items-center gap-1.5">
                      <span className="material-symbols-outlined text-outline text-base">
                        {deliveryModeIcon[record.delivery_mode || "export"]}
                      </span>
                      <span className="text-on-surface-variant capitalize">
                        {record.delivery_mode || "—"}
                      </span>
                    </div>
                    {record.local_bind_port && (
                      <p className="font-technical text-xs text-outline mt-0.5">
                        ::{record.local_bind_port}
                      </p>
                    )}
                  </div>

                  <div>
                    <p className="text-on-surface">
                      {format(new Date(record.started_at), "MMM d, HH:mm")}
                    </p>
                    <p className="text-xs text-on-surface-variant mt-0.5">
                      {formatDistanceToNow(new Date(record.started_at), { addSuffix: true })}
                    </p>
                  </div>

                  <div className="font-technical text-on-surface-variant">
                    {durationMin != null ? `${durationMin}m` : "—"}
                  </div>

                  <div>
                    {reason ? (
                      <span className={clsx("text-xs font-medium", reason.color)}>
                        {reason.label}
                      </span>
                    ) : (
                      <span className="text-xs text-on-surface-variant">—</span>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
