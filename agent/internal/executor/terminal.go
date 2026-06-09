package executor

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var termUpgrader = websocket.Upgrader{
	// Auth is enforced by the caller middleware; no origin check needed here.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// resizeMsg is sent by the client to resize the terminal window.
type resizeMsg struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// TerminalWebSocket starts an authenticated bash PTY session over WebSocket.
// The client sends raw keystrokes as binary frames and JSON resize frames.
// The server streams PTY output as binary frames.
func TerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := termUpgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("terminal ws upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	shell := "/bin/bash"
	if _, err := os.Stat(shell); err != nil {
		shell = "/bin/sh"
	}

	cmd := exec.Command(shell, "-i")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("\r\nFehler: PTY konnte nicht gestartet werden: "+err.Error()+"\r\n"))
		return
	}
	defer func() {
		_ = ptmx.Close()
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	// PTY → WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				if err2 := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err2 != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// WebSocket → PTY
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			return
		}

		if msgType == websocket.TextMessage {
			var resize resizeMsg
			if json.Unmarshal(data, &resize) == nil && resize.Type == "resize" && resize.Cols > 0 && resize.Rows > 0 {
				_ = pty.Setsize(ptmx, &pty.Winsize{Cols: resize.Cols, Rows: resize.Rows})
				continue
			}
		}
		if _, err := ptmx.Write(data); err != nil {
			return
		}
	}
}
