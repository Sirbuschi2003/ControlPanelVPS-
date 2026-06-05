"use client";

import { useEffect, useState } from "react";
import { Terminal, Copy, Server as ServerIcon, X, AlertCircle } from "lucide-react";
import { api, type Server } from "@/lib/api";

export default function TerminalPage() {
  const [servers, setServers] = useState<Server[]>([]);
  const [selectedServer, setSelectedServer] = useState<Server | null>(null);
  const [error, setError] = useState("");
  const [copied, setCopied] = useState("");

  useEffect(() => {
    api.get<Server[]>("/servers")
      .then((sv) => {
        setServers(sv);
        if (sv.length > 0) setSelectedServer(sv[0]);
      })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : "Fehler beim Laden"));
  }, []);

  function copyToClipboard(text: string, key: string) {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(key);
      setTimeout(() => setCopied(""), 2000);
    });
  }

  const sshCommand = selectedServer
    ? `ssh root@${selectedServer.ip_address}`
    : "";

  const sshCommandPort = selectedServer
    ? `ssh -p 22 root@${selectedServer.ip_address}`
    : "";

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Terminal</h1>
          <p className="text-muted-foreground text-sm mt-1">SSH-Verbindungsinformationen</p>
        </div>
        {servers.length > 0 && (
          <select
            value={selectedServer?.id || ""}
            onChange={(e) => {
              const sv = servers.find((s) => s.id === e.target.value);
              setSelectedServer(sv || null);
            }}
            className="bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
          >
            {servers.map((s) => (
              <option key={s.id} value={s.id}>{s.name}</option>
            ))}
          </select>
        )}
      </div>

      {error && (
        <div className="flex items-center gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-lg text-destructive text-sm mb-4">
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          {error}
          <button onClick={() => setError("")} className="ml-auto"><X className="w-4 h-4" /></button>
        </div>
      )}

      {!selectedServer ? (
        <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
          <Terminal className="w-12 h-12 mb-4 opacity-30" />
          <p className="font-medium">Kein Server verfügbar</p>
          <p className="text-sm mt-1">Fügen Sie zuerst einen Server hinzu</p>
        </div>
      ) : (
        <div className="space-y-6 max-w-2xl">
          {/* Info Card */}
          <div className="bg-card border border-border rounded-xl p-6">
            <div className="flex items-start gap-4">
              <div className="w-10 h-10 bg-primary/10 border border-primary/20 rounded-lg flex items-center justify-center flex-shrink-0">
                <ServerIcon className="w-5 h-5 text-primary" />
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-foreground">{selectedServer.name}</h3>
                <div className="mt-3 space-y-2 text-sm">
                  <div className="flex items-center gap-2">
                    <span className="text-muted-foreground w-24">IP-Adresse:</span>
                    <span className="font-mono text-foreground">{selectedServer.ip_address}</span>
                    <button
                      onClick={() => copyToClipboard(selectedServer.ip_address, "ip")}
                      className="text-muted-foreground hover:text-foreground transition-colors"
                    >
                      <Copy className="w-3.5 h-3.5" />
                    </button>
                    {copied === "ip" && <span className="text-xs text-green-400">Kopiert!</span>}
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-muted-foreground w-24">Hostname:</span>
                    <span className="font-mono text-foreground">{selectedServer.hostname}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-muted-foreground w-24">SSH-Port:</span>
                    <span className="font-mono text-foreground">22</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-muted-foreground w-24">Status:</span>
                    <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${
                      selectedServer.status === "online" ? "text-green-400" :
                      selectedServer.status === "offline" ? "text-red-400" : "text-yellow-400"
                    }`}>
                      <span className={`w-1.5 h-1.5 rounded-full ${
                        selectedServer.status === "online" ? "bg-green-400" :
                        selectedServer.status === "offline" ? "bg-red-400" : "bg-yellow-400"
                      }`} />
                      {selectedServer.status === "online" ? "Online" :
                       selectedServer.status === "offline" ? "Offline" : "Unbekannt"}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* SSH Connection */}
          <div className="bg-card border border-border rounded-xl p-6">
            <div className="flex items-center gap-3 mb-4">
              <Terminal className="w-5 h-5 text-primary" />
              <h3 className="font-semibold text-foreground">Verbindung über SSH herstellen</h3>
            </div>
            <p className="text-sm text-muted-foreground mb-4">
              Öffnen Sie ein Terminal und verwenden Sie einen der folgenden Befehle, um sich mit dem Server zu verbinden.
            </p>

            <div className="space-y-3">
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5 uppercase tracking-wide">Standard SSH</label>
                <div className="flex items-center gap-2 bg-zinc-950 border border-border rounded-lg px-4 py-3">
                  <code className="flex-1 text-sm font-mono text-green-400">{sshCommand}</code>
                  <button
                    onClick={() => copyToClipboard(sshCommand, "ssh")}
                    className="text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
                    title="Befehl kopieren"
                  >
                    <Copy className="w-4 h-4" />
                  </button>
                </div>
                {copied === "ssh" && <p className="text-xs text-green-400 mt-1">Befehl kopiert!</p>}
              </div>

              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5 uppercase tracking-wide">Mit explizitem Port</label>
                <div className="flex items-center gap-2 bg-zinc-950 border border-border rounded-lg px-4 py-3">
                  <code className="flex-1 text-sm font-mono text-green-400">{sshCommandPort}</code>
                  <button
                    onClick={() => copyToClipboard(sshCommandPort, "sshp")}
                    className="text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
                    title="Befehl kopieren"
                  >
                    <Copy className="w-4 h-4" />
                  </button>
                </div>
                {copied === "sshp" && <p className="text-xs text-green-400 mt-1">Befehl kopiert!</p>}
              </div>

              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5 uppercase tracking-wide">Mit Identitätsdatei (SSH-Key)</label>
                <div className="flex items-center gap-2 bg-zinc-950 border border-border rounded-lg px-4 py-3">
                  <code className="flex-1 text-sm font-mono text-green-400">
                    {`ssh -i ~/.ssh/id_rsa root@${selectedServer.ip_address}`}
                  </code>
                  <button
                    onClick={() => copyToClipboard(`ssh -i ~/.ssh/id_rsa root@${selectedServer.ip_address}`, "sshkey")}
                    className="text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
                  >
                    <Copy className="w-4 h-4" />
                  </button>
                </div>
                {copied === "sshkey" && <p className="text-xs text-green-400 mt-1">Befehl kopiert!</p>}
              </div>
            </div>
          </div>

          {/* Web Terminal Notice */}
          <div className="bg-yellow-500/5 border border-yellow-500/20 rounded-xl p-4">
            <div className="flex items-start gap-3">
              <AlertCircle className="w-5 h-5 text-yellow-400 flex-shrink-0 mt-0.5" />
              <div>
                <p className="text-sm font-medium text-yellow-400">Web-Terminal nicht verfügbar</p>
                <p className="text-sm text-muted-foreground mt-1">
                  Ein browserbasiertes Web-Terminal ist in dieser Version noch nicht verfügbar.
                  Verwenden Sie bitte einen SSH-Client wie OpenSSH, PuTTY oder Windows Terminal,
                  um sich direkt mit dem Server zu verbinden.
                </p>
              </div>
            </div>
          </div>

          {/* SSH Client Info */}
          <div className="bg-card border border-border rounded-xl p-6">
            <h3 className="font-semibold text-foreground mb-3">Empfohlene SSH-Clients</h3>
            <div className="grid grid-cols-2 gap-3 text-sm">
              {[
                { name: "Windows Terminal / OpenSSH", platforms: "Windows 10/11", note: "Vorinstalliert" },
                { name: "PuTTY", platforms: "Windows", note: "Kostenlos herunterladen" },
                { name: "Terminal (macOS/Linux)", platforms: "macOS, Linux", note: "Vorinstalliert" },
                { name: "Termius", platforms: "Windows, macOS, iOS, Android", note: "Mobilfreundlich" },
              ].map((client) => (
                <div key={client.name} className="p-3 bg-background border border-border rounded-lg">
                  <div className="font-medium text-foreground">{client.name}</div>
                  <div className="text-xs text-muted-foreground mt-1">{client.platforms}</div>
                  <div className="text-xs text-primary mt-0.5">{client.note}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
