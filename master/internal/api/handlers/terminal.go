package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

var termUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// TerminalHandler proxies WebSocket terminal sessions to the agent.
type TerminalHandler struct {
	db *pgxpool.Pool
}

// NewTerminalHandler creates a new TerminalHandler.
func NewTerminalHandler(db *pgxpool.Pool) *TerminalHandler {
	return &TerminalHandler{db: db}
}

// WebSocket handles GET /api/terminal/ws?server_id=...
// It authenticates the browser connection (JWT already validated by middleware),
// looks up the target agent, and proxies the WebSocket connection.
func (h *TerminalHandler) WebSocket(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	if serverID == "" {
		http.Error(w, "server_id required", http.StatusBadRequest)
		return
	}

	var agentURL, agentToken string
	err := h.db.QueryRow(r.Context(),
		`SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID,
	).Scan(&agentURL, &agentToken)
	if err != nil {
		http.Error(w, "server not found", http.StatusNotFound)
		return
	}

	// Build agent WebSocket URL: replace http(s) scheme with ws(s)
	agentWS, err := buildWSURL(agentURL, "/terminal")
	if err != nil {
		http.Error(w, "invalid agent URL", http.StatusInternalServerError)
		return
	}

	// Connect to agent
	agentHeaders := http.Header{"Authorization": {"Bearer " + agentToken}}
	agentConn, _, err := websocket.DefaultDialer.DialContext(r.Context(), agentWS, agentHeaders)
	if err != nil {
		slog.Warn("terminal: could not connect to agent", "error", err, "url", agentWS)
		http.Error(w, fmt.Sprintf("agent unreachable: %v", err), http.StatusBadGateway)
		return
	}
	defer agentConn.Close()

	// Upgrade browser connection
	browserConn, err := termUpgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("terminal: browser upgrade failed", "error", err)
		return
	}
	defer browserConn.Close()

	done := make(chan struct{}, 2)

	// Agent → Browser
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			mt, msg, err := agentConn.ReadMessage()
			if err != nil {
				return
			}
			if err := browserConn.WriteMessage(mt, msg); err != nil {
				return
			}
		}
	}()

	// Browser → Agent
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			mt, msg, err := browserConn.ReadMessage()
			if err != nil {
				return
			}
			if err := agentConn.WriteMessage(mt, msg); err != nil {
				return
			}
		}
	}()

	<-done
}

func buildWSURL(agentURL, path string) (string, error) {
	u, err := url.Parse(agentURL)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(u.Scheme) {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	u.Path = path
	return u.String(), nil
}
