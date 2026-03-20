import { clsx } from "clsx";
import type { DeviceStatus } from "@/types";

interface StatusOrbProps {
  status: DeviceStatus | "active" | "idle";
  size?: "sm" | "md";
  showLabel?: boolean;
  animate?: boolean;
}

const statusConfig = {
  online: {
    color: "bg-tertiary",
    glow: "shadow-[0_0_8px_2px_rgba(78,222,163,0.4)]",
    label: "Online",
  },
  active: {
    color: "bg-tertiary",
    glow: "shadow-[0_0_8px_2px_rgba(78,222,163,0.4)]",
    label: "Active",
  },
  offline: {
    color: "bg-error",
    glow: "shadow-[0_0_8px_2px_rgba(255,180,171,0.4)]",
    label: "Offline",
  },
  unknown: {
    color: "bg-outline",
    glow: "",
    label: "Unknown",
  },
  maintenance: {
    color: "bg-amber-400",
    glow: "shadow-[0_0_8px_2px_rgba(251,191,36,0.4)]",
    label: "Maintenance",
  },
  idle: {
    color: "bg-outline",
    glow: "",
    label: "Idle",
  },
};

export function StatusOrb({
  status,
  size = "sm",
  showLabel = false,
  animate = true,
}: StatusOrbProps) {
  const cfg = statusConfig[status] || statusConfig.unknown;
  const isAlive = ["online", "active"].includes(status);

  return (
    <div className="flex items-center gap-2">
      <span
        className={clsx(
          "rounded-full flex-shrink-0",
          cfg.color,
          cfg.glow,
          size === "sm" ? "w-2 h-2" : "w-3 h-3",
          animate && isAlive ? "pulse-glow" : "",
        )}
      />
      {showLabel && (
        <span
          className={clsx(
            "font-technical uppercase tracking-wider",
            size === "sm" ? "text-[10px]" : "text-xs",
            status === "online" || status === "active"
              ? "text-tertiary"
              : status === "offline"
                ? "text-error"
                : "text-outline",
          )}
        >
          {cfg.label}
        </span>
      )}
    </div>
  );
}
