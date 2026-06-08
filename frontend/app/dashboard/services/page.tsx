"use client";

import { useEffect, useState, useCallback } from "react";
import { Activity, Play, Square, RefreshCw, ToggleLeft, ToggleRight, X, AlertCircle } from "lucide-react";
import { api, type SystemService, type Server } from "@/lib/api";

function Skeleton({ className }: { className?: string }) {
  return <div className={`bg-secondary animate-pulse rounded ${className}`} />;
}

const KEY_SERVICES = ["nginx", "apache2", "mysql", "mariadb", "postgresql", "redis", "redis-server", "postfix", "fail2ban", "php8.2-fpm", "php8.1-fpm", "ssh", "ufw"];

export default function ServicesPage() {
  const [servers, setServers] = useState<Server[]>([]);
  const [selectedServer, setSelectedServer] = useState("");
  const [services, setServices] = useState<SystemService[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null);

  async function loadServers() {
    try {
      const sv = await api.get<Server[]>("/servers");
      setServers(sv);
      if (sv.length > 0) setSelectedServer(sv[0].id);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Laden der Server");
    }
  }

  const loadServices = useCallback(async (serverId?: string) => {
    const id = serverId || selectedServer;
    if (!id) return;
    setLoading(true);
    try {
      const svcs = await api.get<SystemService[]>(`/services?server_id=${id}`);
      setServices(svcs);
      setLastRefresh(new Date());
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Laden der Dienste");
    } finally {
      setLoading(false);
    }
  }, [selectedServer]);

  useEffect(() => {
    loadServers();
  }, []);

  useEffect(() => {
    if (selectedServer) loadServices(selectedServer);
  }, [selectedServer]);

  useEffect(() => {
    if (!autoRefresh) return;
    const interval = setInterval(() => loadServices(), 30000);
    return () => clearInterval(interval);
  }, [autoRefresh, loadServices]);

  async function handleAction(name: string, action: "start" | "stop" | "restart" | "enable" | "disable") {
    setActionLoading(`${name}-${action}`);
    try {
      await api.post(`/services/${name}/action`, { server_id: selectedServer, action });
      await loadServices();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : `Fehler bei Aktion ${action}`);
    } finally {
      setActionLoading(null);
    }
  }

  const isKey = (name: string) => KEY_SERVICES.some((k) => name.toLowerCase().includes(k));

  const sortedServices = [...services].sort((a, b) => {
    const aKey = isKey(a.name) ? 0 : 1;
    const bKey = isKey(b.name) ? 0 : 1;
    if (aKey !== bKey) return aKey - bKey;
    return a.name.localeCompare(b.name);
  });

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Dienste</h1>
          <p className="text-muted-foreground text-sm mt-1">Systemdienste überwachen und steuern</p>
        </div>
        <div className="flex items-center gap-3">
          {lastRefresh && (
            <span className="text-xs text-muted-foreground">
              Zuletzt: {lastRefresh.toLocaleTimeString("de-DE")}
            </span>
          )}
          <button
            onClick={() => setAutoRefresh(!autoRefresh)}
            className={`flex items-center gap-2 px-3 py-2 text-sm border rounded-lg transition-colors ${
              autoRefresh
                ? "border-primary text-primary bg-primary/10"
                : "border-border text-muted-foreground hover:text-foreground hover:bg-accent"
            }`}
          >
            <RefreshCw className={`w-4 h-4 ${autoRefresh ? "animate-spin" : ""}`} />
            Auto (30s)
          </button>
          <button
            onClick={() => loadServices()}
            disabled={loading}
            className="flex items-center gap-2 px-3 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors text-muted-foreground hover:text-foreground disabled:opacity-50"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? "animate-spin" : ""}`} />
            Aktualisieren
          </button>
          {servers.length > 0 && (
            <select
              value={selectedServer}
              onChange={(e) => setSelectedServer(e.target.value)}
              className="bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
            >
              {servers.map((s) => (
                <option key={s.id} value={s.id}>{s.name}</option>
              ))}
            </select>
          )}
        </div>
      </div>

      {error && (
        <div className="flex items-center gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-lg text-destructive text-sm mb-4">
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          {error}
          <button onClick={() => setError("")} className="ml-auto"><X className="w-4 h-4" /></button>
        </div>
      )}

      {!selectedServer ? (
        <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
          <Activity className="w-12 h-12 mb-4 opacity-30" />
          <p className="font-medium">Kein Server verfügbar</p>
          <p className="text-sm mt-1">Fügen Sie zuerst einen Server hinzu</p>
        </div>
      ) : loading && services.length === 0 ? (
        <div className="space-y-2">
          {[1, 2, 3, 4, 5, 6].map((i) => <Skeleton key={i} className="h-16 w-full rounded-xl" />)}
        </div>
      ) : services.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
          <Activity className="w-12 h-12 mb-4 opacity-30" />
          <p className="font-medium">Keine Dienste gefunden</p>
        </div>
      ) : (
        <div className="bg-card border border-border rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-muted-foreground">
                <th className="text-left px-4 py-3 font-medium">Dienst</th>
                <th className="text-left px-4 py-3 font-medium">Status</th>
                <th className="text-left px-4 py-3 font-medium">Aktiviert</th>
                <th className="text-right px-4 py-3 font-medium">Aktionen</th>
              </tr>
            </thead>
            <tbody>
              {sortedServices.map((svc) => {
                const key = isKey(svc.name);
                return (
                  <tr
                    key={svc.name}
                    className={`border-b border-border last:border-0 hover:bg-accent/50 transition-colors ${key ? "bg-primary/5" : ""}`}
                  >
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        {key && <span className="w-1.5 h-1.5 rounded-full bg-primary flex-shrink-0" />}
                        <div>
                          <div className="font-medium text-foreground font-mono text-xs">{svc.name}</div>
                          {svc.description && (
                            <div className="text-xs text-muted-foreground mt-0.5">{svc.description}</div>
                          )}
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <span className={`w-2 h-2 rounded-full flex-shrink-0 ${svc.active ? "bg-green-400" : "bg-red-400"}`} />
                        <span className={`text-sm ${svc.active ? "text-green-400" : "text-red-400"}`}>
                          {svc.active ? "Aktiv" : "Inaktiv"}
                        </span>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <button
                        onClick={() => handleAction(svc.name, svc.enabled ? "disable" : "enable")}
                        disabled={actionLoading !== null}
                        className="text-muted-foreground hover:text-foreground transition-colors"
                      >
                        {svc.enabled
                          ? <ToggleRight className="w-5 h-5 text-green-400" />
                          : <ToggleLeft className="w-5 h-5" />}
                      </button>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end gap-2">
                        {!svc.active ? (
                          <button
                            onClick={() => handleAction(svc.name, "start")}
                            disabled={actionLoading !== null}
                            className="flex items-center gap-1 px-2 py-1 text-xs bg-green-500/20 text-green-400 border border-green-500/30 rounded hover:bg-green-500/30 transition-colors disabled:opacity-50"
                            title="Starten"
                          >
                            <Play className="w-3 h-3" />
                            Starten
                          </button>
                        ) : (
                          <>
                            <button
                              onClick={() => handleAction(svc.name, "restart")}
                              disabled={actionLoading !== null}
                              className="flex items-center gap-1 px-2 py-1 text-xs border border-border rounded hover:bg-accent transition-colors text-muted-foreground hover:text-foreground disabled:opacity-50"
                              title="Neustarten"
                            >
                              <RefreshCw className={`w-3 h-3 ${actionLoading === `${svc.name}-restart` ? "animate-spin" : ""}`} />
                              Neustart
                            </button>
                            <button
                              onClick={() => handleAction(svc.name, "stop")}
                              disabled={actionLoading !== null}
                              className="flex items-center gap-1 px-2 py-1 text-xs bg-red-500/20 text-red-400 border border-red-500/30 rounded hover:bg-red-500/30 transition-colors disabled:opacity-50"
                              title="Stoppen"
                            >
                              <Square className="w-3 h-3" />
                              Stoppen
                            </button>
                          </>
                        )}
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
