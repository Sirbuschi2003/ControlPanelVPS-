package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
)

// CronHandler handles HTTP requests for cron job management.
type CronHandler struct {
	svc *services.CronService
}

// NewCronHandler creates a new CronHandler.
func NewCronHandler(svc *services.CronService) *CronHandler {
	return &CronHandler{svc: svc}
}

// List handles GET /api/crons?server_id=... (server_id optional, returns all when omitted)
func (h *CronHandler) List(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	jobs, err := h.svc.List(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list cron jobs: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

type createCronRequest struct {
	ServerID  string `json:"server_id"`
	Name      string `json:"name"`
	Command   string `json:"command"`
	Schedule  string `json:"schedule"`
	RunAsUser string `json:"run_as_user"`
}

// Create handles POST /api/crons
func (h *CronHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createCronRequest
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
	if req.Command == "" {
		writeError(w, http.StatusBadRequest, "command is required")
		return
	}
	if req.Schedule == "" {
		writeError(w, http.StatusBadRequest, "schedule is required")
		return
	}
	if req.RunAsUser == "" {
		req.RunAsUser = "www-data"
	}

	job, err := h.svc.Create(r.Context(), req.ServerID, req.Name, req.Command, req.Schedule, req.RunAsUser)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create cron job: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

type updateCronRequest struct {
	Command  string `json:"command"`
	Schedule string `json:"schedule"`
	Enabled  bool   `json:"enabled"`
}

// Update handles PUT /api/crons/{id}
func (h *CronHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req updateCronRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Command == "" {
		writeError(w, http.StatusBadRequest, "command is required")
		return
	}
	if req.Schedule == "" {
		writeError(w, http.StatusBadRequest, "schedule is required")
		return
	}

	if err := h.svc.Update(r.Context(), id, req.Command, req.Schedule, req.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update cron job: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "cron job updated"})
}

// Delete handles DELETE /api/crons/{id}
func (h *CronHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete cron job: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "cron job deleted"})
}
