"use client";

import { useEffect, useState, useRef } from "react";
import { Folder, File, ChevronRight, Home, Plus, Upload, Trash2, Download, Edit, X, AlertCircle, RefreshCw } from "lucide-react";
import { api, type Server } from "@/lib/api";

interface FileEntry {
  name: string;
  type: "file" | "dir";
  size: number;
  mode: string;
  modified: string;
}

function Skeleton({ className }: { className?: string }) {
  return <div className={`bg-secondary animate-pulse rounded ${className}`} />;
}

function Modal({ title, onClose, children, wide }: { title: string; onClose: () => void; children: React.ReactNode; wide?: boolean }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div className={`bg-card border border-border rounded-xl mx-4 shadow-xl flex flex-col ${wide ? "w-full max-w-3xl" : "w-full max-w-lg"}`}>
        <div className="flex items-center justify-between p-4 border-b border-border">
          <h2 className="font-semibold text-foreground">{title}</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="w-5 h-5" />
          </button>
        </div>
        <div className="p-4 overflow-auto">{children}</div>
      </div>
    </div>
  );
}

function formatSize(bytes: number): string {
  if (bytes === 0) return "-";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

export default function FilesPage() {
  const [servers, setServers] = useState<Server[]>([]);
  const [selectedServer, setSelectedServer] = useState("");
  const [path, setPath] = useState("/var/www");
  const [entries, setEntries] = useState<FileEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const [showNewFolder, setShowNewFolder] = useState(false);
  const [newFolderName, setNewFolderName] = useState("");
  const [showEdit, setShowEdit] = useState<FileEntry | null>(null);
  const [editContent, setEditContent] = useState("");
  const [editLoading, setEditLoading] = useState(false);
  const [savingEdit, setSavingEdit] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<FileEntry | null>(null);

  const uploadRef = useRef<HTMLInputElement>(null);

  async function loadServers() {
    try {
      const sv = await api.get<Server[]>("/servers");
      setServers(sv);
      if (sv.length > 0) setSelectedServer(sv[0].id);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Laden");
    }
  }

  async function loadDir(serverId: string, dirPath: string) {
    if (!serverId) return;
    setLoading(true);
    setError("");
    try {
      const result = await api.get<FileEntry[]>(`/servers/${serverId}/files?path=${encodeURIComponent(dirPath)}`);
      setEntries(result || []);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Laden");
      setEntries([]);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { loadServers(); }, []);
  useEffect(() => { if (selectedServer) loadDir(selectedServer, path); }, [selectedServer]);

  function navigate(entry: FileEntry) {
    if (entry.type === "dir") {
      const newPath = path.endsWith("/") ? `${path}${entry.name}` : `${path}/${entry.name}`;
      setPath(newPath);
      loadDir(selectedServer, newPath);
    }
  }

  function navigateTo(p: string) {
    setPath(p);
    loadDir(selectedServer, p);
  }

  function navigateUp() {
    const parts = path.split("/").filter(Boolean);
    parts.pop();
    const newPath = "/" + parts.join("/") || "/";
    setPath(newPath);
    loadDir(selectedServer, newPath);
  }

  function getBreadcrumbs() {
    const parts = path.split("/").filter(Boolean);
    return [{ name: "/", path: "/" }, ...parts.map((p, i) => ({
      name: p,
      path: "/" + parts.slice(0, i + 1).join("/"),
    }))];
  }

  async function handleNewFolder() {
    if (!newFolderName) return;
    try {
      await api.post(`/servers/${selectedServer}/files/mkdir`, {
        path: `${path}/${newFolderName}`,
      });
      setShowNewFolder(false);
      setNewFolderName("");
      loadDir(selectedServer, path);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler");
    }
  }

  async function openEdit(entry: FileEntry) {
    setShowEdit(entry);
    setEditLoading(true);
    setEditContent("");
    try {
      const result = await api.get<{ content: string }>(
        `/servers/${selectedServer}/files/read?path=${encodeURIComponent(`${path}/${entry.name}`)}`
      );
      setEditContent(result.content || "");
    } catch (e: unknown) {
      setEditContent(`Fehler: ${e instanceof Error ? e.message : "Unbekannt"}`);
    } finally {
      setEditLoading(false);
    }
  }

  async function handleSaveEdit() {
    if (!showEdit) return;
    setSavingEdit(true);
    try {
      await api.put(`/servers/${selectedServer}/files/write`, {
        path: `${path}/${showEdit.name}`,
        content: editContent,
      });
      setShowEdit(null);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Speichern");
    } finally {
      setSavingEdit(false);
    }
  }

  async function handleUpload(file: File) {
    const reader = new FileReader();
    reader.onload = async (e) => {
      const content = e.target?.result as string;
      try {
        await api.post(`/servers/${selectedServer}/files/upload`, {
          path: `${path}/${file.name}`,
          content,
        });
        loadDir(selectedServer, path);
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : "Upload-Fehler");
      }
    };
    reader.readAsText(file);
  }

  async function handleDelete(entry: FileEntry) {
    try {
      await api.delete(`/servers/${selectedServer}/files?path=${encodeURIComponent(`${path}/${entry.name}`)}`);
      setDeleteTarget(null);
      loadDir(selectedServer, path);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Löschen");
    }
  }

  async function handleDownload(entry: FileEntry) {
    try {
      const result = await api.get<{ content: string }>(
        `/servers/${selectedServer}/files/read?path=${encodeURIComponent(`${path}/${entry.name}`)}`
      );
      const blob = new Blob([result.content], { type: "text/plain" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = entry.name;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Fehler beim Download");
    }
  }

  const sortedEntries = [...entries].sort((a, b) => {
    if (a.type !== b.type) return a.type === "dir" ? -1 : 1;
    return a.name.localeCompare(b.name);
  });

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Dateien</h1>
          <p className="text-muted-foreground text-sm mt-1">Server-Dateisystem durchsuchen</p>
        </div>
        <div className="flex items-center gap-3">
          {servers.length > 0 && (
            <select
              value={selectedServer}
              onChange={(e) => { setSelectedServer(e.target.value); setPath("/var/www"); }}
              className="bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
            >
              {servers.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
            </select>
          )}
          <button
            onClick={() => loadDir(selectedServer, path)}
            disabled={loading}
            className="p-2 border border-border rounded-lg hover:bg-accent transition-colors text-muted-foreground hover:text-foreground disabled:opacity-50"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? "animate-spin" : ""}`} />
          </button>
          <button
            onClick={() => setShowNewFolder(true)}
            className="flex items-center gap-2 px-3 py-2 border border-border rounded-lg text-sm hover:bg-accent transition-colors text-muted-foreground hover:text-foreground"
          >
            <Plus className="w-4 h-4" />
            Ordner
          </button>
          <button
            onClick={() => uploadRef.current?.click()}
            className="flex items-center gap-2 px-3 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors"
          >
            <Upload className="w-4 h-4" />
            Hochladen
          </button>
          <input
            ref={uploadRef}
            type="file"
            className="hidden"
            onChange={(e) => e.target.files?.[0] && handleUpload(e.target.files[0])}
          />
        </div>
      </div>

      {error && (
        <div className="flex items-center gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-lg text-destructive text-sm mb-4">
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          {error}
          <button onClick={() => setError("")} className="ml-auto"><X className="w-4 h-4" /></button>
        </div>
      )}

      {/* Breadcrumb */}
      <div className="flex items-center gap-1 mb-4 bg-card border border-border rounded-lg px-3 py-2 text-sm overflow-x-auto">
        {getBreadcrumbs().map((b, i) => (
          <span key={b.path} className="flex items-center gap-1 flex-shrink-0">
            {i > 0 && <ChevronRight className="w-3 h-3 text-muted-foreground" />}
            <button
              onClick={() => navigateTo(b.path)}
              className={`hover:text-foreground transition-colors ${i === getBreadcrumbs().length - 1 ? "text-foreground font-medium" : "text-muted-foreground"}`}
            >
              {i === 0 ? <Home className="w-4 h-4" /> : b.name}
            </button>
          </span>
        ))}
      </div>

      {/* File table */}
      <div className="bg-card border border-border rounded-xl overflow-hidden">
        {loading ? (
          <div className="p-4 space-y-2">
            {[1, 2, 3, 4, 5].map((i) => <Skeleton key={i} className="h-10 w-full" />)}
          </div>
        ) : !selectedServer ? (
          <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
            <Folder className="w-12 h-12 mb-4 opacity-30" />
            <p>Kein Server ausgewählt</p>
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-muted-foreground">
                <th className="text-left px-4 py-3 font-medium">Name</th>
                <th className="text-left px-4 py-3 font-medium">Größe</th>
                <th className="text-left px-4 py-3 font-medium">Berechtigungen</th>
                <th className="text-left px-4 py-3 font-medium">Geändert</th>
                <th className="text-right px-4 py-3 font-medium">Aktionen</th>
              </tr>
            </thead>
            <tbody>
              {path !== "/" && (
                <tr
                  className="border-b border-border hover:bg-accent/50 transition-colors cursor-pointer"
                  onClick={navigateUp}
                >
                  <td className="px-4 py-2.5" colSpan={5}>
                    <div className="flex items-center gap-2 text-muted-foreground">
                      <Folder className="w-4 h-4" />
                      <span>..</span>
                    </div>
                  </td>
                </tr>
              )}
              {sortedEntries.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-4 py-10 text-center text-muted-foreground">
                    Verzeichnis ist leer
                  </td>
                </tr>
              ) : (
                sortedEntries.map((entry) => (
                  <tr
                    key={entry.name}
                    className="border-b border-border last:border-0 hover:bg-accent/50 transition-colors"
                  >
                    <td className="px-4 py-2.5">
                      <div
                        className={`flex items-center gap-2 ${entry.type === "dir" ? "cursor-pointer" : ""}`}
                        onClick={() => entry.type === "dir" && navigate(entry)}
                      >
                        {entry.type === "dir"
                          ? <Folder className="w-4 h-4 text-blue-400 flex-shrink-0" />
                          : <File className="w-4 h-4 text-muted-foreground flex-shrink-0" />}
                        <span className={`font-medium ${entry.type === "dir" ? "text-blue-400 hover:text-blue-300" : "text-foreground"}`}>
                          {entry.name}
                        </span>
                      </div>
                    </td>
                    <td className="px-4 py-2.5 text-muted-foreground font-mono text-xs">
                      {entry.type === "dir" ? "-" : formatSize(entry.size)}
                    </td>
                    <td className="px-4 py-2.5 font-mono text-xs text-muted-foreground">{entry.mode}</td>
                    <td className="px-4 py-2.5 text-muted-foreground text-xs">
                      {entry.modified ? new Date(entry.modified).toLocaleString("de-DE") : "-"}
                    </td>
                    <td className="px-4 py-2.5">
                      <div className="flex items-center justify-end gap-2">
                        {entry.type === "file" && (
                          <>
                            <button
                              onClick={() => openEdit(entry)}
                              className="text-muted-foreground hover:text-foreground transition-colors"
                              title="Bearbeiten"
                            >
                              <Edit className="w-4 h-4" />
                            </button>
                            <button
                              onClick={() => handleDownload(entry)}
                              className="text-muted-foreground hover:text-foreground transition-colors"
                              title="Herunterladen"
                            >
                              <Download className="w-4 h-4" />
                            </button>
                          </>
                        )}
                        <button
                          onClick={() => setDeleteTarget(entry)}
                          className="text-muted-foreground hover:text-destructive transition-colors"
                          title="Löschen"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        )}
      </div>

      {/* New Folder Modal */}
      {showNewFolder && (
        <Modal title="Neuen Ordner erstellen" onClose={() => setShowNewFolder(false)}>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">Ordnername</label>
              <input
                type="text"
                value={newFolderName}
                onChange={(e) => setNewFolderName(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleNewFolder()}
                placeholder="neuer-ordner"
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
                autoFocus
              />
            </div>
            <div className="flex justify-end gap-3">
              <button onClick={() => setShowNewFolder(false)} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">Abbrechen</button>
              <button
                onClick={handleNewFolder}
                disabled={!newFolderName}
                className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                Erstellen
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Edit File Modal */}
      {showEdit && (
        <Modal title={`Datei bearbeiten – ${showEdit.name}`} onClose={() => setShowEdit(null)} wide>
          <div className="space-y-4">
            {editLoading ? (
              <Skeleton className="h-80 w-full" />
            ) : (
              <textarea
                value={editContent}
                onChange={(e) => setEditContent(e.target.value)}
                className="w-full h-96 bg-zinc-950 border border-border rounded-lg p-3 text-xs font-mono text-foreground resize-none focus:outline-none focus:ring-1 focus:ring-primary"
                spellCheck={false}
              />
            )}
            <div className="flex justify-end gap-3">
              <button onClick={() => setShowEdit(null)} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">Schließen</button>
              <button
                onClick={handleSaveEdit}
                disabled={savingEdit || editLoading}
                className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {savingEdit ? "Wird gespeichert..." : "Speichern"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Delete Confirm */}
      {deleteTarget && (
        <Modal title={`${deleteTarget.type === "dir" ? "Ordner" : "Datei"} löschen`} onClose={() => setDeleteTarget(null)}>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Möchten Sie <span className="text-foreground font-medium">{deleteTarget.name}</span> wirklich löschen?
              {deleteTarget.type === "dir" && " Alle enthaltenen Dateien werden ebenfalls gelöscht."}
            </p>
            <div className="flex justify-end gap-3">
              <button onClick={() => setDeleteTarget(null)} className="px-4 py-2 text-sm border border-border rounded-lg hover:bg-accent transition-colors">Abbrechen</button>
              <button onClick={() => handleDelete(deleteTarget)} className="px-4 py-2 text-sm bg-destructive text-white rounded-lg hover:bg-destructive/90 transition-colors">Löschen</button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
}
