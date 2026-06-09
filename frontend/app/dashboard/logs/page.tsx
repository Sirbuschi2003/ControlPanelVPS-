"use client";

import { useEffect, useState, useRef, useCallback } from "react";
import { FileText, RefreshCw, X, AlertCircle, ChevronDown, Server } from "lucide-react";
import { api, type Server as ServerType } from "@/lib/api";

function Skeleton({ className }: { className?: string }) {
  return <div className={`bg-secondary animate-pulse rounded ${className}`} />;
}

const LINE_OPTIONS = [50, 100, 200, 500, 1000];

const LOG_LABELS: Record<string, string> = {
  "nginx-access": "Nginx Access",
  "nginx-error":  "Nginx Error",
  syslog:         "Syslog",
  auth:           "Auth",
  mail:           "Mail",
  mysql:          "MySQL",
  fail2ban:       "Fail2ban",
  dpkg:           "DPKG",
};

export default function LogsPage() {
  const [servers, setServers] = useState<ServerType[]>([]);
  const [selectedServer, setSelectedServer] = useState("");
  const [availableLogs, setAvailableLogs] = useState<string[]>([]);
  const [selectedLog, setSelectedLog] = useState("");
  const [lines, setLines] = useState(200);
  const [logContent, setLogContent] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [logsLoading, setLogsLoading] = useState(false);
  const [error, setError] = useState("");
  const [autoRefresh, setAutoRefresh] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    api.get<ServerType[]>("/servers")
      .then(setServers)
      .catch((e: unknown) => setError(e instanceof Error ? e.message : "Fehler beim Laden der Server"));
  }, []);

  useEffect(() => {
    if (!selectedServer) {
      setAvailableLogs([]);
      setSelectedLog("");
      return;
    }
    setLogsLoading(true);
    api.get<string[]>(`/logs?server_id=${selectedServer}`)
      .then((logs) => {
        setAvailableLogs(logs);
        if (logs.length > 0) setSelectedLog(logs[0]);
      })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : "Fehler beim Laden der Logs"))
      .finally(() => setLogsLoading(false));
  }, [selectedServer]);

  const fetchLog = useCallback(async () => {
    if (!selectedServer || !selectedLog) return;
    setLoading(true);
    try {
      const data = await api.get<string[]>(`/logs/${selectedServer}/${selectedLog}?lines=${lines}`);
      setLogContent(data);
      setError("");
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Laden des Logs");
    } finally {
      setLoading(false);
    }
  }, [selectedServer, selectedLog, lines]);

  useEffect(() => {
    if (selectedLog) fetchLog();
  }, [fetchLog]);

  useEffect(() => {
    if (autoRefresh && selectedLog) {
      intervalRef.current = setInterval(fetchLog, 5000);
    } else {
      if (intervalRef.current) clearInterval(intervalRef.current);
    }
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [autoRefresh, fetchLog]);

  useEffect(() => {
    if (logContent.length > 0) {
      bottomRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [logContent]);

  const logLabel = (name: string) => LOG_LABELS[name] ?? name;

  return (
    <div className="flex flex-col h-full gap-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Logs</h1>
          <p className="text-muted-foreground text-sm mt-1">Server-Logs in Echtzeit einsehen</p>
        </div>
      </div>

      {error && (
        <div className="flex items-center gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-lg text-destructive text-sm">
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          {error}
          <button onClick={() => setError("")} className="ml-auto"><X className="w-4 h-4" /></button>
        </div>
      )}

      {/* Controls */}
      <div className="flex flex-wrap gap-3 items-end">
        {/* Server select */}
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-muted-foreground">Server</label>
          <div className="relative">
            <Server className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground pointer-events-none" />
            <select
              value={selectedServer}
              onChange={(e) => setSelectedServer(e.target.value)}
              className="pl-8 pr-8 py-2 bg-background border border-border rounded-lg text-sm text-foreground appearance-none min-w-[200px]"
            >
              <option value="">Server auswählen...</option>
              {servers.map((s) => (
                <option key={s.id} value={s.id}>{s.name} ({s.ip_address})</option>
              ))}
            </select>
            <ChevronDown className="absolute right-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground pointer-events-none" />
          </div>
        </div>

        {/* Log select */}
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-muted-foreground">Log-Datei</label>
          <div className="relative">
            <FileText className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground pointer-events-none" />
            <select
              value={selectedLog}
              onChange={(e) => setSelectedLog(e.target.value)}
              disabled={!selectedServer || logsLoading}
              className="pl-8 pr-8 py-2 bg-background border border-border rounded-lg text-sm text-foreground appearance-none min-w-[180px] disabled:opacity-50"
            >
              {logsLoading ? (
                <option>Lade...</option>
              ) : availableLogs.length === 0 ? (
                <option value="">Keine Logs verfügbar</option>
              ) : (
                availableLogs.map((l) => (
                  <option key={l} value={l}>{logLabel(l)}</option>
                ))
              )}
            </select>
            <ChevronDown className="absolute right-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground pointer-events-none" />
          </div>
        </div>

        {/* Lines select */}
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-muted-foreground">Zeilen</label>
          <div className="relative">
            <select
              value={lines}
              onChange={(e) => setLines(Number(e.target.value))}
              className="px-3 pr-8 py-2 bg-background border border-border rounded-lg text-sm text-foreground appearance-none"
            >
              {LINE_OPTIONS.map((n) => (
                <option key={n} value={n}>{n} Zeilen</option>
              ))}
            </select>
            <ChevronDown className="absolute right-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground pointer-events-none" />
          </div>
        </div>

        {/* Refresh button */}
        <button
          onClick={fetchLog}
          disabled={loading || !selectedLog}
          className="flex items-center gap-2 px-3 py-2 border border-border rounded-lg text-sm text-foreground hover:bg-accent transition-colors disabled:opacity-50"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${loading ? "animate-spin" : ""}`} />
          Aktualisieren
        </button>

        {/* Auto-refresh toggle */}
        <label className="flex items-center gap-2 px-3 py-2 border border-border rounded-lg text-sm cursor-pointer hover:bg-accent transition-colors select-none">
          <input
            type="checkbox"
            checked={autoRefresh}
            onChange={(e) => setAutoRefresh(e.target.checked)}
            className="accent-primary"
          />
          <span className="text-foreground">Auto-Refresh (5s)</span>
        </label>
      </div>

      {/* Log viewer */}
      <div className="flex-1 bg-zinc-950 border border-border rounded-xl overflow-hidden flex flex-col min-h-[400px]">
        {/* Header bar */}
        <div className="flex items-center justify-between px-4 py-2 bg-zinc-900 border-b border-border">
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-red-500" />
            <div className="w-3 h-3 rounded-full bg-yellow-500" />
            <div className="w-3 h-3 rounded-full bg-green-500" />
          </div>
          <span className="text-xs text-zinc-400 font-mono">
            {selectedLog ? `${logLabel(selectedLog)}  —  ${logContent.length} Zeilen` : "Kein Log ausgewählt"}
          </span>
          {autoRefresh && (
            <span className="flex items-center gap-1 text-xs text-green-400">
              <span className="w-1.5 h-1.5 rounded-full bg-green-400 animate-pulse" />
              Live
            </span>
          )}
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4 font-mono text-xs text-zinc-300 leading-relaxed">
          {!selectedServer || !selectedLog ? (
            <div className="flex flex-col items-center justify-center h-full text-zinc-600">
              <FileText className="w-10 h-10 mb-3 opacity-40" />
              <p>Server und Log-Datei auswählen</p>
            </div>
          ) : loading && logContent.length === 0 ? (
            <div className="space-y-2">
              {Array.from({ length: 12 }).map((_, i) => (
                <Skeleton key={i} className="h-4 w-full opacity-20" />
              ))}
            </div>
          ) : logContent.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full text-zinc-600">
              <FileText className="w-10 h-10 mb-3 opacity-40" />
              <p>Log-Datei ist leer</p>
            </div>
          ) : (
            <>
              {logContent.map((line, i) => (
                <div
                  key={i}
                  className={`whitespace-pre-wrap break-all ${
                    /error|critical|fatal/i.test(line)
                      ? "text-red-400"
                      : /warn/i.test(line)
                      ? "text-yellow-400"
                      : ""
                  }`}
                >
                  <span className="text-zinc-600 select-none mr-3">{String(i + 1).padStart(4, " ")}</span>
                  {line}
                </div>
              ))}
              <div ref={bottomRef} />
            </>
          )}
        </div>
      </div>
    </div>
  );
}
