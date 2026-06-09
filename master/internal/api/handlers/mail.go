package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
)

// MailHandler handles HTTP requests for mail domain, account and alias management.
type MailHandler struct {
	svc *services.MailService
}

// NewMailHandler creates a new MailHandler.
func NewMailHandler(svc *services.MailService) *MailHandler {
	return &MailHandler{svc: svc}
}

// ---- Domains ----

// ListDomains handles GET /api/mail/domains?server_id=... (server_id optional, returns all when omitted)
func (h *MailHandler) ListDomains(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	domains, err := h.svc.ListDomains(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list mail domains: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, domains)
}

type createMailDomainRequest struct {
	ServerID string `json:"server_id"`
	Domain   string `json:"domain"`
}

// CreateDomain handles POST /api/mail/domains
func (h *MailHandler) CreateDomain(w http.ResponseWriter, r *http.Request) {
	var req createMailDomainRequest
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

	domain, err := h.svc.CreateDomain(r.Context(), req.ServerID, req.Domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create mail domain: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, domain)
}

// DeleteDomain handles DELETE /api/mail/domains/{id}
func (h *MailHandler) DeleteDomain(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.svc.DeleteDomain(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete mail domain: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "mail domain deleted"})
}

// ---- Accounts ----

// ListAccounts handles GET /api/mail/accounts?domain_id=...
func (h *MailHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	domainID := r.URL.Query().Get("domain_id")
	if domainID == "" {
		writeError(w, http.StatusBadRequest, "domain_id is required")
		return
	}

	accounts, err := h.svc.ListAccounts(r.Context(), domainID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list mail accounts: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, accounts)
}

type createMailAccountRequest struct {
	DomainID string `json:"domain_id"`
	Username string `json:"username"`
	Password string `json:"password"`
	QuotaMB  int    `json:"quota_mb"`
}

// CreateAccount handles POST /api/mail/accounts
func (h *MailHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req createMailAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DomainID == "" {
		writeError(w, http.StatusBadRequest, "domain_id is required")
		return
	}
	if req.Username == "" {
		writeError(w, http.StatusBadRequest, "username is required")
		return
	}
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}
	// quota_mb=0 means unlimited — no default override

	account, err := h.svc.CreateAccount(r.Context(), req.DomainID, req.Username, req.Password, req.QuotaMB)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create mail account: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, account)
}

// DeleteAccount handles DELETE /api/mail/accounts/{id}
func (h *MailHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.svc.DeleteAccount(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete mail account: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "mail account deleted"})
}

// ---- Aliases ----

// ListAliases handles GET /api/mail/aliases?domain_id=...
func (h *MailHandler) ListAliases(w http.ResponseWriter, r *http.Request) {
	domainID := r.URL.Query().Get("domain_id")
	if domainID == "" {
		writeError(w, http.StatusBadRequest, "domain_id is required")
		return
	}

	aliases, err := h.svc.ListAliases(r.Context(), domainID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list mail aliases: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, aliases)
}

type createMailAliasRequest struct {
	DomainID    string `json:"domain_id"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

// CreateAlias handles POST /api/mail/aliases
func (h *MailHandler) CreateAlias(w http.ResponseWriter, r *http.Request) {
	var req createMailAliasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DomainID == "" {
		writeError(w, http.StatusBadRequest, "domain_id is required")
		return
	}
	if req.Source == "" {
		writeError(w, http.StatusBadRequest, "source is required")
		return
	}
	if req.Destination == "" {
		writeError(w, http.StatusBadRequest, "destination is required")
		return
	}

	alias, err := h.svc.CreateAlias(r.Context(), req.DomainID, req.Source, req.Destination)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create mail alias: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, alias)
}

// DeleteAlias handles DELETE /api/mail/aliases/{id}
func (h *MailHandler) DeleteAlias(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.svc.DeleteAlias(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete mail alias: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "mail alias deleted"})
}
