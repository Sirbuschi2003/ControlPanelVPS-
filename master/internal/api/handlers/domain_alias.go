package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
)

type DomainAliasHandler struct{ svc *services.DomainAliasService }

func NewDomainAliasHandler(svc *services.DomainAliasService) *DomainAliasHandler {
	return &DomainAliasHandler{svc: svc}
}

func (h *DomainAliasHandler) List(w http.ResponseWriter, r *http.Request) {
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

func (h *DomainAliasHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DomainID string `json:"domain_id"`
		Alias    string `json:"alias"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.DomainID == "" || req.Alias == "" {
		writeError(w, http.StatusBadRequest, "domain_id and alias are required")
		return
	}
	a, err := h.svc.Create(r.Context(), req.DomainID, req.Alias)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (h *DomainAliasHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}
