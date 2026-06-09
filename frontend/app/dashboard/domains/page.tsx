"use client";

import { useEffect, useState } from "react";
import { api, type Domain, type Server, type User } from "@/lib/api";
import Link from "next/link";
import {
  Layers,
  Plus,
  Trash2,
  CheckCircle2,
  AlertCircle,
  Clock,
  Globe,
  Mail,
  Database,
  Server as ServerIcon,
} from "lucide-react";

const STATUS_ICON: Record<string, React.ReactNode> = {
  active: <CheckCircle2 className="w-4 h-4 text-green-400" />,
  error: <AlertCircle className="w-4 h-4 text-red-400" />,
  partial: <AlertCircle className="w-4 h-4 text-yellow-400" />,
  provisioning: <Clock className="w-4 h-4 text-blue-400" />,
};

export default function DomainsPage() {
  const [domains, setDomains] = useState<Domain[]>([]);
  const [servers, setServers] = useState<Server[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [creating, setCreating] = useState(false);

  const [form, setForm] = useState({
    server_id: "",
    name: "",
    owner_user_id: "",
    php_version: "8.2",
    provision_web: true,
    provision_dns: true,
    provision_mail: true,
  });

  async function load() {
    try {
      const [d, s] = await Promise.all([
        api.get<Domain[]>("/domains"),
        api.get<Server[]>("/servers"),
      ]);
      setDomains(d);
      setServers(s);
      try {
        const u = await api.get<User[]>("/users");
        setUsers(u);
      } catch {
        // non-admin users won't have access to /users
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Laden");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { load(); }, []);

  async function handleCreate() {
    if (!form.server_id || !form.name) return;
    setCreating(true);
    try {
      await api.post("/domains", form);
      setShowCreate(false);
      setForm({ server_id: "", name: "", owner_user_id: "", php_version: "8.2", provision_web: true, provision_dns: true, provision_mail: true });
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Erstellen");
    } finally {
      setCreating(false);
    }
  }

  async function handleDelete(id: string, name: string) {
    if (!confirm(`Domain "${name}" wirklich löschen? Alle zugehörigen Ressourcen werden entfernt.`)) return;
    try {
      await api.delete(`/domains/${id}`);
      setDomains(prev => prev.filter(d => d.id !== id));
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Löschen");
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Layers className="w-6 h-6 text-blue-400" />
          <h1 className="text-xl font-semibold text-white">Domains</h1>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm transition-colors"
        >
          <Plus className="w-4 h-4" />
          Domain hinzufügen
        </button>
      </div>

      {error && (
        <div className="bg-red-900/30 border border-red-700 text-red-300 rounded-lg p-3 text-sm">
          {error}
          <button className="ml-2 underline" onClick={() => setError("")}>Schließen</button>
        </div>
      )}

      {loading ? (
        <div className="text-gray-400 text-sm">Lade Domains…</div>
      ) : domains.length === 0 ? (
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-8 text-center text-gray-400">
          <Layers className="w-10 h-10 mx-auto mb-3 opacity-40" />
          <p className="text-sm">Noch keine Domains vorhanden.</p>
          <button
            onClick={() => setShowCreate(true)}
            className="mt-3 text-blue-400 hover:underline text-sm"
          >
            Erste Domain anlegen
          </button>
        </div>
      ) : (
        <div className="bg-gray-800 border border-gray-700 rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-700 text-gray-400 text-left">
                <th className="px-4 py-3">Domain</th>
                <th className="px-4 py-3">Server</th>
                <th className="px-4 py-3">Eigentümer</th>
                <th className="px-4 py-3">Ressourcen</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3">PHP</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody>
              {domains.map(d => (
                <tr key={d.id} className="border-b border-gray-700/50 hover:bg-gray-700/30">
                  <td className="px-4 py-3">
                    <Link href={`/dashboard/domains/${d.id}`} className="text-blue-400 hover:underline font-medium">
                      {d.name}
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-gray-300">
                    <span className="flex items-center gap-1">
                      <ServerIcon className="w-3 h-3 opacity-50" />
                      {d.server_name ?? d.server_id.slice(0, 8)}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-gray-400">{d.owner_name ?? "—"}</td>
                  <td className="px-4 py-3">
                    <div className="flex gap-2">
                      {d.website_id && <span title="Website"><Globe className="w-4 h-4 text-green-400" /></span>}
                      {d.dns_zone_id && <span title="DNS"><Globe className="w-4 h-4 text-purple-400" /></span>}
                      {d.mail_domain_id && <span title="Mail"><Mail className="w-4 h-4 text-yellow-400" /></span>}
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <span className="flex items-center gap-1">
                      {STATUS_ICON[d.status] ?? <AlertCircle className="w-4 h-4 text-gray-500" />}
                      <span className="text-gray-300 capitalize">{d.status}</span>
                    </span>
                  </td>
                  <td className="px-4 py-3 text-gray-400">PHP {d.php_version}</td>
                  <td className="px-4 py-3 text-right">
                    <button
                      onClick={() => handleDelete(d.id, d.name)}
                      className="p-1 text-gray-500 hover:text-red-400 transition-colors"
                      title="Domain löschen"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Create Modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-gray-800 border border-gray-700 rounded-xl p-6 w-full max-w-md space-y-4">
            <h2 className="text-white font-semibold flex items-center gap-2">
              <Layers className="w-5 h-5 text-blue-400" />
              Neue Domain anlegen
            </h2>

            <div className="space-y-3">
              <div>
                <label className="block text-xs text-gray-400 mb-1">Server *</label>
                <select
                  value={form.server_id}
                  onChange={e => setForm(f => ({ ...f, server_id: e.target.value }))}
                  className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-blue-500"
                >
                  <option value="">Server wählen…</option>
                  {servers.map(s => (
                    <option key={s.id} value={s.id}>{s.name} ({s.ip_address})</option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-xs text-gray-400 mb-1">Domain-Name *</label>
                <input
                  type="text"
                  placeholder="beispiel.de"
                  value={form.name}
                  onChange={e => setForm(f => ({ ...f, name: e.target.value.toLowerCase().trim() }))}
                  className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-blue-500"
                />
              </div>

              {users.length > 0 && (
                <div>
                  <label className="block text-xs text-gray-400 mb-1">Eigentümer (optional)</label>
                  <select
                    value={form.owner_user_id}
                    onChange={e => setForm(f => ({ ...f, owner_user_id: e.target.value }))}
                    className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-blue-500"
                  >
                    <option value="">Kein Eigentümer</option>
                    {users.map(u => (
                      <option key={u.id} value={u.id}>{u.name} ({u.email})</option>
                    ))}
                  </select>
                </div>
              )}

              <div>
                <label className="block text-xs text-gray-400 mb-1">PHP-Version</label>
                <select
                  value={form.php_version}
                  onChange={e => setForm(f => ({ ...f, php_version: e.target.value }))}
                  className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-blue-500"
                >
                  {["8.3", "8.2", "8.1", "8.0", "7.4"].map(v => (
                    <option key={v} value={v}>PHP {v}</option>
                  ))}
                </select>
              </div>

              <div className="space-y-2">
                <label className="block text-xs text-gray-400">Automatisch anlegen</label>
                {[
                  { key: "provision_web", label: "Website (Nginx vhost)" },
                  { key: "provision_dns", label: "DNS-Zone (BIND9)" },
                  { key: "provision_mail", label: "Mail-Domain (Postfix/Dovecot)" },
                ].map(({ key, label }) => (
                  <label key={key} className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={form[key as keyof typeof form] as boolean}
                      onChange={e => setForm(f => ({ ...f, [key]: e.target.checked }))}
                      className="rounded border-gray-600"
                    />
                    <span className="text-sm text-gray-300">{label}</span>
                  </label>
                ))}
              </div>
            </div>

            <div className="flex justify-end gap-3 pt-2">
              <button
                onClick={() => setShowCreate(false)}
                className="px-4 py-2 text-gray-300 hover:text-white text-sm transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={handleCreate}
                disabled={creating || !form.server_id || !form.name}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-sm transition-colors"
              >
                {creating ? "Erstelle…" : "Domain anlegen"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
