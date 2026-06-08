"use client";

import { useEffect, useState } from "react";
import { Shield, Plus, Trash2, ToggleLeft, ToggleRight, RefreshCw, X, AlertCircle } from "lucide-react";
import { api, type FirewallRule, type Server } from "@/lib/api";

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

const defaultRules = [
  { port: "22", protocol: "tcp", comment: "SSH", action: "allow" },
  { port: "80", protocol: "tcp", comment: "HTTP", action: "allow" },
  { port: "443", protocol: "tcp", comment: "HTTPS", action: "allow" },
  { port: "25", protocol: "tcp", comment: "SMTP", action: "allow" },
];

export default function FirewallPage() {
  const [rules, setRules] = useState<FirewallRule[]>([]);
  const [servers, setServers] = useState<Server[]>([]);
  const [selectedServer, setSelectedServer] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showAdd, setShowAdd] = useState(false);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [reloading, setReloading] = useState(false);

  const [form, setForm] = useState({
    server_id: "",
    action: "allow" as "allow" | "deny",
    direction: "in" as "in" | "out",
    protocol: "tcp" as "tcp" | "udp" | "icmp" | "any",
    source: "",
    dest_port: "",
    comment: "",
  });

  async function load() {
    try {
      const [r, sv] = await Promise.all([
        api.get<FirewallRule[]>("/firewall"),
        api.get<Server[]>("/servers"),
      ]);
      setRules(r);
      setServers(sv);
      if (sv.length > 0 && !selectedServer) setSelectedServer(sv[0].id);
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
      await api.post("/firewall", form);
      setShowAdd(false);
      setForm({ server_id: "", action: "allow", direction: "in", protocol: "tcp", source: "", dest_port: "", comment: "" });
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSaving(false);
    }
  }

  async function handleToggle(id: string, enabled: boolean) {
    try {
      await api.post(`/firewall/${id}/toggle`, {});
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    }
  }

  async function handleDelete(id: string) {
    try {
      await api.delete(`/firewall/${id}`);
      setDeleteId(null);
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Löschen");
    }
  }

  async function handleReload() {
    if (!selectedServer) return;
    setReloading(true);
    try {
      await api.post(`/firewall/reload`, { server_id: selectedServer });
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Neuladen");
    } finally {
      setReloading(false);
    }
  }

  const serverName = (id: string) => servers.find((s) => s.id === id)?.name || id;
  const filteredRules = selectedServer ? rules.filter((r) => r.server_id === selectedServer) : rules;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Firewall</h1>
          <p className="text-muted-foreground text-sm mt-1">Netzwerkregeln verwalten</p>
        </div>
        <div className="flex gap-3">
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
          <button
            onClick={handleReload}
            disabled={reloading || !selectedServer}
            className="flex items-center gap-2 px-4 py-2 border border-border rounded-lg text-sm hover:bg-accent transition-colors text-muted-foreground hover:text-foreground disabled:opacity-50"
          >
            <RefreshCw className={`w-4 h-4 ${reloading ? "animate-spin" : ""}`} />
            Neu laden
          </button>
          <button
            onClick={() => { setForm({ ...form, server_id: selectedServer }); setShowAdd(true); }}
            className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors"
          >
            <Plus className="w-4 h-4" />
            Regel hinzufügen
          </button>
        </div>
      </div>

      {error && (
        <div className="flex items-center gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-lg text-destructive text-sm mb-4">
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          {error}
          <button onClick={() => setError("")} className="ml-auto"><X className="w-4 h-4" /></button>
        </div>
      )}

      {/* Default Rules Info */}
      <div className="bg-card border border-border rounded-xl p-4 mb-6">
        <h3 className="text-sm font-medium text-foreground mb-3">Empfohlene Standardregeln (Webserver)</h3>
        <div className="flex flex-wrap gap-2">
          {defaultRules.map((r) => (
            <div key={r.port} className="flex items-center gap-2 px-3 py-1.5 bg-background border border-border rounded-lg text-xs">
              <span className="text-green-400 font-medium">{r.action.toUpperCase()}</span>
              <span className="text-muted-foreground">{r.protocol.toUpperCase()}</span>
              <span className="text-foreground font-mono">{r.port}</span>
              <span className="text-muted-foreground">{r.comment}</span>
            </div>
          ))}
        </div>
      </div>

      {loading ? (
        <div className="space-y-3">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="bg-card border border-border rounded-xl p-4 flex gap-4">
              <Skeleton className="h-6 w-8" />
              <Skeleton className="h-6 w-16" />
              <Skeleton className="h-6 w-12" />
              <Skeleton className="h-6 w-32 ml-auto" />
            </div>
          ))}
        </div>
      ) : filteredRules.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
          <Shield className="w-12 h-12 mb-4 opacity-30" />
          <p className="font-medium">Keine Firewall-Regeln vorhanden</p>
          <p className="text-sm mt-1">Fügen Sie Ihre erste Regel hinzu</p>
        </div>
      ) : (
        <div className="bg-card border border-border rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-muted-foreground">
                <th className="text-left px-4 py-3 font-medium">#</th>
                <th className="text-left px-4 py-3 font-medium">Aktion</th>
                <th className="text-left px-4 py-3 font-medium">Richtung</th>
                <th className="text-left px-4 py-3 font-medium">Protokoll</th>
                <th className="text-left px-4 py-3 font-medium">Quelle</th>
                <th className="text-left px-4 py-3 font-medium">Port</th>
                <th className="text-left px-4 py-3 font-medium">Kommentar</th>
                <th className="text-left px-4 py-3 font-medium">Server</th>
                <th className="text-right px-4 py-3 font-medium">Aktionen</th>
              </tr>
            </thead>
            <tbody>
              {filteredRules.sort((a, b) => a.order - b.order).map((r) => (
                <tr key={r.id} className={`border-b border-border last:border-0 hover:bg-accent/50 transition-colors ${!r.enabled ? "opacity-50" : ""}`}>
                  <td className="px-4 py-3 text-muted-foreground">{r.order}</td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                      r.action === "allow" ? "bg-green-500/20 text-green-400" : "bg-red-500/20 text-red-400"
                    }`}>
                      {r.action === "allow" ? "Erlauben" : "Ablehnen"}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className="text-xs text-muted-foreground uppercase">{r.direction}</span>
                  </td>
                  <td className="px-4 py-3">
                    <span className="text-xs font-mono text-muted-foreground uppercase">{r.protocol}</span>
                  </td>
                  <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{r.source || "any"}</td>
                  <td className="px-4 py-3 font-mono text-xs text-foreground">{r.dest_port || "any"}</td>
                  <td className="px-4 py-3 text-muted-foreground">{r.comment || "-"}</td>
                  <td className="px-4 py-3 text-muted-foreground text-xs">{r.server_name || serverName(r.server_id)}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => handleToggle(r.id, r.enabled)}
                        className="text-muted-foreground hover:text-foreground transition-colors"
                        title={r.enabled ? "Deaktivieren" : "Aktivieren"}
                      >
                        {r.enabled ? <ToggleRight className="w-5 h-5 text-green-400" /> : <ToggleLeft className="w-5 h-5" />}
                      </button>
                      <button
                        onClick={() => setDeleteId(r.id)}
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
        <Modal title="Firewall-Regel hinzufügen" onClose={() => setShowAdd(false)}>
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
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">Aktion</label>
                <select
                  value={form.action}
                  onChange={(e) => setForm({ ...form, action: e.target.value as "allow" | "deny" })}
                  className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
                >
                  <option value="allow">Erlauben</option>
                  <option value="deny">Ablehnen</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">Richtung</label>
                <select
                  value={form.direction}
                  onChange={(e) => setForm({ ...form, direction: e.target.value as "in" | "out" })}
                  className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
                >
                  <option value="in">Eingehend (in)</option>
                  <option value="out">Ausgehend (out)</option>
                </select>
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Protokoll</label>
              <select
                value={form.protocol}
                onChange={(e) => setForm({ ...form, protocol: e.target.value as "tcp" | "udp" | "icmp" | "any" })}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
              >
                <option value="tcp">TCP</option>
                <option value="udp">UDP</option>
                <option value="icmp">ICMP</option>
                <option value="any">Alle</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Quell-IP (leer = alle)</label>
              <input
                type="text"
                value={form.source}
                onChange={(e) => setForm({ ...form, source: e.target.value })}
                placeholder="0.0.0.0/0"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Ziel-Port</label>
              <input
                type="text"
                value={form.dest_port}
                onChange={(e) => setForm({ ...form, dest_port: e.target.value })}
                placeholder="80 oder 8000:9000"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Kommentar</label>
              <input
                type="text"
                value={form.comment}
                onChange={(e) => setForm({ ...form, comment: e.target.value })}
                placeholder="z.B. HTTP Traffic"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button onClick={() => setShowAdd(false)} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">Abbrechen</button>
              <button
                onClick={handleAdd}
                disabled={saving || !form.server_id}
                className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {saving ? "Wird erstellt..." : "Hinzufügen"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Delete Confirm */}
      {deleteId && (
        <Modal title="Regel löschen" onClose={() => setDeleteId(null)}>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">Möchten Sie diese Firewall-Regel wirklich löschen?</p>
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
