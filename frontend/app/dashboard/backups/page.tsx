"use client";

import { useEffect, useState } from "react";
import { HardDrive, Plus, Trash2, Play, ToggleLeft, ToggleRight, X, AlertCircle } from "lucide-react";
import { api, type BackupConfig, type BackupJob, type Server, formatBytes } from "@/lib/api";

function Skeleton({ className }: { className?: string }) {
  return <div className={`bg-secondary animate-pulse rounded ${className}`} />;
}

function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: React.ReactNode }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div className="bg-card border border-border rounded-xl w-full max-w-2xl mx-4 shadow-xl max-h-[90vh] flex flex-col">
        <div className="flex items-center justify-between p-4 border-b border-border">
          <h2 className="font-semibold text-foreground">{title}</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="w-5 h-5" />
          </button>
        </div>
        <div className="p-4 overflow-y-auto">{children}</div>
      </div>
    </div>
  );
}

function jobStatusBadge(status: BackupJob["status"]) {
  const map = {
    running: "bg-blue-500/20 text-blue-400",
    success: "bg-green-500/20 text-green-400",
    failed: "bg-red-500/20 text-red-400",
    pending: "bg-yellow-500/20 text-yellow-400",
  };
  const labels = { running: "Läuft", success: "Erfolgreich", failed: "Fehler", pending: "Ausstehend" };
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${map[status]}`}>
      {labels[status]}
    </span>
  );
}

const storageTypeColors: Record<string, string> = {
  local: "bg-zinc-500/20 text-zinc-400",
  s3: "bg-orange-500/20 text-orange-400",
  sftp: "bg-blue-500/20 text-blue-400",
};

const cronPresets = [
  { label: "Jede Minute", value: "* * * * *" },
  { label: "Stündlich", value: "0 * * * *" },
  { label: "Täglich (02:00)", value: "0 2 * * *" },
  { label: "Wöchentlich (So 02:00)", value: "0 2 * * 0" },
  { label: "Monatlich (1. 02:00)", value: "0 2 1 * *" },
];

type Tab = "configs" | "jobs";

export default function BackupsPage() {
  const [tab, setTab] = useState<Tab>("configs");
  const [configs, setConfigs] = useState<BackupConfig[]>([]);
  const [jobs, setJobs] = useState<BackupJob[]>([]);
  const [servers, setServers] = useState<Server[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showAdd, setShowAdd] = useState(false);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [runningId, setRunningId] = useState<string | null>(null);

  const [form, setForm] = useState({
    server_id: "",
    name: "",
    storage_type: "local" as "local" | "s3" | "sftp",
    schedule: "0 2 * * *",
    retention_days: "30",
    include_paths: "/var/www,/etc",
    encrypt: false,
    s3_bucket: "",
    s3_region: "",
    s3_access_key: "",
    s3_secret_key: "",
    sftp_host: "",
    sftp_user: "",
    sftp_password: "",
    sftp_path: "",
  });

  async function load() {
    try {
      const [c, j, sv] = await Promise.all([
        api.get<BackupConfig[]>("/backups/configs"),
        api.get<BackupJob[]>("/backups/jobs"),
        api.get<Server[]>("/servers"),
      ]);
      setConfigs(c);
      setJobs(j);
      setServers(sv);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Laden");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { load(); }, []);

  async function handleAdd() {
    setSaving(true);
    try {
      await api.post("/backups/configs", {
        ...form,
        retention_days: parseInt(form.retention_days) || 30,
        include_paths: form.include_paths.split(",").map((s) => s.trim()),
      });
      setShowAdd(false);
      setForm({
        server_id: "", name: "", storage_type: "local", schedule: "0 2 * * *",
        retention_days: "30", include_paths: "/var/www,/etc", encrypt: false,
        s3_bucket: "", s3_region: "", s3_access_key: "", s3_secret_key: "",
        sftp_host: "", sftp_user: "", sftp_password: "", sftp_path: "",
      });
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSaving(false);
    }
  }

  async function handleToggle(id: string, enabled: boolean) {
    try {
      await api.put(`/backups/configs/${id}`, { enabled: !enabled });
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    }
  }

  async function handleDelete(id: string) {
    try {
      await api.delete(`/backups/configs/${id}`);
      setDeleteId(null);
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Löschen");
    }
  }

  async function handleRun(id: string) {
    setRunningId(id);
    try {
      await api.post(`/backups/configs/${id}/run`, {});
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Starten");
    } finally {
      setRunningId(null);
    }
  }

  function duration(job: BackupJob): string {
    if (!job.finished_at) return "-";
    const ms = new Date(job.finished_at).getTime() - new Date(job.started_at).getTime();
    const s = Math.floor(ms / 1000);
    if (s < 60) return `${s}s`;
    const m = Math.floor(s / 60);
    return `${m}m ${s % 60}s`;
  }

  const serverName = (id: string) => servers.find((s) => s.id === id)?.name || id;
  const configName = (id: string) => configs.find((c) => c.id === id)?.name || id;

  const tabs: { key: Tab; label: string }[] = [
    { key: "configs", label: "Konfigurationen" },
    { key: "jobs", label: "Verlauf" },
  ];

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Backups</h1>
          <p className="text-muted-foreground text-sm mt-1">Sicherungskonfigurationen und Verlauf</p>
        </div>
        {tab === "configs" && (
          <button
            onClick={() => setShowAdd(true)}
            className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors"
          >
            <Plus className="w-4 h-4" />
            Konfiguration erstellen
          </button>
        )}
      </div>

      {error && (
        <div className="flex items-center gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-lg text-destructive text-sm mb-4">
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          {error}
          <button onClick={() => setError("")} className="ml-auto"><X className="w-4 h-4" /></button>
        </div>
      )}

      <div className="flex border-b border-border mb-4">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              tab === t.key ? "border-primary text-primary" : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => <Skeleton key={i} className="h-16 w-full" />)}
        </div>
      ) : tab === "configs" ? (
        configs.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
            <HardDrive className="w-12 h-12 mb-4 opacity-30" />
            <p className="font-medium">Keine Backup-Konfigurationen vorhanden</p>
          </div>
        ) : (
          <div className="bg-card border border-border rounded-xl overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border text-muted-foreground">
                  <th className="text-left px-4 py-3 font-medium">Name</th>
                  <th className="text-left px-4 py-3 font-medium">Speicher</th>
                  <th className="text-left px-4 py-3 font-medium">Zeitplan</th>
                  <th className="text-left px-4 py-3 font-medium">Aufbewahrung</th>
                  <th className="text-left px-4 py-3 font-medium">Server</th>
                  <th className="text-right px-4 py-3 font-medium">Aktionen</th>
                </tr>
              </thead>
              <tbody>
                {configs.map((c) => (
                  <tr key={c.id} className={`border-b border-border last:border-0 hover:bg-accent/50 transition-colors ${!c.enabled ? "opacity-60" : ""}`}>
                    <td className="px-4 py-3">
                      <div className="font-medium text-foreground">{c.name}</div>
                      {c.encrypt && <div className="text-xs text-muted-foreground mt-0.5">Verschlüsselt</div>}
                    </td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${storageTypeColors[c.storage_type]}`}>
                        {c.storage_type.toUpperCase()}
                      </span>
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{c.schedule}</td>
                    <td className="px-4 py-3 text-muted-foreground">{c.retention_days} Tage</td>
                    <td className="px-4 py-3 text-muted-foreground">{c.server_name || serverName(c.server_id)}</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end gap-2">
                        <button
                          onClick={() => handleRun(c.id)}
                          disabled={runningId === c.id}
                          className="flex items-center gap-1 px-2 py-1 text-xs border border-border rounded hover:bg-accent transition-colors text-muted-foreground hover:text-foreground disabled:opacity-50"
                          title="Backup starten"
                        >
                          <Play className={`w-3 h-3 ${runningId === c.id ? "animate-pulse" : ""}`} />
                          Starten
                        </button>
                        <button
                          onClick={() => handleToggle(c.id, c.enabled)}
                          className="text-muted-foreground hover:text-foreground transition-colors"
                        >
                          {c.enabled ? <ToggleRight className="w-5 h-5 text-green-400" /> : <ToggleLeft className="w-5 h-5" />}
                        </button>
                        <button
                          onClick={() => setDeleteId(c.id)}
                          className="text-muted-foreground hover:text-destructive transition-colors"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )
      ) : (
        jobs.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
            <HardDrive className="w-12 h-12 mb-4 opacity-30" />
            <p className="font-medium">Kein Backup-Verlauf vorhanden</p>
          </div>
        ) : (
          <div className="bg-card border border-border rounded-xl overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border text-muted-foreground">
                  <th className="text-left px-4 py-3 font-medium">Konfiguration</th>
                  <th className="text-left px-4 py-3 font-medium">Status</th>
                  <th className="text-left px-4 py-3 font-medium">Größe</th>
                  <th className="text-left px-4 py-3 font-medium">Gestartet</th>
                  <th className="text-left px-4 py-3 font-medium">Dauer</th>
                  <th className="text-left px-4 py-3 font-medium">Fehler</th>
                </tr>
              </thead>
              <tbody>
                {jobs.map((j) => (
                  <tr key={j.id} className="border-b border-border last:border-0 hover:bg-accent/50 transition-colors">
                    <td className="px-4 py-3 font-medium text-foreground">{j.config_name || configName(j.config_id)}</td>
                    <td className="px-4 py-3">{jobStatusBadge(j.status)}</td>
                    <td className="px-4 py-3 text-muted-foreground">{j.size_bytes ? formatBytes(j.size_bytes) : "-"}</td>
                    <td className="px-4 py-3 text-muted-foreground">{new Date(j.started_at).toLocaleString("de-DE")}</td>
                    <td className="px-4 py-3 text-muted-foreground">{duration(j)}</td>
                    <td className="px-4 py-3 text-destructive text-xs">{j.error || "-"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )
      )}

      {/* Add Config Modal */}
      {showAdd && (
        <Modal title="Backup-Konfiguration erstellen" onClose={() => setShowAdd(false)}>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">Server</label>
                <select
                  value={form.server_id}
                  onChange={(e) => setForm({ ...form, server_id: e.target.value })}
                  className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
                >
                  <option value="">Server auswählen...</option>
                  {servers.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">Name</label>
                <input
                  type="text"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  placeholder="Mein Backup"
                  className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
                />
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Speichertyp</label>
              <select
                value={form.storage_type}
                onChange={(e) => setForm({ ...form, storage_type: e.target.value as "local" | "s3" | "sftp" })}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
              >
                <option value="local">Lokal</option>
                <option value="s3">Amazon S3</option>
                <option value="sftp">SFTP</option>
              </select>
            </div>

            {form.storage_type === "s3" && (
              <div className="space-y-3 p-3 bg-background border border-border rounded-lg">
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">S3-Einstellungen</p>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-medium text-foreground mb-1">Bucket</label>
                    <input type="text" value={form.s3_bucket} onChange={(e) => setForm({ ...form, s3_bucket: e.target.value })} placeholder="my-bucket" className="w-full bg-card border border-border rounded px-2 py-1.5 text-sm text-foreground placeholder:text-muted-foreground" />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-foreground mb-1">Region</label>
                    <input type="text" value={form.s3_region} onChange={(e) => setForm({ ...form, s3_region: e.target.value })} placeholder="eu-central-1" className="w-full bg-card border border-border rounded px-2 py-1.5 text-sm text-foreground placeholder:text-muted-foreground" />
                  </div>
                </div>
                <div>
                  <label className="block text-xs font-medium text-foreground mb-1">Access Key</label>
                  <input type="text" value={form.s3_access_key} onChange={(e) => setForm({ ...form, s3_access_key: e.target.value })} className="w-full bg-card border border-border rounded px-2 py-1.5 text-sm text-foreground" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-foreground mb-1">Secret Key</label>
                  <input type="password" value={form.s3_secret_key} onChange={(e) => setForm({ ...form, s3_secret_key: e.target.value })} className="w-full bg-card border border-border rounded px-2 py-1.5 text-sm text-foreground" />
                </div>
              </div>
            )}

            {form.storage_type === "sftp" && (
              <div className="space-y-3 p-3 bg-background border border-border rounded-lg">
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">SFTP-Einstellungen</p>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-medium text-foreground mb-1">Host</label>
                    <input type="text" value={form.sftp_host} onChange={(e) => setForm({ ...form, sftp_host: e.target.value })} placeholder="backup.example.com" className="w-full bg-card border border-border rounded px-2 py-1.5 text-sm text-foreground placeholder:text-muted-foreground" />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-foreground mb-1">Benutzer</label>
                    <input type="text" value={form.sftp_user} onChange={(e) => setForm({ ...form, sftp_user: e.target.value })} placeholder="backup" className="w-full bg-card border border-border rounded px-2 py-1.5 text-sm text-foreground placeholder:text-muted-foreground" />
                  </div>
                </div>
                <div>
                  <label className="block text-xs font-medium text-foreground mb-1">Passwort</label>
                  <input type="password" value={form.sftp_password} onChange={(e) => setForm({ ...form, sftp_password: e.target.value })} className="w-full bg-card border border-border rounded px-2 py-1.5 text-sm text-foreground" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-foreground mb-1">Pfad</label>
                  <input type="text" value={form.sftp_path} onChange={(e) => setForm({ ...form, sftp_path: e.target.value })} placeholder="/backups" className="w-full bg-card border border-border rounded px-2 py-1.5 text-sm text-foreground placeholder:text-muted-foreground" />
                </div>
              </div>
            )}

            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Zeitplan (Cron)</label>
              <input
                type="text"
                value={form.schedule}
                onChange={(e) => setForm({ ...form, schedule: e.target.value })}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm font-mono text-foreground"
              />
              <div className="flex flex-wrap gap-1 mt-2">
                {cronPresets.map((p) => (
                  <button
                    key={p.value}
                    type="button"
                    onClick={() => setForm({ ...form, schedule: p.value })}
                    className="px-2 py-0.5 text-xs border border-border rounded hover:bg-accent transition-colors text-muted-foreground hover:text-foreground"
                  >
                    {p.label}
                  </button>
                ))}
              </div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">Aufbewahrung (Tage)</label>
                <input
                  type="number"
                  value={form.retention_days}
                  onChange={(e) => setForm({ ...form, retention_days: e.target.value })}
                  className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
                />
              </div>
              <div className="flex items-end pb-1">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={form.encrypt}
                    onChange={(e) => setForm({ ...form, encrypt: e.target.checked })}
                    className="w-4 h-4 rounded border-border"
                  />
                  <span className="text-sm text-foreground">Verschlüsseln</span>
                </label>
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Einzuschließende Pfade (kommagetrennt)</label>
              <input
                type="text"
                value={form.include_paths}
                onChange={(e) => setForm({ ...form, include_paths: e.target.value })}
                placeholder="/var/www,/etc,/home"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button onClick={() => setShowAdd(false)} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">Abbrechen</button>
              <button
                onClick={handleAdd}
                disabled={saving || !form.server_id || !form.name}
                className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {saving ? "Wird erstellt..." : "Erstellen"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Delete Confirm */}
      {deleteId && (
        <Modal title="Konfiguration löschen" onClose={() => setDeleteId(null)}>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">Möchten Sie diese Backup-Konfiguration wirklich löschen?</p>
            <div className="flex justify-end gap-3">
              <button onClick={() => setDeleteId(null)} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">Abbrechen</button>
              <button onClick={() => handleDelete(deleteId)} className="px-4 py-2 text-sm bg-destructive text-white rounded-lg hover:bg-destructive/90 transition-colors">Löschen</button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
}
