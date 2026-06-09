package api

import (
	"net/http"
	"strings"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/api/handlers"
	authmw "github.com/Sirbuschi2003/ControlPanelVPS/master/internal/api/middleware"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/config"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(cfg *config.Config, db *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Limit request body to 10 MB to prevent DoS via oversized payloads.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)
			next.ServeHTTP(w, r)
		})
	})

	// CORS: explicit origin list; never use a wildcard with credentials.
	allowedOrigins := []string{"http://localhost:3000"}
	if cfg.AllowedOrigins != "" {
		allowedOrigins = strings.Split(cfg.AllowedOrigins, ",")
		for i, o := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(o)
		}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// ---- Services ----
	authSvc := services.NewAuthService(db, cfg.JWTSecret)
	serverSvc := services.NewServerService(db)
	websiteSvc := services.NewWebsiteService(db)
	sslSvc := services.NewSSLService(db)
	dbSvc := services.NewDatabaseService(db)
	dnsSvc := services.NewDNSService(db)
	mailSvc := services.NewMailService(db)
	firewallSvc := services.NewFirewallService(db)
	backupSvc := services.NewBackupService(db)
	sysSvc := services.NewSystemServiceManager(db)
	cronSvc := services.NewCronService(db)
	domainSvc := services.NewDomainService(db, websiteSvc, dnsSvc, mailSvc)
	logSvc := services.NewLogService(db)
	fileSvc := services.NewFileService(db)
	packageSvc := services.NewPackageService(db)
	userSvc := services.NewUserService(db)
	settingsSvc := services.NewSettingsService(db)
	systemUpdateSvc := services.NewSystemUpdateService(db)
	monitoringSvc := services.NewMonitoringService(db)
	panelUpdateSvc := services.NewPanelUpdateService(cfg.InstallDir, cfg.GitHubRepo)

	// ---- Handlers ----
	terminalHandler := handlers.NewTerminalHandler(db)
	domainHandler := handlers.NewDomainHandler(domainSvc)
	authHandler := handlers.NewAuthHandler(authSvc)
	serverHandler := handlers.NewServerHandler(serverSvc)
	websiteHandler := handlers.NewWebsiteHandler(websiteSvc)
	sslHandler := handlers.NewSSLHandler(sslSvc)
	dbHandler := handlers.NewDatabaseHandler(dbSvc)
	dnsHandler := handlers.NewDNSHandler(dnsSvc)
	mailHandler := handlers.NewMailHandler(mailSvc)
	firewallHandler := handlers.NewFirewallHandler(firewallSvc)
	backupHandler := handlers.NewBackupHandler(backupSvc)
	sysHandler := handlers.NewSystemServiceHandler(sysSvc)
	cronHandler := handlers.NewCronHandler(cronSvc)
	logHandler := handlers.NewLogHandler(logSvc)
	fileHandler := handlers.NewFileHandler(fileSvc)
	packageHandler := handlers.NewPackageHandler(packageSvc)
	userHandler := handlers.NewUserHandler(userSvc)
	settingsHandler := handlers.NewSettingsHandler(settingsSvc)
	systemHandler := handlers.NewSystemHandler(systemUpdateSvc)
	monitoringHandler := handlers.NewMonitoringHandler(monitoringSvc)
	panelUpdateHandler := handlers.NewPanelUpdateHandler(panelUpdateSvc, settingsSvc)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Public routes with rate limiting
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(10, 60))
		r.Post("/api/auth/login", authHandler.Login)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authmw.Auth(authSvc))

		r.Get("/api/auth/me", authHandler.Me)

		// Servers
		r.Get("/api/servers", serverHandler.List)
		r.Post("/api/servers", serverHandler.Create)
		r.Put("/api/servers/{id}", serverHandler.Update)
		r.Delete("/api/servers/{id}", serverHandler.Delete)
		r.Get("/api/servers/{id}/metrics", serverHandler.GetMetrics)

		// Domains (Plesk-like subscriptions)
		r.Get("/api/domains", domainHandler.List)
		r.Get("/api/domains/{id}", domainHandler.Get)
		r.Get("/api/domains/{id}/resources", domainHandler.GetResources)
		r.Group(func(r chi.Router) {
			r.Use(authmw.AdminOnly)
			r.Post("/api/domains", domainHandler.Create)
			r.Delete("/api/domains/{id}", domainHandler.Delete)
			r.Get("/api/domains/{id}/users", domainHandler.ListUsers)
			r.Post("/api/domains/{id}/users", domainHandler.AssignUser)
			r.Delete("/api/domains/{id}/users/{user_id}", domainHandler.RemoveUser)
		})

		// Websites
		r.Get("/api/websites", websiteHandler.List)
		r.Post("/api/websites", websiteHandler.Create)
		r.Put("/api/websites/{id}", websiteHandler.Update)
		r.Delete("/api/websites/{id}", websiteHandler.Delete)
		r.Post("/api/websites/{id}/toggle", websiteHandler.Toggle)
		r.Post("/api/websites/{id}/ssl", websiteHandler.EnableSSL)

		// SSL Certificates
		r.Get("/api/ssl", sslHandler.List)
		r.Post("/api/ssl", sslHandler.Issue)
		r.Post("/api/ssl/{id}/renew", sslHandler.Renew)
		r.Delete("/api/ssl/{id}", sslHandler.Delete)

		// Databases
		r.Get("/api/databases", dbHandler.List)
		r.Post("/api/databases", dbHandler.Create)
		r.Delete("/api/databases/{id}", dbHandler.Delete)
		r.Get("/api/databases/{id}/password", dbHandler.GetPassword)

		// DNS
		r.Get("/api/dns/zones", dnsHandler.ListZones)
		r.Post("/api/dns/zones", dnsHandler.CreateZone)
		r.Get("/api/dns/zones/{id}", dnsHandler.GetZone)
		r.Delete("/api/dns/zones/{id}", dnsHandler.DeleteZone)
		r.Get("/api/dns/zones/{id}/records", dnsHandler.GetRecords)
		r.Post("/api/dns/zones/{id}/records", dnsHandler.AddRecord)
		r.Delete("/api/dns/records/{id}", dnsHandler.DeleteRecord)

		// Mail
		r.Get("/api/mail/domains", mailHandler.ListDomains)
		r.Post("/api/mail/domains", mailHandler.CreateDomain)
		r.Delete("/api/mail/domains/{id}", mailHandler.DeleteDomain)
		r.Get("/api/mail/accounts", mailHandler.ListAccounts)
		r.Post("/api/mail/accounts", mailHandler.CreateAccount)
		r.Delete("/api/mail/accounts/{id}", mailHandler.DeleteAccount)
		r.Get("/api/mail/aliases", mailHandler.ListAliases)
		r.Post("/api/mail/aliases", mailHandler.CreateAlias)
		r.Delete("/api/mail/aliases/{id}", mailHandler.DeleteAlias)

		// Firewall
		r.Get("/api/firewall", firewallHandler.List)
		r.Post("/api/firewall", firewallHandler.Create)
		r.Delete("/api/firewall/{id}", firewallHandler.Delete)
		r.Post("/api/firewall/{id}/toggle", firewallHandler.Toggle)
		r.Post("/api/firewall/reload", firewallHandler.Reload)

		// Backups
		r.Get("/api/backups/configs", backupHandler.ListConfigs)
		r.Post("/api/backups/configs", backupHandler.CreateConfig)
		r.Delete("/api/backups/configs/{id}", backupHandler.DeleteConfig)
		r.Put("/api/backups/configs/{id}", backupHandler.ToggleConfig)
		r.Post("/api/backups/configs/{id}/run", backupHandler.RunBackup)
		r.Get("/api/backups/jobs", backupHandler.ListJobs)

		// System Services
		r.Get("/api/services", sysHandler.List)
		r.Post("/api/services/{name}/action", sysHandler.Action)

		// Cron Jobs
		r.Get("/api/crons", cronHandler.List)
		r.Post("/api/crons", cronHandler.Create)
		r.Put("/api/crons/{id}", cronHandler.Update)
		r.Delete("/api/crons/{id}", cronHandler.Delete)

		// Logs
		r.Get("/api/logs", logHandler.List)
		r.Get("/api/logs/{serverID}/{logName}", logHandler.GetLog)

		// Files
		r.Get("/api/files", fileHandler.List)
		r.Get("/api/files/content", fileHandler.Read)
		r.Post("/api/files/content", fileHandler.Write)
		r.Delete("/api/files", fileHandler.Delete)
		r.Post("/api/files/mkdir", fileHandler.Mkdir)

		// Packages
		r.Get("/api/packages/updates", packageHandler.ListUpdates)
		r.Post("/api/packages/update", packageHandler.ApplyUpdates)

		// Users — admin-only: privilege escalation risk without RBAC
		r.Group(func(r chi.Router) {
			r.Use(authmw.AdminOnly)
			r.Get("/api/users", userHandler.List)
			r.Post("/api/users", userHandler.Create)
			r.Put("/api/users/{id}", userHandler.Update)
			r.Delete("/api/users/{id}", userHandler.Delete)
			r.Post("/api/users/{id}/password", userHandler.ChangePassword)
			r.Post("/api/users/{id}/totp/setup", userHandler.SetupTOTP)
			r.Post("/api/users/{id}/totp/verify", userHandler.VerifyTOTP)
			r.Delete("/api/users/{id}/totp", userHandler.DisableTOTP)
		})

		// Settings — admin-only: contains SMTP credentials and encryption key
		r.Group(func(r chi.Router) {
			r.Use(authmw.AdminOnly)
			r.Get("/api/settings", settingsHandler.Get)
			r.Put("/api/settings", settingsHandler.Set)
			r.Get("/api/settings/info", settingsHandler.Info)
			r.Post("/api/settings/test-smtp", settingsHandler.TestSMTP)
		})

		// System Update
		r.Get("/api/system/info", systemHandler.Info)
		r.Get("/api/system/check-updates", systemHandler.CheckUpdates)
		r.Post("/api/system/update", systemHandler.RunUpdate)

		// Panel self-update
		r.Get("/api/panel/info", panelUpdateHandler.Info)
		r.Get("/api/panel/update-status", panelUpdateHandler.UpdateStatus)
		r.Get("/api/panel/check-update", panelUpdateHandler.CheckUpdate)
		r.Post("/api/panel/update", panelUpdateHandler.RunUpdate)
		r.Get("/api/panel/auto-update", panelUpdateHandler.GetAutoUpdate)
		r.Put("/api/panel/auto-update", panelUpdateHandler.SetAutoUpdate)

		// Monitoring & Mail Security
		r.Get("/api/monitoring/health", monitoringHandler.HealthCheck)
		r.Post("/api/mail/setup-tls", monitoringHandler.SetupMailTLS)
		r.Post("/api/mail/setup-rspamd", monitoringHandler.SetupRspamd)
		r.Post("/api/mail/dkim/{domain}", monitoringHandler.SetupDKIM)
		r.Get("/api/mail/rspamd/status", monitoringHandler.RspamdStatus)

		// Terminal — WebSocket proxy to agent PTY
		r.Get("/api/terminal/ws", terminalHandler.WebSocket)
	})

	return r
}
