package handlers

import (
	"net/http"
	"strconv"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
)

// LogHandler handles HTTP requests for log retrieval.
type LogHandler struct {
	svc *services.LogService
}

// NewLogHandler creates a new LogHandler.
func NewLogHandler(svc *services.LogService) *LogHandler {
	return &LogHandler{svc: svc}
}

// List handles GET /api/logs?server_id=...
// Returns the names of available logs on the server.
func (h *LogHandler) List(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}

	logs, err := h.svc.ListAvailable(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to list logs: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

// GetLog handles GET /api/logs/{serverID}/{logName}?lines=200
func (h *LogHandler) GetLog(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "serverID")
	logName := chi.URLParam(r, "logName")

	if serverID == "" {
		writeError(w, http.StatusBadRequest, "serverID is required")
		return
	}
	if logName == "" {
		writeError(w, http.StatusBadRequest, "logName is required")
		return
	}

	lines := 200
	if linesStr := r.URL.Query().Get("lines"); linesStr != "" {
		n, err := strconv.Atoi(linesStr)
		if err == nil && n > 0 {
			lines = n
		}
	}

	logLines, err := h.svc.GetLog(r.Context(), serverID, logName, lines)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to get log: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, logLines)
}
