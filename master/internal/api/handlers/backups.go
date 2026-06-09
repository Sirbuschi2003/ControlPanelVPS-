package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
)

// BackupHandler handles HTTP requests for backup configuration and job management.
type BackupHandler struct {
	svc *services.BackupService
}

// NewBackupHandler creates a new BackupHandler.
func NewBackupHandler(svc *services.BackupService) *BackupHandler {
	return &BackupHandler{svc: svc}
}

// ListConfigs handles GET /api/backups/configs?server_id=... (server_id optional, returns all when omitted)
func (h *BackupHandler) ListConfigs(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	configs, err := h.svc.ListConfigs(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list backup configs: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, configs)
}

type createBackupConfigRequest struct {
	ServerID      string            `json:"server_id"`
	Name          string            `json:"name"`
	StorageType   string            `json:"storage_type"`
	Schedule      string            `json:"schedule"`
	RetentionDays int               `json:"retention_days"`
	IncludePaths  []string          `json:"include_paths"`
	StorageConfig map[string]string `json:"storage_config"`
	Encrypt       bool              `json:"encrypt"`
}

// CreateConfig handles POST /api/backups/configs
func (h *BackupHandler) CreateConfig(w http.ResponseWriter, r *http.Request) {
	var req createBackupConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ServerID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.StorageType == "" {
		req.StorageType = "local"
	}
	if req.Schedule == "" {
		req.Schedule = "0 2 * * *"
	}
	if req.RetentionDays == 0 {
		req.RetentionDays = 7
	}

	config, err := h.svc.CreateConfig(r.Context(), req.ServerID, req.Name, req.StorageType, req.Schedule,
		req.RetentionDays, req.IncludePaths, req.StorageConfig, req.Encrypt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create backup config: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, config)
}

type toggleBackupConfigRequest struct {
	Enabled bool `json:"enabled"`
}

// ToggleConfig handles PUT /api/backups/configs/{id}
func (h *BackupHandler) ToggleConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	var req toggleBackupConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.svc.ToggleConfig(r.Context(), id, req.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to toggle backup config: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "backup config updated"})
}

// DeleteConfig handles DELETE /api/backups/configs/{id}
func (h *BackupHandler) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.svc.DeleteConfig(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete backup config: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "backup config deleted"})
}

// RunBackup handles POST /api/backups/configs/{id}/run
func (h *BackupHandler) RunBackup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	job, err := h.svc.RunBackup(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to run backup: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

// ListJobs handles GET /api/backups/jobs?config_id=...
func (h *BackupHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	configID := r.URL.Query().Get("config_id")
	if configID == "" {
		writeError(w, http.StatusBadRequest, "config_id is required")
		return
	}

	jobs, err := h.svc.ListJobs(r.Context(), configID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list backup jobs: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}
