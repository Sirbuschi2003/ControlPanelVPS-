package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Sirbuschi2003/ControlPanelVPS/agent/internal/collector"
	"github.com/Sirbuschi2003/ControlPanelVPS/agent/internal/executor"
)

// Handler is the root HTTP handler for the agent API.
type Handler struct {
	token  string
	nodeID string
	mux    *http.ServeMux
}

// NewHandler wires up all routes and returns a ready-to-use Handler.
func NewHandler(token, nodeID string) *Handler {
	h := &Handler{token: token, nodeID: nodeID, mux: http.NewServeMux()}
	h.registerRoutes()
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// registerRoutes maps every API path to its handler function.
func (h *Handler) registerRoutes() {
	auth := h.authMiddleware

	// Health — public, no auth
	h.mux.HandleFunc("GET /health", h.health)

	// Terminal — WebSocket PTY session
	h.mux.HandleFunc("GET /terminal", auth(h.terminal))

	// Metrics
	h.mux.HandleFunc("GET /metrics", auth(h.metrics))

	// System Update
	h.mux.HandleFunc("GET /system/info", auth(h.systemInfo))
	h.mux.HandleFunc("GET /system/check-updates", auth(h.systemCheckUpdates))
	h.mux.HandleFunc("POST /system/update", auth(h.systemRunUpdate))

	// Nginx
	h.mux.HandleFunc("GET /nginx/vhosts", auth(h.listVhosts))
	h.mux.HandleFunc("POST /nginx/vhosts", auth(h.createVhost))
	h.mux.HandleFunc("PUT /nginx/vhosts/{domain}", auth(h.updateVhost))
	h.mux.HandleFunc("DELETE /nginx/vhosts/{domain}", auth(h.deleteVhost))
	h.mux.HandleFunc("POST /nginx/vhosts/{domain}/toggle", auth(h.toggleVhost))
	h.mux.HandleFunc("POST /nginx/reload", auth(h.reloadNginx))

	// SSL
	h.mux.HandleFunc("POST /ssl/issue", auth(h.issueSSL))
	h.mux.HandleFunc("POST /ssl/renew/{domain}", auth(h.renewSSL))
	h.mux.HandleFunc("DELETE /ssl/{domain}", auth(h.deleteSSL))
	h.mux.HandleFunc("GET /ssl", auth(h.listSSL))

	// Databases
	h.mux.HandleFunc("POST /databases", auth(h.createDatabase))
	h.mux.HandleFunc("DELETE /databases/{name}", auth(h.dropDatabase))
	h.mux.HandleFunc("GET /databases", auth(h.listDatabases))

	// DNS
	h.mux.HandleFunc("POST /dns/zones", auth(h.createZone))
	h.mux.HandleFunc("DELETE /dns/zones/{name}", auth(h.deleteZone))
	h.mux.HandleFunc("POST /dns/zones/{name}/records", auth(h.addDNSRecord))
	h.mux.HandleFunc("DELETE /dns/records/{id}", auth(h.deleteDNSRecord))

	// Mail
	h.mux.HandleFunc("POST /mail/domains", auth(h.addMailDomain))
	h.mux.HandleFunc("DELETE /mail/domains/{domain}", auth(h.removeMailDomain))
	h.mux.HandleFunc("POST /mail/accounts", auth(h.createMailAccount))
	h.mux.HandleFunc("PUT /mail/accounts/{email}", auth(h.updateMailAccount))
	h.mux.HandleFunc("DELETE /mail/accounts/{email}", auth(h.deleteMailAccount))
	h.mux.HandleFunc("POST /mail/aliases", auth(h.addAlias))
	h.mux.HandleFunc("DELETE /mail/aliases/{source}", auth(h.removeAlias))
	h.mux.HandleFunc("POST /mail/setup-tls", auth(h.setupMailTLS))
	h.mux.HandleFunc("POST /mail/setup-rspamd", auth(h.setupRspamd))
	h.mux.HandleFunc("GET /mail/rspamd/status", auth(h.rspamdStatus))
	h.mux.HandleFunc("GET /mail/spam/config", auth(h.getSpamConfig))
	h.mux.HandleFunc("PUT /mail/spam/config", auth(h.setSpamConfig))
	h.mux.HandleFunc("POST /mail/dkim/{domain}", auth(h.setupDKIM))
	h.mux.HandleFunc("GET /mail/dkim/{domain}", auth(h.getDKIMKey))

	// Monitoring
	h.mux.HandleFunc("GET /monitoring/health", auth(h.healthCheck))

	// Firewall
	h.mux.HandleFunc("POST /firewall/rules", auth(h.addFirewallRule))
	h.mux.HandleFunc("DELETE /firewall/rules", auth(h.deleteFirewallRule))
	h.mux.HandleFunc("POST /firewall/reload", auth(h.reloadFirewall))
	h.mux.HandleFunc("GET /firewall/status", auth(h.firewallStatus))

	// Backups
	h.mux.HandleFunc("POST /backups", auth(h.runBackup))
	h.mux.HandleFunc("GET /backups", auth(h.listBackups))
	h.mux.HandleFunc("DELETE /backups/{filename}", auth(h.deleteBackup))

	// Services
	h.mux.HandleFunc("GET /services", auth(h.listServices))
	h.mux.HandleFunc("POST /services/{name}/action", auth(h.serviceAction))

	// Crons
	h.mux.HandleFunc("GET /crons", auth(h.listCrons))
	h.mux.HandleFunc("POST /crons", auth(h.createCron))
	h.mux.HandleFunc("PUT /crons/{id}", auth(h.updateCron))
	h.mux.HandleFunc("DELETE /crons/{id}", auth(h.deleteCron))

	// Logs
	h.mux.HandleFunc("GET /logs", auth(h.listLogs))
	h.mux.HandleFunc("GET /logs/{name}", auth(h.getLog))

	// File Manager
	h.mux.HandleFunc("GET /files", auth(h.listFiles))
	h.mux.HandleFunc("GET /files/content", auth(h.readFile))
	h.mux.HandleFunc("POST /files/content", auth(h.writeFile))
	h.mux.HandleFunc("DELETE /files", auth(h.deleteFile))
	h.mux.HandleFunc("POST /files/mkdir", auth(h.makeDir))

	// Packages
	h.mux.HandleFunc("GET /packages/updates", auth(h.listPackageUpdates))
	h.mux.HandleFunc("POST /packages/update", auth(h.applyPackageUpdates))

	// Subdomains
	h.mux.HandleFunc("POST /subdomains", auth(h.createSubdomain))
	h.mux.HandleFunc("DELETE /subdomains/{name}", auth(h.deleteSubdomain))

	// PHP settings
	h.mux.HandleFunc("PUT /php/settings/{domain}", auth(h.updatePHPSettings))
	h.mux.HandleFunc("DELETE /php/settings/{domain}", auth(h.deletePHPSettings))

	// FTP
	h.mux.HandleFunc("POST /ftp/accounts", auth(h.createFTPAccount))
	h.mux.HandleFunc("DELETE /ftp/accounts/{username}", auth(h.deleteFTPAccount))
	h.mux.HandleFunc("PUT /ftp/accounts/{username}/password", auth(h.updateFTPPassword))
	h.mux.HandleFunc("POST /ftp/setup", auth(h.setupFTP))

	// DNS record edit
	h.mux.HandleFunc("PUT /dns/zones/{zone}/records/{id}", auth(h.updateDNSRecord))
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeBody(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

// ─── Middleware ───────────────────────────────────────────────────────────────

func (h *Handler) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") || strings.TrimPrefix(header, "Bearer ") != h.token {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r)
	}
}

// ─── Health & Metrics ────────────────────────────────────────────────────────

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "ok",
		"node_id":  h.nodeID,
		"hostname": hostname,
		"os":       "linux",
	})
}

