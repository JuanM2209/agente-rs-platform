"use client";

export const MIN_SESSION_HOURS = 8;
export const MAX_SESSION_HOURS = 24;
export const DEFAULT_SESSION_HOURS = 8;

const STORAGE_KEY = "nucleus_portal_preferences_v1";

export interface PortalPreferences {
  defaultSessionHours: number;
}

function defaultPreferences(): PortalPreferences {
  return {
    defaultSessionHours: DEFAULT_SESSION_HOURS,
  };
}

export function clampSessionHours(value: number): number {
  if (!Number.isFinite(value)) return DEFAULT_SESSION_HOURS;
  return Math.min(MAX_SESSION_HOURS, Math.max(MIN_SESSION_HOURS, Math.round(value)));
}

export function getPortalPreferences(): PortalPreferences {
  if (typeof window === "undefined") {
    return defaultPreferences();
  }

  const raw = window.localStorage.getItem(STORAGE_KEY);
  if (!raw) return defaultPreferences();

  try {
    const parsed = JSON.parse(raw) as Partial<PortalPreferences>;
    return {
      defaultSessionHours: clampSessionHours(parsed.defaultSessionHours ?? DEFAULT_SESSION_HOURS),
    };
  } catch {
    return defaultPreferences();
  }
}

export function savePortalPreferences(preferences: PortalPreferences): PortalPreferences {
  const normalized: PortalPreferences = {
    defaultSessionHours: clampSessionHours(preferences.defaultSessionHours),
  };

  if (typeof window !== "undefined") {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(normalized));
  }

  return normalized;
}

export function sessionHoursToSeconds(hours: number): number {
  return clampSessionHours(hours) * 60 * 60;
}

export function formatHoursLabel(hours: number): string {
  const normalized = clampSessionHours(hours);
  return `${normalized} hour${normalized === 1 ? "" : "s"}`;
}
