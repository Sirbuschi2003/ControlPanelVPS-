"use client";

import { useEffect, useState, useCallback } from "react";
import {
  RefreshCw, GitCommit, CheckCircle, AlertTriangle,
  ArrowUpCircle, Terminal, Clock, Server, Package,
  ShieldCheck, X,
} from "lucide-react";
import {
  api,
  type Server as ServerType,
  type PanelInfo,
  type PanelUpdateCheck,
  type PanelUpdateResult,
} from "@/lib/api";
import { Bell } from "lucide-react";

// ── Types for agent / system updates ─────────────────────────────────────────

interface SystemInfo {
  commit: string;
  branch: string;
  commit_date: string;
  node_id: string;
  hostname: string;
  os: string;
}

interface AgentUpdateCheck {
  available: boolean;
  current_commit: string;
  latest_commit: string;
}

interface AgentUpdateResult {
  previous_commit: string;
  new_commit: string;
  changed_files: number;
  output: string;
  duration: string;
}

// ── Small reusable components ─────────────────────────────────────────────────

function InfoItem({ icon, label, value }: { icon: React.ReactNode; label: string; value: React.ReactNode }) {
  return (
    <div className="space-y-1">
      <p className="text-xs text-muted-foreground flex items-center gap-1.5">{icon}{label}</p>
      <p className="text-sm text-foreground font-medium">{value}</p>
    </div>
  );
}

function ErrorBanner({ msg, onClose }: { msg: string; onClose: () => void }) {
  return (
    <div className="flex items-center gap-2 px-4 py-3 bg-destructive/10 border border-destructive/20 rounded-xl text-destructive text-sm">
      <AlertTriangle className="w-4 h-4 shrink-0" />
      {msg}
      <button onClick={onClose} className="ml-auto"><X className="w-4 h-4" /></button>
    </div>
  );
}

function Skeleton({ className, style }: { className?: string; style?: React.CSSProperties }) {
  return <div className={`bg-secondary animate-pulse rounded ${className}`} style={style} />;
}

// ── Tab: Panel Software ────────────────────────────────────────────────────────