func (h *Handler) metrics(w http.ResponseWriter, r *http.Request) {
	m, err := collector.Collect()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "collection failed")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// ─── Nginx ───────────────────────────────────────────────────────────────────

func (h *Handler) listVhosts(w http.ResponseWriter, r *http.Request) {
	vhosts, err := executor.ListVhosts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, vhosts)
}

func (h *Handler) createVhost(w http.ResponseWriter, r *http.Request) {
	var cfg executor.VhostConfig
	if err := decodeBody(r, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if cfg.Domain == "" {
		writeError(w, http.StatusBadRequest, "domain is required")
		return
	}
	if err := executor.CreateVhost(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created", "domain": cfg.Domain})
}

func (h *Handler) updateVhost(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	var cfg executor.VhostConfig
	if err := decodeBody(r, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.UpdateVhost(domain, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "domain": domain})
}

func (h *Handler) deleteVhost(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	if err := executor.DeleteVhost(domain); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "domain": domain})
}

func (h *Handler) toggleVhost(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.ToggleVhost(domain, req.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "domain": domain, "enabled": req.Enabled})
}

func (h *Handler) reloadNginx(w http.ResponseWriter, r *http.Request) {
	if err := executor.ReloadNginx(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reloaded"})
}

// ─── SSL ─────────────────────────────────────────────────────────────────────

func (h *Handler) issueSSL(w http.ResponseWriter, r *http.Request) {
	var req executor.SSLIssueRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	info, err := executor.IssueSSL(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (h *Handler) renewSSL(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	if err := executor.RenewSSL(domain); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "renewed", "domain": domain})
}

func (h *Handler) deleteSSL(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	if err := executor.DeleteSSL(domain); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "domain": domain})
}

