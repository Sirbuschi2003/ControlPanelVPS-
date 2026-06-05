"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import {
  Server,
  LayoutDashboard,
  Globe,
  Database,
  Mail,
  Shield,
  HardDrive,
  Terminal,
  Settings,
  LogOut,
  ChevronLeft,
  Activity,
} from "lucide-react";
import { api, type User } from "@/lib/api";

const navItems = [
  { href: "/dashboard", icon: LayoutDashboard, label: "Übersicht" },
  { href: "/dashboard/servers", icon: Server, label: "Server" },
  { href: "/dashboard/websites", icon: Globe, label: "Websites", disabled: true },
  { href: "/dashboard/databases", icon: Database, label: "Datenbanken", disabled: true },
  { href: "/dashboard/mail", icon: Mail, label: "E-Mail", disabled: true },
  { href: "/dashboard/firewall", icon: Shield, label: "Firewall", disabled: true },
  { href: "/dashboard/backups", icon: HardDrive, label: "Backups", disabled: true },
  { href: "/dashboard/terminal", icon: Terminal, label: "Terminal", disabled: true },
];

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [user, setUser] = useState<User | null>(null);
  const [collapsed, setCollapsed] = useState(false);

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (!token) {
      router.push("/login");
      return;
    }
    api.get<User>("/auth/me")
      .then(setUser)
      .catch(() => router.push("/login"));
  }, [router]);

  function logout() {
    localStorage.removeItem("token");
    router.push("/login");
  }

  return (
    <div className="flex h-screen bg-background">
      {/* Sidebar */}
      <aside
        className={`flex flex-col border-r border-border bg-card transition-all duration-200 ${
          collapsed ? "w-16" : "w-56"
        }`}
      >
        {/* Logo */}
        <div className="flex items-center gap-3 px-4 py-4 border-b border-border h-14">
          <div className="flex-shrink-0 w-7 h-7 bg-primary/10 border border-primary/20 rounded-lg flex items-center justify-center">
            <Activity className="w-4 h-4 text-primary" />
          </div>
          {!collapsed && (
            <span className="font-semibold text-sm text-foreground truncate">ControlPanel</span>
          )}
          <button
            onClick={() => setCollapsed(!collapsed)}
            className="ml-auto text-muted-foreground hover:text-foreground flex-shrink-0"
          >
            <ChevronLeft className={`w-4 h-4 transition-transform ${collapsed ? "rotate-180" : ""}`} />
          </button>
        </div>

        {/* Nav */}
        <nav className="flex-1 p-2 space-y-0.5 overflow-y-auto">
          {navItems.map(({ href, icon: Icon, label, disabled }) => {
            const active = pathname === href;
            return (
              <Link
                key={href}
                href={disabled ? "#" : href}
                className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
                  disabled
                    ? "opacity-40 cursor-not-allowed text-muted-foreground"
                    : active
                    ? "bg-primary/10 text-primary"
                    : "text-muted-foreground hover:text-foreground hover:bg-accent"
                }`}
                onClick={(e) => disabled && e.preventDefault()}
                title={collapsed ? label : undefined}
              >
                <Icon className="w-4 h-4 flex-shrink-0" />
                {!collapsed && <span>{label}</span>}
                {!collapsed && disabled && (
                  <span className="ml-auto text-xs text-muted-foreground/50">bald</span>
                )}
              </Link>
            );
          })}
        </nav>

        {/* Footer */}
        <div className="border-t border-border p-2">
          <Link
            href="/dashboard/settings"
            className="flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
            title={collapsed ? "Einstellungen" : undefined}
          >
            <Settings className="w-4 h-4 flex-shrink-0" />
            {!collapsed && <span>Einstellungen</span>}
          </Link>
          <button
            onClick={logout}
            className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
            title={collapsed ? "Abmelden" : undefined}
          >
            <LogOut className="w-4 h-4 flex-shrink-0" />
            {!collapsed && <span>Abmelden</span>}
          </button>
        </div>
      </aside>

      {/* Main */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Top bar */}
        <header className="h-14 border-b border-border bg-card flex items-center px-6 gap-4">
          <div className="flex-1" />
          {user && (
            <div className="flex items-center gap-2">
              <div className="w-7 h-7 rounded-full bg-primary/20 flex items-center justify-center text-primary text-xs font-semibold">
                {user.name.charAt(0).toUpperCase()}
              </div>
              <div className="text-sm">
                <div className="text-foreground font-medium">{user.name}</div>
              </div>
            </div>
          )}
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-auto p-6">
          {children}
        </main>
      </div>
    </div>
  );
}