function PanelTab() {
  const [info, setInfo] = useState<PanelInfo | null>(null);
  const [check, setCheck] = useState<PanelUpdateCheck | null>(null);
  const [result, setResult] = useState<PanelUpdateResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [checking, setChecking] = useState(false);
  const [updating, setUpdating] = useState(false);
  const [error, setError] = useState("");
  const [autoUpdate, setAutoUpdate] = useState(false);
  const [savingAuto, setSavingAuto] = useState(false);

  useEffect(() => {
    Promise.all([
      api.get<PanelInfo>("/panel/info"),
      api.get<{ enabled: boolean }>("/panel/auto-update"),
    ]).then(([inf, au]) => {
      setInfo(inf);
      setAutoUpdate(au.enabled);
    }).catch((e) => setError(e instanceof Error ? e.message : "Fehler beim Laden"))
      .finally(() => setLoading(false));
  }, []);

  async function toggleAutoUpdate(enabled: boolean) {
    setSavingAuto(true);
    try {
      await api.put<{ enabled: boolean }>("/panel/auto-update", { enabled });
      setAutoUpdate(enabled);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Fehler beim Speichern");
    } finally {
      setSavingAuto(false);
    }
  }

  async function checkUpdate() {
    setChecking(true);
    setError("");
    try {
      setCheck(await api.get<PanelUpdateCheck>("/panel/check-update"));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Prüfung fehlgeschlagen");
    } finally {
      setChecking(false);
    }
  }

  async function runUpdate() {
    if (!confirm("Panel-Software jetzt aktualisieren? Das System startet kurz neu.")) return;
    setUpdating(true);
    setError("");
    setResult(null);
    try {
      const r = await api.post<PanelUpdateResult>("/panel/update", {});
      setResult(r);
      setCheck(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Update fehlgeschlagen");
    } finally {
      setUpdating(false);
    }
  }

  const isDevBuild = info?.commit === "dev";

  return (
    <div className="space-y-5">
      {error && <ErrorBanner msg={error} onClose={() => setError("")} />}

      {/* Current version */}
      <div className="bg-card border border-border rounded-xl p-5">
        <h2 className="text-sm font-semibold text-foreground mb-4 flex items-center gap-2">
          <Server className="w-4 h-4 text-primary" />
          Installierte Version
        </h2>
        {loading ? (
          <div className="space-y-2">{[1,2,3].map(i => <Skeleton key={i} className="h-5" style={{width:`${50+i*12}%`}} />)}</div>
        ) : info ? (
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            <InfoItem icon={<GitCommit className="w-4 h-4" />} label="Commit"
              value={<code className="text-primary font-mono">{info.commit === "dev" ? "dev (lokal)" : info.commit}</code>} />
            <InfoItem icon={<Clock className="w-4 h-4" />} label="Build-Datum"
              value={info.date === "unknown" ? "—" : new Date(info.date).toLocaleString("de-DE")} />
            <InfoItem icon={<Terminal className="w-4 h-4" />} label="Installationsverzeichnis"
              value={<code className="text-xs font-mono">{info.install_dir}</code>} />
          </div>
        ) : <p className="text-muted-foreground text-sm">Keine Daten verfügbar</p>}
      </div>

      {/* Update check */}
      <div className="bg-card border border-border rounded-xl p-5">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-semibold text-foreground flex items-center gap-2">
            <ArrowUpCircle className="w-4 h-4 text-primary" />
            Update-Prüfung
          </h2>
          <button
            onClick={checkUpdate}
            disabled={checking || isDevBuild}
            title={isDevBuild ? "Nicht verfügbar im Dev-Build" : undefined}
            className="flex items-center gap-2 px-3 py-1.5 bg-secondary border border-border rounded-lg text-sm text-foreground hover:bg-accent transition-colors disabled:opacity-50"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${checking ? "animate-spin" : ""}`} />
            {checking ? "Prüfe..." : "Jetzt prüfen"}
          </button>
        </div>

        {isDevBuild && (
          <p className="text-sm text-muted-foreground">
            Update-Prüfung ist nur für produktive Builds verfügbar (nicht für lokale Dev-Builds).
          </p>
        )}

        {check && (
          <div className="space-y-4">
            <div className={`flex items-center gap-3 px-4 py-3 rounded-lg border ${
              check.available
                ? "bg-yellow-500/10 border-yellow-500/20 text-yellow-500"
                : "bg-green-500/10 border-green-500/20 text-green-500"
            }`}>
              {check.available
                ? <AlertTriangle className="w-5 h-5 shrink-0" />
                : <CheckCircle className="w-5 h-5 shrink-0" />}
              <div>
                <p className="font-medium text-sm">
                  {check.available ? "Update verfügbar!" : "Bereits aktuell"}
                </p>
                <p className="text-xs opacity-80 mt-0.5">
                  Lokal: <code className="font-mono">{check.current_commit || "—"}</code>
                  {check.available && <> → GitHub: <code className="font-mono">{check.latest_commit}</code></>}
                  {" · "}Release: {new Date(check.published_at).toLocaleString("de-DE")}
                </p>
              </div>
            </div>

            {check.available && (
              <button
                onClick={runUpdate}
                disabled={updating}
                className="flex items-center gap-2 px-4 py-2.5 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {updating ? <RefreshCw className="w-4 h-4 animate-spin" /> : <ArrowUpCircle className="w-4 h-4" />}
                {updating ? "Wird aktualisiert..." : "Update installieren"}
              </button>
            )}
          </div>
        )}

        {!check && !isDevBuild && (
          <p className="text-sm text-muted-foreground">Klicke auf "Jetzt prüfen" um nach Updates zu suchen.</p>
        )}
      </div>

      {/* Update result */}
      {result && (
        <div className="bg-card border border-green-500/20 rounded-xl p-5">
          <h2 className="text-sm font-semibold text-green-500 flex items-center gap-2 mb-4">
            <CheckCircle className="w-4 h-4" />
            Update erfolgreich — {result.duration}
          </h2>
          <div className="grid grid-cols-2 gap-4 mb-3">
            <InfoItem icon={<GitCommit className="w-4 h-4" />} label="Vorher"
              value={<code className="font-mono text-muted-foreground">{result.previous_commit}</code>} />
            <InfoItem icon={<GitCommit className="w-4 h-4" />} label="Nachher"
              value={<code className="font-mono text-primary">{result.new_commit}</code>} />
          </div>
          <p className="text-sm text-muted-foreground">
            Die Dienste werden neu gestartet. Die Seite ist in wenigen Sekunden wieder erreichbar.
          </p>
        </div>
      )}

      {/* Auto-update toggle */}
      <div className="bg-card border border-border rounded-xl p-5">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className={`p-2 rounded-lg ${autoUpdate ? "bg-green-500/10" : "bg-secondary"}`}>
              <Bell className={`w-4 h-4 ${autoUpdate ? "text-green-500" : "text-muted-foreground"}`} />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">Automatische Updates</p>
              <p className="text-xs text-muted-foreground mt-0.5">
                {autoUpdate
                  ? "Updates werden täglich geprüft und automatisch eingespielt."
                  : "Updates werden täglich geprüft — Einspielen nur manuell."}
              </p>
            </div>
          </div>
          <button
            onClick={() => toggleAutoUpdate(!autoUpdate)}
            disabled={savingAuto || loading}
            aria-label="Auto-Update umschalten"
            className={`relative w-11 h-6 rounded-full transition-colors focus:outline-none disabled:opacity-50 ${
              autoUpdate ? "bg-green-500" : "bg-secondary border border-border"
            }`}
          >
            <span className={`absolute top-0.5 w-5 h-5 rounded-full bg-white shadow transition-transform ${
              autoUpdate ? "translate-x-5" : "translate-x-0.5"
            }`} />
          </button>
        </div>
        {autoUpdate && (
          <p className="mt-3 text-xs text-yellow-500 bg-yellow-500/10 border border-yellow-500/20 rounded-lg px-3 py-2">
            Automatische Updates starten den Dienst ohne Vorwarnung neu. Nur für produktive Systeme mit stabiler Internetverbindung empfohlen.
          </p>
        )}
      </div>

      {/* Manual fallback */}
      <div className="bg-card border border-border rounded-xl p-5">
        <h2 className="text-sm font-semibold text-foreground mb-2 flex items-center gap-2">
          <Terminal className="w-4 h-4" />
          Manuelles Update per SSH
        </h2>
        <pre className="bg-secondary text-foreground text-xs font-mono p-4 rounded-lg whitespace-pre-wrap">
{`# Binary ersetzen und Dienst neu starten
curl -fL https://github.com/Sirbuschi2003/ControlPanelVPS-/releases/download/latest/master \\
  -o /tmp/master.new
chmod +x /tmp/master.new
mv /tmp/master.new /opt/controlpanel/bin/master
systemctl restart cpanel-master cpanel-frontend`}
        </pre>
      </div>
    </div>
  );
}

// ── Tab: Server-Komponenten (Agent Updates) ────────────────────────────────────

function AgentTab() {
  const [servers, setServers] = useState<ServerType[]>([]);
  const [selectedServer, setSelectedServer] = useState("");
  const [info, setInfo] = useState<SystemInfo | null>(null);
  const [check, setCheck] = useState<AgentUpdateCheck | null>(null);
  const [result, setResult] = useState<AgentUpdateResult | null>(null);
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
      setInfo(await api.get<SystemInfo>(`/system/info?server_id=${selectedServer}`));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Fehler beim Laden");
    } finally {
      setLoading(false);
    }
  }, [selectedServer]);

  useEffect(() => { loadInfo(); }, [loadInfo]);

  async function checkUpdate() {
    setChecking(true);
    setError("");
    try {
      setCheck(await api.get<AgentUpdateCheck>(`/system/check-updates?server_id=${selectedServer}`));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Prüfung fehlgeschlagen");
    } finally {
      setChecking(false);
    }
  }

  async function runUpdate() {
    if (!confirm("Agent und Systempakete auf diesem Server aktualisieren? Der Agent wird kurz neu gestartet.")) return;
    setUpdating(true);
    setError("");
    setResult(null);
    try {
      const r = await api.post<AgentUpdateResult>(`/system/update?server_id=${selectedServer}`, {});
      setResult(r);
      await loadInfo();
      setCheck(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Update fehlgeschlagen");
    } finally {
      setUpdating(false);
    }
  }

  return (
    <div className="space-y-5">
      {error && <ErrorBanner msg={error} onClose={() => setError("")} />}

      {/* Server selector */}
      <div className="flex items-center gap-3">
        {servers.length > 1 && (
          <select
            value={selectedServer}
            onChange={(e) => { setSelectedServer(e.target.value); setCheck(null); setResult(null); }}
            className="px-3 py-2 bg-secondary border border-border rounded-lg text-foreground text-sm focus:outline-none"
          >
            {servers.map((s) => (
              <option key={s.id} value={s.id}>{s.name} ({s.ip_address})</option>
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

      {/* Agent version info */}
      <div className="bg-card border border-border rounded-xl p-5">
        <h2 className="text-sm font-semibold text-foreground mb-4 flex items-center gap-2">
          <Package className="w-4 h-4 text-primary" />
          Agent-Version
        </h2>
        {loading ? (
          <div className="space-y-2">{[1,2,3].map(i => <Skeleton key={i} className="h-5" style={{width:`${50+i*12}%`}} />)}</div>
        ) : info ? (
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            <InfoItem icon={<GitCommit className="w-4 h-4" />} label="Commit"
              value={<code className="text-primary font-mono">{info.commit}</code>} />
            <InfoItem icon={<Terminal className="w-4 h-4" />} label="OS" value={info.os} />
            <InfoItem icon={<Server className="w-4 h-4" />} label="Hostname" value={info.hostname} />
            <InfoItem icon={<ShieldCheck className="w-4 h-4" />} label="Branch" value={info.branch} />
            <InfoItem icon={<Clock className="w-4 h-4" />} label="Commit-Datum"
              value={info.commit_date ? info.commit_date.slice(0, 10) : "—"} />
          </div>
        ) : (
          <p className="text-muted-foreground text-sm">Kein Server ausgewählt oder Agent nicht erreichbar.</p>
        )}
      </div>

      {/* Update check */}
      <div className="bg-card border border-border rounded-xl p-5">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-semibold text-foreground flex items-center gap-2">
            <ArrowUpCircle className="w-4 h-4 text-primary" />
            Systemkomponenten prüfen
          </h2>
          <button
            onClick={checkUpdate}
            disabled={checking || !selectedServer}
            className="flex items-center gap-2 px-3 py-1.5 bg-secondary border border-border rounded-lg text-sm text-foreground hover:bg-accent transition-colors disabled:opacity-50"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${checking ? "animate-spin" : ""}`} />
            {checking ? "Prüfe..." : "Jetzt prüfen"}
          </button>
        </div>

        {check ? (
          <div className="space-y-4">
            <div className={`flex items-center gap-3 px-4 py-3 rounded-lg border ${
              check.available
                ? "bg-yellow-500/10 border-yellow-500/20 text-yellow-500"
                : "bg-green-500/10 border-green-500/20 text-green-500"
            }`}>
              {check.available
                ? <AlertTriangle className="w-5 h-5 shrink-0" />
                : <CheckCircle className="w-5 h-5 shrink-0" />}
              <div>
                <p className="font-medium text-sm">{check.available ? "Update verfügbar!" : "Bereits aktuell"}</p>
                <p className="text-xs opacity-80 mt-0.5">
                  Aktuell: <code className="font-mono">{check.current_commit}</code>
                  {check.available && <> → Neu: <code className="font-mono">{check.latest_commit}</code></>}
                </p>
              </div>
            </div>
            {check.available && (
              <button
                onClick={runUpdate}
                disabled={updating}
                className="flex items-center gap-2 px-4 py-2.5 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {updating ? <RefreshCw className="w-4 h-4 animate-spin" /> : <ArrowUpCircle className="w-4 h-4" />}
                {updating ? "Wird installiert..." : "Update installieren"}
              </button>
            )}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">Klicke auf "Jetzt prüfen" um nach Updates zu suchen.</p>
        )}
      </div>

      {/* Update result */}
      {result && (
        <div className="bg-card border border-green-500/20 rounded-xl p-5">
          <h2 className="text-sm font-semibold text-green-500 flex items-center gap-2 mb-4">
            <CheckCircle className="w-4 h-4" />
            Update erfolgreich — {result.duration}
          </h2>
          <div className="grid grid-cols-3 gap-4 mb-4">
            <InfoItem icon={<GitCommit className="w-4 h-4" />} label="Vorher"
              value={<code className="font-mono text-muted-foreground">{result.previous_commit}</code>} />
            <InfoItem icon={<GitCommit className="w-4 h-4" />} label="Nachher"
              value={<code className="font-mono text-primary">{result.new_commit}</code>} />
            <InfoItem icon={<Package className="w-4 h-4" />} label="Geänderte Dateien"
              value={String(result.changed_files)} />
          </div>
          {result.output && (
            <pre className="bg-secondary text-foreground text-xs font-mono p-4 rounded-lg overflow-auto max-h-60 whitespace-pre-wrap">
              {result.output}
            </pre>
          )}
        </div>
      )}
    </div>
  );
}

// ── Main page ──────────────────────────────────────────────────────────────────

type Tab = "panel" | "agent";

export default function UpdatesPage() {
  const [tab, setTab] = useState<Tab>("panel");

  return (
    <div className="space-y-6 max-w-4xl">
      <div>
        <h1 className="text-2xl font-bold text-foreground">Updates</h1>
        <p className="text-muted-foreground text-sm mt-1">Panel-Software und Server-Komponenten aktuell halten</p>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 p-1 bg-secondary rounded-xl w-fit">
        {([
          { id: "panel", label: "Panel-Software", icon: <ArrowUpCircle className="w-4 h-4" /> },
          { id: "agent", label: "Server-Komponenten", icon: <Package className="w-4 h-4" /> },
        ] as { id: Tab; label: string; icon: React.ReactNode }[]).map(({ id, label, icon }) => (
          <button
            key={id}
            onClick={() => setTab(id)}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
              tab === id
                ? "bg-card text-foreground shadow-sm"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            {icon}
            {label}
          </button>
        ))}
      </div>

      {tab === "panel" ? <PanelTab /> : <AgentTab />}
    </div>
  );
}