func (h *Handler) listSSL(w http.ResponseWriter, r *http.Request) {
	certs, err := executor.ListSSL()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, certs)
}

// ─── Databases ───────────────────────────────────────────────────────────────

func (h *Handler) createDatabase(w http.ResponseWriter, r *http.Request) {
	var req executor.DBCreateRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.CreateDatabase(req); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created", "name": req.Name})
}

func (h *Handler) dropDatabase(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	dbType := r.URL.Query().Get("db_type")
	dbUser := r.URL.Query().Get("db_user")
	if err := executor.DropDatabase(name, dbType, dbUser); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "dropped", "name": name})
}

func (h *Handler) listDatabases(w http.ResponseWriter, r *http.Request) {
	dbType := r.URL.Query().Get("db_type")
	dbs, err := executor.ListDatabases(dbType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dbs)
}

// ─── DNS ─────────────────────────────────────────────────────────────────────

func (h *Handler) createZone(w http.ResponseWriter, r *http.Request) {
	var cfg executor.ZoneConfig
	if err := decodeBody(r, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.CreateZone(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created", "zone": cfg.Name})
}

func (h *Handler) deleteZone(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := executor.DeleteZone(name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "zone": name})
}

func (h *Handler) addDNSRecord(w http.ResponseWriter, r *http.Request) {
	zoneName := r.PathValue("name")
	var rec executor.RecordRequest
	if err := decodeBody(r, &rec); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.AddRecord(zoneName, rec); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "added", "zone": zoneName})
}

func (h *Handler) deleteDNSRecord(w http.ResponseWriter, r *http.Request) {
	// id format: {zone}:{name}:{type} — zone is the first segment
	id := r.PathValue("id")
	parts := strings.SplitN(id, ":", 2)
	if len(parts) < 2 {
		writeError(w, http.StatusBadRequest, "invalid record id format: expected zone:name:type")
		return
	}
	zoneName := parts[0]
	recordID := parts[1]

	if err := executor.DeleteRecord(zoneName, recordID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

// ─── Mail ─────────────────────────────────────────────────────────────────────

func (h *Handler) addMailDomain(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domain string `json:"domain"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.AddMailDomain(req.Domain); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "added", "domain": req.Domain})
}

func (h *Handler) removeMailDomain(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	if err := executor.RemoveMailDomain(domain); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed", "domain": domain})
}

func (h *Handler) createMailAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		QuotaMB  int    `json:"quota_mb"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// Hash password before storing
	hashed := executor.HashPassword(req.Password)
	if hashed == "" {
		writeError(w, http.StatusInternalServerError, "password hashing failed")
		return
	}
	if err := executor.CreateMailAccount(req.Email, hashed, req.QuotaMB); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created", "email": req.Email})
}

func (h *Handler) updateMailAccount(w http.ResponseWriter, r *http.Request) {
	email := r.PathValue("email")
	var req struct {
		Password string `json:"password"`
		QuotaMB  int    `json:"quota_mb"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.UpdateMailAccount(email, req.Password, req.QuotaMB); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "email": email})
}

func (h *Handler) deleteMailAccount(w http.ResponseWriter, r *http.Request) {
	email := r.PathValue("email")
	if err := executor.DeleteMailAccount(email); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "email": email})
}

func (h *Handler) addAlias(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.AddAlias(req.Source, req.Destination); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "added", "source": req.Source})
}

func (h *Handler) removeAlias(w http.ResponseWriter, r *http.Request) {
	source := r.PathValue("source")
	if err := executor.RemoveAlias(source); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed", "source": source})
}

// ─── Firewall ─────────────────────────────────────────────────────────────────

func (h *Handler) addFirewallRule(w http.ResponseWriter, r *http.Request) {
	var rule executor.FirewallRule
	if err := decodeBody(r, &rule); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.AddRule(rule); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "added"})
}

func (h *Handler) deleteFirewallRule(w http.ResponseWriter, r *http.Request) {
	var rule executor.FirewallRule
	if err := decodeBody(r, &rule); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.DeleteRule(rule); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) reloadFirewall(w http.ResponseWriter, r *http.Request) {
	if err := executor.ReloadFirewall(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reloaded"})
}

func (h *Handler) firewallStatus(w http.ResponseWriter, r *http.Request) {
	status, err := executor.GetStatus()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}

// ─── Backups ─────────────────────────────────────────────────────────────────

func (h *Handler) runBackup(w http.ResponseWriter, r *http.Request) {
	var req executor.BackupRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := executor.RunBackup(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) listBackups(w http.ResponseWriter, r *http.Request) {
	files, err := executor.ListBackups()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, files)
}

func (h *Handler) deleteBackup(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	if err := executor.DeleteBackup(filename); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "filename": filename})
}

// ─── Services ─────────────────────────────────────────────────────────────────

func (h *Handler) listServices(w http.ResponseWriter, r *http.Request) {
	services, err := executor.ListServices()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, services)
}

func (h *Handler) serviceAction(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req struct {
		Action string `json:"action"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.ServiceAction(name, req.Action); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": name, "action": req.Action})
}

// ─── Crons ───────────────────────────────────────────────────────────────────

func (h *Handler) listCrons(w http.ResponseWriter, r *http.Request) {
	crons, err := executor.ListCrons()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, crons)
}

func (h *Handler) createCron(w http.ResponseWriter, r *http.Request) {
	var entry executor.CronEntry
	if err := decodeBody(r, &entry); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.CreateCron(entry); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created", "id": entry.ID})
}

func (h *Handler) updateCron(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var entry executor.CronEntry
	if err := decodeBody(r, &entry); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	entry.ID = id
	if err := executor.UpdateCron(entry); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "id": id})
}

