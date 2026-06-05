"use client";

import { useEffect, useState, useCallback } from "react";
import {
  RefreshCw,
  GitBranch,
  GitCommit,
  CheckCircle,
  AlertTriangle,
  ArrowUpCircle,
  Terminal,
  Clock,
  Server,
} from "lucide-react";
import { api, type Server as ServerType } from "@/lib/api";

interface SystemInfo {
  commit: string;
  branch: string;
  commit_date: string;
  node_id: string;
  hostname: string;
  os: string;
}

interface UpdateCheck {
  available: boolean;
  current_commit: string;
  latest_commit: string;
}

interface UpdateResult {
  previous_commit: string;
  new_commit: string;
  changed_files: number;
  output: string;
  duration: string;
  restarted_at: string;
}

export default function UpdatesPage() {
  const [servers, setServers] = useState<ServerType[]>([]);
  const [selectedServer, setSelectedServer] = useState<string>("");
  const [info, setInfo] = useState<SystemInfo | null>(null);
  const [check, setCheck] = useState<UpdateCheck | null>(null);
  const [updateResult, setUpdateResult] = useState<UpdateResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [checking, setChecking] = useState(false);
  const [updating, setUpdating] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    api.get<ServerType[]>("/servers").then((data) => {
      setServers(data);
      if (data.length > 0) setSelectedServer(data[0].id);
    });
  }, []);

  const loadInfo = useCallback(async () => {
    if (!selectedServer) return;
    setLoading(true);
    setError("");
    try {
      const data = await api.get<SystemInfo>(
        `/system/info?server_id=${selectedServer}`
      );
      setInfo(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Fehler beim Laden");
    } finally {
      setLoading(false);
    }
  }, [selectedServer]);

  useEffect(() => {
    loadInfo();
  }, [loadInfo]);

  async function checkForUpdates() {
    setChecking(true);
    setError("");
    try {
      const data = await api.get<UpdateCheck>(
        `/system/check-updates?server_id=${selectedServer}`
      );
      setCheck(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Fehler beim Prüfen");
    } finally {
      setChecking(false);
    }
  }

  async function runUpdate() {
    if (
      !confirm(
        "Update jetzt installieren? Der Server wird kurz neu gestartet. Fortfahren?"
      )
    )
      return;
    setUpdating(true);
    setError("");
    setUpdateResult(null);
    try {
      const data = await api.post<UpdateResult>(
        `/system/update?server_id=${selectedServer}`,
        {}
      );
      setUpdateResult(data);
      await loadInfo();
      setCheck(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Update fehlgeschlagen");
    } finally {
      setUpdating(false);
    }
  }

  return (
    <div className="space-y-6 max-w-4xl">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">
            Panel-Updates
          </h1>
          <p className="text-muted-foreground text-sm mt-1">
            ControlPanelVPS aktuell halten
          </p>
        </div>
        <div className="flex items-center gap-2">
          {servers.length > 1 && (
            <select
              value={selectedServer}
              onChange={(e) => setSelectedServer(e.target.value)}
              className="px-3 py-2 bg-secondary border border-border rounded-lg text-foreground text-sm focus:outline-none focus:ring-2 focus:ring-primary"
            >
              {servers.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
          )}
          <button
            onClick={loadInfo}
            disabled={loading}
            className="flex items-center gap-2 px-3 py-2 border border-border rounded-lg text-sm text-muted-foreground hover:bg-accent transition-colors"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? "animate-spin" : ""}`} />
            Aktualisieren
          </button>
        </div>
      </div>

      {error && (
        <div className="px-4 py-3 bg-destructive/10 border border-destructive/20 rounded-xl text-destructive text-sm">
          {error}
        </div>
      )}

      {/* Current version card */}
      <div className="bg-card border border-border rounded-xl p-5">
        <h2 className="text-sm font-semibold text-foreground mb-4 flex items-center gap-2">
          <Server className="w-4 h-4 text-primary" />
          Aktuelle Version
        </h2>
        {loading ? (
          <div className="space-y-2">
            {[1, 2, 3].map((i) => (
              <div
                key={i}
                className="h-5 bg-secondary rounded animate-pulse"
                style={{ width: `${60 + i * 10}%` }}
              />
            ))}
          </div>
        ) : info ? (
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            <InfoItem
              icon={<GitCommit className="w-4 h-4" />}
              label="Commit"
              value={
                <code className="text-primary font-mono">{info.commit}</code>
              }
            />
            <InfoItem
              icon={<GitBranch className="w-4 h-4" />}
              label="Branch"
              value={info.branch}
            />
            <InfoItem
              icon={<Clock className="w-4 h-4" />}
              label="Datum"
              value={info.commit_date ? info.commit_date.slice(0, 10) : "—"}
            />
            <InfoItem
              icon={<Server className="w-4 h-4" />}
              label="Hostname"
              value={info.hostname}
            />
            <InfoItem
              icon={<Terminal className="w-4 h-4" />}
              label="Betriebssystem"
              value={info.os}
            />
            <InfoItem
              icon={<GitBranch className="w-4 h-4" />}
              label="Node ID"
              value={info.node_id}
            />
          </div>
        ) : (
          <p className="text-muted-foreground text-sm">Keine Daten</p>
        )}
      </div>

      {/* Update check card */}
      <div className="bg-card border border-border rounded-xl p-5">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-semibold text-foreground flex items-center gap-2">
            <ArrowUpCircle className="w-4 h-4 text-primary" />
            Updates prüfen
          </h2>
          <button
            onClick={checkForUpdates}
            disabled={checking || !selectedServer}
            className="flex items-center gap-2 px-3 py-1.5 bg-secondary border border-border rounded-lg text-sm text-foreground hover:bg-accent transition-colors disabled:opacity-50"
          >
            <RefreshCw
              className={`w-3.5 h-3.5 ${checking ? "animate-spin" : ""}`}
            />
            {checking ? "Prüfe..." : "Jetzt prüfen"}
          </button>
        </div>

        {check ? (
          <div className="space-y-4">
            <div
              className={`flex items-center gap-3 px-4 py-3 rounded-lg border ${
                check.available
                  ? "bg-yellow-500/10 border-yellow-500/20 text-yellow-500"
                  : "bg-green-500/10 border-green-500/20 text-green-500"
              }`}
            >
              {check.available ? (
                <AlertTriangle className="w-5 h-5 flex-shrink-0" />
              ) : (
                <CheckCircle className="w-5 h-5 flex-shrink-0" />
              )}
              <div>
                <p className="font-medium text-sm">
                  {check.available
                    ? "Update verfügbar!"
                    : "Bereits aktuell"}
                </p>
                <p className="text-xs opacity-80 mt-0.5">
                  Aktuell:{" "}
                  <code className="font-mono">{check.current_commit}</code>
                  {check.available && (
                    <>
                      {" "}
                      → Neu:{" "}
                      <code className="font-mono">{check.latest_commit}</code>
                    </>
                  )}
                </p>
              </div>
            </div>

            {check.available && (
              <button
                onClick={runUpdate}
                disabled={updating}
                className="flex items-center gap-2 px-4 py-2.5 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {updating ? (
                  <RefreshCw className="w-4 h-4 animate-spin" />
                ) : (
                  <ArrowUpCircle className="w-4 h-4" />
                )}
                {updating
                  ? "Update wird installiert..."
                  : "Update jetzt installieren"}
              </button>
            )}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            Klicke auf "Jetzt prüfen" um nach Updates zu suchen.
          </p>
        )}
      </div>

      {/* Update result */}
      {updateResult && (
        <div className="bg-card border border-green-500/20 rounded-xl p-5">
          <h2 className="text-sm font-semibold text-green-500 flex items-center gap-2 mb-4">
            <CheckCircle className="w-4 h-4" />
            Update erfolgreich — {updateResult.duration}
          </h2>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4 mb-4">
            <InfoItem
              icon={<GitCommit className="w-4 h-4" />}
              label="Vorher"
              value={
                <code className="font-mono text-muted-foreground">
                  {updateResult.previous_commit}
                </code>
              }
            />
            <InfoItem
              icon={<GitCommit className="w-4 h-4" />}
              label="Nachher"
              value={
                <code className="font-mono text-primary">
                  {updateResult.new_commit}
                </code>
              }
            />
            <InfoItem
              icon={<GitBranch className="w-4 h-4" />}
              label="Geänderte Dateien"
              value={String(updateResult.changed_files)}
            />
          </div>
          <div>
            <p className="text-xs text-muted-foreground mb-1.5 font-medium">
              Build-Log
            </p>
            <pre className="bg-secondary text-foreground text-xs font-mono p-4 rounded-lg overflow-auto max-h-80 whitespace-pre-wrap">
              {updateResult.output}
            </pre>
          </div>
        </div>
      )}

      {/* Manual update instructions */}
      <div className="bg-card border border-border rounded-xl p-5">
        <h2 className="text-sm font-semibold text-foreground mb-3">
          Manuelles Update per SSH
        </h2>
        <p className="text-sm text-muted-foreground mb-3">
          Als Alternative zum Panel-Update kannst du das Update-Script direkt
          auf dem Server ausführen:
        </p>
        <pre className="bg-secondary text-foreground text-xs font-mono p-4 rounded-lg">
          {`ssh root@DEIN-SERVER
bash /opt/controlpanel/deploy/update.sh`}
        </pre>
        <p className="text-xs text-muted-foreground mt-2">
          Das Script führt git pull, rebuild und service restart automatisch
          durch.
        </p>
      </div>
    </div>
  );
}

function InfoItem({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode;
  label: string;
  value: React.ReactNode;
}) {
  return (
    <div className="space-y-1">
      <p className="text-xs text-muted-foreground flex items-center gap-1.5">
        {icon}
        {label}
      </p>
      <p className="text-sm text-foreground font-medium">{value}</p>
    </div>
  );
}
