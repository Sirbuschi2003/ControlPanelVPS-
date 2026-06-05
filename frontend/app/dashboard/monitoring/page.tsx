"use client";

import { useState, useEffect, useCallback } from "react";
import { api, HealthReport, Alert } from "@/lib/api";
import {
  ShieldCheck,
  ShieldAlert,
  AlertTriangle,
  RefreshCw,
  Server,
  CheckCircle2,
} from "lucide-react";

function AlertBadge({ level }: { level: Alert["level"] }) {
  if (level === "critical")
    return (
      <span className="px-2 py-0.5 text-xs font-semibold rounded bg-red-900 text-red-200">
        Kritisch
      </span>
    );
  if (level === "warning")
    return (
      <span className="px-2 py-0.5 text-xs font-semibold rounded bg-yellow-900 text-yellow-200">
        Warnung
      </span>
    );
  return (
    <span className="px-2 py-0.5 text-xs font-semibold rounded bg-blue-900 text-blue-200">
      Info
    </span>
  );
}

function ScoreRing({ score }: { score: number }) {
  const color =
    score >= 80 ? "text-green-400" : score >= 50 ? "text-yellow-400" : "text-red-400";
  return (
    <div className={`text-5xl font-bold ${color}`}>
      {score}
      <span className="text-xl text-gray-400">/100</span>
    </div>
  );
}

export default function MonitoringPage() {
  const [reports, setReports] = useState<HealthReport[]>([]);
  const [servers, setServers] = useState<{ id: string; name: string }[]>([]);
  const [loading, setLoading] = useState(false);
  const [lastCheck, setLastCheck] = useState<Date | null>(null);

  const fetchServers = useCallback(async () => {
    try {
      const data = await api.get<{ id: string; name: string }[]>("/servers");
      setServers(data);
      return data;
    } catch {
      return [];
    }
  }, []);

  const runChecks = useCallback(async (srvList?: { id: string }[]) => {
    setLoading(true);
    const list = srvList ?? servers;
    const results = await Promise.allSettled(
      list.map((s) =>
        api.get<HealthReport>(`/monitoring/health?server_id=${s.id}`)
      )
    );
    const ok = results
      .filter((r): r is PromiseFulfilledResult<HealthReport> => r.status === "fulfilled")
      .map((r) => r.value);
    setReports(ok);
    setLastCheck(new Date());
    setLoading(false);
  }, [servers]);

  useEffect(() => {
    fetchServers().then((srvs) => runChecks(srvs));
    const iv = setInterval(() => runChecks(), 60_000);
    return () => clearInterval(iv);
  }, []);  // eslint-disable-line react-hooks/exhaustive-deps

  const allHealthy = reports.every((r) => r.healthy);
  const criticalCount = reports.flatMap((r) => r.alerts).filter((a) => a.level === "critical").length;
  const warningCount = reports.flatMap((r) => r.alerts).filter((a) => a.level === "warning").length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Monitoring</h1>
          <p className="text-gray-400 text-sm mt-1">
            {lastCheck
              ? `Letzter Check: ${lastCheck.toLocaleTimeString("de-DE")}`
              : "Noch kein Check durchgeführt"}
          </p>
        </div>
        <button
          onClick={() => runChecks()}
          disabled={loading}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white rounded-lg transition-colors"
        >
          <RefreshCw size={16} className={loading ? "animate-spin" : ""} />
          Jetzt prüfen
        </button>
      </div>

      {/* Summary bar */}
      <div className="grid grid-cols-3 gap-4">
        <div
          className={`rounded-xl p-4 flex items-center gap-3 ${
            allHealthy ? "bg-green-900/30 border border-green-700" : "bg-red-900/30 border border-red-700"
          }`}
        >
          {allHealthy ? (
            <ShieldCheck className="text-green-400" size={28} />
          ) : (
            <ShieldAlert className="text-red-400" size={28} />
          )}
          <div>
            <div className="font-semibold text-white">
              {allHealthy ? "Alle Systeme OK" : "Probleme erkannt"}
            </div>
            <div className="text-sm text-gray-400">
              {reports.length} Server geprüft
            </div>
          </div>
        </div>
        <div className="rounded-xl p-4 bg-gray-800 border border-gray-700">
          <div className="flex items-center gap-2 text-red-400 mb-1">
            <ShieldAlert size={18} />
            <span className="font-semibold">Kritisch</span>
          </div>
          <div className="text-3xl font-bold text-white">{criticalCount}</div>
        </div>
        <div className="rounded-xl p-4 bg-gray-800 border border-gray-700">
          <div className="flex items-center gap-2 text-yellow-400 mb-1">
            <AlertTriangle size={18} />
            <span className="font-semibold">Warnungen</span>
          </div>
          <div className="text-3xl font-bold text-white">{warningCount}</div>
        </div>
      </div>

      {/* Per-server reports */}
      {reports.length === 0 && !loading && (
        <div className="text-center py-16 text-gray-500">
          <Server size={40} className="mx-auto mb-3 opacity-40" />
          <p>Keine Server gefunden oder keine Verbindung zum Agenten.</p>
        </div>
      )}

      {reports.map((report) => {
        const serverName =
          servers.find((s) => s.id === report.server_id)?.name ?? report.server_id;
        return (
          <div key={report.server_id} className="bg-gray-800 rounded-xl border border-gray-700 p-5">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-3">
                <Server size={20} className="text-blue-400" />
                <h2 className="font-semibold text-white">{serverName}</h2>
                {report.healthy ? (
                  <span className="flex items-center gap-1 text-green-400 text-sm">
                    <CheckCircle2 size={14} /> Gesund
                  </span>
                ) : (
                  <span className="flex items-center gap-1 text-red-400 text-sm">
                    <ShieldAlert size={14} /> Problem
                  </span>
                )}
              </div>
              <ScoreRing score={report.score} />
            </div>

            {report.alerts.length === 0 ? (
              <div className="text-green-400 text-sm flex items-center gap-2">
                <CheckCircle2 size={16} />
                Keine Probleme gefunden
              </div>
            ) : (
              <div className="space-y-2">
                {report.alerts.map((alert, i) => (
                  <div
                    key={i}
                    className={`flex items-start justify-between rounded-lg p-3 ${
                      alert.level === "critical"
                        ? "bg-red-900/20 border border-red-800"
                        : "bg-yellow-900/20 border border-yellow-800"
                    }`}
                  >
                    <div className="flex items-start gap-3">
                      {alert.level === "critical" ? (
                        <ShieldAlert size={16} className="text-red-400 mt-0.5 shrink-0" />
                      ) : (
                        <AlertTriangle size={16} className="text-yellow-400 mt-0.5 shrink-0" />
                      )}
                      <div>
                        <div className="text-white text-sm font-medium">{alert.message}</div>
                        <div className="text-gray-400 text-xs mt-0.5">
                          Aktuell: <span className="font-mono">{alert.value}</span>
                          {" · "}Schwellwert: <span className="font-mono">{alert.threshold}</span>
                          {" · "}Kategorie: {alert.category}
                        </div>
                      </div>
                    </div>
                    <AlertBadge level={alert.level} />
                  </div>
                ))}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
