"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { logout } from "@/lib/api";

interface TopBarProps {
  userName?: string;
  tenantName?: string;
}

export function TopBar({ userName = "Admin Console", tenantName = "Global Operations" }: TopBarProps) {
  const router = useRouter();

  const handleLogout = async () => {
    try {
      await logout();
    } catch {
      // ignore
    } finally {
      localStorage.removeItem("nucleus_token");
      localStorage.removeItem("nucleus_user");
      router.push("/login");
    }
  };

  return (
    <header className="flex justify-between items-center w-full px-6 py-3 bg-background sticky top-0 z-40 border-b border-outline-variant/5">
      <div className="flex items-center space-x-4">
        <h1 className="text-2xl font-headline font-bold tracking-tight text-primary">
          Nucleus
        </h1>
        <div className="h-6 w-px bg-outline-variant/30 mx-2" />
        <button className="flex items-center space-x-2 px-3 py-1.5 rounded-lg bg-surface-container-low hover:bg-surface-container-high transition-colors text-on-surface-variant text-sm font-medium">
          <span>{tenantName}</span>
          <span className="material-symbols-outlined text-lg">expand_more</span>
        </button>
      </div>

      <div className="flex items-center space-x-4">
        <div className="flex items-center space-x-1">
          <button className="p-2 text-slate-400 hover:bg-surface-container-high rounded-full transition-colors">
            <span className="material-symbols-outlined">notifications</span>
          </button>
          <Link
            href="/settings"
            className="p-2 text-slate-400 hover:bg-surface-container-high rounded-full transition-colors"
          >
            <span className="material-symbols-outlined">settings</span>
          </Link>
        </div>

        <button
          onClick={handleLogout}
          className="flex items-center space-x-3 bg-surface-container-low pl-1 pr-4 py-1 rounded-full border border-outline-variant/10 hover:bg-surface-container-high transition-colors"
          title="Click to logout"
        >
          <div className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center border border-primary/20">
            <span className="material-symbols-outlined text-primary text-base">
              person
            </span>
          </div>
          <span className="text-sm font-medium text-on-surface">{userName}</span>
        </button>
      </div>
    </header>
  );
}
