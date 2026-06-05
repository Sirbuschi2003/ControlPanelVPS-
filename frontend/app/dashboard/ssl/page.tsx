"use client";

import { useEffect, useState } from "react";
import { Lock, Plus, Trash2, RefreshCw, X, AlertCircle, CheckCircle, Clock, AlertTriangle } from "lucide-react";
import { api, type SSLCert, type Server } from "@/lib/api";

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

function expiryColor(expiresAt: string): string {
  const days = Math.floor((new Date(expiresAt).getTime() - Date.now()) / 86400000);
  if (days > 30) return "text-green-400";
  if (days > 7) return "text-yellow-400";
  return "text-red-400";
}

function expiryIcon(expiresAt: string) {
  const days = Math.floor((new Date(expiresAt).getTime() - Date.now()) / 86400000);
  if (days > 30) return <CheckCircle className="w-4 h-4 text-green-400" />;
  if (days > 7) return <Clock className="w-4 h-4 text-yellow-400" />;
  return <AlertTriangle className="w-4 h-4 text-red-400" />;
}

function statusBadge(status: SSLCert["status"]) {
  const map = {
    active: "bg-green-500/20 text-green-400",
    pending: "bg-yellow-500/20 text-yellow-400",
    expired: "bg-red-500/20 text-red-400",
    failed: "bg-red-500/20 text-red-400",
  };
  const labels = { active: "Aktiv", pending: "Ausstehend", expired: "Abgelaufen", failed: "Fehler" };
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${map[status]}`}>
      {labels[status]}
    </span>
  );
}

export default function SSLPage() {
  const [certs, setCerts] = useState<SSLCert[]>([]);
  const [servers, setServers] = useState<Server[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showAdd, setShowAdd] = useState(false);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [renewingId, setRenewingId] = useState<string | null>(null);

  const [form, setForm] = useState({
    server_id: "",
    domain: "",
    san_domains: "",
    email: "",
  });

  async function load() {
    try {
      const [cs, sv] = await Promise.all([
        api.get<SSLCert[]>("/ssl"),
        api.get<Server[]>("/servers"),
      ]);
      setCerts(cs);
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
      await api.post("/ssl", {
        ...form,
        san_domains: form.san_domains ? form.san_domains.split(",").map((s) => s.trim()) : [],
      });
      setShowAdd(false);
      setForm({ server_id: "", domain: "", san_domains: "", email: "" });
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSaving(false);
    }
  }

  async function handleRenew(id: string) {
    setRenewingId(id);
    try {
      await api.post(`/ssl/${id}/renew`, {});
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Erneuern");
    } finally {
      setRenewingId(null);
    }
  }

  async function handleDelete(id: string) {
    try {
      await api.delete(`/ssl/${id}`);
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
          <h1 className="text-2xl font-bold text-foreground">SSL/TLS</h1>
          <p className="text-muted-foreground text-sm mt-1">Zertifikatsverwaltung für Ihre Domains</p>
        </div>
        <button
          onClick={() => setShowAdd(true)}
          className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors"
        >
          <Plus className="w-4 h-4" />
          Zertifikat ausstellen
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
              <Skeleton className="h-6 w-32" />
              <Skeleton className="h-6 w-24 ml-auto" />
            </div>
          ))}
        </div>
      ) : certs.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
          <Lock className="w-12 h-12 mb-4 opacity-30" />
          <p className="font-medium">Keine Zertifikate vorhanden</p>
          <p className="text-sm mt-1">Stellen Sie Ihr erstes SSL-Zertifikat aus</p>
        </div>
      ) : (
        <div className="bg-card border border-border rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-muted-foreground">
                <th className="text-left px-4 py-3 font-medium">Domain</th>
                <th className="text-left px-4 py-3 font-medium">Status</th>
                <th className="text-left px-4 py-3 font-medium">Aussteller</th>
                <th className="text-left px-4 py-3 font-medium">Läuft ab</th>
                <th className="text-left px-4 py-3 font-medium">Auto-Erneuerung</th>
                <th className="text-left px-4 py-3 font-medium">Server</th>
                <th className="text-right px-4 py-3 font-medium">Aktionen</th>
              </tr>
            </thead>
            <tbody>
              {certs.map((c) => (
                <tr key={c.id} className="border-b border-border last:border-0 hover:bg-accent/50 transition-colors">
                  <td className="px-4 py-3">
                    <div className="font-medium text-foreground">{c.domain}</div>
                    {c.san_domains && c.san_domains.length > 0 && (
                      <div className="text-xs text-muted-foreground mt-0.5">{c.san_domains.join(", ")}</div>
                    )}
                  </td>
                  <td className="px-4 py-3">{statusBadge(c.status)}</td>
                  <td className="px-4 py-3 text-muted-foreground">{c.issuer || "Let's Encrypt"}</td>
                  <td className="px-4 py-3">
                    <div className={`flex items-center gap-1.5 ${expiryColor(c.expires_at)}`}>
                      {expiryIcon(c.expires_at)}
                      <span className="text-sm">
                        {new Date(c.expires_at).toLocaleDateString("de-DE")}
                      </span>
                    </div>
                    <div className="text-xs text-muted-foreground mt-0.5">
                      {Math.floor((new Date(c.expires_at).getTime() - Date.now()) / 86400000)} Tage
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                      c.auto_renew ? "bg-green-500/20 text-green-400" : "bg-zinc-500/20 text-zinc-400"
                    }`}>
                      {c.auto_renew ? "Aktiv" : "Deaktiviert"}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {c.server_name || serverName(c.server_id)}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => handleRenew(c.id)}
                        disabled={renewingId === c.id}
                        className="flex items-center gap-1 px-2 py-1 text-xs border border-border rounded hover:bg-accent transition-colors text-muted-foreground hover:text-foreground disabled:opacity-50"
                        title="Erneuern"
                      >
                        <RefreshCw className={`w-3 h-3 ${renewingId === c.id ? "animate-spin" : ""}`} />
                        Erneuern
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

      {/* Add Modal */}
      {showAdd && (
        <Modal title="SSL-Zertifikat ausstellen" onClose={() => setShowAdd(false)}>
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
              <label className="block text-sm font-medium text-foreground mb-1">SAN-Domains (kommagetrennt)</label>
              <input
                type="text"
                value={form.san_domains}
                onChange={(e) => setForm({ ...form, san_domains: e.target.value })}
                placeholder="www.example.com, mail.example.com"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">E-Mail (für Let's Encrypt)</label>
              <input
                type="email"
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
                placeholder="admin@example.com"
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
                disabled={saving || !form.server_id || !form.domain || !form.email}
                className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {saving ? "Wird ausgestellt..." : "Ausstellen"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Delete Confirm */}
      {deleteId && (
        <Modal title="Zertifikat löschen" onClose={() => setDeleteId(null)}>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Möchten Sie dieses Zertifikat wirklich löschen? Diese Aktion kann nicht rückgängig gemacht werden.
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
