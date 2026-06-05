package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
)

// FileHandler handles HTTP requests for remote file management.
type FileHandler struct {
	svc *services.FileService
}

// NewFileHandler creates a new FileHandler.
func NewFileHandler(svc *services.FileService) *FileHandler {
	return &FileHandler{svc: svc}
}

// List handles GET /api/files?server_id=...&path=...
func (h *FileHandler) List(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	path := r.URL.Query().Get("path")

	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}
	if path == "" {
		path = "/"
	}

	entries, err := h.svc.List(r.Context(), serverID, path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list files: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

// Read handles GET /api/files/content?server_id=...&path=...
func (h *FileHandler) Read(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	path := r.URL.Query().Get("path")

	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}
	if path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	content, err := h.svc.Read(r.Context(), serverID, path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read file: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"content": content})
}

type writeFileRequest struct {
	ServerID string `json:"server_id"`
	Path     string `json:"path"`
	Content  string `json:"content"`
}

// Write handles POST /api/files/content
func (h *FileHandler) Write(w http.ResponseWriter, r *http.Request) {
	var req writeFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ServerID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}
	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	if err := h.svc.Write(r.Context(), req.ServerID, req.Path, req.Content); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write file: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "file written"})
}

// Delete handles DELETE /api/files?server_id=...&path=...
func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	path := r.URL.Query().Get("path")

	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}
	if path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	if err := h.svc.Delete(r.Context(), serverID, path); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete file: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "file deleted"})
}

type mkdirRequest struct {
	ServerID string `json:"server_id"`
	Path     string `json:"path"`
}

// Mkdir handles POST /api/files/mkdir
func (h *FileHandler) Mkdir(w http.ResponseWriter, r *http.Request) {
	var req mkdirRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ServerID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}
	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	if err := h.svc.Mkdir(r.Context(), req.ServerID, req.Path); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create directory: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"message": "directory created"})
}
