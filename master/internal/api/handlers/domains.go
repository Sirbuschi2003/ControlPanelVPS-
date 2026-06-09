package handlers

import (
	"encoding/json"
	"net/http"

	authmw "github.com/Sirbuschi2003/ControlPanelVPS/master/internal/api/middleware"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
)

// DomainHandler exposes domain management endpoints.
type DomainHandler struct {
	svc *services.DomainService
}

// NewDomainHandler creates a DomainHandler.
func NewDomainHandler(svc *services.DomainService) *DomainHandler {
	return &DomainHandler{svc: svc}
}

// List returns all domains visible to the authenticated user.
func (h *DomainHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := authmw.GetClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	domains, err := h.svc.List(r.Context(), claims.UserID, claims.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list domains: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, domains)
}

// Create provisions a new domain (admin only).
func (h *DomainHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ServerID      string `json:"server_id"`
		Name          string `json:"name"`
		OwnerUserID   string `json:"owner_user_id"`
		PHPVersion    string `json:"php_version"`
		ProvisionWeb  bool   `json:"provision_web"`
		ProvisionDNS  bool   `json:"provision_dns"`
		ProvisionMail bool   `json:"provision_mail"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ServerID == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "server_id and name are required")
		return
	}
	if req.PHPVersion == "" {
		req.PHPVersion = "8.2"
	}

	domain, err := h.svc.Create(r.Context(), req.ServerID, req.Name, req.OwnerUserID, req.PHPVersion, req.ProvisionWeb, req.ProvisionDNS, req.ProvisionMail)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create domain: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, domain)
}

// Get returns a single domain.
func (h *DomainHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims := authmw.GetClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.checkAccess(r, id, claims); err != nil {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	domain, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "domain not found")
		return
	}
	writeJSON(w, http.StatusOK, domain)
}

// Delete removes a domain and all its resources (admin only).
func (h *DomainHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete domain: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetResources returns all sub-resources for a domain.
func (h *DomainHandler) GetResources(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims := authmw.GetClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.checkAccess(r, id, claims); err != nil {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	res, err := h.svc.GetResources(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get resources: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ListUsers returns all users assigned to a domain (admin only).
func (h *DomainHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	users, err := h.svc.ListUsers(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list domain users: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// AssignUser grants a user access to a domain (admin only).
func (h *DomainHandler) AssignUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if err := h.svc.AssignUser(r.Context(), id, req.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to assign user: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RemoveUser revokes a user's access to a domain (admin only).
func (h *DomainHandler) RemoveUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "user_id")
	if err := h.svc.RemoveUser(r.Context(), id, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove user: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// checkAccess returns an error if the user is non-admin and not assigned to the domain.
func (h *DomainHandler) checkAccess(r *http.Request, domainID string, claims *services.Claims) error {
	if claims.Role == "admin" {
		return nil
	}
	ids, err := h.svc.AccessibleDomainIDs(r.Context(), claims.UserID, claims.Role)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if id == domainID {
			return nil
		}
	}
	return &accessDenied{}
}

type accessDenied struct{}

func (e *accessDenied) Error() string { return "access denied" }
