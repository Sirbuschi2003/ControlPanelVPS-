"use client";

import { useEffect, useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import {
  api,
  type DomainResources,
  type DomainUser,
  type User,
  type DNSRecord,
  type MailAccount,
  type MailAlias,
  type SSLCert,
  type ManagedDatabase,
  type CronJob,
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
  Server,
  Plus,
  Trash2,
  X,
  CheckCircle2,
  AlertCircle,
  Eye,
  EyeOff,
} from "lucide-react";

type Tab = "overview" | "website" | "dns" | "mail" | "databases" | "ssl" | "crons" | "users";

const TABS: { id: Tab; label: string; icon: React.ElementType }[] = [
  { id: "overview", label: "Übersicht", icon: Layers },
  { id: "website", label: "Website", icon: Globe },
  { id: "dns", label: "DNS", icon: Globe },
  { id: "mail", label: "E-Mail", icon: Mail },
  { id: "databases", label: "Datenbanken", icon: Database },
  { id: "ssl", label: "SSL/TLS", icon: Shield },
  { id: "crons", label: "Cron Jobs", icon: Clock },
  { id: "users", label: "Benutzer", icon: Users },
];

function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: React.ReactNode }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div className="bg-card border border-border rounded-xl w-full max-w-lg mx-4 shadow-xl">
        <div className="flex items-center justify-between p-4 border-b border-border">
          <h2 className="font-semibold text-foreground">{title}</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground"><X className="w-5 h-5" /></button>
        </div>
        <div className="p-4 space-y-3">{children}</div>
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1">
      <label className="text-xs font-medium text-muted-foreground">{label}</label>
      {children}
    </div>
  );
}

function Input(props: React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      {...props}
      className={`w-full bg-secondary border border-border rounded-lg px-3 py-2 text-sm text-foreground focus:outline-none focus:border-primary ${props.className ?? ""}`}
    />
  );
}

function Select({ children, ...props }: React.SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      {...props}
      className="w-full bg-secondary border border-border rounded-lg px-3 py-2 text-sm text-foreground focus:outline-none focus:border-primary"
    >
      {children}
    </select>
  );
}

