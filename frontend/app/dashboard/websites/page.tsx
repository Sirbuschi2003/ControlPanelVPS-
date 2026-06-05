"use client";

import { useEffect, useState } from "react";
import { Globe, Plus, Trash2, ToggleLeft, ToggleRight, Lock, X, AlertCircle, CheckCircle } from "lucide-react";
import { api, type Website, type Server, type SSLCert } from "@/lib/api";

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

function StatusBadge({ enabled, trueLabel, falseLabel }: { enabled: boolean; trueLabel: string; falseLabel: string }) {
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
      enabled ? "bg-green-500/20 text-green-400" : "bg-zinc-500/20 text-zinc-400"
    }`}>
      {enabled ? trueLabel : falseLabel}
    </span>
  );
}

export default function WebsitesPage() {
  const [websites, setWebsites] = useState<Website[]>([]);
  const [servers, setServers] = useState<Server[]>([]);
  const [certs, setCerts] = useState<SSLCert[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showAdd, setShowAdd] = useState(false);
  const [showSSL, setShowSSL] = useState<string | null>(null);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const [form, setForm] = useState({
    server_id: "",
    domain: "",
    php_version: "8.2",
    document_root: "",
    aliases: "",
  });
  const [sslCertId, setSslCertId] = useState("");

  async function load() {
    try {
      const [ws, sv] = await Promise.all([
        api.get<Website[]>("/websites"),
        api.get<Server[]>("/servers"),
      ]);
      setWebsites(ws);
      setServers(sv);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Laden");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function loadCerts() {
    try {
      const cs = await api.get<SSLCert[]>("/ssl");
      setCerts(cs);
    } catch {
      // ignore
    }
  }

  async function handleAdd() {
    setSaving(true);
    try {
      await api.post("/websites", {
        ...form,
        aliases: form.aliases ? form.aliases.split(",").map((s) => s.trim()) : [],
      });
      setShowAdd(false);
      setForm({ server_id: "", domain: "", php_version: "8.2", document_root: "", aliases: "" });
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSaving(false);
    }
  }

  async function handleToggle(id: string, enabled: boolean) {
    try {
      await api.put(`/websites/${id}`, { enabled: !enabled });
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    }
  }

  async function handleDelete(id: string) {
    try {
      await api.delete(`/websites/${id}`);
      setDeleteId(null);
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    }
  }

  async function handleEnableSSL(id: string) {
    setSaving(true);
    try {
      await api.put(`/websites/${id}/ssl`, { cert_id: sslCertId });
      setShowSSL(null);
      setSslCertId("");
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSaving(false);
    }
  }

  const serverName = (id: string) => servers.find((s) => s.id === id)?.name || id;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Websites</h1>
          <p className="text-muted-foreground text-sm mt-1">Verwaltung aller gehosteten Websites</p>
        </div>
        <button
          onClick={() => setShowAdd(true)}
          className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors"
        >
          <Plus className="w-4 h-4" />
          Website hinzufügen
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
          {[1, 2, 3].map((i) => (
            <div key={i} className="bg-card border border-border rounded-xl p-4 flex gap-4">
              <Skeleton className="h-6 w-48" />
              <Skeleton className="h-6 w-20" />
              <Skeleton className="h-6 w-20" />
              <Skeleton className="h-6 w-24 ml-auto" />
            </div>
          ))}
        </div>
      ) : websites.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
          <Globe className="w-12 h-12 mb-4 opacity-30" />
          <p className="font-medium">Keine Websites vorhanden</p>
          <p className="text-sm mt-1">Fügen Sie Ihre erste Website hinzu</p>
        </div>
      ) : (
        <div className="bg-card border border-border rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-muted-foreground">
                <th className="text-left px-4 py-3 font-medium">Domain</th>
                <th className="text-left px-4 py-3 font-medium">PHP</th>
                <th className="text-left px-4 py-3 font-medium">SSL</th>
                <th className="text-left px-4 py-3 font-medium">Status</th>
                <th className="text-left px-4 py-3 font-medium">Server</th>
                <th className="text-right px-4 py-3 font-medium">Aktionen</th>
              </tr>
            </thead>
            <tbody>
              {websites.map((w) => (
                <tr key={w.id} className="border-b border-border last:border-0 hover:bg-accent/50 transition-colors">
                  <td className="px-4 py-3">
                    <div className="font-medium text-foreground">{w.domain}</div>
                    {w.aliases && w.aliases.length > 0 && (
                      <div className="text-xs text-muted-foreground mt-0.5">{w.aliases.join(", ")}</div>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-blue-500/20 text-blue-400">
                      PHP {w.php_version}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge enabled={w.ssl_enabled} trueLabel="Aktiv" falseLabel="Kein SSL" />
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge enabled={w.enabled} trueLabel="Aktiv" falseLabel="Deaktiviert" />
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {w.server_name || serverName(w.server_id)}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center justify-end gap-2">
                      {!w.ssl_enabled && (
                        <button
                          onClick={async () => { await loadCerts(); setShowSSL(w.id); }}
                          className="flex items-center gap-1 px-2 py-1 text-xs border border-border rounded hover:bg-accent transition-colors text-muted-foreground hover:text-foreground"
                          title="SSL aktivieren"
                        >
                          <Lock className="w-3 h-3" />
                          SSL
                        </button>
                      )}
                      <button
                        onClick={() => handleToggle(w.id, w.enabled)}
                        className="text-muted-foreground hover:text-foreground transition-colors"
                        title={w.enabled ? "Deaktivieren" : "Aktivieren"}
                      >
                        {w.enabled ? <ToggleRight className="w-5 h-5 text-green-400" /> : <ToggleLeft className="w-5 h-5" />}
                      </button>
                      <button
                        onClick={() => setDeleteId(w.id)}
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

      {/* Add Modal */}
      {showAdd && (
        <Modal title="Website hinzufügen" onClose={() => setShowAdd(false)}>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Server</label>
              <select
                value={form.server_id}
                onChange={(e) => setForm({ ...form, server_id: e.target.value })}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
              >
                <option value="">Server auswählen...</option>
                {servers.map((s) => (
                  <option key={s.id} value={s.id}>{s.name} ({s.ip_address})</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Domain</label>
              <input
                type="text"
                value={form.domain}
                onChange={(e) => setForm({ ...form, domain: e.target.value })}
                placeholder="example.com"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">PHP Version</label>
              <select
                value={form.php_version}
                onChange={(e) => setForm({ ...form, php_version: e.target.value })}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
              >
                {["7.4", "8.1", "8.2", "8.3"].map((v) => (
                  <option key={v} value={v}>PHP {v}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Document Root</label>
              <input
                type="text"
                value={form.document_root}
                onChange={(e) => setForm({ ...form, document_root: e.target.value })}
                placeholder="/var/www/example.com/public"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Aliases (kommagetrennt)</label>
              <input
                type="text"
                value={form.aliases}
                onChange={(e) => setForm({ ...form, aliases: e.target.value })}
                placeholder="www.example.com, mail.example.com"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button
                onClick={() => setShowAdd(false)}
                className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={handleAdd}
                disabled={saving || !form.server_id || !form.domain}
                className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {saving ? "Wird erstellt..." : "Hinzufügen"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* SSL Modal */}
      {showSSL && (
        <Modal title="SSL aktivieren" onClose={() => setShowSSL(null)}>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">Wählen Sie ein SSL-Zertifikat für diese Website aus.</p>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Zertifikat</label>
              <select
                value={sslCertId}
                onChange={(e) => setSslCertId(e.target.value)}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
              >
                <option value="">Zertifikat auswählen...</option>
                {certs.map((c) => (
                  <option key={c.id} value={c.id}>{c.domain} ({c.status})</option>
                ))}
              </select>
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button
                onClick={() => setShowSSL(null)}
                className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={() => handleEnableSSL(showSSL)}
                disabled={saving || !sslCertId}
                className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {saving ? "Wird aktiviert..." : "SSL aktivieren"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Delete Confirm */}
      {deleteId && (
        <Modal title="Website löschen" onClose={() => setDeleteId(null)}>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Möchten Sie diese Website wirklich löschen? Diese Aktion kann nicht rückgängig gemacht werden.
            </p>
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setDeleteId(null)}
                className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={() => handleDelete(deleteId)}
                className="px-4 py-2 text-sm bg-destructive text-white rounded-lg hover:bg-destructive/90 transition-colors"
              >
                Löschen
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
}
