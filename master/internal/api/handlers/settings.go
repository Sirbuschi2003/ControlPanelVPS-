package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
)

// SettingsHandler handles HTTP requests for panel settings management.
type SettingsHandler struct {
	svc *services.SettingsService
}

// NewSettingsHandler creates a new SettingsHandler.
func NewSettingsHandler(svc *services.SettingsService) *SettingsHandler {
	return &SettingsHandler{svc: svc}
}

// Get handles GET /api/settings
// Returns all settings as a key/value map.
func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	settings, err := h.svc.GetAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load settings: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

type setSettingRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Set handles PUT /api/settings
// Creates or updates a single setting.
func (h *SettingsHandler) Set(w http.ResponseWriter, r *http.Request) {
	var req setSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}

	if err := h.svc.Set(r.Context(), req.Key, req.Value); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save setting: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"key":   req.Key,
		"value": req.Value,
	})
}
