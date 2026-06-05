"use client";

import { useEffect, useState } from "react";
import { Clock, Plus, Trash2, Edit, ToggleLeft, ToggleRight, X, AlertCircle, CheckCircle, XCircle } from "lucide-react";
import { api, type CronJob, type Server } from "@/lib/api";

function Skeleton({ className }: { className?: string }) {
  return <div className={`bg-secondary animate-pulse rounded ${className}`} />;
}

function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: React.ReactNode }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div className="bg-card border border-border rounded-xl w-full max-w-lg mx-4 shadow-xl">
        <div className="flex items-center justify-between p-4 border-b border-border">
          <h2 className="font-semibold text-foreground">{title}</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="w-5 h-5" />
          </button>
        </div>
        <div className="p-4">{children}</div>
      </div>
    </div>
  );
}

const cronPresets = [
  { label: "Jede Minute", value: "* * * * *", description: "Jede Minute" },
  { label: "Stündlich", value: "0 * * * *", description: "Jede Stunde um :00" },
  { label: "Täglich", value: "0 2 * * *", description: "Täglich um 02:00" },
  { label: "Wöchentlich", value: "0 2 * * 0", description: "Sonntags um 02:00" },
  { label: "Monatlich", value: "0 2 1 * *", description: "1. des Monats um 02:00" },
];

function parseCron(schedule: string): string {
  const preset = cronPresets.find((p) => p.value === schedule);
  if (preset) return preset.description;
  const parts = schedule.split(" ");
  if (parts.length !== 5) return schedule;
  return schedule;
}

function lastStatusIcon(status?: "success" | "failed" | "running") {
  if (!status) return null;
  if (status === "success") return <CheckCircle className="w-4 h-4 text-green-400" />;
  if (status === "failed") return <XCircle className="w-4 h-4 text-red-400" />;
  return <Clock className="w-4 h-4 text-blue-400 animate-pulse" />;
}

type FormData = {
  server_id: string;
  name: string;
  schedule: string;
  command: string;
  run_as_user: string;
};

