"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { clsx } from "clsx";

interface NavItem {
  href: string;
  icon: string;
  label: string;
}

const navItems: NavItem[] = [
  { href: "/dashboard", icon: "home", label: "Home" },
  { href: "/sessions", icon: "sensors", label: "Active Sessions" },
  { href: "/history", icon: "history", label: "Audit History" },
  { href: "/admin", icon: "admin_panel_settings", label: "Admin Overview" },
];

const bottomNavItems: NavItem[] = [
  { href: "/support", icon: "help", label: "Support" },
  { href: "/settings", icon: "settings", label: "Settings" },
];

export function SideNav() {
  const pathname = usePathname();

  return (
    <aside className="fixed left-0 top-0 h-full w-20 hover:w-64 transition-all duration-300 glass border-r border-outline-variant/10 flex flex-col items-center py-8 z-50 surface-shadow group overflow-hidden">
      {/* Logo */}
      <div className="mb-10 px-6 w-full flex items-center space-x-4">
        <div className="w-10 h-10 rounded-xl gradient-primary flex items-center justify-center flex-shrink-0">
          <span className="material-symbols-outlined text-on-primary text-2xl">
            hub
          </span>
        </div>
        <div className="opacity-0 group-hover:opacity-100 transition-opacity duration-300">
          <p className="font-headline font-bold text-on-surface leading-none">
            Nucleus Portal
          </p>
          <p className="text-[10px] text-on-surface-variant uppercase tracking-widest mt-1">
            Remote Access
          </p>
        </div>
      </div>

      {/* Main nav */}
      <nav className="flex-1 w-full px-3 space-y-2">
        {navItems.map((item) => {
          const isActive = pathname.startsWith(item.href);
          return (
            <Link
              key={item.href}
              href={item.href}
              className={clsx(
                "rounded-xl flex items-center h-12 w-full transition-colors overflow-hidden",
                isActive
                  ? "bg-primary-container/10 text-primary"
                  : "text-slate-500 hover:text-slate-200 hover:bg-surface-container-high",
              )}
            >
              <div className="w-14 flex-shrink-0 flex justify-center">
                <span className="material-symbols-outlined">{item.icon}</span>
              </div>
              <span className="opacity-0 group-hover:opacity-100 transition-opacity duration-300 font-medium whitespace-nowrap">
                {item.label}
              </span>
            </Link>
          );
        })}
      </nav>

      {/* Bottom nav */}
      <div className="mt-auto w-full px-3 space-y-2 border-t border-outline-variant/10 pt-6">
        {bottomNavItems.map((item) => (
          <Link
            key={item.href}
            href={item.href}
            className="text-slate-500 hover:text-slate-200 hover:bg-surface-container-high rounded-xl flex items-center h-12 w-full transition-colors overflow-hidden"
          >
            <div className="w-14 flex-shrink-0 flex justify-center">
              <span className="material-symbols-outlined">{item.icon}</span>
            </div>
            <span className="opacity-0 group-hover:opacity-100 transition-opacity duration-300 font-medium whitespace-nowrap">
              {item.label}
            </span>
          </Link>
        ))}
      </div>
    </aside>
  );
}
