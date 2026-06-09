package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/api"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/config"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/db"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/updater"
	"github.com/joho/godotenv"
)

func warnIfDefaultCredentials(cfg *config.Config) {
	defaultPw := "ControlPanel2024!"
	defaultJWT := "dev_secret_change_in_production_32c"

	if cfg.AdminPassword == "" || cfg.AdminPassword == defaultPw {
		slog.Warn("SECURITY: default admin password is active — change ADMIN_PASSWORD immediately")
	}
	if cfg.JWTSecret == defaultJWT {
		slog.Warn("SECURITY: default JWT secret is active — set JWT_SECRET to a random 32+ char string")
	}
}

func main() {
	_ = godotenv.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := config.Load()

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	if err := db.SeedAdmin(database, cfg.AdminEmail, cfg.AdminPassword); err != nil {
		slog.Error("seed admin failed", "error", err)
		os.Exit(1)
	}

	warnIfDefaultCredentials(cfg)

	// Start background update checker (checks every 24h, auto-updates if setting enabled)
	panelSvc := services.NewPanelUpdateService(cfg.InstallDir, cfg.GitHubRepo)
	settingsSvc := services.NewSettingsService(database)
	updater.Start(panelSvc, settingsSvc)

	router := api.NewRouter(cfg, database)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("master server starting", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
