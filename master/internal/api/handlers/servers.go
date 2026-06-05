package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
)

type ServerHandler struct {
	serverSvc *services.ServerService
}

func NewServerHandler(serverSvc *services.ServerService) *ServerHandler {
	return &ServerHandler{serverSvc: serverSvc}
}

func (h *ServerHandler) List(w http.ResponseWriter, r *http.Request) {
	servers, err := h.serverSvc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list servers")
		return
	}
	writeJSON(w, http.StatusOK, servers)
}

type createServerRequest struct {
	Name       string `json:"name"`
	Hostname   string `json:"hostname"`
	IPAddress  string `json:"ip_address"`
	AgentURL   string `json:"agent_url"`
	AgentToken string `json:"agent_token"`
	Role       string `json:"role"`
}

func (h *ServerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.IPAddress == "" || req.AgentURL == "" {
		writeError(w, http.StatusBadRequest, "name, ip_address and agent_url are required")
		return
	}

	if req.Role == "" {
		req.Role = "general"
	}

	srv, err := h.serverSvc.Create(r.Context(), req.Name, req.Hostname, req.IPAddress, req.AgentURL, req.AgentToken, req.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create server")
		return
	}
	writeJSON(w, http.StatusCreated, srv)
}

func (h *ServerHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	metrics, err := h.serverSvc.GetMetrics(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch metrics: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}
