"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import {
  Server,
  LayoutDashboard,
  Globe,
  Globe2,
  Database,
  Mail,
  Shield,
  HardDrive,
  Terminal,
  Settings,
  LogOut,
  ChevronLeft,
  Activity,
  Lock,
  Clock,
  FileText,
  Folder,
  Users,
  ArrowUpCircle,
} from "lucide-react";
import { api, type User } from "@/lib/api";

const navItems = [
  { href: "/dashboard", icon: LayoutDashboard, label: "Übersicht" },
  { href: "/dashboard/servers", icon: Server, label: "Server" },
  { href: "/dashboard/websites", icon: Globe, label: "Websites" },
  { href: "/dashboard/ssl", icon: Lock, label: "SSL/TLS" },
  { href: "/dashboard/databases", icon: Database, label: "Datenbanken" },
  { href: "/dashboard/dns", icon: Globe2, label: "DNS" },
  { href: "/dashboard/mail", icon: Mail, label: "E-Mail" },
  { href: "/dashboard/firewall", icon: Shield, label: "Firewall" },
  { href: "/dashboard/backups", icon: HardDrive, label: "Backups" },
  { href: "/dashboard/services", icon: Activity, label: "Dienste" },
  { href: "/dashboard/crons", icon: Clock, label: "Cron Jobs" },
  { href: "/dashboard/logs", icon: FileText, label: "Logs" },
  { href: "/dashboard/files", icon: Folder, label: "Dateien" },
  { href: "/dashboard/terminal", icon: Terminal, label: "Terminal" },
  { href: "/dashboard/users", icon: Users, label: "Benutzer" },
  { href: "/dashboard/updates", icon: ArrowUpCircle, label: "Updates" },
  { href: "/dashboard/settings", icon: Settings, label: "Einstellungen" },
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
          {navItems.map(({ href, icon: Icon, label }) => {
            const active = pathname === href || (href !== "/dashboard" && pathname.startsWith(href));
            return (
              <Link
                key={href}
                href={href}
                className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
                  active
                    ? "bg-primary/10 text-primary"
                    : "text-muted-foreground hover:text-foreground hover:bg-accent"
                }`}
                title={collapsed ? label : undefined}
              >
                <Icon className="w-4 h-4 flex-shrink-0" />
                {!collapsed && <span>{label}</span>}
              </Link>
            );
          })}
        </nav>

        {/* Footer */}
        <div className="border-t border-border p-2">
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
