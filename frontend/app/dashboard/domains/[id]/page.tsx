"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import {
  api,
  type DomainResources,
  type DomainUser,
  type User,
} from "@/lib/api";
import {
  Layers,
  Globe,
  Mail,
  Database,
  Shield,
  Clock,
  Users,
  ArrowLeft,
  CheckCircle2,
  AlertCircle,
  ExternalLink,
  Plus,
  Trash2,
  Server,
} from "lucide-react";

type Tab = "overview" | "website" | "dns" | "mail" | "databases" | "ssl" | "users";

const TABS: { id: Tab; label: string; icon: React.ElementType }[] = [
  { id: "overview", label: "Übersicht", icon: Layers },
  { id: "website", label: "Website", icon: Globe },
  { id: "dns", label: "DNS", icon: Globe },
  { id: "mail", label: "E-Mail", icon: Mail },
  { id: "databases", label: "Datenbanken", icon: Database },
  { id: "ssl", label: "SSL/TLS", icon: Shield },
  { id: "users", label: "Benutzer", icon: Users },
];

export default function DomainDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [tab, setTab] = useState<Tab>("overview");
  const [resources, setResources] = useState<DomainResources | null>(null);
  const [domainUsers, setDomainUsers] = useState<DomainUser[]>([]);
  const [allUsers, setAllUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [addUserID, setAddUserID] = useState("");
  const [addingUser, setAddingUser] = useState(false);

  async function load() {
    try {
      const res = await api.get<DomainResources>(`/domains/${id}/resources`);
      setResources(res);
      try {
        const [du, au] = await Promise.all([
          api.get<DomainUser[]>(`/domains/${id}/users`),
          api.get<User[]>("/users"),
        ]);
        setDomainUsers(du);
        setAllUsers(au);
      } catch {
        // non-admin — users tab will be hidden
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Laden");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { load(); }, [id]);

  async function handleAssignUser() {
    if (!addUserID) return;
    setAddingUser(true);
    try {
      await api.post(`/domains/${id}/users`, { user_id: addUserID });
      setAddUserID("");
      const du = await api.get<DomainUser[]>(`/domains/${id}/users`);
      setDomainUsers(du);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setAddingUser(false);
    }
  }

  async function handleRemoveUser(userID: string) {
    try {
      await api.delete(`/domains/${id}/users/${userID}`);
      setDomainUsers(prev => prev.filter(u => u.user_id !== userID));
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    }
  }

  if (loading) return <div className="text-gray-400 text-sm p-4">Lade Domain…</div>;
  if (error && !resources) return (
    <div className="text-red-400 text-sm p-4">{error}</div>
  );
  if (!resources) return null;

  const { domain, website, dns_zone, mail_domain, ssl_certs, databases, cron_jobs } = resources;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <button onClick={() => router.push("/dashboard/domains")} className="text-gray-400 hover:text-white transition-colors">
            <ArrowLeft className="w-5 h-5" />
          </button>
          <Layers className="w-5 h-5 text-blue-400" />
          <h1 className="text-xl font-semibold text-white">{domain.name}</h1>
          <span className={`px-2 py-0.5 rounded text-xs font-medium ${
            domain.status === "active" ? "bg-green-900/40 text-green-400" :
            domain.status === "error" ? "bg-red-900/40 text-red-400" :
            domain.status === "partial" ? "bg-yellow-900/40 text-yellow-400" :
            "bg-blue-900/40 text-blue-400"
          }`}>
            {domain.status}
          </span>
        </div>
        <div className="text-sm text-gray-400 flex items-center gap-1">
          <Server className="w-4 h-4" />
          {domain.server_name ?? domain.server_id.slice(0, 8)}
          {domain.server_ip && <span className="text-gray-500">({domain.server_ip})</span>}
        </div>
      </div>

      {error && (
        <div className="bg-red-900/30 border border-red-700 text-red-300 rounded-lg p-3 text-sm">
          {error}
          <button className="ml-2 underline" onClick={() => setError("")}>Schließen</button>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-700 overflow-x-auto">
        {TABS.map(t => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`flex items-center gap-2 px-4 py-2 text-sm font-medium transition-colors whitespace-nowrap ${
              tab === t.id
                ? "text-blue-400 border-b-2 border-blue-400"
                : "text-gray-400 hover:text-white"
            }`}
          >
            <t.icon className="w-4 h-4" />
            {t.label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      {tab === "overview" && (
        <div className="grid grid-cols-2 gap-4">
          <InfoCard title="Domain-Name" value={domain.name} />
          <InfoCard title="Status" value={domain.status} />
          <InfoCard title="PHP-Version" value={`PHP ${domain.php_version}`} />
          <InfoCard title="Document Root" value={domain.document_root} mono />
          <InfoCard title="Eigentümer" value={domain.owner_name ?? "—"} />
          <InfoCard title="Server-IP" value={domain.server_ip ?? "—"} />
          <InfoCard title="Website" value={website ? "Aktiv" : "Nicht angelegt"} ok={!!website} />
          <InfoCard title="DNS-Zone" value={dns_zone ? dns_zone.name : "Nicht angelegt"} ok={!!dns_zone} />
          <InfoCard title="Mail-Domain" value={mail_domain ? "Aktiv" : "Nicht angelegt"} ok={!!mail_domain} />
          <InfoCard title="SSL-Zertifikate" value={`${ssl_certs.length} Zertifikat${ssl_certs.length !== 1 ? "e" : ""}`} />
          <InfoCard title="Datenbanken" value={`${databases.length} Datenbank${databases.length !== 1 ? "en" : ""}`} />
          <InfoCard title="Cron Jobs" value={`${cron_jobs.length} Job${cron_jobs.length !== 1 ? "s" : ""}`} />
        </div>
      )}

      {tab === "website" && (
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-5 space-y-3">
          {website ? (
            <>
              <div className="flex items-center justify-between">
                <h3 className="text-white font-medium">Website-Konfiguration</h3>
                <Link href={`/dashboard/websites`} className="text-blue-400 hover:underline text-sm flex items-center gap-1">
                  <ExternalLink className="w-3 h-3" /> In Websites öffnen
                </Link>
              </div>
              <table className="w-full text-sm">
                <tbody className="divide-y divide-gray-700">
                  <Row label="Domain" value={website.domain} />
                  <Row label="PHP" value={`PHP ${website.php_version}`} />
                  <Row label="Document Root" value={website.document_root} mono />
                  <Row label="SSL aktiv" value={website.ssl_enabled ? "Ja" : "Nein"} />

                  <Row label="Status" value={website.enabled ? "Aktiv" : "Deaktiviert"} />
                </tbody>
              </table>
            </>
          ) : (
            <EmptyState icon={Globe} text="Noch keine Website angelegt." />
          )}
        </div>
      )}

      {tab === "dns" && (
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-5 space-y-3">
          {dns_zone ? (
            <>
              <div className="flex items-center justify-between">
                <h3 className="text-white font-medium">DNS-Zone</h3>
                <Link href={`/dashboard/dns`} className="text-blue-400 hover:underline text-sm flex items-center gap-1">
                  <ExternalLink className="w-3 h-3" /> In DNS öffnen
                </Link>
              </div>
              <table className="w-full text-sm">
                <tbody className="divide-y divide-gray-700">
                  <Row label="Zone" value={dns_zone.name} />
                  <Row label="Nameserver" value={dns_zone.nameserver} />
                  <Row label="Admin-E-Mail" value={dns_zone.admin_email} />
                  <Row label="Serial" value={String(dns_zone.serial)} />
                </tbody>
              </table>
            </>
          ) : (
            <EmptyState icon={Globe} text="Noch keine DNS-Zone angelegt." />
          )}
        </div>
      )}

      {tab === "mail" && (
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-5 space-y-3">
          {mail_domain ? (
            <>
              <div className="flex items-center justify-between">
                <h3 className="text-white font-medium">Mail-Domain</h3>
                <Link href={`/dashboard/mail`} className="text-blue-400 hover:underline text-sm flex items-center gap-1">
                  <ExternalLink className="w-3 h-3" /> In E-Mail öffnen
                </Link>
              </div>
              <table className="w-full text-sm">
                <tbody className="divide-y divide-gray-700">
                  <Row label="Domain" value={mail_domain.domain} />
                </tbody>
              </table>
            </>
          ) : (
            <EmptyState icon={Mail} text="Noch keine Mail-Domain angelegt." />
          )}
        </div>
      )}

      {tab === "databases" && (
        <div className="bg-gray-800 border border-gray-700 rounded-lg overflow-hidden">
          {databases.length > 0 ? (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-700 text-gray-400 text-left">
                  <th className="px-4 py-3">Name</th>
                  <th className="px-4 py-3">Typ</th>
                  <th className="px-4 py-3">Benutzer</th>
                  <th className="px-4 py-3">Größe</th>
                </tr>
              </thead>
              <tbody>
                {databases.map(db => (
                  <tr key={db.id} className="border-b border-gray-700/50">
                    <td className="px-4 py-3 text-white font-mono">{db.name}</td>
                    <td className="px-4 py-3 text-gray-300 uppercase">{db.db_type}</td>
                    <td className="px-4 py-3 text-gray-300">{db.db_user}</td>
                    <td className="px-4 py-3 text-gray-400">{(db.size_bytes / 1024 / 1024).toFixed(1)} MB</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <div className="p-6">
              <EmptyState icon={Database} text="Noch keine Datenbanken für diese Domain." />
            </div>
          )}
          <div className="p-4 border-t border-gray-700">
            <Link href="/dashboard/databases" className="text-blue-400 hover:underline text-sm flex items-center gap-1">
              <ExternalLink className="w-3 h-3" /> Datenbank in Datenbanken anlegen
            </Link>
          </div>
        </div>
      )}

      {tab === "ssl" && (
        <div className="bg-gray-800 border border-gray-700 rounded-lg overflow-hidden">
          {ssl_certs.length > 0 ? (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-700 text-gray-400 text-left">
                  <th className="px-4 py-3">Domain</th>
                  <th className="px-4 py-3">Aussteller</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3">Ablauf</th>
                </tr>
              </thead>
              <tbody>
                {ssl_certs.map(cert => (
                  <tr key={cert.id} className="border-b border-gray-700/50">
                    <td className="px-4 py-3 text-white">{cert.domain}</td>
                    <td className="px-4 py-3 text-gray-300">{cert.issuer ?? "—"}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-0.5 rounded text-xs ${cert.status === "active" ? "bg-green-900/40 text-green-400" : "bg-yellow-900/40 text-yellow-400"}`}>
                        {cert.status}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-gray-400">
                      {cert.expires_at ? new Date(cert.expires_at).toLocaleDateString("de-DE") : "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <div className="p-6">
              <EmptyState icon={Shield} text="Noch keine SSL-Zertifikate für diese Domain." />
            </div>
          )}
          <div className="p-4 border-t border-gray-700">
            <Link href="/dashboard/ssl" className="text-blue-400 hover:underline text-sm flex items-center gap-1">
              <ExternalLink className="w-3 h-3" /> Zertifikat in SSL/TLS ausstellen
            </Link>
          </div>
        </div>
      )}

      {tab === "users" && (
        <div className="space-y-4">
          {/* Assign user */}
          {allUsers.length > 0 && (
            <div className="bg-gray-800 border border-gray-700 rounded-lg p-4 flex gap-3">
              <select
                value={addUserID}
                onChange={e => setAddUserID(e.target.value)}
                className="flex-1 bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-blue-500"
              >
                <option value="">Benutzer auswählen…</option>
                {allUsers
                  .filter(u => !domainUsers.some(du => du.user_id === u.id))
                  .map(u => (
                    <option key={u.id} value={u.id}>{u.name} ({u.email})</option>
                  ))}
              </select>
              <button
                onClick={handleAssignUser}
                disabled={!addUserID || addingUser}
                className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-sm transition-colors"
              >
                <Plus className="w-4 h-4" />
                Zuweisen
              </button>
            </div>
          )}

          <div className="bg-gray-800 border border-gray-700 rounded-lg overflow-hidden">
            {domainUsers.length > 0 ? (
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-700 text-gray-400 text-left">
                    <th className="px-4 py-3">Benutzer</th>
                    <th className="px-4 py-3">E-Mail</th>
                    <th className="px-4 py-3">Zugewiesen am</th>
                    <th className="px-4 py-3"></th>
                  </tr>
                </thead>
                <tbody>
                  {domainUsers.map(u => (
                    <tr key={u.user_id} className="border-b border-gray-700/50">
                      <td className="px-4 py-3 text-white">{u.user_name ?? u.user_id.slice(0, 8)}</td>
                      <td className="px-4 py-3 text-gray-400">{u.user_email ?? "—"}</td>
                      <td className="px-4 py-3 text-gray-400">
                        {new Date(u.granted_at).toLocaleDateString("de-DE")}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <button
                          onClick={() => handleRemoveUser(u.user_id)}
                          className="p-1 text-gray-500 hover:text-red-400 transition-colors"
                          title="Zugriff entziehen"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            ) : (
              <div className="p-6">
                <EmptyState icon={Users} text="Noch keine Benutzer dieser Domain zugewiesen." />
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function InfoCard({ title, value, mono, ok }: { title: string; value: string; mono?: boolean; ok?: boolean }) {
  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
      <div className="text-xs text-gray-400 mb-1">{title}</div>
      <div className={`text-sm font-medium ${mono ? "font-mono text-blue-300" : "text-white"} ${ok === false ? "text-gray-500" : ""}`}>
        {value}
      </div>
    </div>
  );
}

function Row({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <tr>
      <td className="py-2 text-gray-400 w-1/3">{label}</td>
      <td className={`py-2 text-white ${mono ? "font-mono text-blue-300 text-xs" : ""}`}>{value}</td>
    </tr>
  );
}

function EmptyState({ icon: Icon, text }: { icon: React.ElementType; text: string }) {
  return (
    <div className="text-center text-gray-400 py-4">
      <Icon className="w-8 h-8 mx-auto mb-2 opacity-30" />
      <p className="text-sm">{text}</p>
    </div>
  );
}