export default function CronsPage() {
  const [crons, setCrons] = useState<CronJob[]>([]);
  const [servers, setServers] = useState<Server[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const emptyForm: FormData = { server_id: "", name: "", schedule: "0 2 * * *", command: "", run_as_user: "www-data" };
  const [form, setForm] = useState<FormData>(emptyForm);

  async function load() {
    try {
      const [c, sv] = await Promise.all([
        api.get<CronJob[]>("/crons"),
        api.get<Server[]>("/servers"),
      ]);
      setCrons(c);
      setServers(sv);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Laden");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { load(); }, []);

  function openAdd() {
    setForm(emptyForm);
    setEditId(null);
    setShowForm(true);
  }

  function openEdit(cron: CronJob) {
    setForm({
      server_id: cron.server_id,
      name: cron.name,
      schedule: cron.schedule,
      command: cron.command,
      run_as_user: cron.run_as_user,
    });
    setEditId(cron.id);
    setShowForm(true);
  }

  async function handleSave() {
    setSaving(true);
    try {
      if (editId) {
        await api.put(`/crons/${editId}`, form);
      } else {
        await api.post("/crons", form);
      }
      setShowForm(false);
      setEditId(null);
      setForm(emptyForm);
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSaving(false);
    }
  }

  async function handleToggle(id: string, enabled: boolean) {
    try {
      await api.put(`/crons/${id}`, { enabled: !enabled });
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    }
  }

  async function handleDelete(id: string) {
    try {
      await api.delete(`/crons/${id}`);
      setDeleteId(null);
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Löschen");
    }
  }

  const serverName = (id: string) => servers.find((s) => s.id === id)?.name || id;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Cron Jobs</h1>
          <p className="text-muted-foreground text-sm mt-1">Geplante Aufgaben verwalten</p>
        </div>
        <button
          onClick={openAdd}
          className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors"
        >
          <Plus className="w-4 h-4" />
          Cron Job hinzufügen
        </button>
      </div>

      {error && (
        <div className="flex items-center gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-lg text-destructive text-sm mb-4">
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          {error}
          <button onClick={() => setError("")} className="ml-auto"><X className="w-4 h-4" /></button>
        </div>
      )}

      {loading ? (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => <Skeleton key={i} className="h-14 w-full rounded-xl" />)}
        </div>
      ) : crons.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
          <Clock className="w-12 h-12 mb-4 opacity-30" />
          <p className="font-medium">Keine Cron Jobs vorhanden</p>
          <p className="text-sm mt-1">Fügen Sie Ihren ersten Cron Job hinzu</p>
        </div>
      ) : (
        <div className="bg-card border border-border rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-muted-foreground">
                <th className="text-left px-4 py-3 font-medium">Name</th>
                <th className="text-left px-4 py-3 font-medium">Zeitplan</th>
                <th className="text-left px-4 py-3 font-medium">Befehl</th>
                <th className="text-left px-4 py-3 font-medium">Benutzer</th>
                <th className="text-left px-4 py-3 font-medium">Letzter Lauf</th>
                <th className="text-left px-4 py-3 font-medium">Server</th>
                <th className="text-right px-4 py-3 font-medium">Aktionen</th>
              </tr>
            </thead>
            <tbody>
              {crons.map((c) => (
                <tr key={c.id} className={`border-b border-border last:border-0 hover:bg-accent/50 transition-colors ${!c.enabled ? "opacity-60" : ""}`}>
                  <td className="px-4 py-3 font-medium text-foreground">{c.name}</td>
                  <td className="px-4 py-3">
                    <div className="font-mono text-xs text-muted-foreground">{c.schedule}</div>
                    <div className="text-xs text-muted-foreground/70 mt-0.5">{parseCron(c.schedule)}</div>
                  </td>
                  <td className="px-4 py-3">
                    <code className="text-xs text-foreground bg-background px-1.5 py-0.5 rounded max-w-xs truncate block">{c.command}</code>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground font-mono text-xs">{c.run_as_user}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-1.5">
                      {lastStatusIcon(c.last_status)}
                      <span className="text-xs text-muted-foreground">
                        {c.last_run ? new Date(c.last_run).toLocaleString("de-DE") : "Noch nie"}
                      </span>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground text-xs">{c.server_name || serverName(c.server_id)}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => openEdit(c)}
                        className="text-muted-foreground hover:text-foreground transition-colors"
                        title="Bearbeiten"
                      >
                        <Edit className="w-4 h-4" />
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
                        title="Löschen"
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
      )}

      {/* Add/Edit Modal */}
      {showForm && (
        <Modal title={editId ? "Cron Job bearbeiten" : "Cron Job hinzufügen"} onClose={() => setShowForm(false)}>
          <div className="space-y-4">
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
                placeholder="z.B. Datenbank-Backup"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
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
                    className={`px-2 py-0.5 text-xs border rounded transition-colors ${
                      form.schedule === p.value
                        ? "border-primary text-primary bg-primary/10"
                        : "border-border text-muted-foreground hover:text-foreground hover:bg-accent"
                    }`}
                  >
                    {p.label}
                  </button>
                ))}
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Befehl</label>
              <input
                type="text"
                value={form.command}
                onChange={(e) => setForm({ ...form, command: e.target.value })}
                placeholder="/usr/bin/php /var/www/app/artisan schedule:run"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm font-mono text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Ausführen als Benutzer</label>
              <input
                type="text"
                value={form.run_as_user}
                onChange={(e) => setForm({ ...form, run_as_user: e.target.value })}
                placeholder="www-data"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm font-mono text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button onClick={() => setShowForm(false)} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">Abbrechen</button>
              <button
                onClick={handleSave}
                disabled={saving || !form.server_id || !form.name || !form.command}
                className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {saving ? "Wird gespeichert..." : editId ? "Speichern" : "Erstellen"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Delete Confirm */}
      {deleteId && (
        <Modal title="Cron Job löschen" onClose={() => setDeleteId(null)}>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">Möchten Sie diesen Cron Job wirklich löschen?</p>
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