function Btn({ children, variant = "primary", ...props }: React.ButtonHTMLAttributes<HTMLButtonElement> & { variant?: "primary" | "danger" | "ghost" }) {
  const cls = {
    primary: "bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50",
    danger: "bg-destructive text-white hover:bg-destructive/90",
    ghost: "bg-secondary text-foreground hover:bg-accent",
  }[variant];
  return <button {...props} className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${cls} ${props.className ?? ""}`}>{children}</button>;
}

function InfoCard({ label, value, mono, ok }: { label: string; value: string; mono?: boolean; ok?: boolean }) {
  return (
    <div className="bg-secondary border border-border rounded-lg p-3">
      <div className="text-xs text-muted-foreground mb-1">{label}</div>
      <div className={`text-sm font-medium flex items-center gap-1 ${ok === true ? "text-green-400" : ok === false ? "text-yellow-400" : "text-foreground"} ${mono ? "font-mono" : ""}`}>
        {ok === true && <CheckCircle2 className="w-3 h-3" />}
        {ok === false && <AlertCircle className="w-3 h-3" />}
        {value}
      </div>
    </div>
  );
}

const recordTypeColors: Record<string, string> = {
  A: "bg-blue-500/20 text-blue-400", AAAA: "bg-indigo-500/20 text-indigo-400",
  CNAME: "bg-purple-500/20 text-purple-400", MX: "bg-orange-500/20 text-orange-400",
  TXT: "bg-yellow-500/20 text-yellow-400", SRV: "bg-pink-500/20 text-pink-400",
  CAA: "bg-red-500/20 text-red-400",
};

const cronPresets = [
  { label: "Jede Minute", value: "* * * * *" },
  { label: "Stündlich", value: "0 * * * *" },
  { label: "Täglich 02:00", value: "0 2 * * *" },
  { label: "Wöchentlich", value: "0 2 * * 0" },
  { label: "Monatlich", value: "0 2 1 * *" },
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

  // DNS
  const [dnsRecords, setDnsRecords] = useState<DNSRecord[]>([]);
  const [dnsLoading, setDnsLoading] = useState(false);
  const [showAddRecord, setShowAddRecord] = useState(false);
  const [deleteRecordId, setDeleteRecordId] = useState<string | null>(null);
  const [recordForm, setRecordForm] = useState({ name: "@", type: "A" as DNSRecord["type"], content: "", ttl: "3600", priority: "" });
  const [savingRecord, setSavingRecord] = useState(false);

  // Mail
  const [mailAccounts, setMailAccounts] = useState<MailAccount[]>([]);
  const [mailAliases, setMailAliases] = useState<MailAlias[]>([]);
  const [mailLoading, setMailLoading] = useState(false);
  const [mailSubTab, setMailSubTab] = useState<"accounts" | "aliases">("accounts");
  const [showAddAccount, setShowAddAccount] = useState(false);
  const [showAddAlias, setShowAddAlias] = useState(false);
  const [deleteMailTarget, setDeleteMailTarget] = useState<{ type: "account" | "alias"; id: string } | null>(null);
  const [accountForm, setAccountForm] = useState({ username: "", password: "", quota_mb: "1000" });
  const [aliasForm, setAliasForm] = useState({ source: "", destination: "" });
  const [showAccountPw, setShowAccountPw] = useState(false);
  const [savingMail, setSavingMail] = useState(false);

  // SSL
  const [showAddSSL, setShowAddSSL] = useState(false);
  const [deleteSSLId, setDeleteSSLId] = useState<string | null>(null);
  const [sslForm, setSslForm] = useState({ email: "" });
  const [savingSSL, setSavingSSL] = useState(false);

  // DB
  const [showAddDB, setShowAddDB] = useState(false);
  const [deleteDBId, setDeleteDBId] = useState<string | null>(null);
  const [dbForm, setDbForm] = useState({ name: "", db_type: "mysql" as "mysql" | "postgresql", db_user: "", db_password: "" });
  const [showDBPw, setShowDBPw] = useState(false);
  const [savingDB, setSavingDB] = useState(false);

  // Crons
  const [showAddCron, setShowAddCron] = useState(false);
  const [deleteCronId, setDeleteCronId] = useState<string | null>(null);
  const [cronForm, setCronForm] = useState({ name: "", schedule: "0 2 * * *", command: "", run_as_user: "www-data" });
  const [savingCron, setSavingCron] = useState(false);

  const loadResources = useCallback(async () => {
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
      } catch { /* non-admin */ }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Laden");
    } finally {
      setLoading(false);
    }
  }, [id]);

  const loadDNSRecords = useCallback(async (zoneId: string) => {
    setDnsLoading(true);
    try {
      const r = await api.get<DNSRecord[]>(`/dns/zones/${zoneId}/records`);
      setDnsRecords(r);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "DNS-Fehler");
    } finally {
      setDnsLoading(false);
    }
  }, []);

  const loadMail = useCallback(async (mailDomainId: string) => {
    setMailLoading(true);
    try {
      const [acc, ali] = await Promise.all([
        api.get<MailAccount[]>(`/mail/accounts?domain_id=${mailDomainId}`),
        api.get<MailAlias[]>(`/mail/aliases?domain_id=${mailDomainId}`),
      ]);
      setMailAccounts(acc);
      setMailAliases(ali);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Mail-Fehler");
    } finally {
      setMailLoading(false);
    }
  }, []);

  useEffect(() => { loadResources(); }, [loadResources]);

  useEffect(() => {
    if (!resources) return;
    if (tab === "dns" && resources.dns_zone) loadDNSRecords(resources.dns_zone.id);
    if (tab === "mail" && resources.mail_domain) loadMail(resources.mail_domain.id);
  }, [tab, resources, loadDNSRecords, loadMail]);

  async function handleAddRecord() {
    if (!resources?.dns_zone) return;
    setSavingRecord(true);
    try {
      await api.post(`/dns/zones/${resources.dns_zone.id}/records`, {
        ...recordForm,
        ttl: parseInt(recordForm.ttl) || 3600,
        priority: recordForm.priority ? parseInt(recordForm.priority) : undefined,
      });
      setShowAddRecord(false);
      setRecordForm({ name: "@", type: "A", content: "", ttl: "3600", priority: "" });
      await loadDNSRecords(resources.dns_zone.id);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSavingRecord(false);
    }
  }

  async function handleDeleteRecord(recordId: string) {
    if (!resources?.dns_zone) return;
    try {
      await api.delete(`/dns/records/${recordId}`);
      setDeleteRecordId(null);
      await loadDNSRecords(resources.dns_zone.id);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    }
  }

  async function handleAddAccount() {
    if (!resources?.mail_domain) return;
    setSavingMail(true);
    try {
      await api.post("/mail/accounts", {
        domain_id: resources.mail_domain.id,
        username: accountForm.username,
        password: accountForm.password,
        quota_mb: parseInt(accountForm.quota_mb) || 1000,
      });
      setShowAddAccount(false);
      setAccountForm({ username: "", password: "", quota_mb: "1000" });
      await loadMail(resources.mail_domain.id);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSavingMail(false);
    }
  }

  async function handleAddAlias() {
    if (!resources?.mail_domain) return;
    setSavingMail(true);
    try {
      await api.post("/mail/aliases", {
        domain_id: resources.mail_domain.id,
        source: aliasForm.source,
        destination: aliasForm.destination,
      });
      setShowAddAlias(false);
      setAliasForm({ source: "", destination: "" });
      await loadMail(resources.mail_domain.id);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSavingMail(false);
    }
  }

  async function handleDeleteMail() {
    if (!deleteMailTarget || !resources?.mail_domain) return;
    try {
      const path = deleteMailTarget.type === "account"
        ? `/mail/accounts/${deleteMailTarget.id}`
        : `/mail/aliases/${deleteMailTarget.id}`;
      await api.delete(path);
      setDeleteMailTarget(null);
      await loadMail(resources.mail_domain.id);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    }
  }

  async function handleAddSSL() {
    if (!resources?.domain) return;
    setSavingSSL(true);
    try {
      await api.post("/ssl", {
        server_id: resources.domain.server_id,
        domain: resources.domain.name,
        san_domains: [],
        email: sslForm.email,
        domain_id: id,
      });
      setShowAddSSL(false);
      setSslForm({ email: "" });
      await loadResources();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSavingSSL(false);
    }
  }

  async function handleDeleteSSL(certId: string) {
    try {
      await api.delete(`/ssl/${certId}`);
      setDeleteSSLId(null);
      await loadResources();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    }
  }

  async function handleAddDB() {
    if (!resources?.domain) return;
    setSavingDB(true);
    try {
      await api.post("/databases", {
        server_id: resources.domain.server_id,
        domain_id: id,
        ...dbForm,
      });
      setShowAddDB(false);
      setDbForm({ name: "", db_type: "mysql", db_user: "", db_password: "" });
      await loadResources();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSavingDB(false);
    }
  }

  async function handleDeleteDB(dbId: string) {
    try {
      await api.delete(`/databases/${dbId}`);
      setDeleteDBId(null);
      await loadResources();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    }
  }

  async function handleAddCron() {
    if (!resources?.domain) return;
    setSavingCron(true);
    try {
      await api.post("/crons", {
        server_id: resources.domain.server_id,
        domain_id: id,
        ...cronForm,
      });
      setShowAddCron(false);
      setCronForm({ name: "", schedule: "0 2 * * *", command: "", run_as_user: "www-data" });
      await loadResources();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSavingCron(false);
    }
  }

  async function handleDeleteCron(cronId: string) {
    try {
      await api.delete(`/crons/${cronId}`);
      setDeleteCronId(null);
      await loadResources();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    }
  }

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

  if (loading) return <div className="text-muted-foreground text-sm p-4">Lade Domain…</div>;
  if (error && !resources) return <div className="text-destructive text-sm p-4">{error}</div>;
  if (!resources) return null;

  const { domain, website, dns_zone, mail_domain, ssl_certs, databases, cron_jobs } = resources;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <button onClick={() => router.push("/dashboard/domains")} className="text-muted-foreground hover:text-foreground transition-colors">
            <ArrowLeft className="w-5 h-5" />
          </button>
          <Layers className="w-5 h-5 text-primary" />
          <h1 className="text-xl font-semibold text-foreground">{domain.name}</h1>
          <span className={`px-2 py-0.5 rounded text-xs font-medium ${
            domain.status === "active" ? "bg-green-500/20 text-green-400" :
            domain.status === "error" ? "bg-destructive/20 text-destructive" :
            domain.status === "partial" ? "bg-yellow-500/20 text-yellow-400" :
            "bg-primary/20 text-primary"
          }`}>{domain.status}</span>
        </div>
        <div className="text-sm text-muted-foreground flex items-center gap-1">
          <Server className="w-4 h-4" />
          {domain.server_name ?? domain.server_id.slice(0, 8)}
          {domain.server_ip && <span className="text-muted-foreground/60 ml-1">({domain.server_ip})</span>}
        </div>
      </div>

      {error && (
        <div className="bg-destructive/10 border border-destructive/30 text-destructive rounded-lg p-3 text-sm flex items-center justify-between">
          {error}
          <button onClick={() => setError("")}><X className="w-4 h-4" /></button>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-border overflow-x-auto">
        {TABS.map(t => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`flex items-center gap-2 px-4 py-2 text-sm font-medium transition-colors whitespace-nowrap ${
              tab === t.id ? "text-primary border-b-2 border-primary" : "text-muted-foreground hover:text-foreground"
            }`}
          >
            <t.icon className="w-4 h-4" />
            {t.label}
          </button>
        ))}
      </div>

      {/* ─── ÜBERSICHT ─── */}
      {tab === "overview" && (
        <div className="grid grid-cols-2 gap-4">
          <InfoCard label="Domain-Name" value={domain.name} />
          <InfoCard label="Status" value={domain.status} ok={domain.status === "active"} />
          <InfoCard label="PHP-Version" value={`PHP ${domain.php_version}`} />
          <InfoCard label="Document Root" value={domain.document_root} mono />
          <InfoCard label="Eigentümer" value={domain.owner_name ?? "—"} />
          <InfoCard label="Server-IP" value={domain.server_ip ?? "—"} />
          <InfoCard label="Website" value={website ? "Aktiv" : "Nicht angelegt"} ok={!!website} />
          <InfoCard label="DNS-Zone" value={dns_zone ? dns_zone.name : "Nicht angelegt"} ok={!!dns_zone} />
          <InfoCard label="Mail-Domain" value={mail_domain ? "Aktiv" : "Nicht angelegt"} ok={!!mail_domain} />
          <InfoCard label="SSL-Zertifikate" value={`${ssl_certs.length} Zertifikat${ssl_certs.length !== 1 ? "e" : ""}`} />
          <InfoCard label="Datenbanken" value={`${databases.length} Datenbank${databases.length !== 1 ? "en" : ""}`} />
          <InfoCard label="Cron Jobs" value={`${cron_jobs.length} Job${cron_jobs.length !== 1 ? "s" : ""}`} />
        </div>
      )}

      {/* ─── WEBSITE ─── */}
      {tab === "website" && (
        <div className="bg-card border border-border rounded-lg p-5">
          {website ? (
            <div className="space-y-4">
              <h3 className="font-medium text-foreground">Website-Konfiguration</h3>
              <div className="grid grid-cols-2 gap-3">
                <InfoCard label="Domain" value={website.domain} />
                <InfoCard label="PHP" value={`PHP ${website.php_version}`} />
                <InfoCard label="Document Root" value={website.document_root} mono />
                <InfoCard label="SSL aktiv" value={website.ssl_enabled ? "Ja" : "Nein"} ok={website.ssl_enabled} />
                <InfoCard label="Status" value={website.enabled ? "Aktiv" : "Deaktiviert"} ok={website.enabled} />
              </div>
            </div>
          ) : (
            <div className="flex flex-col items-center gap-2 py-8 text-muted-foreground">
              <Globe className="w-8 h-8" />
              <p className="text-sm">Noch keine Website für diese Domain angelegt.</p>
              <p className="text-xs">Die Website wurde beim Domain-Erstellen automatisch provisioniert.</p>
            </div>
          )}
        </div>
      )}

      {/* ─── DNS ─── */}
      {tab === "dns" && (
        <div className="space-y-4">
          {dns_zone ? (
            <>
              <div className="bg-card border border-border rounded-lg p-4">
                <div className="grid grid-cols-2 gap-3">
                  <InfoCard label="Zone" value={dns_zone.name} />
                  <InfoCard label="Typ" value={dns_zone.zone_type} />
                  <InfoCard label="Nameserver" value={dns_zone.nameserver} />
                  <InfoCard label="Admin-E-Mail" value={dns_zone.admin_email} />
                  {dns_zone.master_ip && <InfoCard label="Master-IP" value={dns_zone.master_ip} />}
                  <InfoCard label="Serial" value={String(dns_zone.serial)} />
                </div>
              </div>

              <div className="bg-card border border-border rounded-lg overflow-hidden">
                <div className="flex items-center justify-between px-4 py-3 border-b border-border">
                  <h3 className="font-medium text-foreground text-sm">DNS-Einträge</h3>
                  <Btn onClick={() => setShowAddRecord(true)}><Plus className="w-3 h-3 mr-1 inline" />Eintrag hinzufügen</Btn>
                </div>
                {dnsLoading ? (
                  <div className="p-4 text-sm text-muted-foreground">Lade Einträge…</div>
                ) : dnsRecords.length > 0 ? (
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-border text-muted-foreground text-left text-xs">
                        <th className="px-4 py-2">Name</th>
                        <th className="px-4 py-2">Typ</th>
                        <th className="px-4 py-2">Inhalt</th>
                        <th className="px-4 py-2">TTL</th>
                        <th className="px-4 py-2">Prio</th>
                        <th className="px-4 py-2 w-8"></th>
                      </tr>
                    </thead>
                    <tbody>
                      {dnsRecords.map(r => (
                        <tr key={r.id} className="border-b border-border/50 hover:bg-secondary/30">
                          <td className="px-4 py-2 font-mono text-foreground">{r.name}</td>
                          <td className="px-4 py-2">
                            <span className={`px-1.5 py-0.5 rounded text-xs font-mono ${recordTypeColors[r.type] ?? "bg-secondary text-foreground"}`}>{r.type}</span>
                          </td>
                          <td className="px-4 py-2 text-muted-foreground font-mono text-xs max-w-xs truncate">{r.content}</td>
                          <td className="px-4 py-2 text-muted-foreground">{r.ttl}</td>
                          <td className="px-4 py-2 text-muted-foreground">{r.priority ?? "—"}</td>
                          <td className="px-4 py-2">
                            {deleteRecordId === r.id ? (
                              <div className="flex gap-1">
                                <button onClick={() => handleDeleteRecord(r.id)} className="text-destructive text-xs hover:underline">Ja</button>
                                <button onClick={() => setDeleteRecordId(null)} className="text-muted-foreground text-xs hover:underline">Nein</button>
                              </div>
                            ) : (
                              <button onClick={() => setDeleteRecordId(r.id)} className="text-muted-foreground hover:text-destructive"><Trash2 className="w-3.5 h-3.5" /></button>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                ) : (
                  <div className="p-6 text-center text-sm text-muted-foreground">Keine DNS-Einträge vorhanden.</div>
                )}
              </div>
            </>
          ) : (
            <div className="bg-card border border-border rounded-lg p-8 flex flex-col items-center gap-2 text-muted-foreground">
              <Globe className="w-8 h-8" />
              <p className="text-sm">Noch keine DNS-Zone für diese Domain.</p>
            </div>
          )}
        </div>
      )}

      {/* ─── E-MAIL ─── */}
      {tab === "mail" && (
        <div className="space-y-4">
          {mail_domain ? (
            <>
              <div className="bg-card border border-border rounded-lg p-4">
                <div className="grid grid-cols-2 gap-3">
                  <InfoCard label="Mail-Domain" value={mail_domain.domain} />
                  <InfoCard label="Status" value={mail_domain.enabled ? "Aktiv" : "Deaktiviert"} ok={mail_domain.enabled} />
                </div>
              </div>

              {/* Sub-tabs */}
              <div className="flex gap-1 border-b border-border">
                {(["accounts", "aliases"] as const).map(st => (
                  <button key={st} onClick={() => setMailSubTab(st)}
                    className={`px-4 py-2 text-sm font-medium transition-colors ${mailSubTab === st ? "text-primary border-b-2 border-primary" : "text-muted-foreground hover:text-foreground"}`}>
                    {st === "accounts" ? "Postfächer" : "Weiterleitungen"}
                  </button>
                ))}
              </div>

              {/* Accounts */}
              {mailSubTab === "accounts" && (
                <div className="bg-card border border-border rounded-lg overflow-hidden">
                  <div className="flex items-center justify-between px-4 py-3 border-b border-border">
                    <h3 className="font-medium text-foreground text-sm">Postfächer</h3>
                    <Btn onClick={() => setShowAddAccount(true)}><Plus className="w-3 h-3 mr-1 inline" />Postfach anlegen</Btn>
                  </div>
                  {mailLoading ? (
                    <div className="p-4 text-sm text-muted-foreground">Lade Postfächer…</div>
                  ) : mailAccounts.length > 0 ? (
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b border-border text-muted-foreground text-xs text-left">
                          <th className="px-4 py-2">E-Mail-Adresse</th>
                          <th className="px-4 py-2">Quota</th>
                          <th className="px-4 py-2 w-8"></th>
                        </tr>
                      </thead>
                      <tbody>
                        {mailAccounts.map(acc => (
                          <tr key={acc.id} className="border-b border-border/50 hover:bg-secondary/30">
                            <td className="px-4 py-2 text-foreground">{acc.username}@{mail_domain.domain}</td>
                            <td className="px-4 py-2 text-muted-foreground">{acc.quota_mb} MB</td>
                            <td className="px-4 py-2">
                              {deleteMailTarget?.id === acc.id ? (
                                <div className="flex gap-1">
                                  <button onClick={handleDeleteMail} className="text-destructive text-xs hover:underline">Ja</button>
                                  <button onClick={() => setDeleteMailTarget(null)} className="text-muted-foreground text-xs hover:underline">Nein</button>
                                </div>
                              ) : (
                                <button onClick={() => setDeleteMailTarget({ type: "account", id: acc.id })} className="text-muted-foreground hover:text-destructive"><Trash2 className="w-3.5 h-3.5" /></button>
                              )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  ) : (
                    <div className="p-6 text-center text-sm text-muted-foreground">Noch keine Postfächer angelegt.</div>
                  )}
                </div>
              )}

              {/* Aliases */}
              {mailSubTab === "aliases" && (
                <div className="bg-card border border-border rounded-lg overflow-hidden">
                  <div className="flex items-center justify-between px-4 py-3 border-b border-border">
                    <h3 className="font-medium text-foreground text-sm">Weiterleitungen</h3>
                    <Btn onClick={() => setShowAddAlias(true)}><Plus className="w-3 h-3 mr-1 inline" />Weiterleitung anlegen</Btn>
                  </div>
                  {mailLoading ? (
                    <div className="p-4 text-sm text-muted-foreground">Lade Weiterleitungen…</div>
                  ) : mailAliases.length > 0 ? (
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b border-border text-muted-foreground text-xs text-left">
                          <th className="px-4 py-2">Von</th>
                          <th className="px-4 py-2">An</th>
                          <th className="px-4 py-2 w-8"></th>
                        </tr>
                      </thead>
                      <tbody>
                        {mailAliases.map(alias => (
                          <tr key={alias.id} className="border-b border-border/50 hover:bg-secondary/30">
                            <td className="px-4 py-2 text-foreground">{alias.source}</td>
                            <td className="px-4 py-2 text-muted-foreground">{alias.destination}</td>
                            <td className="px-4 py-2">
                              {deleteMailTarget?.id === alias.id ? (
                                <div className="flex gap-1">
                                  <button onClick={handleDeleteMail} className="text-destructive text-xs hover:underline">Ja</button>
                                  <button onClick={() => setDeleteMailTarget(null)} className="text-muted-foreground text-xs hover:underline">Nein</button>
                                </div>
                              ) : (
                                <button onClick={() => setDeleteMailTarget({ type: "alias", id: alias.id })} className="text-muted-foreground hover:text-destructive"><Trash2 className="w-3.5 h-3.5" /></button>
                              )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  ) : (
                    <div className="p-6 text-center text-sm text-muted-foreground">Noch keine Weiterleitungen angelegt.</div>
                  )}
                </div>
              )}
            </>
          ) : (
            <div className="bg-card border border-border rounded-lg p-8 flex flex-col items-center gap-2 text-muted-foreground">
              <Mail className="w-8 h-8" />
              <p className="text-sm">Noch keine Mail-Domain für diese Domain.</p>
            </div>
          )}
        </div>
      )}

      {/* ─── DATENBANKEN ─── */}
      {tab === "databases" && (
        <div className="bg-card border border-border rounded-lg overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b border-border">
            <h3 className="font-medium text-foreground text-sm">Datenbanken</h3>
            <Btn onClick={() => setShowAddDB(true)}><Plus className="w-3 h-3 mr-1 inline" />Datenbank anlegen</Btn>
          </div>
          {databases.length > 0 ? (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border text-muted-foreground text-xs text-left">
                  <th className="px-4 py-2">Name</th>
                  <th className="px-4 py-2">Typ</th>
                  <th className="px-4 py-2">Benutzer</th>
                  <th className="px-4 py-2">Größe</th>
                  <th className="px-4 py-2 w-8"></th>
                </tr>
              </thead>
              <tbody>
                {databases.map(db => (
                  <tr key={db.id} className="border-b border-border/50 hover:bg-secondary/30">
                    <td className="px-4 py-2 text-foreground font-mono">{db.name}</td>
                    <td className="px-4 py-2 text-muted-foreground uppercase text-xs">{db.db_type}</td>
                    <td className="px-4 py-2 text-muted-foreground">{db.db_user}</td>
                    <td className="px-4 py-2 text-muted-foreground">{(db.size_bytes / 1024 / 1024).toFixed(1)} MB</td>
                    <td className="px-4 py-2">
                      {deleteDBId === db.id ? (
                        <div className="flex gap-1">
                          <button onClick={() => handleDeleteDB(db.id)} className="text-destructive text-xs hover:underline">Ja</button>
                          <button onClick={() => setDeleteDBId(null)} className="text-muted-foreground text-xs hover:underline">Nein</button>
                        </div>
                      ) : (
                        <button onClick={() => setDeleteDBId(db.id)} className="text-muted-foreground hover:text-destructive"><Trash2 className="w-3.5 h-3.5" /></button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <div className="p-6 text-center text-sm text-muted-foreground">Noch keine Datenbanken für diese Domain.</div>
          )}
        </div>
      )}

      {/* ─── SSL/TLS ─── */}
      {tab === "ssl" && (
        <div className="bg-card border border-border rounded-lg overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b border-border">
            <h3 className="font-medium text-foreground text-sm">SSL-Zertifikate</h3>
            <Btn onClick={() => setShowAddSSL(true)}><Plus className="w-3 h-3 mr-1 inline" />Zertifikat ausstellen</Btn>
          </div>
          {ssl_certs.length > 0 ? (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border text-muted-foreground text-xs text-left">
                  <th className="px-4 py-2">Domain</th>
                  <th className="px-4 py-2">Aussteller</th>
                  <th className="px-4 py-2">Status</th>
                  <th className="px-4 py-2">Ablauf</th>
                  <th className="px-4 py-2 w-8"></th>
                </tr>
              </thead>
              <tbody>
                {ssl_certs.map(cert => (
                  <tr key={cert.id} className="border-b border-border/50 hover:bg-secondary/30">
                    <td className="px-4 py-2 text-foreground">{cert.domain}</td>
                    <td className="px-4 py-2 text-muted-foreground">{cert.issuer ?? "—"}</td>
                    <td className="px-4 py-2">
                      <span className={`px-2 py-0.5 rounded-full text-xs ${cert.status === "active" ? "bg-green-500/20 text-green-400" : cert.status === "pending" ? "bg-yellow-500/20 text-yellow-400" : "bg-destructive/20 text-destructive"}`}>
                        {cert.status}
                      </span>
                    </td>
                    <td className="px-4 py-2 text-muted-foreground">
                      {cert.expires_at ? new Date(cert.expires_at).toLocaleDateString("de-DE") : "—"}
                    </td>
                    <td className="px-4 py-2">
                      {deleteSSLId === cert.id ? (
                        <div className="flex gap-1">
                          <button onClick={() => handleDeleteSSL(cert.id)} className="text-destructive text-xs hover:underline">Ja</button>
                          <button onClick={() => setDeleteSSLId(null)} className="text-muted-foreground text-xs hover:underline">Nein</button>
                        </div>
                      ) : (
                        <button onClick={() => setDeleteSSLId(cert.id)} className="text-muted-foreground hover:text-destructive"><Trash2 className="w-3.5 h-3.5" /></button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <div className="p-6 text-center text-sm text-muted-foreground">Noch keine SSL-Zertifikate für diese Domain.</div>
          )}
        </div>
      )}

      {/* ─── CRON JOBS ─── */}
      {tab === "crons" && (
        <div className="bg-card border border-border rounded-lg overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b border-border">
            <h3 className="font-medium text-foreground text-sm">Cron Jobs</h3>
            <Btn onClick={() => setShowAddCron(true)}><Plus className="w-3 h-3 mr-1 inline" />Cron Job anlegen</Btn>
          </div>
          {cron_jobs.length > 0 ? (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border text-muted-foreground text-xs text-left">
                  <th className="px-4 py-2">Name</th>
                  <th className="px-4 py-2">Zeitplan</th>
                  <th className="px-4 py-2">Befehl</th>
                  <th className="px-4 py-2">Status</th>
                  <th className="px-4 py-2 w-8"></th>
                </tr>
              </thead>
              <tbody>
                {cron_jobs.map((job: CronJob) => (
                  <tr key={job.id} className="border-b border-border/50 hover:bg-secondary/30">
                    <td className="px-4 py-2 text-foreground">{job.name}</td>
                    <td className="px-4 py-2 text-muted-foreground font-mono text-xs">{job.schedule}</td>
                    <td className="px-4 py-2 text-muted-foreground font-mono text-xs max-w-xs truncate">{job.command}</td>
                    <td className="px-4 py-2">
                      <span className={`px-1.5 py-0.5 rounded text-xs ${job.enabled ? "bg-green-500/20 text-green-400" : "bg-secondary text-muted-foreground"}`}>
                        {job.enabled ? "Aktiv" : "Deaktiviert"}
                      </span>
                    </td>
                    <td className="px-4 py-2">
                      {deleteCronId === job.id ? (
                        <div className="flex gap-1">
                          <button onClick={() => handleDeleteCron(job.id)} className="text-destructive text-xs hover:underline">Ja</button>
                          <button onClick={() => setDeleteCronId(null)} className="text-muted-foreground text-xs hover:underline">Nein</button>
                        </div>
                      ) : (
                        <button onClick={() => setDeleteCronId(job.id)} className="text-muted-foreground hover:text-destructive"><Trash2 className="w-3.5 h-3.5" /></button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <div className="p-6 text-center text-sm text-muted-foreground">Noch keine Cron Jobs für diese Domain.</div>
          )}
        </div>
      )}

      {/* ─── BENUTZER ─── */}
      {tab === "users" && (
        <div className="space-y-4">
          {allUsers.length > 0 && (
            <div className="bg-card border border-border rounded-lg p-4 flex gap-3">
              <Select value={addUserID} onChange={e => setAddUserID(e.target.value)}>
                <option value="">Benutzer auswählen…</option>
                {allUsers.filter(u => !domainUsers.some(du => du.user_id === u.id)).map(u => (
                  <option key={u.id} value={u.id}>{u.name} ({u.email})</option>
                ))}
              </Select>
              <Btn onClick={handleAssignUser} disabled={!addUserID || addingUser}>
                {addingUser ? "…" : "Zuweisen"}
              </Btn>
            </div>
          )}
          {domainUsers.length > 0 ? (
            <div className="bg-card border border-border rounded-lg overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border text-muted-foreground text-xs text-left">
                    <th className="px-4 py-2">Name</th>
                    <th className="px-4 py-2">E-Mail</th>
                    <th className="px-4 py-2">Zugewiesen</th>
                    <th className="px-4 py-2 w-8"></th>
                  </tr>
                </thead>
                <tbody>
                  {domainUsers.map(du => (
                    <tr key={du.user_id} className="border-b border-border/50">
                      <td className="px-4 py-2 text-foreground">{du.user_name ?? "—"}</td>
                      <td className="px-4 py-2 text-muted-foreground">{du.user_email ?? "—"}</td>
                      <td className="px-4 py-2 text-muted-foreground text-xs">
                        {new Date(du.granted_at).toLocaleDateString("de-DE")}
                      </td>
                      <td className="px-4 py-2">
                        <button onClick={() => handleRemoveUser(du.user_id)} className="text-muted-foreground hover:text-destructive">
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="bg-card border border-border rounded-lg p-6 text-center text-sm text-muted-foreground">
              Keine Benutzer dieser Domain zugewiesen.
            </div>
          )}
        </div>
      )}

      {/* ─── MODALS ─── */}

      {showAddRecord && (
        <Modal title="DNS-Eintrag hinzufügen" onClose={() => setShowAddRecord(false)}>
          <Field label="Name (@ für Root)">
            <Input value={recordForm.name} onChange={e => setRecordForm({ ...recordForm, name: e.target.value })} placeholder="@ oder subdomain" />
          </Field>
          <Field label="Typ">
            <Select value={recordForm.type} onChange={e => setRecordForm({ ...recordForm, type: e.target.value as DNSRecord["type"] })}>
              {["A", "AAAA", "CNAME", "MX", "TXT", "SRV", "CAA"].map(t => <option key={t}>{t}</option>)}
            </Select>
          </Field>
          <Field label="Inhalt (IP/Ziel)">
            <Input value={recordForm.content} onChange={e => setRecordForm({ ...recordForm, content: e.target.value })} placeholder="z.B. 1.2.3.4" />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="TTL (Sekunden)">
              <Input type="number" value={recordForm.ttl} onChange={e => setRecordForm({ ...recordForm, ttl: e.target.value })} />
            </Field>
            {(recordForm.type === "MX" || recordForm.type === "SRV") && (
              <Field label="Priorität">
                <Input type="number" value={recordForm.priority} onChange={e => setRecordForm({ ...recordForm, priority: e.target.value })} placeholder="10" />
              </Field>
            )}
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Btn variant="ghost" onClick={() => setShowAddRecord(false)}>Abbrechen</Btn>
            <Btn onClick={handleAddRecord} disabled={savingRecord || !recordForm.content}>{savingRecord ? "Speichere…" : "Hinzufügen"}</Btn>
          </div>
        </Modal>
      )}

      {showAddAccount && (
        <Modal title="Postfach anlegen" onClose={() => setShowAddAccount(false)}>
          <Field label="Benutzername">
            <div className="flex items-center">
              <Input value={accountForm.username} onChange={e => setAccountForm({ ...accountForm, username: e.target.value })} placeholder="info" className="rounded-r-none" />
              <span className="bg-secondary border border-l-0 border-border rounded-r-lg px-3 py-2 text-sm text-muted-foreground">@{mail_domain?.domain}</span>
            </div>
          </Field>
          <Field label="Passwort">
            <div className="relative">
              <Input type={showAccountPw ? "text" : "password"} value={accountForm.password} onChange={e => setAccountForm({ ...accountForm, password: e.target.value })} />
              <button type="button" className="absolute right-3 top-2 text-muted-foreground" onClick={() => setShowAccountPw(!showAccountPw)}>
                {showAccountPw ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
          </Field>
          <Field label="Quota (MB)">
            <Input type="number" value={accountForm.quota_mb} onChange={e => setAccountForm({ ...accountForm, quota_mb: e.target.value })} />
          </Field>
          <div className="flex justify-end gap-2 pt-2">
            <Btn variant="ghost" onClick={() => setShowAddAccount(false)}>Abbrechen</Btn>
            <Btn onClick={handleAddAccount} disabled={savingMail || !accountForm.username || !accountForm.password}>{savingMail ? "Speichere…" : "Anlegen"}</Btn>
          </div>
        </Modal>
      )}

      {showAddAlias && (
        <Modal title="Weiterleitung anlegen" onClose={() => setShowAddAlias(false)}>
          <Field label="Von (Absender-Adresse)">
            <Input value={aliasForm.source} onChange={e => setAliasForm({ ...aliasForm, source: e.target.value })} placeholder={`info@${mail_domain?.domain}`} />
          </Field>
          <Field label="An (Ziel-Adresse)">
            <Input value={aliasForm.destination} onChange={e => setAliasForm({ ...aliasForm, destination: e.target.value })} placeholder="ziel@example.com" />
          </Field>
          <div className="flex justify-end gap-2 pt-2">
            <Btn variant="ghost" onClick={() => setShowAddAlias(false)}>Abbrechen</Btn>
            <Btn onClick={handleAddAlias} disabled={savingMail || !aliasForm.source || !aliasForm.destination}>{savingMail ? "Speichere…" : "Anlegen"}</Btn>
          </div>
        </Modal>
      )}

      {showAddSSL && (
        <Modal title={`SSL-Zertifikat für ${domain.name}`} onClose={() => setShowAddSSL(false)}>
          <div className="bg-secondary/50 border border-border rounded-lg p-3 text-sm text-muted-foreground">
            Domain: <span className="text-foreground font-medium">{domain.name}</span>
          </div>
          <Field label="E-Mail-Adresse (für Let's Encrypt)">
            <Input type="email" value={sslForm.email} onChange={e => setSslForm({ email: e.target.value })} placeholder="admin@example.com" />
          </Field>
          <div className="flex justify-end gap-2 pt-2">
            <Btn variant="ghost" onClick={() => setShowAddSSL(false)}>Abbrechen</Btn>
            <Btn onClick={handleAddSSL} disabled={savingSSL || !sslForm.email}>{savingSSL ? "Ausstelle…" : "Zertifikat ausstellen"}</Btn>
          </div>
        </Modal>
      )}

      {showAddDB && (
        <Modal title="Datenbank anlegen" onClose={() => setShowAddDB(false)}>
          <Field label="Datenbankname">
            <Input value={dbForm.name} onChange={e => setDbForm({ ...dbForm, name: e.target.value })} placeholder="meine_db" />
          </Field>
          <Field label="Typ">
            <Select value={dbForm.db_type} onChange={e => setDbForm({ ...dbForm, db_type: e.target.value as "mysql" | "postgresql" })}>
              <option value="mysql">MySQL</option>
              <option value="postgresql">PostgreSQL</option>
            </Select>
          </Field>
          <Field label="DB-Benutzer">
            <Input value={dbForm.db_user} onChange={e => setDbForm({ ...dbForm, db_user: e.target.value })} placeholder="db_user" />
          </Field>
          <Field label="DB-Passwort">
            <div className="relative">
              <Input type={showDBPw ? "text" : "password"} value={dbForm.db_password} onChange={e => setDbForm({ ...dbForm, db_password: e.target.value })} />
              <button type="button" className="absolute right-3 top-2 text-muted-foreground" onClick={() => setShowDBPw(!showDBPw)}>
                {showDBPw ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
          </Field>
          <div className="flex justify-end gap-2 pt-2">
            <Btn variant="ghost" onClick={() => setShowAddDB(false)}>Abbrechen</Btn>
            <Btn onClick={handleAddDB} disabled={savingDB || !dbForm.name || !dbForm.db_user || !dbForm.db_password}>{savingDB ? "Anlege…" : "Anlegen"}</Btn>
          </div>
        </Modal>
      )}

      {showAddCron && (
        <Modal title="Cron Job anlegen" onClose={() => setShowAddCron(false)}>
          <Field label="Name">
            <Input value={cronForm.name} onChange={e => setCronForm({ ...cronForm, name: e.target.value })} placeholder="Backup täglich" />
          </Field>
          <Field label="Zeitplan (Cron-Ausdruck)">
            <Input value={cronForm.schedule} onChange={e => setCronForm({ ...cronForm, schedule: e.target.value })} placeholder="0 2 * * *" />
            <div className="flex flex-wrap gap-1 mt-1">
              {cronPresets.map(p => (
                <button key={p.value} type="button" onClick={() => setCronForm({ ...cronForm, schedule: p.value })}
                  className={`px-2 py-0.5 rounded text-xs transition-colors ${cronForm.schedule === p.value ? "bg-primary text-primary-foreground" : "bg-secondary text-muted-foreground hover:text-foreground"}`}>
                  {p.label}
                </button>
              ))}
            </div>
          </Field>
          <Field label="Befehl">
            <Input value={cronForm.command} onChange={e => setCronForm({ ...cronForm, command: e.target.value })} placeholder="/usr/bin/php /var/www/cron.php" className="font-mono" />
          </Field>
          <Field label="Ausführen als">
            <Input value={cronForm.run_as_user} onChange={e => setCronForm({ ...cronForm, run_as_user: e.target.value })} placeholder="www-data" />
          </Field>
          <div className="flex justify-end gap-2 pt-2">
            <Btn variant="ghost" onClick={() => setShowAddCron(false)}>Abbrechen</Btn>
            <Btn onClick={handleAddCron} disabled={savingCron || !cronForm.name || !cronForm.command}>{savingCron ? "Anlege…" : "Anlegen"}</Btn>
          </div>
        </Modal>
      )}
    </div>
  );
}
