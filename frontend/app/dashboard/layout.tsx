"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import {
  Server,
  LayoutDashboard,
  Globe,
  Globe2,
  Layers,
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
  HeartPulse,
  UserCircle,
  KeyRound,
  ChevronDown,
} from "lucide-react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { api, type User, type PanelUpdateStatus } from "@/lib/api";

const navItems = [
  { href: "/dashboard", icon: LayoutDashboard, label: "Übersicht" },
  { href: "/dashboard/servers", icon: Server, label: "Server" },
  { href: "/dashboard/domains", icon: Layers, label: "Domains" },
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
  { href: "/dashboard/monitoring", icon: HeartPulse, label: "Monitoring" },
  { href: "/dashboard/settings", icon: Settings, label: "Einstellungen" },
];

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [user, setUser] = useState<User | null>(null);
  const [collapsed, setCollapsed] = useState(false);
  const [updateAvailable, setUpdateAvailable] = useState(false);

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

  // Poll cached update status every 5 minutes — no GitHub API calls, instant
  useEffect(() => {
    function fetchStatus() {
      api.get<PanelUpdateStatus>("/panel/update-status")
        .then((s) => setUpdateAvailable(s.available))
        .catch(() => {});
    }
    fetchStatus();
    const iv = setInterval(fetchStatus, 5 * 60 * 1000);
    return () => clearInterval(iv);
  }, []);

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
            const showBadge = href === "/dashboard/updates" && updateAvailable;
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
                <span className="relative flex-shrink-0">
                  <Icon className="w-4 h-4" />
                  {showBadge && (
                    <span className="absolute -top-1 -right-1 w-2 h-2 rounded-full bg-yellow-400" />
                  )}
                </span>
                {!collapsed && (
                  <span className="flex items-center gap-2 flex-1 min-w-0">
                    <span className="truncate">{label}</span>
                    {showBadge && (
                      <span className="ml-auto text-xs font-semibold px-1.5 py-0.5 rounded-full bg-yellow-400/20 text-yellow-500 leading-none">
                        NEU
                      </span>
                    )}
                  </span>
                )}
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
            <DropdownMenu.Root>
              <DropdownMenu.Trigger asChild>
                <button className="flex items-center gap-2 rounded-lg px-2 py-1 hover:bg-accent transition-colors outline-none">
                  <div className="w-7 h-7 rounded-full bg-primary/20 flex items-center justify-center text-primary text-xs font-semibold">
                    {user.name.charAt(0).toUpperCase()}
                  </div>
                  <div className="text-sm text-left">
                    <div className="text-foreground font-medium leading-none">{user.name}</div>
                    <div className="text-muted-foreground text-xs mt-0.5">{user.email}</div>
                  </div>
                  <ChevronDown className="w-3.5 h-3.5 text-muted-foreground ml-1" />
                </button>
              </DropdownMenu.Trigger>
              <DropdownMenu.Portal>
                <DropdownMenu.Content
                  align="end"
                  sideOffset={6}
                  className="z-50 min-w-[180px] rounded-lg border border-border bg-card shadow-lg p-1 text-sm"
                >
                  <DropdownMenu.Item asChild>
                    <Link
                      href="/dashboard/profile"
                      className="flex items-center gap-2 px-3 py-2 rounded-md text-foreground hover:bg-accent cursor-pointer outline-none"
                    >
                      <UserCircle className="w-4 h-4" />
                      Mein Profil
                    </Link>
                  </DropdownMenu.Item>
                  <DropdownMenu.Item asChild>
                    <Link
                      href="/dashboard/profile#password"
                      className="flex items-center gap-2 px-3 py-2 rounded-md text-foreground hover:bg-accent cursor-pointer outline-none"
                    >
                      <KeyRound className="w-4 h-4" />
                      Passwort ändern
                    </Link>
                  </DropdownMenu.Item>
                  <DropdownMenu.Separator className="my-1 h-px bg-border" />
                  <DropdownMenu.Item asChild>
                    <button
                      onClick={logout}
                      className="w-full flex items-center gap-2 px-3 py-2 rounded-md text-destructive hover:bg-destructive/10 cursor-pointer outline-none"
                    >
                      <LogOut className="w-4 h-4" />
                      Abmelden
                    </button>
                  </DropdownMenu.Item>
                </DropdownMenu.Content>
              </DropdownMenu.Portal>
            </DropdownMenu.Root>
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
