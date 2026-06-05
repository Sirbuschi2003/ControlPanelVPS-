package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
)

// UserHandler handles HTTP requests for panel user management.
type UserHandler struct {
	svc *services.UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(svc *services.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// List handles GET /api/users
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, users)
}

type createUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

// Create handles POST /api/users
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Role == "" {
		req.Role = "admin"
	}

	user, err := h.svc.Create(r.Context(), req.Email, req.Password, req.Name, req.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

type updateUserRequest struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

// Update handles PUT /api/users/{id}
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Role == "" {
		writeError(w, http.StatusBadRequest, "role is required")
		return
	}

	if err := h.svc.Update(r.Context(), id, req.Name, req.Role); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "user updated"})
}

// Delete handles DELETE /api/users/{id}
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete user: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}

type changePasswordRequest struct {
	NewPassword string `json:"new_password"`
}

// ChangePassword handles POST /api/users/{id}/password
func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "new_password is required")
		return
	}

	if err := h.svc.ChangePassword(r.Context(), id, req.NewPassword); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to change password: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "password changed"})
}

// SetupTOTP handles POST /api/users/{id}/totp/setup
func (h *UserHandler) SetupTOTP(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	secret, qrURL, err := h.svc.SetupTOTP(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to setup TOTP: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"secret":    secret,
		"qr_url":    qrURL,
	})
}

type verifyTOTPRequest struct {
	Code string `json:"code"`
}

// VerifyTOTP handles POST /api/users/{id}/totp/verify
func (h *UserHandler) VerifyTOTP(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req verifyTOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	if err := h.svc.VerifyAndEnableTOTP(r.Context(), id, req.Code); err != nil {
		if err == services.ErrInvalidTOTP {
			writeError(w, http.StatusUnauthorized, "invalid TOTP code")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to verify TOTP: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "TOTP enabled"})
}

// DisableTOTP handles DELETE /api/users/{id}/totp
func (h *UserHandler) DisableTOTP(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.svc.DisableTOTP(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to disable TOTP: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "TOTP disabled"})
}
