package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
)

type PHPHandler struct{ svc *services.PHPService }

func NewPHPHandler(svc *services.PHPService) *PHPHandler { return &PHPHandler{svc: svc} }

func (h *PHPHandler) Get(w http.ResponseWriter, r *http.Request) {
	domainID := r.URL.Query().Get("domain_id")
	if domainID == "" {
		writeError(w, http.StatusBadRequest, "domain_id is required")
		return
	}
	p, err := h.svc.Get(r.Context(), domainID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *PHPHandler) Save(w http.ResponseWriter, r *http.Request) {
	var p models.PHPSettings
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if p.DomainID == "" {
		writeError(w, http.StatusBadRequest, "domain_id is required")
		return
	}
	result, err := h.svc.Save(r.Context(), p)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
