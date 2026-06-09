package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
)

// SSLHandler handles HTTP requests for SSL certificate management.
type SSLHandler struct {
	svc *services.SSLService
}

// NewSSLHandler creates a new SSLHandler.
func NewSSLHandler(svc *services.SSLService) *SSLHandler {
	return &SSLHandler{svc: svc}
}

// List handles GET /api/ssl?server_id=... (server_id optional, returns all when omitted)
func (h *SSLHandler) List(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	certs, err := h.svc.List(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list SSL certs: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, certs)
}

type issueSSLRequest struct {
	ServerID   string   `json:"server_id"`
	Domain     string   `json:"domain"`
	SANDomains []string `json:"san_domains"`
	Email      string   `json:"email"`
}

// Issue handles POST /api/ssl
func (h *SSLHandler) Issue(w http.ResponseWriter, r *http.Request) {
	var req issueSSLRequest
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
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	cert, err := h.svc.Issue(r.Context(), req.ServerID, req.Domain, req.SANDomains, req.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue SSL cert: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cert)
}

// Renew handles POST /api/ssl/{id}/renew
func (h *SSLHandler) Renew(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.svc.Renew(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to renew SSL cert: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "renewal initiated"})
}

// Delete handles DELETE /api/ssl/{id}
func (h *SSLHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete SSL cert: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "SSL cert deleted"})
}
