package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
)

var startTime = time.Now()

// SettingsHandler handles HTTP requests for panel settings management.
type SettingsHandler struct {
	svc *services.SettingsService
}

// NewSettingsHandler creates a new SettingsHandler.
func NewSettingsHandler(svc *services.SettingsService) *SettingsHandler {
	return &SettingsHandler{svc: svc}
}

// sensitiveKeys holds setting keys whose values must never be returned in plaintext.
// A non-empty value is replaced with "***" so the UI can detect "is set" vs "empty".
var sensitiveKeys = map[string]bool{
	"smtp_pass":              true,
	"backup_encryption_key": true,
}

// Get handles GET /api/settings
// Returns all settings as a key/value map with sensitive values masked.
func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	settings, err := h.svc.GetAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load settings: "+err.Error())
		return
	}
	for key := range sensitiveKeys {
		if v, exists := settings[key]; exists && v != "" {
			settings[key] = "***"
		}
	}
	writeJSON(w, http.StatusOK, settings)
}

// Set handles PUT /api/settings — accepts a map of key/value pairs for bulk update.
func (h *SettingsHandler) Set(w http.ResponseWriter, r *http.Request) {
	var bulk map[string]string
	if err := json.NewDecoder(r.Body).Decode(&bulk); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.svc.SetBulk(r.Context(), bulk); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save settings: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "settings saved"})
}

// Info handles GET /api/settings/info — returns runtime panel info.
func (h *SettingsHandler) Info(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(startTime).Round(time.Second).String()
	writeJSON(w, http.StatusOK, map[string]string{
		"version":         "1.0.0",
		"uptime":          uptime,
		"database_status": "ok",
		"go_version":      "go1.22",
	})
}

// TestSMTP handles POST /api/settings/test-smtp — stub, SMTP sending not yet implemented.
func (h *SettingsHandler) TestSMTP(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "SMTP test not yet implemented")
}
