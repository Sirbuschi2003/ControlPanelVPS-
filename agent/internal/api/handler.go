package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Sirbuschi2003/ControlPanelVPS/agent/internal/collector"
)

type Handler struct {
	token  string
	nodeID string
	mux    *http.ServeMux
}

func NewHandler(token, nodeID string) *Handler {
	h := &Handler{token: token, nodeID: nodeID, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /health", h.health)
	h.mux.HandleFunc("GET /metrics", h.authMiddleware(h.metrics))
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "node_id": h.nodeID})
}

func (h *Handler) metrics(w http.ResponseWriter, r *http.Request) {
	m, err := collector.Collect()
	if err != nil {
		http.Error(w, `{"error":"collection failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(m)
}

func (h *Handler) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") || strings.TrimPrefix(header, "Bearer ") != h.token {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
