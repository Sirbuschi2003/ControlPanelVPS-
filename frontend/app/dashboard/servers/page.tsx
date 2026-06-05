"use client";

import { useEffect, useState, useCallback } from "react";
import { Plus, RefreshCw, Server, Cpu, MemoryStick, HardDrive, Clock } from "lucide-react";
import { api, type Server as ServerType, type ServerMetrics, formatBytes, formatUptime } from "@/lib/api";

export default function ServersPage() {
  const [servers, setServers] = useState<ServerType[]>([]);
  const [metrics, setMetrics] = useState<Record<string, ServerMetrics>>({});
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [showAdd, setShowAdd] = useState(false);

  const loadServers = useCallback(async () => {
    const data = await api.get<ServerType[]>("/servers");
    setServers(data);
    return data;
  }, []);

  const loadMetrics = useCallback(async (serverList: ServerType[]) => {
    const results = await Promise.allSettled(
      serverList.map((s) => api.get<ServerMetrics>(`/servers/${s.id}/metrics`))
    );
    const map: Record<string, ServerMetrics> = {};
    results.forEach((r, i) => {
      if (r.status === "fulfilled") {
        map[serverList[i].id] = r.value;
      }
    });
    setMetrics(map);
  }, []);

  useEffect(() => {
    loadServers()
      .then(loadMetrics)
      .catch(console.error)
      .finally(() => setLoading(false));
  }, [loadServers, loadMetrics]);

  async function refresh() {
    setRefreshing(true);
    try {
      const data = await loadServers();
      await loadMetrics(data);
    } finally {
      setRefreshing(false);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Server</h1>
          <p className="text-muted-foreground text-sm mt-1">
            {servers.length} Server verwaltet
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={refresh}
            disabled={refreshing}
            className="flex items-center gap-2 px-3 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors text-muted-foreground disabled:opacity-50"
          >
            <RefreshCw className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`} />
            Aktualisieren
          </button>
          <button
            onClick={() => setShowAdd(true)}
            className="flex items-center gap-2 px-3 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
          >
            <Plus className="w-4 h-4" />
            Server hinzufügen
          </button>
        </div>
      </div>

      {loading ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {[1, 2].map((i) => (
            <div key={i} className="h-48 bg-card border border-border rounded-xl animate-pulse" />
          ))}
        </div>
      ) : servers.length === 0 ? (
        <EmptyState onAdd={() => setShowAdd(true)} />
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {servers.map((server) => (
            <ServerCard key={server.id} server={server} metrics={metrics[server.id]} />
          ))}
        </div>
      )}

      {showAdd && <AddServerModal onClose={() => setShowAdd(false)} onAdded={refresh} />}
    </div>
  );
}

function ServerCard({ server, metrics }: { server: ServerType; metrics?: ServerMetrics }) {
  return (
    <div className="bg-card border border-border rounded-xl p-5 space-y-4">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 bg-secondary rounded-lg flex items-center justify-center">
            <Server className="w-5 h-5 text-primary" />
          </div>
          <div>
            <h3 className="font-semibold text-foreground text-sm">{server.name}</h3>
            <p className="text-xs text-muted-foreground">{server.ip_address}</p>
          </div>
        </div>
        <StatusBadge status={server.status} />
      </div>

      {/* Metrics */}
      {metrics ? (
        <div className="grid grid-cols-2 gap-3">
          <MetricItem
            icon={<Cpu className="w-3.5 h-3.5" />}
            label="CPU"
            value={`${metrics.cpu.usage_percent.toFixed(1)}%`}
            percent={metrics.cpu.usage_percent}
          />
          <MetricItem
            icon={<MemoryStick className="w-3.5 h-3.5" />}
            label="RAM"
            value={`${formatBytes(metrics.memory.used_bytes)} / ${formatBytes(metrics.memory.total_bytes)}`}
            percent={metrics.memory.usage_percent}
          />
          <MetricItem
            icon={<HardDrive className="w-3.5 h-3.5" />}
            label="Disk"
            value={`${formatBytes(metrics.disk.used_bytes)} / ${formatBytes(metrics.disk.total_bytes)}`}
            percent={metrics.disk.usage_percent}
          />
          <MetricItem
            icon={<Clock className="w-3.5 h-3.5" />}
            label="Uptime"
            value={formatUptime(metrics.uptime)}
          />
        </div>
      ) : server.status === "offline" ? (
        <div className="text-xs text-muted-foreground text-center py-4">
          Agent nicht erreichbar
        </div>
      ) : (
        <div className="grid grid-cols-2 gap-3">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="h-12 bg-secondary rounded-lg animate-pulse" />
          ))}
        </div>
      )}

      {/* OS info */}
      {metrics && (
        <p className="text-xs text-muted-foreground border-t border-border pt-3">
          {metrics.os} · Kernel {metrics.kernel_version} · Load {metrics.load_avg.load1.toFixed(2)}
        </p>
      )}
    </div>
  );
}

function MetricItem({
  icon,
  label,
  value,
  percent,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  percent?: number;
}) {
  const color =
    percent === undefined ? "bg-primary"
    : percent > 90 ? "bg-red-500"
    : percent > 70 ? "bg-yellow-500"
    : "bg-green-500";

  return (
    <div className="bg-secondary rounded-lg px-3 py-2 space-y-1">
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
        {icon}
        <span>{label}</span>
      </div>
      <p className="text-sm font-medium text-foreground">{value}</p>
      {percent !== undefined && (
        <div className="h-1 bg-border rounded-full overflow-hidden">
          <div
            className={`h-full rounded-full transition-all ${color}`}
            style={{ width: `${Math.min(percent, 100)}%` }}
          />
        </div>
      )}
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const styles =
    status === "online"
      ? "bg-green-500/10 text-green-500 border-green-500/20"
      : status === "offline"
      ? "bg-red-500/10 text-red-500 border-red-500/20"
      : "bg-yellow-500/10 text-yellow-500 border-yellow-500/20";

  return (
    <span className={`text-xs px-2 py-0.5 rounded-full border font-medium ${styles}`}>
      {status === "online" ? "Online" : status === "offline" ? "Offline" : "Unbekannt"}
    </span>
  );
}

function EmptyState({ onAdd }: { onAdd: () => void }) {
  return (
    <div className="bg-card border border-dashed border-border rounded-xl p-12 text-center">
      <Server className="w-12 h-12 text-muted-foreground/40 mx-auto mb-4" />
      <h3 className="text-base font-semibold text-foreground mb-1">Kein Server vorhanden</h3>
      <p className="text-sm text-muted-foreground mb-4">
        Füge deinen ersten Server hinzu, um ihn zu verwalten.
      </p>
      <button
        onClick={onAdd}
        className="px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors"
      >
        Server hinzufügen
      </button>
    </div>
  );
}

function AddServerModal({ onClose, onAdded }: { onClose: () => void; onAdded: () => void }) {
  const [form, setForm] = useState({
    name: "",
    hostname: "",
    ip_address: "",
    agent_url: "",
    agent_token: "",
    role: "general",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      await api.post("/servers", form);
      onAdded();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Fehler beim Hinzufügen");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
      <div className="bg-card border border-border rounded-xl w-full max-w-md shadow-2xl">
        <div className="flex items-center justify-between px-5 py-4 border-b border-border">
          <h2 className="font-semibold text-foreground">Server hinzufügen</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">✕</button>
        </div>
        <form onSubmit={handleSubmit} className="p-5 space-y-3">
          {[
            { key: "name", label: "Name", placeholder: "Mein VPS" },
            { key: "hostname", label: "Hostname", placeholder: "server1.example.com" },
            { key: "ip_address", label: "IP-Adresse", placeholder: "1.2.3.4" },
            { key: "agent_url", label: "Agent URL", placeholder: "http://1.2.3.4:8087" },
            { key: "agent_token", label: "Agent Token", placeholder: "Geheimes Token" },
          ].map(({ key, label, placeholder }) => (
            <div key={key}>
              <label className="block text-xs font-medium text-muted-foreground mb-1">{label}</label>
              <input
                type="text"
                value={form[key as keyof typeof form]}
                onChange={(e) => setForm({ ...form, [key]: e.target.value })}
                placeholder={placeholder}
                className="w-full px-3 py-2 bg-secondary border border-border rounded-lg text-foreground placeholder-muted-foreground text-sm focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>
          ))}
          {error && (
            <p className="text-xs text-destructive">{error}</p>
          )}
          <div className="flex gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 border border-border rounded-lg text-sm text-muted-foreground hover:bg-accent transition-colors"
            >
              Abbrechen
            </button>
            <button
              type="submit"
              disabled={loading}
              className="flex-1 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              {loading ? "Hinzufügen..." : "Hinzufügen"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
