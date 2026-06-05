"use client";

import { useEffect, useState } from "react";
import { Database, Plus, Trash2, Eye, EyeOff, Copy, X, AlertCircle } from "lucide-react";
import { api, type ManagedDatabase, type Server, formatBytes } from "@/lib/api";

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

export default function DatabasesPage() {
  const [databases, setDatabases] = useState<ManagedDatabase[]>([]);
  const [servers, setServers] = useState<Server[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showAdd, setShowAdd] = useState(false);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [passwordModal, setPasswordModal] = useState<{ id: string; db: ManagedDatabase } | null>(null);
  const [passwordValue, setPasswordValue] = useState("");
  const [passwordLoading, setPasswordLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [copied, setCopied] = useState("");

  const [form, setForm] = useState({
    server_id: "",
    name: "",
    db_type: "mysql" as "mysql" | "postgresql",
    db_user: "",
    db_password: "",
  });
  const [showFormPassword, setShowFormPassword] = useState(false);

  async function load() {
    try {
      const [dbs, sv] = await Promise.all([
        api.get<ManagedDatabase[]>("/databases"),
        api.get<Server[]>("/servers"),
      ]);
      setDatabases(dbs);
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
      await api.post("/databases", form);
      setShowAdd(false);
      setForm({ server_id: "", name: "", db_type: "mysql", db_user: "", db_password: "" });
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: string) {
    try {
      await api.delete(`/databases/${id}`);
      setDeleteId(null);
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Löschen");
    }
  }

  async function handleShowPassword(db: ManagedDatabase) {
    setPasswordModal({ id: db.id, db });
    setPasswordLoading(true);
    setPasswordValue("");
    setShowPassword(false);
    try {
      const result = await api.get<{ password: string }>(`/databases/${db.id}/password`);
      setPasswordValue(result.password);
    } catch (e: unknown) {
      setPasswordValue("Fehler: " + (e instanceof Error ? e.message : "Unbekannt"));
    } finally {
      setPasswordLoading(false);
    }
  }

  function getConnectionString(db: ManagedDatabase, password: string): string {
    const server = servers.find((s) => s.id === db.server_id);
    const host = server?.ip_address || "localhost";
    if (db.db_type === "mysql") {
      return `mysql://${db.db_user}:${password}@${host}:3306/${db.name}`;
    }
    return `postgresql://${db.db_user}:${password}@${host}:5432/${db.name}`;
  }

  function copyToClipboard(text: string, key: string) {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(key);
      setTimeout(() => setCopied(""), 2000);
    });
  }

  const serverName = (id: string) => servers.find((s) => s.id === id)?.name || id;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Datenbanken</h1>
          <p className="text-muted-foreground text-sm mt-1">Verwaltung aller Datenbanken</p>
        </div>
        <button
          onClick={() => setShowAdd(true)}
          className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors"
        >
          <Plus className="w-4 h-4" />
          Datenbank erstellen
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
              <Skeleton className="h-6 w-40" />
              <Skeleton className="h-6 w-20" />
              <Skeleton className="h-6 w-28" />
              <Skeleton className="h-6 w-16 ml-auto" />
            </div>
          ))}
        </div>
      ) : databases.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
          <Database className="w-12 h-12 mb-4 opacity-30" />
          <p className="font-medium">Keine Datenbanken vorhanden</p>
          <p className="text-sm mt-1">Erstellen Sie Ihre erste Datenbank</p>
        </div>
      ) : (
        <div className="bg-card border border-border rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-muted-foreground">
                <th className="text-left px-4 py-3 font-medium">Name</th>
                <th className="text-left px-4 py-3 font-medium">Typ</th>
                <th className="text-left px-4 py-3 font-medium">Benutzer</th>
                <th className="text-left px-4 py-3 font-medium">Größe</th>
                <th className="text-left px-4 py-3 font-medium">Server</th>
                <th className="text-right px-4 py-3 font-medium">Aktionen</th>
              </tr>
            </thead>
            <tbody>
              {databases.map((db) => (
                <tr key={db.id} className="border-b border-border last:border-0 hover:bg-accent/50 transition-colors">
                  <td className="px-4 py-3 font-medium text-foreground">{db.name}</td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                      db.db_type === "mysql"
                        ? "bg-orange-500/20 text-orange-400"
                        : "bg-blue-500/20 text-blue-400"
                    }`}>
                      {db.db_type === "mysql" ? "MySQL" : "PostgreSQL"}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground font-mono text-xs">{db.db_user}</td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {db.size_bytes ? formatBytes(db.size_bytes) : "-"}
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {db.server_name || serverName(db.server_id)}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => handleShowPassword(db)}
                        className="flex items-center gap-1 px-2 py-1 text-xs border border-border rounded hover:bg-accent transition-colors text-muted-foreground hover:text-foreground"
                        title="Passwort anzeigen"
                      >
                        <Eye className="w-3 h-3" />
                        Passwort
                      </button>
                      <button
                        onClick={() => setDeleteId(db.id)}
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
        <Modal title="Datenbank erstellen" onClose={() => setShowAdd(false)}>
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
              <label className="block text-sm font-medium text-foreground mb-1">Datenbankname</label>
              <input
                type="text"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder="my_database"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Datenbanktyp</label>
              <select
                value={form.db_type}
                onChange={(e) => setForm({ ...form, db_type: e.target.value as "mysql" | "postgresql" })}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
              >
                <option value="mysql">MySQL</option>
                <option value="postgresql">PostgreSQL</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Datenbankbenutzer</label>
              <input
                type="text"
                value={form.db_user}
                onChange={(e) => setForm({ ...form, db_user: e.target.value })}
                placeholder="db_user"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Passwort</label>
              <div className="relative">
                <input
                  type={showFormPassword ? "text" : "password"}
                  value={form.db_password}
                  onChange={(e) => setForm({ ...form, db_password: e.target.value })}
                  placeholder="Sicheres Passwort"
                  className="w-full bg-background border border-border rounded-lg px-3 py-2 pr-10 text-sm text-foreground placeholder:text-muted-foreground"
                />
                <button
                  type="button"
                  onClick={() => setShowFormPassword(!showFormPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                >
                  {showFormPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
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
                disabled={saving || !form.server_id || !form.name || !form.db_user || !form.db_password}
                className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {saving ? "Wird erstellt..." : "Erstellen"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Password Modal */}
      {passwordModal && (
        <Modal title={`Passwort – ${passwordModal.db.name}`} onClose={() => setPasswordModal(null)}>
          <div className="space-y-4">
            {passwordLoading ? (
              <Skeleton className="h-10 w-full" />
            ) : (
              <>
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Passwort</label>
                  <div className="flex gap-2">
                    <input
                      type={showPassword ? "text" : "password"}
                      readOnly
                      value={passwordValue}
                      className="flex-1 bg-background border border-border rounded-lg px-3 py-2 text-sm font-mono text-foreground"
                    />
                    <button
                      onClick={() => setShowPassword(!showPassword)}
                      className="px-3 border border-border rounded-lg hover:bg-accent transition-colors text-muted-foreground hover:text-foreground"
                    >
                      {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </button>
                    <button
                      onClick={() => copyToClipboard(passwordValue, "password")}
                      className="px-3 border border-border rounded-lg hover:bg-accent transition-colors text-muted-foreground hover:text-foreground"
                      title="Kopieren"
                    >
                      <Copy className="w-4 h-4" />
                    </button>
                  </div>
                  {copied === "password" && <p className="text-xs text-green-400 mt-1">Kopiert!</p>}
                </div>
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Connection String</label>
                  <div className="flex gap-2">
                    <input
                      readOnly
                      value={getConnectionString(passwordModal.db, passwordValue)}
                      className="flex-1 bg-background border border-border rounded-lg px-3 py-2 text-xs font-mono text-foreground"
                    />
                    <button
                      onClick={() => copyToClipboard(getConnectionString(passwordModal.db, passwordValue), "conn")}
                      className="px-3 border border-border rounded-lg hover:bg-accent transition-colors text-muted-foreground hover:text-foreground"
                      title="Kopieren"
                    >
                      <Copy className="w-4 h-4" />
                    </button>
                  </div>
                  {copied === "conn" && <p className="text-xs text-green-400 mt-1">Kopiert!</p>}
                </div>
              </>
            )}
            <div className="flex justify-end">
              <button
                onClick={() => setPasswordModal(null)}
                className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors"
              >
                Schließen
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Delete Confirm */}
      {deleteId && (
        <Modal title="Datenbank löschen" onClose={() => setDeleteId(null)}>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Möchten Sie diese Datenbank wirklich löschen? Alle Daten gehen verloren.
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
