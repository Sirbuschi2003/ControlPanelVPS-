"use client";

import { useEffect, useRef } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "@xterm/xterm/css/xterm.css";

interface Props {
  ws: WebSocket;
  onDisconnect: () => void;
}

export default function XTerm({ ws, onDisconnect }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitRef = useRef<FitAddon | null>(null);

  useEffect(() => {
    if (!containerRef.current || !ws) return;

    const term = new Terminal({
      theme: {
        background: "#09090b",
        foreground: "#e4e4e7",
        cursor: "#a1a1aa",
        selectionBackground: "#3f3f46",
      },
      fontFamily: '"JetBrains Mono", "Fira Code", "Cascadia Code", monospace',
      fontSize: 13,
      lineHeight: 1.2,
      cursorBlink: true,
      convertEol: true,
      scrollback: 5000,
    });

    const fit = new FitAddon();
    term.loadAddon(fit);
    term.loadAddon(new WebLinksAddon());

    term.open(containerRef.current);
    fit.fit();
    term.focus();

    termRef.current = term;
    fitRef.current = fit;

    // Send initial resize
    sendResize(ws, term.cols, term.rows);

    // Resize observer
    const observer = new ResizeObserver(() => {
      fit.fit();
      sendResize(ws, term.cols, term.rows);
    });
    observer.observe(containerRef.current);

    // Terminal input → WebSocket
    const disposeInput = term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(new TextEncoder().encode(data));
      }
    });

    // WebSocket output → Terminal
    ws.onmessage = (e) => {
      if (e.data instanceof ArrayBuffer) {
        term.write(new Uint8Array(e.data));
      } else if (typeof e.data === "string") {
        term.write(e.data);
      }
    };
    ws.onclose = () => {
      term.write("\r\n\x1b[31m[Verbindung getrennt]\x1b[0m\r\n");
      onDisconnect();
    };

    return () => {
      disposeInput.dispose();
      observer.disconnect();
      term.dispose();
      termRef.current = null;
      fitRef.current = null;
    };
  }, [ws, onDisconnect]);

  return <div ref={containerRef} className="w-full h-full p-1" />;
}

function sendResize(ws: WebSocket, cols: number, rows: number) {
  if (ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: "resize", cols, rows }));
  }
}
