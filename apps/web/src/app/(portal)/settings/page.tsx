"use client";

import { useEffect, useState } from "react";
import type { User } from "@/types";
import {
  DEFAULT_SESSION_HOURS,
  MAX_SESSION_HOURS,
  MIN_SESSION_HOURS,
  clampSessionHours,
  formatHoursLabel,
  getPortalPreferences,
  savePortalPreferences,
} from "@/lib/portal-settings";

export default function SettingsPage() {
  const [user, setUser] = useState<User | null>(null);
  const [defaultSessionHours, setDefaultSessionHours] = useState(DEFAULT_SESSION_HOURS);
  const [savedMessage, setSavedMessage] = useState("");

  useEffect(() => {
    const storedUser = window.localStorage.getItem("nucleus_user");
    if (storedUser) {
      try {
        setUser(JSON.parse(storedUser) as User);
      } catch {
        // ignore malformed local storage
      }
    }

    const preferences = getPortalPreferences();
    setDefaultSessionHours(preferences.defaultSessionHours);
  }, []);

  const isAdmin = user?.role === "admin";

  const persistPreferences = (hours: number) => {
    const saved = savePortalPreferences({ defaultSessionHours: hours });
    setDefaultSessionHours(saved.defaultSessionHours);
    setSavedMessage(`Saved. New sessions now default to ${formatHoursLabel(saved.defaultSessionHours)}.`);
    window.setTimeout(() => setSavedMessage(""), 3000);
  };

  return (
    <div className="p-8 max-w-5xl mx-auto">
      <div className="mb-8">
        <h1 className="font-headline text-3xl font-bold text-on-surface mb-2">Settings</h1>
        <p className="text-on-surface-variant text-sm">
          Tune operator defaults, session behavior, and customer-facing workflow preferences.
        </p>
      </div>

      <div className="grid gap-6 lg:grid-cols-[1.3fr_0.9fr]">
        <section className="bg-surface-container-high rounded-2xl p-6 border border-outline-variant/10">
          <div className="flex items-start justify-between gap-4 mb-6">
            <div>
              <p className="text-xs font-technical uppercase tracking-[0.25em] text-primary mb-2">
                Session Policy
              </p>
              <h2 className="font-headline text-2xl font-bold text-on-surface">
                Default Session Duration
              </h2>
              <p className="text-sm text-on-surface-variant mt-2 max-w-2xl">
                New web sessions and exported ports now start with an 8-hour minimum window.
                Admins can raise the default here for customers who need longer maintenance or support work.
              </p>
            </div>
            <div className="rounded-xl bg-surface-container-highest px-4 py-3 min-w-[160px] text-right">
              <p className="text-[11px] uppercase tracking-[0.2em] text-outline mb-1">Current Default</p>
              <p className="font-headline text-2xl text-on-surface">
                {defaultSessionHours}h
              </p>
            </div>
          </div>

          {isAdmin ? (
            <div className="space-y-6">
              <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                {[8, 12, 16, 24].map((hours) => (
                  <button
                    key={hours}
                    onClick={() => persistPreferences(hours)}
                    className={[
                      "rounded-xl px-4 py-4 text-left transition-all border",
                      defaultSessionHours === hours
                        ? "border-primary bg-primary/10 text-on-surface shadow-primary"
                        : "border-outline-variant/10 bg-surface-container-highest hover:bg-surface-container-low text-on-surface-variant",
                    ].join(" ")}
                  >
                    <p className="font-headline text-xl font-bold">{hours}h</p>
                    <p className="text-xs mt-1">Recommended preset</p>
                  </button>
                ))}
              </div>

              <div className="rounded-2xl bg-surface-container-highest p-5 border border-outline-variant/10">
                <div className="flex flex-col md:flex-row md:items-end gap-4">
                  <div className="flex-1">
                    <label className="block text-sm text-on-surface-variant mb-2">
                      Custom default in hours
                    </label>
                    <input
                      type="number"
                      min={MIN_SESSION_HOURS}
                      max={MAX_SESSION_HOURS}
                      value={defaultSessionHours}
                      onChange={(e) => setDefaultSessionHours(clampSessionHours(Number(e.target.value)))}
                      className="w-full bg-surface-container-low rounded-xl px-4 py-3 text-sm text-on-surface border border-outline-variant/10 focus:border-primary outline-none"
                    />
                    <p className="text-xs text-on-surface-variant mt-2">
                      Allowed range: {MIN_SESSION_HOURS} to {MAX_SESSION_HOURS} hours.
                    </p>
                  </div>
                  <button
                    onClick={() => persistPreferences(defaultSessionHours)}
                    className="gradient-primary text-on-primary font-bold px-6 py-3 rounded-xl text-sm hover:shadow-primary transition-all"
                  >
                    Save Admin Default
                  </button>
                </div>
              </div>

              {savedMessage && (
                <div className="rounded-xl border border-tertiary/30 bg-tertiary/10 px-4 py-3 text-sm text-tertiary">
                  {savedMessage}
                </div>
              )}
            </div>
          ) : (
            <div className="rounded-2xl border border-outline-variant/10 bg-surface-container-highest px-5 py-5">
              <p className="text-sm text-on-surface">
                Your admin controls the default session window for customers.
              </p>
              <p className="text-xs text-on-surface-variant mt-2">
                Current default: {formatHoursLabel(defaultSessionHours)}.
              </p>
            </div>
          )}
        </section>

        <section className="space-y-6">
          <div className="bg-surface-container-high rounded-2xl p-6 border border-outline-variant/10">
            <p className="text-xs font-technical uppercase tracking-[0.25em] text-primary mb-2">
              Operator Experience
            </p>
            <h2 className="font-headline text-xl font-bold text-on-surface mb-3">
              What Changed
            </h2>
            <ul className="space-y-3 text-sm text-on-surface-variant">
              <li>Port actions now separate into clear choices: Open Web Port or Export to Your Laptop.</li>
              <li>Export sessions can target a custom localhost port such as `127.0.0.1:1889`.</li>
              <li>Serial Modbus flows now keep the Node-RED interruption warning front and center.</li>
            </ul>
          </div>

          <div className="bg-surface-container-high rounded-2xl p-6 border border-outline-variant/10">
            <p className="text-xs font-technical uppercase tracking-[0.25em] text-primary mb-2">
              Signed In
            </p>
            <h2 className="font-headline text-xl font-bold text-on-surface mb-4">
              Access Context
            </h2>
            <div className="space-y-3 text-sm">
              <div className="flex items-center justify-between rounded-xl bg-surface-container-highest px-4 py-3">
                <span className="text-on-surface-variant">User</span>
                <span className="text-on-surface font-medium">{user?.display_name || "Unknown"}</span>
              </div>
              <div className="flex items-center justify-between rounded-xl bg-surface-container-highest px-4 py-3">
                <span className="text-on-surface-variant">Role</span>
                <span className="text-on-surface font-technical uppercase">{user?.role || "unknown"}</span>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
}
