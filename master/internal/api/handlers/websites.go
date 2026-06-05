package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
)

// WebsiteHandler handles HTTP requests for website management.
type WebsiteHandler struct {
	svc *services.WebsiteService
}

// NewWebsiteHandler creates a new WebsiteHandler.
func NewWebsiteHandler(svc *services.WebsiteService) *WebsiteHandler {
	return &WebsiteHandler{svc: svc}
}

// List handles GET /api/websites?server_id=...
func (h *WebsiteHandler) List(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	websites, err := h.svc.List(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list websites: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, websites)
}

type createWebsiteRequest struct {
	ServerID     string   `json:"server_id"`
	Domain       string   `json:"domain"`
	Aliases      []string `json:"aliases"`
	PHPVersion   string   `json:"php_version"`
	DocumentRoot string   `json:"document_root"`
}

// Create handles POST /api/websites
func (h *WebsiteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createWebsiteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ServerID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}
	if req.Domain == "" {
		writeError(w, http.StatusBadRequest, "domain is required")
		return
	}
	if req.PHPVersion == "" {
		req.PHPVersion = "8.2"
	}
	if req.DocumentRoot == "" {
		req.DocumentRoot = "/var/www/" + req.Domain
	}

	website, err := h.svc.Create(r.Context(), req.ServerID, req.Domain, req.PHPVersion, req.DocumentRoot, req.Aliases)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create website: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, website)
}

type updateWebsiteRequest struct {
	PHPVersion    string `json:"php_version"`
	DocumentRoot  string `json:"document_root"`
	SSLEnabled    *bool  `json:"ssl_enabled"`
	SSLForceHTTPS *bool  `json:"ssl_force_https"`
}

// Update handles PUT /api/websites/{id}
func (h *WebsiteHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req updateWebsiteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updates := map[string]any{}
	if req.PHPVersion != "" {
		updates["php_version"] = req.PHPVersion
	}
	if req.DocumentRoot != "" {
		updates["document_root"] = req.DocumentRoot
	}
	if req.SSLEnabled != nil {
		updates["ssl_enabled"] = *req.SSLEnabled
	}
	if req.SSLForceHTTPS != nil {
		updates["ssl_force_https"] = *req.SSLForceHTTPS
	}

	website, err := h.svc.Update(r.Context(), id, updates)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update website: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, website)
}

// Delete handles DELETE /api/websites/{id}
func (h *WebsiteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete website: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "website deleted"})
}

type toggleWebsiteRequest struct {
	Enabled bool `json:"enabled"`
}

// Toggle handles POST /api/websites/{id}/toggle
func (h *WebsiteHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req toggleWebsiteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.Toggle(r.Context(), id, req.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to toggle website: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": req.Enabled})
}

type enableSSLRequest struct {
	CertID string `json:"cert_id"`
}

// EnableSSL handles POST /api/websites/{id}/ssl
func (h *WebsiteHandler) EnableSSL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req enableSSLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.CertID == "" {
		writeError(w, http.StatusBadRequest, "cert_id is required")
		return
	}

	if err := h.svc.EnableSSL(r.Context(), id, req.CertID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enable SSL: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "SSL enabled"})
}
