package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
)

// FirewallHandler handles HTTP requests for firewall rule management.
type FirewallHandler struct {
	svc *services.FirewallService
}

// NewFirewallHandler creates a new FirewallHandler.
func NewFirewallHandler(svc *services.FirewallService) *FirewallHandler {
	return &FirewallHandler{svc: svc}
}

// List handles GET /api/firewall?server_id=... (server_id optional, returns all when omitted)
func (h *FirewallHandler) List(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	rules, err := h.svc.List(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list firewall rules: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

type createFirewallRuleRequest struct {
	ServerID  string  `json:"server_id"`
	Action    string  `json:"action"`
	Direction string  `json:"direction"`
	Protocol  string  `json:"protocol"`
	Source    string  `json:"source"`
	DestPort  *string `json:"dest_port"`
	Comment   *string `json:"comment"`
	Order     int     `json:"order"`
}

// Create handles POST /api/firewall
func (h *FirewallHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createFirewallRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ServerID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}
	if req.Action == "" {
		req.Action = "allow"
	}
	if req.Direction == "" {
		req.Direction = "in"
	}
	if req.Protocol == "" {
		req.Protocol = "tcp"
	}
	if req.Source == "" {
		req.Source = "any"
	}
	if req.Order == 0 {
		req.Order = 100
	}

	rule, err := h.svc.Create(r.Context(), req.ServerID, req.Action, req.Direction, req.Protocol, req.Source, req.DestPort, req.Comment, req.Order)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create firewall rule: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

// Delete handles DELETE /api/firewall/{id}
func (h *FirewallHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete firewall rule: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "firewall rule deleted"})
}

type toggleFirewallRequest struct {
	Enabled bool `json:"enabled"`
}

// Toggle handles POST /api/firewall/{id}/toggle
func (h *FirewallHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req toggleFirewallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.Toggle(r.Context(), id, req.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to toggle firewall rule: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": req.Enabled})
}

// Reload handles POST /api/firewall/reload
func (h *FirewallHandler) Reload(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}

	if err := h.svc.Reload(r.Context(), serverID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reload firewall: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "firewall reloaded"})
}
