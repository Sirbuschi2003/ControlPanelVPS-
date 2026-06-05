package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
)

// SystemServiceHandler handles HTTP requests for system service management.
type SystemServiceHandler struct {
	svc *services.SystemServiceManager
}

// NewSystemServiceHandler creates a new SystemServiceHandler.
func NewSystemServiceHandler(svc *services.SystemServiceManager) *SystemServiceHandler {
	return &SystemServiceHandler{svc: svc}
}

// List handles GET /api/services?server_id=...
func (h *SystemServiceHandler) List(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}

	svcList, err := h.svc.List(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to list services: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, svcList)
}

type serviceActionRequest struct {
	ServerID string `json:"server_id"`
	Action   string `json:"action"`
}

// Action handles POST /api/services/{name}/action
func (h *SystemServiceHandler) Action(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "service name is required")
		return
	}

	var req serviceActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ServerID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}
	if req.Action == "" {
		writeError(w, http.StatusBadRequest, "action is required")
		return
	}

	if err := h.svc.Action(r.Context(), req.ServerID, name, req.Action); err != nil {
		writeError(w, http.StatusInternalServerError, "service action failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"service": name,
		"action":  req.Action,
		"status":  "ok",
	})
}
