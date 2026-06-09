package handlers

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"time"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
)

var startTime = time.Now()

// SettingsHandler handles HTTP requests for panel settings management.
type SettingsHandler struct {
	svc *services.SettingsService
}

// NewSettingsHandler creates a new SettingsHandler.
func NewSettingsHandler(svc *services.SettingsService) *SettingsHandler {
	return &SettingsHandler{svc: svc}
}

// sensitiveKeys holds setting keys whose values must never be returned in plaintext.
// A non-empty value is replaced with "***" so the UI can detect "is set" vs "empty".
var sensitiveKeys = map[string]bool{
	"smtp_pass":              true,
	"backup_encryption_key": true,
}

// Get handles GET /api/settings
// Returns all settings as a key/value map with sensitive values masked.
func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	settings, err := h.svc.GetAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load settings: "+err.Error())
		return
	}
	for key := range sensitiveKeys {
		if v, exists := settings[key]; exists && v != "" {
			settings[key] = "***"
		}
	}
	writeJSON(w, http.StatusOK, settings)
}

// Set handles PUT /api/settings — accepts a map of key/value pairs for bulk update.
func (h *SettingsHandler) Set(w http.ResponseWriter, r *http.Request) {
	var bulk map[string]string
	if err := json.NewDecoder(r.Body).Decode(&bulk); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.svc.SetBulk(r.Context(), bulk); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save settings: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "settings saved"})
}

// Info handles GET /api/settings/info — returns runtime panel info.
func (h *SettingsHandler) Info(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(startTime).Round(time.Second).String()
	writeJSON(w, http.StatusOK, map[string]string{
		"version":         "1.0.0",
		"uptime":          uptime,
		"database_status": "ok",
		"go_version":      "go1.22",
	})
}

type smtpTestRequest struct {
	SMTPHost    string `json:"smtp_host"`
	SMTPPort    string `json:"smtp_port"`
	SMTPUser    string `json:"smtp_user"`
	SMTPPass    string `json:"smtp_pass"`
	SMTPFrom    string `json:"smtp_from"`
	NotifyEmail string `json:"notify_email"`
}

// TestSMTP handles POST /api/settings/test-smtp
// Sends a test email using the provided SMTP configuration.
// Supports STARTTLS (ports 25, 587) and implicit TLS (port 465).
func (h *SettingsHandler) TestSMTP(w http.ResponseWriter, r *http.Request) {
	var req smtpTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SMTPHost == "" {
		writeError(w, http.StatusBadRequest, "smtp_host is required")
		return
	}
	if req.SMTPPort == "" {
		req.SMTPPort = "587"
	}
	if req.SMTPFrom == "" {
		req.SMTPFrom = "controlpanel@" + req.SMTPHost
	}
	if req.NotifyEmail == "" {
		writeError(w, http.StatusBadRequest, "notify_email is required")
		return
	}

	msg := buildTestEmail(req.SMTPFrom, req.NotifyEmail)

	addr := net.JoinHostPort(req.SMTPHost, req.SMTPPort)
	var auth smtp.Auth
	if req.SMTPUser != "" {
		auth = smtp.PlainAuth("", req.SMTPUser, req.SMTPPass, req.SMTPHost)
	}

	var sendErr error
	if req.SMTPPort == "465" {
		sendErr = sendMailTLS(addr, req.SMTPHost, auth, req.SMTPFrom, req.NotifyEmail, msg)
	} else {
		sendErr = smtp.SendMail(addr, auth, req.SMTPFrom, []string{req.NotifyEmail}, msg)
	}
	if sendErr != nil {
		writeError(w, http.StatusBadGateway, "SMTP error: "+sendErr.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Test-E-Mail erfolgreich gesendet an %s", req.NotifyEmail),
	})
}

func buildTestEmail(from, to string) []byte {
	subject := "ControlPanel – SMTP Test"
	body := "Diese E-Mail bestätigt, dass die SMTP-Konfiguration korrekt funktioniert.\r\n\r\nGesendet von ControlPanelVPS."
	return []byte(
		"From: " + from + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" +
			body,
	)
}

// sendMailTLS sends email over implicit TLS (port 465, no STARTTLS).
func sendMailTLS(addr, host string, auth smtp.Auth, from, to string, msg []byte) error {
	tlsCfg := &tls.Config{ServerName: host} //nolint:gosec — uses system trust store
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("TLS dial: %w", err)
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP client: %w", err)
	}
	defer c.Close()
	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth: %w", err)
		}
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO: %w", err)
	}
	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err := wc.Write(msg); err != nil {
		return fmt.Errorf("write body: %w", err)
	}
	return wc.Close()
}
