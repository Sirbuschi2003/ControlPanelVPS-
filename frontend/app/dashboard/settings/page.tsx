"use client";

import { useEffect, useState } from "react";
import { Settings, Save, Mail, FlaskConical, X, AlertCircle, CheckCircle, RefreshCw } from "lucide-react";
import { api } from "@/lib/api";

interface PanelSettings {
  panel_name: string;
  panel_timezone: string;
  smtp_host: string;
  smtp_port: string;
  smtp_user: string;
  smtp_pass: string;
  smtp_from: string;
  notify_email: string;
}

interface PanelInfo {
  version: string;
  uptime: string;
  database_status: string;
  go_version: string;
}

function Skeleton({ className }: { className?: string }) {
  return <div className={`bg-secondary animate-pulse rounded ${className}`} />;
}

const TIMEZONES = [
  "UTC",
  "Europe/Berlin",
  "Europe/Vienna",
  "Europe/Zurich",
  "Europe/London",
  "America/New_York",
  "America/Chicago",
  "America/Los_Angeles",
  "Asia/Tokyo",
  "Asia/Shanghai",
  "Australia/Sydney",
];

export default function SettingsPage() {
  const [settings, setSettings] = useState<PanelSettings>({
    panel_name: "",
    panel_timezone: "UTC",
    smtp_host: "",
    smtp_port: "587",
    smtp_user: "",
    smtp_pass: "",
    smtp_from: "",
    notify_email: "",
  });
  const [info, setInfo] = useState<PanelInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testingSmtp, setTestingSmtp] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [smtpTestResult, setSmtpTestResult] = useState<{ ok: boolean; message: string } | null>(null);

  useEffect(() => {
    async function load() {
      try {
        const [s, i] = await Promise.all([
          api.get<PanelSettings>("/settings"),
          api.get<PanelInfo>("/settings/info"),
        ]);
        setSettings(s);
        setInfo(i);
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : "Fehler beim Laden");
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  async function handleSave() {
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      await api.put("/settings", settings);
      setSuccess("Einstellungen wurden gespeichert.");
      setTimeout(() => setSuccess(""), 4000);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Speichern");
    } finally {
      setSaving(false);
    }
  }

  async function handleTestSmtp() {
    setTestingSmtp(true);
    setSmtpTestResult(null);
    try {
      await api.post("/settings/test-smtp", {
        smtp_host: settings.smtp_host,
        smtp_port: settings.smtp_port,
        smtp_user: settings.smtp_user,
        smtp_pass: settings.smtp_pass,
        smtp_from: settings.smtp_from,
        notify_email: settings.notify_email,
      });
      setSmtpTestResult({ ok: true, message: "Test-E-Mail erfolgreich gesendet!" });
    } catch (e: unknown) {
      setSmtpTestResult({ ok: false, message: e instanceof Error ? e.message : "Sendefehler" });
    } finally {
      setTestingSmtp(false);
    }
  }

  function field(label: string, key: keyof PanelSettings, type = "text", placeholder = "") {
    return (
      <div>
        <label className="block text-sm font-medium text-foreground mb-1">{label}</label>
        <input
          type={type}
          value={settings[key]}
          onChange={(e) => setSettings({ ...settings, [key]: e.target.value })}
          placeholder={placeholder}
          className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary"
        />
      </div>
    );
  }

  return (
    <div className="max-w-2xl">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Einstellungen</h1>
          <p className="text-muted-foreground text-sm mt-1">Panel-Konfiguration verwalten</p>
        </div>
        <button
          onClick={handleSave}
          disabled={saving || loading}
          className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
        >
          <Save className="w-4 h-4" />
          {saving ? "Wird gespeichert..." : "Speichern"}
        </button>
      </div>

      {error && (
        <div className="flex items-center gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-lg text-destructive text-sm mb-4">
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          {error}
          <button onClick={() => setError("")} className="ml-auto"><X className="w-4 h-4" /></button>
        </div>
      )}

      {success && (
        <div className="flex items-center gap-2 p-3 bg-green-500/10 border border-green-500/20 rounded-lg text-green-400 text-sm mb-4">
          <CheckCircle className="w-4 h-4 flex-shrink-0" />
          {success}
        </div>
      )}

      {loading ? (
        <div className="space-y-4">
          <Skeleton className="h-64 w-full rounded-xl" />
          <Skeleton className="h-64 w-full rounded-xl" />
        </div>
      ) : (
        <div className="space-y-6">
          {/* Panel Settings */}
          <div className="bg-card border border-border rounded-xl p-6">
            <div className="flex items-center gap-3 mb-5">
              <div className="w-8 h-8 bg-primary/10 border border-primary/20 rounded-lg flex items-center justify-center">
                <Settings className="w-4 h-4 text-primary" />
              </div>
              <h2 className="font-semibold text-foreground">Panel-Einstellungen</h2>
            </div>
            <div className="space-y-4">
              {field("Panel-Name", "panel_name", "text", "ControlPanel VPS")}
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">Zeitzone</label>
                <select
                  value={settings.panel_timezone}
                  onChange={(e) => setSettings({ ...settings, panel_timezone: e.target.value })}
                  className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
                >
                  {TIMEZONES.map((tz) => (
                    <option key={tz} value={tz}>{tz}</option>
                  ))}
                </select>
              </div>
            </div>
          </div>

          {/* SMTP Settings */}
          <div className="bg-card border border-border rounded-xl p-6">
            <div className="flex items-center gap-3 mb-5">
              <div className="w-8 h-8 bg-primary/10 border border-primary/20 rounded-lg flex items-center justify-center">
                <Mail className="w-4 h-4 text-primary" />
              </div>
              <h2 className="font-semibold text-foreground">E-Mail Benachrichtigungen</h2>
            </div>
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-3">
                {field("SMTP-Host", "smtp_host", "text", "smtp.example.com")}
                {field("SMTP-Port", "smtp_port", "text", "587")}
              </div>
              {field("SMTP-Benutzer", "smtp_user", "text", "user@example.com")}
              {field("SMTP-Passwort", "smtp_pass", "password")}
              {field("Absender-Adresse", "smtp_from", "email", "panel@example.com")}
              {field("Benachrichtigungs-E-Mail", "notify_email", "email", "admin@example.com")}
            </div>

            <div className="mt-5 pt-4 border-t border-border">
              <button
                onClick={handleTestSmtp}
                disabled={testingSmtp || !settings.smtp_host || !settings.notify_email}
                className="flex items-center gap-2 px-4 py-2 border border-border rounded-lg text-sm hover:bg-accent transition-colors text-muted-foreground hover:text-foreground disabled:opacity-50"
              >
                {testingSmtp
                  ? <><RefreshCw className="w-4 h-4 animate-spin" />Wird getestet...</>
                  : <><FlaskConical className="w-4 h-4" />SMTP testen</>
                }
              </button>
              {smtpTestResult && (
                <div className={`flex items-center gap-2 mt-3 p-3 rounded-lg text-sm border ${
                  smtpTestResult.ok
                    ? "bg-green-500/10 border-green-500/20 text-green-400"
                    : "bg-destructive/10 border-destructive/20 text-destructive"
                }`}>
                  {smtpTestResult.ok
                    ? <CheckCircle className="w-4 h-4 flex-shrink-0" />
                    : <AlertCircle className="w-4 h-4 flex-shrink-0" />}
                  {smtpTestResult.message}
                </div>
              )}
            </div>
          </div>

          {/* Panel Info */}
          {info && (
            <div className="bg-card border border-border rounded-xl p-6">
              <h2 className="font-semibold text-foreground mb-4">Panel-Informationen</h2>
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div className="space-y-3">
                  <div>
                    <div className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-0.5">Version</div>
                    <div className="font-mono text-foreground">{info.version || "v1.0.0"}</div>
                  </div>
                  <div>
                    <div className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-0.5">Go-Version</div>
                    <div className="font-mono text-foreground">{info.go_version || "-"}</div>
                  </div>
                </div>
                <div className="space-y-3">
                  <div>
                    <div className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-0.5">Laufzeit</div>
                    <div className="text-foreground">{info.uptime || "-"}</div>
                  </div>
                  <div>
                    <div className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-0.5">Datenbank</div>
                    <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${
                      info.database_status === "ok" ? "text-green-400" : "text-red-400"
                    }`}>
                      <span className={`w-1.5 h-1.5 rounded-full ${
                        info.database_status === "ok" ? "bg-green-400" : "bg-red-400"
                      }`} />
                      {info.database_status === "ok" ? "Verbunden" : info.database_status}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Save Button at bottom */}
          <div className="flex justify-end">
            <button
              onClick={handleSave}
              disabled={saving}
              className="flex items-center gap-2 px-6 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Save className="w-4 h-4" />
              {saving ? "Wird gespeichert..." : "Änderungen speichern"}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
