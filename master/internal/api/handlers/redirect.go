package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
)

type RedirectHandler struct{ svc *services.RedirectService }

func NewRedirectHandler(svc *services.RedirectService) *RedirectHandler {
	return &RedirectHandler{svc: svc}
}

func (h *RedirectHandler) List(w http.ResponseWriter, r *http.Request) {
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

func (h *RedirectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DomainID     string `json:"domain_id"`
		SourcePath   string `json:"source_path"`
		TargetURL    string `json:"target_url"`
		RedirectType int    `json:"redirect_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.DomainID == "" || req.TargetURL == "" {
		writeError(w, http.StatusBadRequest, "domain_id and target_url are required")
		return
	}
	if req.SourcePath == "" {
		req.SourcePath = "/"
	}
	if req.RedirectType == 0 {
		req.RedirectType = 301
	}
	red, err := h.svc.Create(r.Context(), req.DomainID, req.SourcePath, req.TargetURL, req.RedirectType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, red)
}

func (h *RedirectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}
