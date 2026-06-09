package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
)

type FTPHandler struct{ svc *services.FTPService }

func NewFTPHandler(svc *services.FTPService) *FTPHandler { return &FTPHandler{svc: svc} }

func (h *FTPHandler) List(w http.ResponseWriter, r *http.Request) {
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

func (h *FTPHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DomainID string `json:"domain_id"`
		Username string `json:"username"`
		Password string `json:"password"`
		HomeDir  string `json:"home_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.DomainID == "" || req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "domain_id, username and password are required")
		return
	}
	f, err := h.svc.Create(r.Context(), req.DomainID, req.Username, req.Password, req.HomeDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

func (h *FTPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *FTPHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}
	if err := h.svc.UpdatePassword(r.Context(), chi.URLParam(r, "id"), req.Password); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}
