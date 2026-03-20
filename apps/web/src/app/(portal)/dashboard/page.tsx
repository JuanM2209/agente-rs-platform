"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { StatusOrb } from "@/components/ui/StatusOrb";

const recentSearches = ["N-1001", "N-1044", "R-9921"];

const recentActivity = [
  {
    user: "Marcus Chen",
    action: "accessed",
    device: "N-1044",
    deviceColor: "text-primary bg-primary/10",
    time: "2 mins ago",
    type: "SSH Tunnel",
  },
  {
    user: "Sarah Vane",
    action: "closed session on",
    device: "R-9921",
    deviceColor: "text-primary bg-primary/10",
    time: "14 mins ago",
    type: "Web UI",
  },
  {
    user: "Auto-Provisioning",
    action: "registered new device",
    device: "Z-0012",
    deviceColor: "text-tertiary bg-tertiary/10",
    time: "1 hour ago",
    type: "System Process",
    isSystem: true,
  },
];

export default function DashboardPage() {
  const router = useRouter();
  const [searchValue, setSearchValue] = useState("");
  const [searching, setSearching] = useState(false);
  const [notFound, setNotFound] = useState(false);

  const handleSearch = async (deviceId?: string) => {
    const id = (deviceId || searchValue).trim().toUpperCase();
    if (!id) return;

    setSearching(true);
    setNotFound(false);

    // Navigate to device detail page
    router.push(`/devices/${id}`);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSearch();
    }
  };

  return (
    <div className="flex flex-col min-h-full">
      {/* Hero Search */}
      <section className="flex flex-col items-center justify-center pt-20 pb-14 px-6">
        <div className="max-w-3xl w-full text-center mb-10">
          <h2 className="font-headline text-5xl font-extrabold text-on-surface mb-4 tracking-tight">
            Access anything,{" "}
            <span className="text-gradient-primary">anywhere.</span>
          </h2>
          <p className="text-on-surface-variant text-lg">
            Centralized telemetry and secure remote orchestration.
          </p>
        </div>

        {/* Search box */}
        <div className="max-w-3xl w-full relative">
          <div className="absolute inset-y-0 left-6 flex items-center pointer-events-none">
            <span className="material-symbols-outlined text-primary text-2xl">
              search
            </span>
          </div>
          <input
            type="text"
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value.toUpperCase())}
            onKeyDown={handleKeyDown}
            placeholder="Enter Device ID (example: N-1001)"
            className="w-full h-20 bg-surface-container-high rounded-xl border-none pl-16 pr-36 text-xl font-body text-on-surface focus:ring-2 focus:ring-primary/40 placeholder:text-outline-variant transition-all shadow-2xl outline-none"
          />
          <div className="absolute inset-y-0 right-4 flex items-center">
            <button
              onClick={() => handleSearch()}
              disabled={!searchValue.trim() || searching}
              className="gradient-primary text-on-primary font-bold px-8 py-3 rounded-xl hover:shadow-primary transition-all active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed text-sm uppercase tracking-wide"
            >
              {searching ? "Searching..." : "CONNECT"}
            </button>
          </div>
        </div>

        {/* Recent searches */}
        {recentSearches.length > 0 && (
          <div className="mt-4 flex items-center gap-3 text-sm text-on-surface-variant">
            <span className="text-outline text-xs">Recent:</span>
            {recentSearches.map((id) => (
              <button
                key={id}
                onClick={() => handleSearch(id)}
                className="font-technical text-xs bg-surface-container-high hover:bg-surface-container-highest px-3 py-1 rounded-lg text-primary transition-colors"
              >
                {id}
              </button>
            ))}
          </div>
        )}

        <div className="mt-4 flex items-center space-x-2 text-sm text-on-surface-variant/60">
          <span className="material-symbols-outlined text-base">info</span>
          <p>Type a Device ID to begin your remote session.</p>
        </div>
      </section>

      {/* Bento grid */}
      <section className="grid grid-cols-12 gap-6 px-8 pb-8 max-w-[1600px] mx-auto w-full">
        {/* Status cards */}
        <div className="col-span-12 lg:col-span-8 grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="bg-surface-container-high p-6 rounded-xl relative overflow-hidden hover:bg-surface-bright transition-colors">
            <div className="flex justify-between items-start mb-4">
              <div className="p-2 bg-tertiary/10 rounded-lg">
                <span className="material-symbols-outlined text-tertiary">cloud_done</span>
              </div>
              <div className="w-3 h-3 bg-tertiary rounded-full pulse-glow" />
            </div>
            <p className="text-3xl font-headline font-bold text-on-surface">142</p>
            <p className="text-sm text-on-surface-variant font-medium">Online Sessions</p>
            <div className="absolute bottom-0 left-0 h-1 w-full bg-gradient-to-r from-tertiary to-transparent opacity-30" />
          </div>

          <div className="bg-surface-container-high p-6 rounded-xl relative overflow-hidden hover:bg-surface-bright transition-colors">
            <div className="flex justify-between items-start mb-4">
              <div className="p-2 bg-error/10 rounded-lg">
                <span className="material-symbols-outlined text-error">timer_off</span>
              </div>
              <span className="text-[10px] font-bold text-error uppercase tracking-tighter bg-error/10 px-2 py-0.5 rounded">
                Urgent
              </span>
            </div>
            <p className="text-3xl font-headline font-bold text-on-surface">08</p>
            <p className="text-sm text-on-surface-variant font-medium">Expiring Sessions</p>
            <div className="absolute bottom-0 left-0 h-1 w-full bg-gradient-to-r from-error to-transparent opacity-30" />
          </div>

          <div className="bg-surface-container-high p-6 rounded-xl relative overflow-hidden hover:bg-surface-bright transition-colors">
            <div className="flex justify-between items-start mb-4">
              <div className="p-2 bg-primary/10 rounded-lg">
                <span className="material-symbols-outlined text-primary">update</span>
              </div>
              <span className="text-[10px] font-headline text-primary-fixed-dim">99.2%</span>
            </div>
            <p className="text-3xl font-headline font-bold text-on-surface">Active</p>
            <p className="text-sm text-on-surface-variant font-medium">Device Freshness</p>
            <div className="absolute bottom-0 left-0 h-1 w-full bg-gradient-to-r from-primary to-transparent opacity-30" />
          </div>
        </div>

        {/* System health */}
        <div className="col-span-12 lg:col-span-4 bg-surface-container-low border border-outline-variant/10 rounded-xl p-6 flex flex-col">
          <div className="flex items-center justify-between mb-6">
            <h3 className="font-headline font-bold text-on-surface flex items-center gap-2">
              <span className="material-symbols-outlined text-primary text-lg">security</span>
              System Health
            </h3>
            <span className="text-[10px] font-technical text-on-surface-variant uppercase bg-surface-container-high px-2 py-1 rounded">
              Admin Only
            </span>
          </div>
          <div className="space-y-5">
            <div className="flex items-center justify-between">
              <span className="text-sm text-on-surface-variant">Auth Gateway</span>
              <div className="flex items-center gap-2">
                <span className="w-1.5 h-1.5 rounded-full bg-tertiary" />
                <span className="text-xs font-technical text-tertiary">STABLE</span>
              </div>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-on-surface-variant">Relay Latency</span>
              <span className="text-xs font-technical text-on-surface">12ms</span>
            </div>
            <div className="w-full bg-surface-container-highest h-1 rounded-full overflow-hidden">
              <div className="bg-primary h-full w-[85%]" />
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-on-surface-variant">Tunnel Capacity</span>
              <span className="text-xs font-technical text-on-surface">85% / 100%</span>
            </div>
          </div>
        </div>

        {/* Recent activity */}
        <div className="col-span-12 lg:col-span-7 bg-surface-container-high rounded-xl p-8">
          <div className="flex items-center justify-between mb-8">
            <h3 className="font-headline text-xl font-bold text-on-surface">Access Registry</h3>
            <a href="/history" className="text-sm text-primary font-medium hover:underline">
              View All Logs
            </a>
          </div>
          <div className="space-y-4">
            {recentActivity.map((item, i) => (
              <div
                key={i}
                className="flex items-center space-x-4 p-4 rounded-xl hover:bg-surface-bright transition-colors group cursor-pointer"
              >
                <div className="w-10 h-10 rounded-full bg-surface-container-highest border border-outline-variant/30 flex items-center justify-center flex-shrink-0">
                  {item.isSystem ? (
                    <span className="material-symbols-outlined text-tertiary text-base">robot</span>
                  ) : (
                    <span className="material-symbols-outlined text-primary text-base">person</span>
                  )}
                </div>
                <div className="flex-1">
                  <p className="text-sm font-semibold text-on-surface">
                    {item.user}{" "}
                    <span className="text-on-surface-variant font-normal">{item.action}</span>{" "}
                    <span className={`font-technical text-xs px-1.5 py-0.5 rounded ${item.deviceColor}`}>
                      {item.device}
                    </span>
                  </p>
                  <p className="text-xs text-outline font-technical mt-1">
                    {item.time} • {item.type}
                  </p>
                </div>
                <span className="material-symbols-outlined text-outline group-hover:text-on-surface transition-colors">
                  chevron_right
                </span>
              </div>
            ))}
          </div>
        </div>

        {/* Resource library */}
        <div className="col-span-12 lg:col-span-5 flex flex-col gap-6">
          <div className="bg-gradient-to-br from-surface-container-high to-surface-container p-8 rounded-xl border border-outline-variant/10">
            <h3 className="font-headline text-xl font-bold text-on-surface mb-6">Resource Library</h3>
            <div className="space-y-3">
              {[
                {
                  icon: "lan",
                  title: "How to open a web port",
                  desc: "Protocol mapping and NAT traversal basics for edge nodes.",
                },
                {
                  icon: "output",
                  title: "Exporting a program port",
                  desc: "Securely expose local services to the Nucleus relay network.",
                },
                {
                  icon: "verified_user",
                  title: "Configuring MFA for Tunnels",
                  desc: "Hardening your session security with biometric handshakes.",
                },
              ].map((item) => (
                <a
                  key={item.icon}
                  href="#"
                  className="group flex items-start space-x-4 bg-surface-container-lowest/50 p-4 rounded-xl hover:bg-primary/5 transition-colors"
                >
                  <div className="p-3 bg-surface-container-highest rounded-lg text-primary flex-shrink-0">
                    <span className="material-symbols-outlined">{item.icon}</span>
                  </div>
                  <div>
                    <p className="text-on-surface font-semibold group-hover:text-primary transition-colors text-sm">
                      {item.title}
                    </p>
                    <p className="text-xs text-on-surface-variant mt-1 leading-relaxed">
                      {item.desc}
                    </p>
                  </div>
                </a>
              ))}
            </div>
          </div>

          <div className="bg-primary/10 p-6 rounded-xl border border-primary/20 flex flex-col items-center text-center space-y-3">
            <div className="w-12 h-12 rounded-full bg-primary/20 flex items-center justify-center">
              <span className="material-symbols-outlined text-primary">bolt</span>
            </div>
            <p className="font-headline font-bold text-on-surface">Quick Onboarding</p>
            <p className="text-xs text-on-surface-variant">
              New to Nucleus? Follow our 5-minute setup guide to connect your first edge agent.
            </p>
            <button className="text-primary text-xs font-bold uppercase tracking-widest hover:text-primary-fixed-dim mt-2 transition-colors">
              Get Started
            </button>
          </div>
        </div>
      </section>
    </div>
  );
}
