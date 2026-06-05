"use client";

import { useEffect, useState } from "react";
import { Server, CheckCircle, XCircle, Clock, Activity } from "lucide-react";
import { api, type Server as ServerType } from "@/lib/api";
import Link from "next/link";

export default function DashboardPage() {
  const [servers, setServers] = useState<ServerType[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.get<ServerType[]>("/servers")
      .then(setServers)
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const online = servers.filter((s) => s.status === "online").length;
  const offline = servers.filter((s) => s.status === "offline").length;
  const unknown = servers.filter((s) => s.status === "unknown").length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">Übersicht</h1>
        <p className="text-muted-foreground text-sm mt-1">
          Alle verwalteten Server auf einen Blick
        </p>
      </div>

      {/* Stat cards */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <StatCard
          icon={<Server className="w-5 h-5 text-primary" />}
          label="Gesamt"
          value={servers.length}
          loading={loading}
        />
        <StatCard
          icon={<CheckCircle className="w-5 h-5 text-green-500" />}
          label="Online"
          value={online}
          loading={loading}
          color="green"
        />
        <StatCard
          icon={<XCircle className="w-5 h-5 text-red-500" />}
          label="Offline"
          value={offline + unknown}
          loading={loading}
          color="red"
        />
      </div>

      {/* Server list */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-base font-semibold text-foreground">Server</h2>
          <Link
            href="/dashboard/servers"
            className="text-sm text-primary hover:underline"
          >
            Alle anzeigen →
          </Link>
        </div>
        {loading ? (
          <div className="space-y-2">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-16 bg-card border border-border rounded-xl animate-pulse" />
            ))}
          </div>
        ) : servers.length === 0 ? (
          <div className="bg-card border border-dashed border-border rounded-xl p-8 text-center">
            <Activity className="w-10 h-10 text-muted-foreground/50 mx-auto mb-3" />
            <p className="text-muted-foreground text-sm">
              Noch kein Server hinzugefügt.{" "}
              <Link href="/dashboard/servers" className="text-primary hover:underline">
                Jetzt hinzufügen →
              </Link>
            </p>
          </div>
        ) : (
          <div className="space-y-2">
            {servers.slice(0, 5).map((server) => (
              <ServerRow key={server.id} server={server} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function StatCard({
  icon,
  label,
  value,
  loading,
  color,
}: {
  icon: React.ReactNode;
  label: string;
  value: number;
  loading: boolean;
  color?: "green" | "red";
}) {
  return (
    <div className="bg-card border border-border rounded-xl p-4 flex items-center gap-4">
      <div className="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center flex-shrink-0">
        {icon}
      </div>
      <div>
        <p className="text-xs text-muted-foreground">{label}</p>
        {loading ? (
          <div className="h-6 w-8 bg-secondary rounded animate-pulse mt-0.5" />
        ) : (
          <p className={`text-2xl font-bold ${color === "green" ? "text-green-500" : color === "red" ? "text-red-500" : "text-foreground"}`}>
            {value}
          </p>
        )}
      </div>
    </div>
  );
}

function ServerRow({ server }: { server: ServerType }) {
  return (
    <Link
      href="/dashboard/servers"
      className="flex items-center gap-4 bg-card border border-border rounded-xl px-4 py-3 hover:bg-accent transition-colors"
    >
      <StatusDot status={server.status} />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium text-foreground truncate">{server.name}</p>
        <p className="text-xs text-muted-foreground truncate">{server.ip_address}</p>
      </div>
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <Clock className="w-3 h-3" />
        {server.last_seen
          ? new Date(server.last_seen).toLocaleTimeString("de-DE", { hour: "2-digit", minute: "2-digit" })
          : "Nie"}
      </div>
    </Link>
  );
}

function StatusDot({ status }: { status: string }) {
  const classes =
    status === "online"
      ? "bg-green-500"
      : status === "offline"
      ? "bg-red-500"
      : "bg-yellow-500";
  return (
    <span className="relative flex h-2.5 w-2.5 flex-shrink-0">
      {status === "online" && (
        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
      )}
      <span className={`relative inline-flex rounded-full h-2.5 w-2.5 ${classes}`} />
    </span>
  );
}
