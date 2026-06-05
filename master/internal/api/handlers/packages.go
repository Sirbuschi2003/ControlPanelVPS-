package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
)

// PackageHandler handles HTTP requests for package update management.
type PackageHandler struct {
	svc *services.PackageService
}

// NewPackageHandler creates a new PackageHandler.
func NewPackageHandler(svc *services.PackageService) *PackageHandler {
	return &PackageHandler{svc: svc}
}

// ListUpdates handles GET /api/packages/updates?server_id=...
func (h *PackageHandler) ListUpdates(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}

	updates, err := h.svc.ListUpdates(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to list package updates: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updates)
}

type applyUpdatesRequest struct {
	ServerID string   `json:"server_id"`
	Packages []string `json:"packages"`
}

// ApplyUpdates handles POST /api/packages/update
func (h *PackageHandler) ApplyUpdates(w http.ResponseWriter, r *http.Request) {
	var req applyUpdatesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ServerID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}

	if err := h.svc.ApplyUpdates(r.Context(), req.ServerID, req.Packages); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to apply package updates: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "package update initiated"})
}
