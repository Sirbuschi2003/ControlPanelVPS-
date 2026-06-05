package handlers

import (
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
)

type SystemHandler struct {
	systemSvc *services.SystemUpdateService
}

func NewSystemHandler(svc *services.SystemUpdateService) *SystemHandler {
	return &SystemHandler{systemSvc: svc}
}

func (h *SystemHandler) Info(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	info, err := h.systemSvc.GetInfo(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (h *SystemHandler) CheckUpdates(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	result, err := h.systemSvc.CheckUpdates(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *SystemHandler) RunUpdate(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	result, err := h.systemSvc.RunUpdate(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
