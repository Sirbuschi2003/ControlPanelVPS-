"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { UserCircle, KeyRound, Save } from "lucide-react";
import { api, type User } from "@/lib/api";

export default function ProfilePage() {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [name, setName] = useState("");
  const [nameStatus, setNameStatus] = useState<"idle" | "saving" | "saved" | "error">("idle");

  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [pwStatus, setPwStatus] = useState<"idle" | "saving" | "saved" | "error">("idle");
  const [pwError, setPwError] = useState("");

  useEffect(() => {
    api.get<User>("/auth/me")
      .then((u) => { setUser(u); setName(u.name); })
      .catch(() => router.push("/login"));
  }, [router]);

  async function saveName(e: React.FormEvent) {
    e.preventDefault();
    if (!user || !name.trim()) return;
    setNameStatus("saving");
    try {
      await api.put(`/users/${user.id}`, { name: name.trim(), role: user.role });
      setNameStatus("saved");
      setTimeout(() => setNameStatus("idle"), 2000);
    } catch {
      setNameStatus("error");
    }
  }

  async function changePassword(e: React.FormEvent) {
    e.preventDefault();
    setPwError("");
    if (newPassword.length < 12) {
      setPwError("Passwort muss mindestens 12 Zeichen lang sein.");
      return;
    }
    if (newPassword !== confirmPassword) {
      setPwError("Passwörter stimmen nicht überein.");
      return;
    }
    if (!user) return;
    setPwStatus("saving");
    try {
      await api.post(`/users/${user.id}/password`, { new_password: newPassword });
      setPwStatus("saved");
      setNewPassword("");
      setConfirmPassword("");
      setTimeout(() => setPwStatus("idle"), 2000);
    } catch {
      setPwStatus("error");
      setPwError("Fehler beim Ändern des Passworts.");
    }
  }

  if (!user) return null;

  return (
    <div className="max-w-xl space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-foreground">Mein Profil</h1>
        <p className="text-sm text-muted-foreground mt-1">Profilinformationen und Passwort verwalten</p>
      </div>

      {/* Profile info */}
      <div className="rounded-xl border border-border bg-card p-6 space-y-4">
        <div className="flex items-center gap-3 pb-3 border-b border-border">
          <UserCircle className="w-5 h-5 text-primary" />
          <span className="font-medium text-foreground">Profilinformationen</span>
        </div>
        <form onSubmit={saveName} className="space-y-4">
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Name</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground outline-none focus:ring-2 focus:ring-primary/30"
            />
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">E-Mail</label>
            <input
              value={user.email}
              disabled
              className="w-full rounded-lg border border-border bg-muted px-3 py-2 text-sm text-muted-foreground cursor-not-allowed"
            />
            <p className="text-xs text-muted-foreground">E-Mail kann nicht geändert werden.</p>
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Rolle</label>
            <input
              value={user.role}
              disabled
              className="w-full rounded-lg border border-border bg-muted px-3 py-2 text-sm text-muted-foreground cursor-not-allowed capitalize"
            />
          </div>
          <button
            type="submit"
            disabled={nameStatus === "saving"}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 disabled:opacity-50 transition-colors"
          >
            <Save className="w-4 h-4" />
            {nameStatus === "saving" ? "Speichern…" : nameStatus === "saved" ? "Gespeichert!" : "Speichern"}
          </button>
          {nameStatus === "error" && <p className="text-sm text-destructive">Fehler beim Speichern.</p>}
        </form>
      </div>

      {/* Password change */}
      <div id="password" className="rounded-xl border border-border bg-card p-6 space-y-4 scroll-mt-6">
        <div className="flex items-center gap-3 pb-3 border-b border-border">
          <KeyRound className="w-5 h-5 text-primary" />
          <span className="font-medium text-foreground">Passwort ändern</span>
        </div>
        <form onSubmit={changePassword} className="space-y-4">
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Neues Passwort</label>
            <input
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              placeholder="Mindestens 12 Zeichen"
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground outline-none focus:ring-2 focus:ring-primary/30"
            />
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Passwort bestätigen</label>
            <input
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              placeholder="Passwort wiederholen"
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground outline-none focus:ring-2 focus:ring-primary/30"
            />
          </div>
          {pwError && <p className="text-sm text-destructive">{pwError}</p>}
          <button
            type="submit"
            disabled={pwStatus === "saving"}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 disabled:opacity-50 transition-colors"
          >
            <KeyRound className="w-4 h-4" />
            {pwStatus === "saving" ? "Speichern…" : pwStatus === "saved" ? "Geändert!" : "Passwort ändern"}
          </button>
          {pwStatus === "error" && !pwError && <p className="text-sm text-destructive">Fehler beim Ändern.</p>}
        </form>
      </div>
    </div>
  );
}
