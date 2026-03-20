"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { SideNav } from "@/components/layout/SideNav";
import { TopBar } from "@/components/layout/TopBar";
import type { User } from "@/types";

export default function PortalLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();

  useEffect(() => {
    const token = localStorage.getItem("nucleus_token");
    if (!token) {
      router.push("/login");
    }
  }, [router]);

  let user: User | null = null;
  if (typeof window !== "undefined") {
    const stored = localStorage.getItem("nucleus_user");
    if (stored) {
      try {
        user = JSON.parse(stored);
      } catch {
        // ignore
      }
    }
  }

  return (
    <div className="min-h-screen bg-background">
      <SideNav />
      <div className="ml-20">
        <TopBar
          userName={user?.display_name || "Console"}
          tenantName="Global Operations"
        />
        <main className="min-h-[calc(100vh-56px)]">{children}</main>
        <footer className="flex justify-between px-8 py-4 items-center border-t border-slate-800/50">
          <div className="text-slate-500 font-body text-[12px]">
            v1.0.0 © 2024 Nucleus Systems
          </div>
          <div className="flex space-x-6">
            <a className="text-[12px] text-slate-500 hover:text-white transition-colors" href="#">
              Support
            </a>
            <a className="text-[12px] text-slate-500 hover:text-white transition-colors" href="#">
              Security
            </a>
            <a className="text-[12px] text-slate-500 hover:text-white transition-colors" href="#">
              Legal
            </a>
          </div>
        </footer>
      </div>
    </div>
  );
}