func (h *Handler) deleteCron(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := executor.DeleteCron(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

// ─── Logs ─────────────────────────────────────────────────────────────────────

func (h *Handler) listLogs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, executor.ListLogs())
}

func (h *Handler) getLog(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	linesStr := r.URL.Query().Get("lines")
	lines := 200
	if linesStr != "" {
		if n, err := strconv.Atoi(linesStr); err == nil && n > 0 {
			lines = n
		}
	}
	logLines, err := executor.GetLog(name, lines)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, logLines)
}

// ─── File Manager ─────────────────────────────────────────────────────────────

func (h *Handler) listFiles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path query parameter is required")
		return
	}
	files, err := executor.ListDirectory(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, files)
}

func (h *Handler) readFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path query parameter is required")
		return
	}
	content, err := executor.ReadFile(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": path, "content": content})
}

func (h *Handler) writeFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.WriteFile(req.Path, req.Content); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "written", "path": req.Path})
}

func (h *Handler) deleteFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path query parameter is required")
		return
	}
	if err := executor.DeletePath(path); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "path": path})
}

func (h *Handler) makeDir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.MakeDir(req.Path); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "created", "path": req.Path})
}

// ─── Packages ─────────────────────────────────────────────────────────────────

func (h *Handler) listPackageUpdates(w http.ResponseWriter, r *http.Request) {
	updates, err := executor.ListUpdates()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updates)
}

func (h *Handler) applyPackageUpdates(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Packages []string `json:"packages"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.ApplyUpdates(req.Packages); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// ─── Mail TLS, DKIM, Rspamd ──────────────────────────────────────────────────

func (h *Handler) setupMailTLS(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Hostname string `json:"hostname"`
		CertPath string `json:"cert_path"`
		KeyPath  string `json:"key_path"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Hostname == "" || req.CertPath == "" || req.KeyPath == "" {
		writeError(w, http.StatusBadRequest, "hostname, cert_path and key_path are required")
		return
	}
	if err := executor.ConfigureMailTLS(req.Hostname, req.CertPath, req.KeyPath); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "configured", "hostname": req.Hostname})
}

func (h *Handler) setupRspamd(w http.ResponseWriter, r *http.Request) {
	if err := executor.SetupRspamd(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "configured"})
}

func (h *Handler) rspamdStatus(w http.ResponseWriter, r *http.Request) {
	stats, err := executor.GetRspamdStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *Handler) getSpamConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := executor.GetSpamConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *Handler) setSpamConfig(w http.ResponseWriter, r *http.Request) {
	var cfg executor.SpamConfig
	if err := decodeBody(r, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.SetSpamConfig(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) setupDKIM(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	pubKey, err := executor.ConfigureDKIM(domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":     "configured",
		"domain":     domain,
		"public_key": pubKey,
		"dns_record": "mail._domainkey." + domain + " TXT \"" + pubKey + "\"",
	})
}

func (h *Handler) getDKIMKey(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	pubKey, err := executor.GetDKIMPublicKey(domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"domain":     domain,
		"public_key": pubKey,
		"dns_record": "mail._domainkey." + domain + " TXT \"" + pubKey + "\"",
	})
}

