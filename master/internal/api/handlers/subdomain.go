package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
)

type SubdomainHandler struct{ svc *services.SubdomainService }

func NewSubdomainHandler(svc *services.SubdomainService) *SubdomainHandler {
	return &SubdomainHandler{svc: svc}
}

func (h *SubdomainHandler) List(w http.ResponseWriter, r *http.Request) {
	domainID := r.URL.Query().Get("domain_id")
	if domainID == "" {
		writeError(w, http.StatusBadRequest, "domain_id is required")
		return
	}
	out, err := h.svc.List(r.Context(), domainID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *SubdomainHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DomainID     string `json:"domain_id"`
		Name         string `json:"name"`
		DocumentRoot string `json:"document_root"`
		PHPVersion   string `json:"php_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.DomainID == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "domain_id and name are required")
		return
	}
	sub, err := h.svc.Create(r.Context(), req.DomainID, req.Name, req.DocumentRoot, req.PHPVersion)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, sub)
}

func (h *SubdomainHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}
