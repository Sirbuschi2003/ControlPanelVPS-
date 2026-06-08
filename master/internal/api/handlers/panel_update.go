package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/updater"
)

// PanelUpdateHandler handles self-update of the panel software.
type PanelUpdateHandler struct {
	svc         *services.PanelUpdateService
	settingsSvc *services.SettingsService
}

func NewPanelUpdateHandler(svc *services.PanelUpdateService, settingsSvc *services.SettingsService) *PanelUpdateHandler {
	return &PanelUpdateHandler{svc: svc, settingsSvc: settingsSvc}
}

// Info handles GET /api/panel/info
func (h *PanelUpdateHandler) Info(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.GetInfo())
}

// CheckUpdate handles GET /api/panel/check-update — calls GitHub API (slower)
func (h *PanelUpdateHandler) CheckUpdate(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.CheckUpdate(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "update check failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// UpdateStatus handles GET /api/panel/update-status — returns cached status (instant)
func (h *PanelUpdateHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, updater.GetStatus())
}

// RunUpdate handles POST /api/panel/update
func (h *PanelUpdateHandler) RunUpdate(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.RunUpdate(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// GetAutoUpdate handles GET /api/panel/auto-update
func (h *PanelUpdateHandler) GetAutoUpdate(w http.ResponseWriter, r *http.Request) {
	val, err := h.settingsSvc.Get(r.Context(), "auto_update")
	if err != nil {
		val = "false"
	}
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": val == "true"})
}

// SetAutoUpdate handles PUT /api/panel/auto-update
func (h *PanelUpdateHandler) SetAutoUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	value := "false"
	if body.Enabled {
		value = "true"
	}
	if err := h.settingsSvc.Set(r.Context(), "auto_update", value); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save setting")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": body.Enabled})
}