// ─── Monitoring ───────────────────────────────────────────────────────────────

func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	report, err := executor.RunHealthCheck()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	status := http.StatusOK
	if !report.Healthy {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, report)
}

// ─── System Update ────────────────────────────────────────────────────────────

func (h *Handler) systemInfo(w http.ResponseWriter, r *http.Request) {
	commit, date, branch := executor.GetVersion()
	hostname, _ := os.Hostname()
	writeJSON(w, http.StatusOK, map[string]string{
		"commit":      commit,
		"branch":      branch,
		"commit_date": date,
		"node_id":     h.nodeID,
		"hostname":    hostname,
		"os":          "linux",
	})
}

func (h *Handler) systemCheckUpdates(w http.ResponseWriter, r *http.Request) {
	available, latestCommit, err := executor.CheckUpdates()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	commit, _, _ := executor.GetVersion()
	writeJSON(w, http.StatusOK, map[string]any{
		"available":      available,
		"current_commit": commit,
		"latest_commit":  latestCommit,
	})
}

func (h *Handler) systemRunUpdate(w http.ResponseWriter, r *http.Request) {
	result, err := executor.RunUpdate()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) terminal(w http.ResponseWriter, r *http.Request) {
	executor.TerminalWebSocket(w, r)
}

// ─── Subdomains ───────────────────────────────────────────────────────────────

func (h *Handler) createSubdomain(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		Domain       string `json:"domain"`
		DocumentRoot string `json:"document_root"`
		PHPVersion   string `json:"php_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.CreateSubdomainVhost(req.Name, req.Domain, req.DocumentRoot, req.PHPVersion); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"message": "subdomain created"})
}

func (h *Handler) deleteSubdomain(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	domain := r.URL.Query().Get("domain")
	if err := executor.DeleteSubdomainVhost(name, domain); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "subdomain deleted"})
}

// ─── PHP Settings ─────────────────────────────────────────────────────────────

func (h *Handler) updatePHPSettings(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	var cfg executor.PHPPoolConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	cfg.Domain = domain
	if err := executor.WritePHPPool(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "php settings updated"})
}

func (h *Handler) deletePHPSettings(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	phpVersion := r.URL.Query().Get("php_version")
	if err := executor.DeletePHPPool(domain, phpVersion); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "php pool deleted"})
}

// ─── FTP ─────────────────────────────────────────────────────────────────────

func (h *Handler) setupFTP(w http.ResponseWriter, r *http.Request) {
	if err := executor.SetupVsftpd(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "vsftpd configured"})
}

func (h *Handler) createFTPAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		HomeDir  string `json:"home_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.CreateFTPAccount(req.Username, req.Password, req.HomeDir); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"message": "ftp account created"})
}

func (h *Handler) deleteFTPAccount(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	if err := executor.DeleteFTPAccount(username); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ftp account deleted"})
}

func (h *Handler) updateFTPPassword(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := executor.UpdateFTPPassword(username, req.Password); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ftp password updated"})
}

// ─── DNS Record Update ────────────────────────────────────────────────────────

func (h *Handler) updateDNSRecord(w http.ResponseWriter, r *http.Request) {
	zoneName := r.PathValue("zone")
	recordID := r.PathValue("id")
	var req struct {
		OldName    string `json:"old_name"`
		OldType    string `json:"old_type"`
		OldContent string `json:"old_content"`
		Name       string `json:"name"`
		Type       string `json:"type"`
		Content    string `json:"content"`
		TTL        int    `json:"ttl"`
		Priority   int    `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// Delete old record, add new one
	delID := req.OldName + ":" + req.OldType
	if req.OldContent != "" {
		delID += ":" + req.OldContent
	}
	_ = recordID
	if err := executor.DeleteRecord(zoneName, delID); err != nil {
		writeError(w, http.StatusInternalServerError, "delete old record: "+err.Error())
		return
	}
	ttl := req.TTL
	if ttl == 0 {
		ttl = 3600
	}
	if err := executor.AddRecord(zoneName, executor.RecordRequest{
		Name:     req.Name,
		Type:     req.Type,
		Content:  req.Content,
		TTL:      ttl,
		Priority: req.Priority,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "add new record: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "dns record updated"})
}
