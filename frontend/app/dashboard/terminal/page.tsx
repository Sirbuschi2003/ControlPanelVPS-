"use client";

import { useEffect, useRef, useState } from "react";
import dynamic from "next/dynamic";
import { Terminal as TerminalIcon, AlertCircle, X, Server as ServerIcon } from "lucide-react";
import { api, type Server } from "@/lib/api";

// xterm must be loaded client-side only (no SSR) — it uses browser APIs
const XTermComponent = dynamic(() => import("@/components/XTerm"), { ssr: false });

export default function TerminalPage() {
  const [servers, setServers] = useState<Server[]>([]);
  const [selectedServer, setSelectedServer] = useState<Server | null>(null);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState("");
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    api.get<Server[]>("/servers")
      .then((sv) => {
        setServers(sv);
        if (sv.length > 0) setSelectedServer(sv[0]);
      })
      .catch(() => setError("Server konnten nicht geladen werden"));
  }, []);

  function connect(server: Server) {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setConnected(false);
    setError("");

    const token = localStorage.getItem("token") || "";
    const wsBase = window.location.origin
      .replace("https://", "wss://")
      .replace("http://", "ws://");
    const wsUrl = `${wsBase}/api/terminal/ws?server_id=${server.id}&token=${encodeURIComponent(token)}`;

    const ws = new WebSocket(wsUrl);
    ws.binaryType = "arraybuffer";
    wsRef.current = ws;

    ws.onopen = () => setConnected(true);
    ws.onclose = () => { setConnected(false); wsRef.current = null; };
    ws.onerror = () => {
      setError("Verbindung zum Terminal fehlgeschlagen");
      setConnected(false);
    };
  }

  function disconnect() {
    wsRef.current?.close();
    wsRef.current = null;
    setConnected(false);
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Terminal</h1>
          <p className="text-muted-foreground text-sm mt-1">Browser-Terminal via SSH-Agent</p>
        </div>
        <div className="flex items-center gap-3">
          {servers.length > 0 && (
            <select
              value={selectedServer?.id || ""}
              onChange={(e) => {
                const sv = servers.find((s) => s.id === e.target.value) || null;
                setSelectedServer(sv);
                if (connected) disconnect();
              }}
              className="bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground"
            >
              {servers.map((s) => (
                <option key={s.id} value={s.id}>{s.name} ({s.ip_address})</option>
              ))}
            </select>
          )}
          {selectedServer && (
            connected ? (
              <button
                onClick={disconnect}
                className="flex items-center gap-2 px-3 py-2 text-sm bg-destructive/10 text-destructive border border-destructive/30 rounded-lg hover:bg-destructive/20 transition-colors"
              >
                <X className="w-4 h-4" /> Trennen
              </button>
            ) : (
              <button
                onClick={() => connect(selectedServer)}
                className="flex items-center gap-2 px-3 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
              >
                <TerminalIcon className="w-4 h-4" /> Verbinden
              </button>
            )
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

      {!selectedServer ? (
        <div className="flex flex-col items-center justify-center flex-1 text-muted-foreground">
          <ServerIcon className="w-12 h-12 mb-4 opacity-30" />
          <p className="font-medium">Kein Server verfügbar</p>
          <p className="text-sm mt-1">Fügen Sie zuerst einen Server hinzu</p>
        </div>
      ) : !connected ? (
        <div className="flex flex-col items-center justify-center flex-1 text-muted-foreground bg-zinc-950 rounded-xl border border-border">
          <TerminalIcon className="w-12 h-12 mb-4 opacity-30" />
          <p className="font-medium">Terminal nicht verbunden</p>
          <p className="text-sm mt-2 text-muted-foreground">
            Server: <span className="text-foreground font-mono">{selectedServer.ip_address}</span>
          </p>
          <button
            onClick={() => connect(selectedServer)}
            className="mt-4 flex items-center gap-2 px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
          >
            <TerminalIcon className="w-4 h-4" /> Verbindung herstellen
          </button>
        </div>
      ) : (
        <div className="flex-1 rounded-xl overflow-hidden border border-border bg-zinc-950 min-h-0">
          <XTermComponent ws={wsRef.current!} onDisconnect={disconnect} />
        </div>
      )}
    </div>
  );
}
