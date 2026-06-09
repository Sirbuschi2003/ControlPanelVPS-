package handlers

import (
	"encoding/json"
	"net/http"

	authmw "github.com/Sirbuschi2003/ControlPanelVPS/master/internal/api/middleware"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
)

type AuthHandler struct {
	authSvc *services.AuthService
}

func NewAuthHandler(authSvc *services.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TOTPCode string `json:"totp_code"`
}

type loginResponse struct {
	Token string `json:"token"`
	User  any    `json:"user"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, user, err := h.authSvc.Login(r.Context(), req.Email, req.Password, req.TOTPCode)
	if err != nil {
		h.authSvc.WriteLoginAudit(r.Context(), "", req.Email, r.RemoteAddr, false)
		switch err {
		case services.ErrInvalidCredentials:
			writeError(w, http.StatusUnauthorized, "invalid email or password")
		case services.ErrTOTPRequired:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "totp_required"})
		case services.ErrInvalidTOTP:
			writeError(w, http.StatusUnauthorized, "invalid 2FA code")
		default:
			writeError(w, http.StatusInternalServerError, "login failed")
		}
		return
	}

	h.authSvc.WriteLoginAudit(r.Context(), user.ID, user.Email, r.RemoteAddr, true)
	writeJSON(w, http.StatusOK, loginResponse{Token: token, User: user})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(authmw.ClaimsKey).(*services.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.authSvc.GetUser(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
