"use client";

import { useEffect, useState } from "react";
import { Globe2, Plus, Trash2, X, AlertCircle } from "lucide-react";
import { api, type DNSZone, type DNSRecord, type Server } from "@/lib/api";

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

const recordTypeColors: Record<string, string> = {
  A: "bg-blue-500/20 text-blue-400",
  AAAA: "bg-indigo-500/20 text-indigo-400",
  CNAME: "bg-purple-500/20 text-purple-400",
  MX: "bg-orange-500/20 text-orange-400",
  TXT: "bg-yellow-500/20 text-yellow-400",
  SRV: "bg-pink-500/20 text-pink-400",
  CAA: "bg-red-500/20 text-red-400",
};

export default function DNSPage() {
  const [zones, setZones] = useState<DNSZone[]>([]);
  const [records, setRecords] = useState<DNSRecord[]>([]);
  const [servers, setServers] = useState<Server[]>([]);
  const [selectedZone, setSelectedZone] = useState<DNSZone | null>(null);
  const [loading, setLoading] = useState(true);
  const [recordsLoading, setRecordsLoading] = useState(false);
  const [error, setError] = useState("");
  const [showAddZone, setShowAddZone] = useState(false);
  const [showAddRecord, setShowAddRecord] = useState(false);
  const [deleteZoneId, setDeleteZoneId] = useState<string | null>(null);
  const [deleteRecordId, setDeleteRecordId] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const [zoneForm, setZoneForm] = useState({ server_id: "", name: "", zone_type: "master", master_ip: "", nameserver: "", admin_email: "" });
  const [recordForm, setRecordForm] = useState({
    name: "",
    type: "A" as DNSRecord["type"],
    content: "",
    ttl: "3600",
    priority: "",
  });

  async function loadZones() {
    try {
      const [z, sv] = await Promise.all([
        api.get<DNSZone[]>("/dns/zones"),
        api.get<Server[]>("/servers"),
      ]);
      setZones(z);
      setServers(sv);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Laden");
    } finally {
      setLoading(false);
    }
  }

  async function loadRecords(zoneId: string) {
    setRecordsLoading(true);
    try {
      const r = await api.get<DNSRecord[]>(`/dns/zones/${zoneId}/records`);
      setRecords(r);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Laden");
    } finally {
      setRecordsLoading(false);
    }
  }

  useEffect(() => { loadZones(); }, []);

  function selectZone(zone: DNSZone) {
    setSelectedZone(zone);
    loadRecords(zone.id);
  }

  async function handleAddZone() {
    setSaving(true);
    try {
      await api.post("/dns/zones", zoneForm);
      setShowAddZone(false);
      setZoneForm({ server_id: "", name: "", zone_type: "master", master_ip: "", nameserver: "", admin_email: "" });
      await loadZones();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSaving(false);
    }
  }

  async function handleAddRecord() {
    if (!selectedZone) return;
    setSaving(true);
    try {
      await api.post(`/dns/zones/${selectedZone.id}/records`, {
        ...recordForm,
        ttl: parseInt(recordForm.ttl) || 3600,
        priority: recordForm.priority ? parseInt(recordForm.priority) : undefined,
      });
      setShowAddRecord(false);
      setRecordForm({ name: "", type: "A", content: "", ttl: "3600", priority: "" });
      await loadRecords(selectedZone.id);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    } finally {
      setSaving(false);
    }
  }

  async function handleDeleteZone(id: string) {
    try {
      await api.delete(`/dns/zones/${id}`);
      setDeleteZoneId(null);
      if (selectedZone?.id === id) {
        setSelectedZone(null);
        setRecords([]);
      }
      await loadZones();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    }
  }

  async function handleDeleteRecord(id: string) {
    if (!selectedZone) return;
    try {
      await api.delete(`/dns/records/${id}`);
      setDeleteRecordId(null);
      await loadRecords(selectedZone.id);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    }
  }

  const serverName = (id: string) => servers.find((s) => s.id === id)?.name || id;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">DNS</h1>
          <p className="text-muted-foreground text-sm mt-1">DNS-Zonen und Einträge verwalten</p>
        </div>
      </div>

      {error && (
        <div className="flex items-center gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-lg text-destructive text-sm mb-4">
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          {error}
          <button onClick={() => setError("")} className="ml-auto"><X className="w-4 h-4" /></button>
        </div>
      )}

      <div className="flex gap-6 h-[calc(100vh-13rem)]">
        {/* Left: Zone list */}
        <div className="w-72 flex flex-col bg-card border border-border rounded-xl overflow-hidden flex-shrink-0">
          <div className="flex items-center justify-between p-3 border-b border-border">
            <span className="text-sm font-medium text-foreground">DNS-Zonen</span>
            <button
              onClick={() => setShowAddZone(true)}
              className="flex items-center gap-1 px-2 py-1 text-xs bg-primary text-primary-foreground rounded hover:bg-primary/90 transition-colors"
            >
              <Plus className="w-3 h-3" />
              Zone
            </button>
          </div>
          <div className="flex-1 overflow-y-auto">
            {loading ? (
              <div className="p-3 space-y-2">
                {[1, 2, 3].map((i) => <Skeleton key={i} className="h-10 w-full" />)}
              </div>
            ) : zones.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-full text-muted-foreground p-4 text-center">
                <Globe2 className="w-8 h-8 mb-2 opacity-30" />
                <p className="text-sm">Keine Zonen vorhanden</p>
              </div>
            ) : (
              zones.map((zone) => (
                <div
                  key={zone.id}
                  onClick={() => selectZone(zone)}
                  className={`flex items-center justify-between px-3 py-2.5 cursor-pointer border-b border-border last:border-0 transition-colors ${
                    selectedZone?.id === zone.id ? "bg-primary/10" : "hover:bg-accent"
                  }`}
                >
                  <div className="min-w-0">
                    <div className={`text-sm font-medium truncate ${selectedZone?.id === zone.id ? "text-primary" : "text-foreground"}`}>
                      {zone.name}
                    </div>
                    <div className="flex items-center gap-1.5 mt-0.5">
                      <span className="text-xs text-muted-foreground">{zone.server_name || serverName(zone.server_id)}</span>
                      <span className={`px-1 py-0.5 rounded text-[10px] font-medium ${zone.zone_type === "slave" ? "bg-orange-500/20 text-orange-400" : "bg-blue-500/20 text-blue-400"}`}>
                        {zone.zone_type ?? "master"}
                      </span>
                    </div>
                  </div>
                  <button
                    onClick={(e) => { e.stopPropagation(); setDeleteZoneId(zone.id); }}
                    className="text-muted-foreground hover:text-destructive transition-colors opacity-0 group-hover:opacity-100"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Right: Records */}
        <div className="flex-1 flex flex-col bg-card border border-border rounded-xl overflow-hidden">
          {!selectedZone ? (
            <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
              <Globe2 className="w-12 h-12 mb-4 opacity-30" />
              <p className="font-medium">Zone auswählen</p>
              <p className="text-sm mt-1">Wählen Sie links eine DNS-Zone aus</p>
            </div>
          ) : (
            <>
              <div className="flex items-center justify-between p-3 border-b border-border">
                <div>
                  <span className="text-sm font-medium text-foreground">{selectedZone.name}</span>
                  <span className="text-xs text-muted-foreground ml-2">DNS-Einträge</span>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => setDeleteZoneId(selectedZone.id)}
                    className="flex items-center gap-1 px-2 py-1 text-xs border border-destructive/40 text-destructive rounded hover:bg-destructive/10 transition-colors"
                  >
                    <Trash2 className="w-3 h-3" />
                    Zone löschen
                  </button>
                  <button
                    onClick={() => setShowAddRecord(true)}
                    className="flex items-center gap-1 px-2 py-1 text-xs bg-primary text-primary-foreground rounded hover:bg-primary/90 transition-colors"
                  >
                    <Plus className="w-3 h-3" />
                    Eintrag hinzufügen
                  </button>
                </div>
              </div>
              <div className="flex-1 overflow-y-auto">
                {recordsLoading ? (
                  <div className="p-4 space-y-2">
                    {[1, 2, 3, 4].map((i) => <Skeleton key={i} className="h-8 w-full" />)}
                  </div>
                ) : records.length === 0 ? (
                  <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
                    <p className="text-sm">Keine Einträge vorhanden</p>
                  </div>
                ) : (
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-border text-muted-foreground">
                        <th className="text-left px-4 py-2.5 font-medium">Name</th>
                        <th className="text-left px-4 py-2.5 font-medium">Typ</th>
                        <th className="text-left px-4 py-2.5 font-medium">Inhalt</th>
                        <th className="text-left px-4 py-2.5 font-medium">TTL</th>
                        <th className="text-left px-4 py-2.5 font-medium">Prio</th>
                        <th className="text-right px-4 py-2.5 font-medium">Aktion</th>
                      </tr>
                    </thead>
                    <tbody>
                      {records.map((r) => (
                        <tr key={r.id} className="border-b border-border last:border-0 hover:bg-accent/50 transition-colors">
                          <td className="px-4 py-2.5 font-mono text-xs text-foreground">{r.name}</td>
                          <td className="px-4 py-2.5">
                            <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium font-mono ${recordTypeColors[r.type] || "bg-zinc-500/20 text-zinc-400"}`}>
                              {r.type}
                            </span>
                          </td>
                          <td className="px-4 py-2.5 font-mono text-xs text-muted-foreground max-w-xs truncate">{r.content}</td>
                          <td className="px-4 py-2.5 text-muted-foreground">{r.ttl}</td>
                          <td className="px-4 py-2.5 text-muted-foreground">{r.priority ?? "-"}</td>
                          <td className="px-4 py-2.5 text-right">
                            <button
                              onClick={() => setDeleteRecordId(r.id)}
                              className="text-muted-foreground hover:text-destructive transition-colors"
                            >
                              <Trash2 className="w-4 h-4" />
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            </>
          )}
        </div>
      </div>

      {/* Add Zone Modal */}
      {showAddZone && (
        <Modal title="DNS-Zone hinzufügen" onClose={() => setShowAddZone(false)}>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Server</label>
              <select
                value={zoneForm.server_id}
                onChange={(e) => setZoneForm({ ...zoneForm, server_id: e.target.value })}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
              >
                <option value="">Server auswählen...</option>
                {servers.map((s) => (
                  <option key={s.id} value={s.id}>{s.name} ({s.ip_address})</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Zonenname (Domain)</label>
              <input
                type="text"
                value={zoneForm.name}
                onChange={(e) => setZoneForm({ ...zoneForm, name: e.target.value })}
                placeholder="example.com"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Zonentyp</label>
              <select
                value={zoneForm.zone_type}
                onChange={(e) => setZoneForm({ ...zoneForm, zone_type: e.target.value, master_ip: "" })}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
              >
                <option value="master">Master (autoritativ)</option>
                <option value="slave">Slave (repliziert vom Master)</option>
              </select>
            </div>
            {zoneForm.zone_type === "slave" && (
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">Master-IP *</label>
                <input
                  type="text"
                  value={zoneForm.master_ip}
                  onChange={(e) => setZoneForm({ ...zoneForm, master_ip: e.target.value })}
                  placeholder="1.2.3.4"
                  className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
                />
                <p className="text-xs text-muted-foreground mt-1">IP des Master-DNS-Servers</p>
              </div>
            )}
            {zoneForm.zone_type === "master" && (
              <>
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Nameserver</label>
                  <input
                    type="text"
                    value={zoneForm.nameserver}
                    onChange={(e) => setZoneForm({ ...zoneForm, nameserver: e.target.value })}
                    placeholder={`ns1.${zoneForm.name || "example.com"}`}
                    className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Admin E-Mail</label>
                  <input
                    type="email"
                    value={zoneForm.admin_email}
                    onChange={(e) => setZoneForm({ ...zoneForm, admin_email: e.target.value })}
                    placeholder={`admin@${zoneForm.name || "example.com"}`}
                    className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
                  />
                </div>
              </>
            )}
            <div className="flex justify-end gap-3 pt-2">
              <button onClick={() => setShowAddZone(false)} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">
                Abbrechen
              </button>
              <button
                onClick={handleAddZone}
                disabled={saving || !zoneForm.server_id || !zoneForm.name || (zoneForm.zone_type === "slave" && !zoneForm.master_ip)}
                className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {saving ? "Wird erstellt..." : "Zone erstellen"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Add Record Modal */}
      {showAddRecord && selectedZone && (
        <Modal title={`Eintrag hinzufügen – ${selectedZone.name}`} onClose={() => setShowAddRecord(false)}>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Name</label>
              <input
                type="text"
                value={recordForm.name}
                onChange={(e) => setRecordForm({ ...recordForm, name: e.target.value })}
                placeholder="@ oder subdomain"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">Typ</label>
                <select
                  value={recordForm.type}
                  onChange={(e) => setRecordForm({ ...recordForm, type: e.target.value as DNSRecord["type"] })}
                  className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
                >
                  {["A", "AAAA", "CNAME", "MX", "TXT", "SRV", "CAA"].map((t) => (
                    <option key={t} value={t}>{t}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">TTL</label>
                <input
                  type="number"
                  value={recordForm.ttl}
                  onChange={(e) => setRecordForm({ ...recordForm, ttl: e.target.value })}
                  className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
                />
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Inhalt</label>
              <input
                type="text"
                value={recordForm.content}
                onChange={(e) => setRecordForm({ ...recordForm, content: e.target.value })}
                placeholder="IP-Adresse, Hostname oder Text"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
              />
            </div>
            {(recordForm.type === "MX" || recordForm.type === "SRV") && (
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">Priorität</label>
                <input
                  type="number"
                  value={recordForm.priority}
                  onChange={(e) => setRecordForm({ ...recordForm, priority: e.target.value })}
                  placeholder="10"
                  className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
                />
              </div>
            )}
            <div className="flex justify-end gap-3 pt-2">
              <button onClick={() => setShowAddRecord(false)} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">
                Abbrechen
              </button>
              <button
                onClick={handleAddRecord}
                disabled={saving || !recordForm.name || !recordForm.content}
                className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {saving ? "Wird erstellt..." : "Hinzufügen"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Delete Zone Confirm */}
      {deleteZoneId && (
        <Modal title="Zone löschen" onClose={() => setDeleteZoneId(null)}>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Möchten Sie diese DNS-Zone und alle zugehörigen Einträge wirklich löschen?
            </p>
            <div className="flex justify-end gap-3">
              <button onClick={() => setDeleteZoneId(null)} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">Abbrechen</button>
              <button onClick={() => handleDeleteZone(deleteZoneId)} className="px-4 py-2 text-sm bg-destructive text-white rounded-lg hover:bg-destructive/90 transition-colors">Löschen</button>
            </div>
          </div>
        </Modal>
      )}

      {/* Delete Record Confirm */}
      {deleteRecordId && (
        <Modal title="Eintrag löschen" onClose={() => setDeleteRecordId(null)}>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">Möchten Sie diesen DNS-Eintrag wirklich löschen?</p>
            <div className="flex justify-end gap-3">
              <button onClick={() => setDeleteRecordId(null)} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">Abbrechen</button>
              <button onClick={() => handleDeleteRecord(deleteRecordId)} className="px-4 py-2 text-sm bg-destructive text-white rounded-lg hover:bg-destructive/90 transition-colors">Löschen</button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
}
