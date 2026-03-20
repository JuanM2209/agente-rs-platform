"use client";

import { useState, useEffect } from "react";
import { StatusOrb } from "@/components/ui/StatusOrb";

// Mock data for admin overview (production would come from API)
const mockStats = {
  active_sessions: 142,
  expiring_sessions: 8,
  online_devices: 1847,
  offline_devices: 23,
  total_tenants: 12,
  relay_latency_ms: 12,
  tunnel_capacity: 85,
  auth_gateway: "stable" as const,
};

const mockTenants = [
  { name: "Alpha Industries", slug: "alpha", devices: 450, sessions: 67, users: 24, status: "healthy" },
  { name: "Beta Controls", slug: "beta", devices: 312, sessions: 43, users: 18, status: "healthy" },
  { name: "Gamma Field Ops", slug: "gamma", devices: 189, sessions: 21, users: 11, status: "warning" },
  { name: "Delta Process", slug: "delta", devices: 234, sessions: 11, users: 15, status: "healthy" },
];

export default function AdminPage() {
  return (
    <div className="max-w-7xl mx-auto px-6 py-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="font-headline text-3xl font-bold text-on-surface">Admin Overview</h1>
          <p className="text-on-surface-variant text-sm mt-1">
            System-wide health, tenant status, and operations monitoring
          </p>
        </div>
        <span className="text-xs font-technical text-on-surface-variant uppercase bg-surface-container-high px-3 py-1.5 rounded-lg">
          Admin Only
        </span>
      </div>

      {/* System health cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
        {[
          { icon: "cloud_done", label: "Active Sessions", value: mockStats.active_sessions, color: "text-tertiary", bg: "bg-tertiary/10", trend: "+12%" },
          { icon: "timer_off", label: "Expiring Soon", value: mockStats.expiring_sessions, color: "text-error", bg: "bg-error/10", trend: null },
          { icon: "devices", label: "Online Devices", value: mockStats.online_devices, color: "text-primary", bg: "bg-primary/10", trend: null },
          { icon: "device_unknown", label: "Offline Devices", value: mockStats.offline_devices, color: "text-amber-400", bg: "bg-amber-400/10", trend: null },
        ].map((stat) => (
          <div key={stat.label} className="bg-surface-container-high rounded-xl p-6 relative overflow-hidden hover:bg-surface-bright transition-colors">
            <div className="flex justify-between items-start mb-4">
              <div className={`p-2 ${stat.bg} rounded-lg`}>
                <span className={`material-symbols-outlined ${stat.color}`}>{stat.icon}</span>
              </div>
              {stat.trend && (
                <span className="text-[10px] text-tertiary bg-tertiary/10 px-2 py-0.5 rounded font-technical">
                  {stat.trend}
                </span>
              )}
            </div>
            <p className="text-3xl font-headline font-bold text-on-surface">{stat.value.toLocaleString()}</p>
            <p className="text-sm text-on-surface-variant font-medium mt-1">{stat.label}</p>
          </div>
        ))}
      </div>

      {/* System health panel */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-8">
        <div className="lg:col-span-1 bg-surface-container-low border border-outline-variant/10 rounded-xl p-6">
          <h3 className="font-headline font-bold text-on-surface flex items-center gap-2 mb-6">
            <span className="material-symbols-outlined text-primary text-lg">health_and_safety</span>
            Infrastructure Health
          </h3>
          <div className="space-y-5">
            {[
              { label: "Auth Gateway", value: "STABLE", color: "text-tertiary", dot: "bg-tertiary" },
              { label: "WebSocket Hub", value: "ACTIVE", color: "text-tertiary", dot: "bg-tertiary" },
              { label: "PostgreSQL", value: "HEALTHY", color: "text-tertiary", dot: "bg-tertiary" },
              { label: "Redis Cache", value: "HEALTHY", color: "text-tertiary", dot: "bg-tertiary" },
              { label: "Cloudflare Tunnel", value: "UP", color: "text-tertiary", dot: "bg-tertiary" },
            ].map((item) => (
              <div key={item.label} className="flex items-center justify-between">
                <span className="text-sm text-on-surface-variant">{item.label}</span>
                <div className="flex items-center gap-2">
                  <span className={`w-1.5 h-1.5 rounded-full ${item.dot}`} />
                  <span className={`text-xs font-technical ${item.color}`}>{item.value}</span>
                </div>
              </div>
            ))}

            <div className="pt-2 border-t border-outline-variant/10">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-on-surface-variant">Relay Latency</span>
                <span className="text-xs font-technical text-on-surface">{mockStats.relay_latency_ms}ms</span>
              </div>
              <div className="w-full bg-surface-container-highest h-1.5 rounded-full overflow-hidden">
                <div
                  className="bg-gradient-to-r from-tertiary to-primary h-full rounded-full"
                  style={{ width: `${Math.min(mockStats.relay_latency_ms / 100 * 100, 100)}%` }}
                />
              </div>
            </div>

            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-on-surface-variant">Tunnel Capacity</span>
                <span className="text-xs font-technical text-on-surface">{mockStats.tunnel_capacity}%</span>
              </div>
              <div className="w-full bg-surface-container-highest h-1.5 rounded-full overflow-hidden">
                <div
                  className="bg-gradient-to-r from-primary to-primary-container h-full rounded-full"
                  style={{ width: `${mockStats.tunnel_capacity}%` }}
                />
              </div>
            </div>
          </div>
        </div>

        {/* Tenant overview */}
        <div className="lg:col-span-2 bg-surface-container-low border border-outline-variant/10 rounded-xl p-6">
          <div className="flex items-center justify-between mb-6">
            <h3 className="font-headline font-bold text-on-surface flex items-center gap-2">
              <span className="material-symbols-outlined text-primary text-lg">business</span>
              Tenant Summary
            </h3>
            <span className="text-xs font-technical text-on-surface-variant">
              {mockStats.total_tenants} tenants
            </span>
          </div>

          <div className="space-y-3">
            {mockTenants.map((tenant) => (
              <div
                key={tenant.slug}
                className="flex items-center justify-between p-4 rounded-xl bg-surface-container hover:bg-surface-container-high transition-colors"
              >
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-lg bg-surface-container-high flex items-center justify-center">
                    <span className="font-headline font-bold text-primary text-sm">
                      {tenant.name.charAt(0)}
                    </span>
                  </div>
                  <div>
                    <p className="text-sm font-medium text-on-surface">{tenant.name}</p>
                    <p className="text-xs text-on-surface-variant font-technical">{tenant.slug}</p>
                  </div>
                </div>

                <div className="flex items-center gap-6">
                  <div className="text-right">
                    <p className="text-xs text-on-surface-variant">Devices</p>
                    <p className="text-sm font-technical text-on-surface">{tenant.devices}</p>
                  </div>
                  <div className="text-right">
                    <p className="text-xs text-on-surface-variant">Sessions</p>
                    <p className="text-sm font-technical text-on-surface">{tenant.sessions}</p>
                  </div>
                  <div className="text-right">
                    <p className="text-xs text-on-surface-variant">Users</p>
                    <p className="text-sm font-technical text-on-surface">{tenant.users}</p>
                  </div>
                  <StatusOrb
                    status={tenant.status === "healthy" ? "online" : "maintenance"}
                    size="sm"
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Recent audit log */}
      <div className="bg-surface-container-low rounded-xl p-6">
        <div className="flex items-center justify-between mb-6">
          <h3 className="font-headline font-bold text-on-surface flex items-center gap-2">
            <span className="material-symbols-outlined text-primary text-lg">policy</span>
            Audit Stream
          </h3>
          <a href="/history" className="text-sm text-primary hover:underline">View full log</a>
        </div>
        <div className="space-y-3">
          {[
            { user: "admin@alpha.com", action: "SESSION_CREATED", device: "N-1044", time: "2m ago", ip: "10.0.1.5" },
            { user: "operator@alpha.com", action: "SESSION_STOPPED", device: "N-1001", time: "7m ago", ip: "10.0.1.12" },
            { user: "system", action: "SESSION_EXPIRED", device: "N-1003", time: "18m ago", ip: "—" },
            { user: "admin@beta.com", action: "DEVICE_SCANNED", device: "N-2001", time: "45m ago", ip: "10.0.2.5" },
            { user: "system", action: "AGENT_CONNECTED", device: "N-1004", time: "1h ago", ip: "—" },
          ].map((log, i) => (
            <div key={i} className="flex items-center gap-4 p-3 rounded-xl hover:bg-surface-container transition-colors text-sm">
              <span className="material-symbols-outlined text-outline text-base">receipt_long</span>
              <span className="font-technical text-xs text-on-surface-variant w-40 truncate">
                {log.user}
              </span>
              <span className={`font-technical text-xs px-2 py-0.5 rounded ${
                log.action.includes("EXPIRED") || log.action.includes("STOPPED")
                  ? "text-amber-400 bg-amber-400/10"
                  : log.action.includes("CONNECTED")
                    ? "text-tertiary bg-tertiary/10"
                    : "text-primary bg-primary/10"
              }`}>
                {log.action}
              </span>
              <span className="font-technical text-xs text-primary bg-primary/10 px-1.5 py-0.5 rounded">
                {log.device}
              </span>
              <span className="text-xs text-outline ml-auto font-technical">{log.ip}</span>
              <span className="text-xs text-on-surface-variant">{log.time}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
