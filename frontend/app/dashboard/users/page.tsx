"use client";

import { useEffect, useState } from "react";
import { Users, Plus, Trash2, Edit, Key, Shield, X, AlertCircle, QrCode } from "lucide-react";
import { api, type User } from "@/lib/api";

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

function roleBadge(role: string) {
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
      role === "admin" ? "bg-blue-500/20 text-blue-400" : "bg-zinc-500/20 text-zinc-400"
    }`}>
      {role === "admin" ? "Admin" : "Betrachter"}
    </span>
  );
}

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  const [showAdd, setShowAdd] = useState(false);
  const [showEdit, setShowEdit] = useState<User | null>(null);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [showPassword, setShowPassword] = useState<User | null>(null);
  const [showTotp, setShowTotp] = useState<User | null>(null);

  const [addForm, setAddForm] = useState({ email: "", name: "", password: "", role: "viewer" });
  const [editForm, setEditForm] = useState({ name: "", role: "viewer" });
  const [newPassword, setNewPassword] = useState("");
  const [totpData, setTotpData] = useState<{ qr_url: string; secret: string } | null>(null);
  const [totpCode, setTotpCode] = useState("");
  const [totpLoading, setTotpLoading] = useState(false);

  async function load() {
    try {
      const u = await api.get<User[]>("/users");
      setUsers(u);
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
      await api.post("/users", addForm);
      setShowAdd(false);
      setAddForm({ email: "", name: "", password: "", role: "viewer" });
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSaving(false);
    }
  }

  async function handleEdit() {
    if (!showEdit) return;
    setSaving(true);
    try {
      await api.put(`/users/${showEdit.id}`, editForm);
      setShowEdit(null);
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: string) {
    try {
      await api.delete(`/users/${id}`);
      setDeleteId(null);
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Löschen");
    }
  }

  async function handleChangePassword() {
    if (!showPassword || !newPassword) return;
    setSaving(true);
    try {
      await api.post(`/users/${showPassword.id}/password`, { new_password: newPassword });
      setShowPassword(null);
      setNewPassword("");
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSaving(false);
    }
  }

  async function openTotp(user: User) {
    setShowTotp(user);
    setTotpData(null);
    setTotpCode("");
    setTotpLoading(true);
    try {
      const data = await api.get<{ qr_url: string; secret: string }>(`/users/${user.id}/totp/setup`);
      setTotpData(data);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Laden des TOTP");
    } finally {
      setTotpLoading(false);
    }
  }

  async function handleVerifyTotp() {
    if (!showTotp || !totpCode) return;
    setSaving(true);
    try {
      await api.post(`/users/${showTotp.id}/totp/verify`, { code: totpCode });
      setShowTotp(null);
      setTotpData(null);
      setTotpCode("");
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Ungültiger Code");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Benutzer</h1>
          <p className="text-muted-foreground text-sm mt-1">Panel-Benutzer verwalten</p>
        </div>
        <button
          onClick={() => setShowAdd(true)}
          className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors"
        >
          <Plus className="w-4 h-4" />
          Benutzer hinzufügen
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
          {[1, 2, 3].map((i) => <Skeleton key={i} className="h-16 w-full rounded-xl" />)}
        </div>
      ) : users.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
          <Users className="w-12 h-12 mb-4 opacity-30" />
          <p className="font-medium">Keine Benutzer vorhanden</p>
        </div>
      ) : (
        <div className="bg-card border border-border rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-muted-foreground">
                <th className="text-left px-4 py-3 font-medium">Benutzer</th>
                <th className="text-left px-4 py-3 font-medium">Rolle</th>
                <th className="text-left px-4 py-3 font-medium">2FA</th>
                <th className="text-left px-4 py-3 font-medium">Erstellt</th>
                <th className="text-right px-4 py-3 font-medium">Aktionen</th>
              </tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.id} className="border-b border-border last:border-0 hover:bg-accent/50 transition-colors">
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center text-primary text-sm font-semibold flex-shrink-0">
                        {u.name.charAt(0).toUpperCase()}
                      </div>
                      <div>
                        <div className="font-medium text-foreground">{u.name}</div>
                        <div className="text-xs text-muted-foreground">{u.email}</div>
                      </div>
                    </div>
                  </td>
                  <td className="px-4 py-3">{roleBadge(u.role)}</td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${
                      u.totp_enabled ? "bg-green-500/20 text-green-400" : "bg-zinc-500/20 text-zinc-400"
                    }`}>
                      <Shield className="w-3 h-3" />
                      {u.totp_enabled ? "Aktiv" : "Deaktiviert"}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground text-xs">
                    {new Date(u.created_at).toLocaleDateString("de-DE")}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => openTotp(u)}
                        className="text-muted-foreground hover:text-foreground transition-colors"
                        title="2FA einrichten"
                      >
                        <QrCode className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => { setShowPassword(u); setNewPassword(""); }}
                        className="text-muted-foreground hover:text-foreground transition-colors"
                        title="Passwort ändern"
                      >
                        <Key className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => { setShowEdit(u); setEditForm({ name: u.name, role: u.role }); }}
                        className="text-muted-foreground hover:text-foreground transition-colors"
                        title="Bearbeiten"
                      >
                        <Edit className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => setDeleteId(u.id)}
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
        <Modal title="Benutzer hinzufügen" onClose={() => setShowAdd(false)}>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Name</label>
              <input
                type="text"
                value={addForm.name}
                onChange={(e) => setAddForm({ ...addForm, name: e.target.value })}
                placeholder="Max Mustermann"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">E-Mail</label>
              <input
                type="email"
                value={addForm.email}
                onChange={(e) => setAddForm({ ...addForm, email: e.target.value })}
                placeholder="max@example.com"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Passwort</label>
              <input
                type="password"
                value={addForm.password}
                onChange={(e) => setAddForm({ ...addForm, password: e.target.value })}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Rolle</label>
              <select
                value={addForm.role}
                onChange={(e) => setAddForm({ ...addForm, role: e.target.value })}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
              >
                <option value="viewer">Betrachter</option>
                <option value="admin">Admin</option>
              </select>
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button onClick={() => setShowAdd(false)} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">Abbrechen</button>
              <button
                onClick={handleAdd}
                disabled={saving || !addForm.email || !addForm.name || !addForm.password}
                className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {saving ? "Wird erstellt..." : "Erstellen"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Edit Modal */}
      {showEdit && (
        <Modal title={`Benutzer bearbeiten – ${showEdit.name}`} onClose={() => setShowEdit(null)}>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Name</label>
              <input
                type="text"
                value={editForm.name}
                onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Rolle</label>
              <select
                value={editForm.role}
                onChange={(e) => setEditForm({ ...editForm, role: e.target.value })}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
              >
                <option value="viewer">Betrachter</option>
                <option value="admin">Admin</option>
              </select>
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button onClick={() => setShowEdit(null)} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">Abbrechen</button>
              <button
                onClick={handleEdit}
                disabled={saving || !editForm.name}
                className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {saving ? "Wird gespeichert..." : "Speichern"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Change Password Modal */}
      {showPassword && (
        <Modal title={`Passwort ändern – ${showPassword.name}`} onClose={() => setShowPassword(null)}>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Neues Passwort</label>
              <input
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder="Mindestens 8 Zeichen"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button onClick={() => setShowPassword(null)} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">Abbrechen</button>
              <button
                onClick={handleChangePassword}
                disabled={saving || !newPassword || newPassword.length < 8}
                className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {saving ? "Wird gespeichert..." : "Passwort ändern"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* TOTP Setup Modal */}
      {showTotp && (
        <Modal title={`2FA einrichten – ${showTotp.name}`} onClose={() => { setShowTotp(null); setTotpData(null); }}>
          <div className="space-y-4">
            {totpLoading ? (
              <div className="flex flex-col items-center py-6">
                <Skeleton className="w-48 h-48 mb-4" />
                <Skeleton className="h-4 w-40" />
              </div>
            ) : totpData ? (
              <>
                <p className="text-sm text-muted-foreground">
                  Scannen Sie den QR-Code mit Ihrer Authenticator-App (z.B. Google Authenticator, Authy).
                </p>
                <div className="flex flex-col items-center py-4">
                  {totpData.qr_url ? (
                    <img
                      src={totpData.qr_url}
                      alt="TOTP QR Code"
                      className="w-48 h-48 bg-white p-2 rounded-lg"
                    />
                  ) : (
                    <div className="w-48 h-48 bg-white/10 border border-border rounded-lg flex items-center justify-center">
                      <QrCode className="w-16 h-16 text-muted-foreground" />
                    </div>
                  )}
                  <div className="mt-3 text-center">
                    <p className="text-xs text-muted-foreground mb-1">Manueller Schlüssel:</p>
                    <code className="text-sm font-mono text-foreground bg-background px-3 py-1 rounded border border-border">
                      {totpData.secret}
                    </code>
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">
                    Bestätigungscode eingeben
                  </label>
                  <input
                    type="text"
                    value={totpCode}
                    onChange={(e) => setTotpCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
                    placeholder="123456"
                    maxLength={6}
                    className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm font-mono text-foreground placeholder:text-muted-foreground text-center text-lg tracking-widest"
                  />
                </div>
                <div className="flex justify-end gap-3 pt-2">
                  <button onClick={() => { setShowTotp(null); setTotpData(null); }} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">Abbrechen</button>
                  <button
                    onClick={handleVerifyTotp}
                    disabled={saving || totpCode.length !== 6}
                    className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
                  >
                    {saving ? "Wird verifiziert..." : "Verifizieren & Aktivieren"}
                  </button>
                </div>
              </>
            ) : (
              <div className="text-center py-6 text-muted-foreground">
                <p>Fehler beim Laden der TOTP-Daten</p>
              </div>
            )}
          </div>
        </Modal>
      )}

      {/* Delete Confirm */}
      {deleteId && (
        <Modal title="Benutzer löschen" onClose={() => setDeleteId(null)}>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">Möchten Sie diesen Benutzer wirklich löschen?</p>
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
