package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
)

type MonitoringHandler struct {
	svc *services.MonitoringService
}

func NewMonitoringHandler(svc *services.MonitoringService) *MonitoringHandler {
	return &MonitoringHandler{svc: svc}
}

func (h *MonitoringHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}
	report, err := h.svc.HealthCheck(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	status := http.StatusOK
	if !report.Healthy {
		status = http.StatusOK // always 200 but healthy flag shows state
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(report)
}

func (h *MonitoringHandler) SetupMailTLS(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ServerID string `json:"server_id"`
		Hostname string `json:"hostname"`
		CertPath string `json:"cert_path"`
		KeyPath  string `json:"key_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ServerID == "" || req.Hostname == "" {
		writeError(w, http.StatusBadRequest, "server_id and hostname are required")
		return
	}
	// Default cert paths for Let's Encrypt
	if req.CertPath == "" {
		req.CertPath = "/etc/letsencrypt/live/" + req.Hostname + "/fullchain.pem"
	}
	if req.KeyPath == "" {
		req.KeyPath = "/etc/letsencrypt/live/" + req.Hostname + "/privkey.pem"
	}
	if err := h.svc.SetupMailTLS(r.Context(), req.ServerID, req.Hostname, req.CertPath, req.KeyPath); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "configured", "hostname": req.Hostname})
}

func (h *MonitoringHandler) SetupRspamd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ServerID string `json:"server_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ServerID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}
	if err := h.svc.SetupRspamd(r.Context(), req.ServerID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "configured"})
}

func (h *MonitoringHandler) SetupDKIM(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	domain := chi.URLParam(r, "domain")
	if serverID == "" || domain == "" {
		writeError(w, http.StatusBadRequest, "server_id and domain are required")
		return
	}
	result, err := h.svc.SetupDKIM(r.Context(), serverID, domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *MonitoringHandler) RspamdStatus(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}
	stats, err := h.svc.GetRspamdStatus(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}
