package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/api/middleware"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
)

// DatabaseHandler handles HTTP requests for managed database operations.
type DatabaseHandler struct {
	svc *services.DatabaseService
}

// NewDatabaseHandler creates a new DatabaseHandler.
func NewDatabaseHandler(svc *services.DatabaseService) *DatabaseHandler {
	return &DatabaseHandler{svc: svc}
}

// List handles GET /api/databases or GET /api/databases?server_id=...
func (h *DatabaseHandler) List(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")

	dbs, err := h.svc.List(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list databases: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dbs)
}

type createDatabaseRequest struct {
	ServerID   string `json:"server_id"`
	Name       string `json:"name"`
	DBType     string `json:"db_type"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DomainID   string `json:"domain_id"`
}

// Create handles POST /api/databases
func (h *DatabaseHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createDatabaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ServerID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.DBType == "" {
		req.DBType = "mysql"
	}
	if req.DBUser == "" {
		writeError(w, http.StatusBadRequest, "db_user is required")
		return
	}
	if req.DBPassword == "" {
		writeError(w, http.StatusBadRequest, "db_password is required")
		return
	}

	db, err := h.svc.Create(r.Context(), req.ServerID, req.Name, req.DBType, req.DBUser, req.DBPassword, req.DomainID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create database: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, db)
}

// Delete handles DELETE /api/databases/{id}
func (h *DatabaseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete database: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "database deleted"})
}

// GetPassword handles GET /api/databases/{id}/password
func (h *DatabaseHandler) GetPassword(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	actorID := ""
	if claims, ok := r.Context().Value(middleware.ClaimsKey).(*services.Claims); ok {
		actorID = claims.UserID
	}

	password, err := h.svc.GetPassword(r.Context(), id, actorID, r.RemoteAddr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve password: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"password": password})
}
