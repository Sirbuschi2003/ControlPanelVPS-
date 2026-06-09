"use client";

import { useEffect, useState } from "react";
import { Mail, Plus, Trash2, X, AlertCircle } from "lucide-react";
import { api, type MailDomain, type MailAccount, type MailAlias, type Server } from "@/lib/api";

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

type Tab = "domains" | "accounts" | "aliases";

export default function MailPage() {
  const [tab, setTab] = useState<Tab>("domains");
  const [domains, setDomains] = useState<MailDomain[]>([]);
  const [accounts, setAccounts] = useState<MailAccount[]>([]);
  const [aliases, setAliases] = useState<MailAlias[]>([]);
  const [servers, setServers] = useState<Server[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [filterDomain, setFilterDomain] = useState("");

  const [showAddDomain, setShowAddDomain] = useState(false);
  const [showAddAccount, setShowAddAccount] = useState(false);
  const [showAddAlias, setShowAddAlias] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{ type: string; id: string } | null>(null);

  const [domainForm, setDomainForm] = useState({ server_id: "", domain: "" });
  const [accountForm, setAccountForm] = useState({ domain_id: "", username: "", password: "", quota_mb: "0", quota_custom: false });
  const [aliasForm, setAliasForm] = useState({ domain_id: "", source: "", destination: "" });

  async function load() {
    try {
      const [d, a, al, sv] = await Promise.all([
        api.get<MailDomain[]>("/mail/domains"),
        api.get<MailAccount[]>("/mail/accounts"),
        api.get<MailAlias[]>("/mail/aliases"),
        api.get<Server[]>("/servers"),
      ]);
      setDomains(d);
      setAccounts(a);
      setAliases(al);
      setServers(sv);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Laden");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { load(); }, []);

  async function handleAddDomain() {
    setSaving(true);
    try {
      await api.post("/mail/domains", domainForm);
      setShowAddDomain(false);
      setDomainForm({ server_id: "", domain: "" });
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSaving(false);
    }
  }

  async function handleAddAccount() {
    setSaving(true);
    try {
      await api.post("/mail/accounts", { ...accountForm, quota_mb: parseInt(accountForm.quota_mb) || 0 });
      setShowAddAccount(false);
      setAccountForm({ domain_id: "", username: "", password: "", quota_mb: "0", quota_custom: false });
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSaving(false);
    }
  }

  async function handleAddAlias() {
    setSaving(true);
    try {
      await api.post("/mail/aliases", aliasForm);
      setShowAddAlias(false);
      setAliasForm({ domain_id: "", source: "", destination: "" });
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    try {
      const paths: Record<string, string> = {
        domain: `/mail/domains/${deleteTarget.id}`,
        account: `/mail/accounts/${deleteTarget.id}`,
        alias: `/mail/aliases/${deleteTarget.id}`,
      };
      await api.delete(paths[deleteTarget.type]);
      setDeleteTarget(null);
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Löschen");
    }
  }

  const serverName = (id: string) => servers.find((s) => s.id === id)?.name || id;
  const domainName = (id: string) => domains.find((d) => d.id === id)?.domain || id;

  const filteredAccounts = filterDomain
    ? accounts.filter((a) => a.domain_id === filterDomain)
    : accounts;

  const tabs: { key: Tab; label: string }[] = [
    { key: "domains", label: "Domains" },
    { key: "accounts", label: "Konten" },
    { key: "aliases", label: "Aliases" },
  ];

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">E-Mail</h1>
          <p className="text-muted-foreground text-sm mt-1">E-Mail-Domains, Konten und Aliases</p>
        </div>
        <div>
          {tab === "domains" && (
            <button onClick={() => setShowAddDomain(true)} className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors">
              <Plus className="w-4 h-4" />Domain hinzufügen
            </button>
          )}
          {tab === "accounts" && (
            <button onClick={() => setShowAddAccount(true)} className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors">
              <Plus className="w-4 h-4" />Konto erstellen
            </button>
          )}
          {tab === "aliases" && (
            <button onClick={() => setShowAddAlias(true)} className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors">
              <Plus className="w-4 h-4" />Alias hinzufügen
            </button>
          )}
        </div>
      </div>

      {error && (
        <div className="flex items-center gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-lg text-destructive text-sm mb-4">
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          {error}
          <button onClick={() => setError("")} className="ml-auto"><X className="w-4 h-4" /></button>
        </div>
      )}

      {/* Tabs */}
      <div className="flex border-b border-border mb-4">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              tab === t.key
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => <Skeleton key={i} className="h-12 w-full" />)}
        </div>
      ) : (
        <>
          {/* Domains Tab */}
          {tab === "domains" && (
            domains.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
                <Mail className="w-12 h-12 mb-4 opacity-30" />
                <p className="font-medium">Keine Mail-Domains vorhanden</p>
              </div>
            ) : (
              <div className="bg-card border border-border rounded-xl overflow-hidden">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border text-muted-foreground">
                      <th className="text-left px-4 py-3 font-medium">Domain</th>
                      <th className="text-left px-4 py-3 font-medium">Server</th>
                      <th className="text-left px-4 py-3 font-medium">Erstellt</th>
                      <th className="text-right px-4 py-3 font-medium">Aktionen</th>
                    </tr>
                  </thead>
                  <tbody>
                    {domains.map((d) => (
                      <tr key={d.id} className="border-b border-border last:border-0 hover:bg-accent/50 transition-colors">
                        <td className="px-4 py-3 font-medium text-foreground">{d.domain}</td>
                        <td className="px-4 py-3 text-muted-foreground">{d.server_name || serverName(d.server_id)}</td>
                        <td className="px-4 py-3 text-muted-foreground">{new Date(d.created_at).toLocaleDateString("de-DE")}</td>
                        <td className="px-4 py-3 text-right">
                          <button onClick={() => setDeleteTarget({ type: "domain", id: d.id })} className="text-muted-foreground hover:text-destructive transition-colors">
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )
          )}

          {/* Accounts Tab */}
          {tab === "accounts" && (
            <div>
              {domains.length > 0 && (
                <div className="mb-4">
                  <select
                    value={filterDomain}
                    onChange={(e) => setFilterDomain(e.target.value)}
                    className="bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
                  >
                    <option value="">Alle Domains</option>
                    {domains.map((d) => (
                      <option key={d.id} value={d.id}>{d.domain}</option>
                    ))}
                  </select>
                </div>
              )}
              {filteredAccounts.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
                  <Mail className="w-12 h-12 mb-4 opacity-30" />
                  <p className="font-medium">Keine E-Mail-Konten vorhanden</p>
                </div>
              ) : (
                <div className="bg-card border border-border rounded-xl overflow-hidden">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-border text-muted-foreground">
                        <th className="text-left px-4 py-3 font-medium">E-Mail-Adresse</th>
                        <th className="text-left px-4 py-3 font-medium">Quota</th>
                        <th className="text-left px-4 py-3 font-medium">Erstellt</th>
                        <th className="text-right px-4 py-3 font-medium">Aktionen</th>
                      </tr>
                    </thead>
                    <tbody>
                      {filteredAccounts.map((a) => (
                        <tr key={a.id} className="border-b border-border last:border-0 hover:bg-accent/50 transition-colors">
                          <td className="px-4 py-3 font-medium text-foreground">
                            {a.username}@{a.domain_name || domainName(a.domain_id)}
                          </td>
                          <td className="px-4 py-3 text-muted-foreground">{a.quota_mb > 0 ? `${a.quota_mb} MB` : "Unbegrenzt"}</td>
                          <td className="px-4 py-3 text-muted-foreground">{new Date(a.created_at).toLocaleDateString("de-DE")}</td>
                          <td className="px-4 py-3 text-right">
                            <button onClick={() => setDeleteTarget({ type: "account", id: a.id })} className="text-muted-foreground hover:text-destructive transition-colors">
                              <Trash2 className="w-4 h-4" />
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}

          {/* Aliases Tab */}
          {tab === "aliases" && (
            aliases.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
                <Mail className="w-12 h-12 mb-4 opacity-30" />
                <p className="font-medium">Keine Aliases vorhanden</p>
              </div>
            ) : (
              <div className="bg-card border border-border rounded-xl overflow-hidden">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border text-muted-foreground">
                      <th className="text-left px-4 py-3 font-medium">Quelle</th>
                      <th className="text-left px-4 py-3 font-medium">Ziel</th>
                      <th className="text-left px-4 py-3 font-medium">Domain</th>
                      <th className="text-right px-4 py-3 font-medium">Aktionen</th>
                    </tr>
                  </thead>
                  <tbody>
                    {aliases.map((a) => (
                      <tr key={a.id} className="border-b border-border last:border-0 hover:bg-accent/50 transition-colors">
                        <td className="px-4 py-3 font-medium text-foreground font-mono text-xs">{a.source}</td>
                        <td className="px-4 py-3 text-muted-foreground font-mono text-xs">{a.destination}</td>
                        <td className="px-4 py-3 text-muted-foreground">{a.domain_name || domainName(a.domain_id)}</td>
                        <td className="px-4 py-3 text-right">
                          <button onClick={() => setDeleteTarget({ type: "alias", id: a.id })} className="text-muted-foreground hover:text-destructive transition-colors">
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )
          )}
        </>
      )}

      {/* Add Domain Modal */}
      {showAddDomain && (
        <Modal title="Mail-Domain hinzufügen" onClose={() => setShowAddDomain(false)}>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Server</label>
              <select
                value={domainForm.server_id}
                onChange={(e) => setDomainForm({ ...domainForm, server_id: e.target.value })}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
              >
                <option value="">Server auswählen...</option>
                {servers.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Domain</label>
              <input
                type="text"
                value={domainForm.domain}
                onChange={(e) => setDomainForm({ ...domainForm, domain: e.target.value })}
                placeholder="example.com"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button onClick={() => setShowAddDomain(false)} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">Abbrechen</button>
              <button
                onClick={handleAddDomain}
                disabled={saving || !domainForm.server_id || !domainForm.domain}
                className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {saving ? "Wird hinzugefügt..." : "Hinzufügen"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Add Account Modal */}
      {showAddAccount && (
        <Modal title="E-Mail-Konto erstellen" onClose={() => setShowAddAccount(false)}>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Domain</label>
              <select
                value={accountForm.domain_id}
                onChange={(e) => setAccountForm({ ...accountForm, domain_id: e.target.value })}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
              >
                <option value="">Domain auswählen...</option>
                {domains.map((d) => <option key={d.id} value={d.id}>{d.domain}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Benutzername</label>
              <input
                type="text"
                value={accountForm.username}
                onChange={(e) => setAccountForm({ ...accountForm, username: e.target.value })}
                placeholder="info"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
              {accountForm.domain_id && (
                <p className="text-xs text-muted-foreground mt-1">
                  Vollständige Adresse: {accountForm.username}@{domainName(accountForm.domain_id)}
                </p>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Passwort</label>
              <input
                type="password"
                value={accountForm.password}
                onChange={(e) => setAccountForm({ ...accountForm, password: e.target.value })}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Postfachgröße</label>
              <select
                value={accountForm.quota_custom ? "custom" : accountForm.quota_mb}
                onChange={e => {
                  const v = e.target.value;
                  if (v === "custom") {
                    setAccountForm({ ...accountForm, quota_custom: true, quota_mb: "500" });
                  } else {
                    setAccountForm({ ...accountForm, quota_custom: false, quota_mb: v });
                  }
                }}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
              >
                <option value="0">Unbegrenzt</option>
                <option value="100">100 MB</option>
                <option value="250">250 MB</option>
                <option value="500">500 MB</option>
                <option value="1024">1 GB</option>
                <option value="2048">2 GB</option>
                <option value="5120">5 GB</option>
                <option value="10240">10 GB</option>
                <option value="custom">Benutzerdefiniert…</option>
              </select>
              {accountForm.quota_custom && (
                <div className="flex items-center gap-2 mt-2">
                  <input type="number" min="1" value={accountForm.quota_mb}
                    onChange={e => setAccountForm({ ...accountForm, quota_mb: e.target.value })}
                    placeholder="z.B. 750"
                    className="flex-1 bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground" />
                  <span className="text-sm text-muted-foreground">MB</span>
                </div>
              )}
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button onClick={() => setShowAddAccount(false)} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">Abbrechen</button>
              <button
                onClick={handleAddAccount}
                disabled={saving || !accountForm.domain_id || !accountForm.username || !accountForm.password}
                className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {saving ? "Wird erstellt..." : "Erstellen"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Add Alias Modal */}
      {showAddAlias && (
        <Modal title="Alias hinzufügen" onClose={() => setShowAddAlias(false)}>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Domain</label>
              <select
                value={aliasForm.domain_id}
                onChange={(e) => setAliasForm({ ...aliasForm, domain_id: e.target.value })}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
              >
                <option value="">Domain auswählen...</option>
                {domains.map((d) => <option key={d.id} value={d.id}>{d.domain}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Quelle (E-Mail-Adresse)</label>
              <input
                type="text"
                value={aliasForm.source}
                onChange={(e) => setAliasForm({ ...aliasForm, source: e.target.value })}
                placeholder="kontakt@example.com"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Ziel (Weiterleitungsadresse)</label>
              <input
                type="text"
                value={aliasForm.destination}
                onChange={(e) => setAliasForm({ ...aliasForm, destination: e.target.value })}
                placeholder="info@example.com"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button onClick={() => setShowAddAlias(false)} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">Abbrechen</button>
              <button
                onClick={handleAddAlias}
                disabled={saving || !aliasForm.domain_id || !aliasForm.source || !aliasForm.destination}
                className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {saving ? "Wird erstellt..." : "Hinzufügen"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Delete Confirm */}
      {deleteTarget && (
        <Modal title="Löschen bestätigen" onClose={() => setDeleteTarget(null)}>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">Möchten Sie diesen Eintrag wirklich löschen?</p>
            <div className="flex justify-end gap-3">
              <button onClick={() => setDeleteTarget(null)} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">Abbrechen</button>
              <button onClick={handleDelete} className="px-4 py-2 text-sm bg-destructive text-white rounded-lg hover:bg-destructive/90 transition-colors">Löschen</button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
}
