package updater

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
)

// Status holds the last cached update check result.
type Status struct {
	Available     bool      `json:"available"`
	CurrentCommit string    `json:"current_commit"`
	LatestCommit  string    `json:"latest_commit"`
	PublishedAt   string    `json:"published_at"`
	CheckedAt     time.Time `json:"checked_at"`
	Error         string    `json:"error,omitempty"`
}

var (
	mu     sync.RWMutex
	cached Status
)

// GetStatus returns the last cached update status without hitting GitHub.
func GetStatus() Status {
	mu.RLock()
	defer mu.RUnlock()
	return cached
}

// Start launches the background update checker goroutine.
// It checks 30 seconds after startup, then every 24 hours.
func Start(panelSvc *services.PanelUpdateService, settingsSvc *services.SettingsService) {
	go run(panelSvc, settingsSvc)
}

func run(panelSvc *services.PanelUpdateService, settingsSvc *services.SettingsService) {
	// Give services time to fully start before first check
	time.Sleep(30 * time.Second)
	runCheck(panelSvc, settingsSvc)

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		runCheck(panelSvc, settingsSvc)
	}
}

func runCheck(panelSvc *services.PanelUpdateService, settingsSvc *services.SettingsService) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	slog.Info("panel update check starting")

	result, err := panelSvc.CheckUpdate(ctx)

	mu.Lock()
	if err != nil {
		cached = Status{Error: err.Error(), CheckedAt: time.Now()}
		mu.Unlock()
		slog.Warn("panel update check failed", "error", err)
		return
	}
	cached = Status{
		Available:     result.Available,
		CurrentCommit: result.CurrentCommit,
		LatestCommit:  result.LatestCommit,
		PublishedAt:   result.PublishedAt,
		CheckedAt:     time.Now(),
	}
	available := result.Available
	mu.Unlock()

	if !available {
		slog.Info("panel is up to date")
		return
	}
	slog.Info("panel update available", "latest", result.LatestCommit)

	// Apply automatically if setting is enabled
	autoUpdate := false
	if settingsSvc != nil {
		val, err := settingsSvc.Get(ctx, "auto_update")
		if err == nil {
			autoUpdate = val == "true"
		}
	}

	if autoUpdate {
		slog.Info("auto-update enabled — applying update")
		updateCtx, updateCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer updateCancel()
		if _, err := panelSvc.RunUpdate(updateCtx); err != nil {
			slog.Error("auto-update failed", "error", err)
		} else {
			slog.Info("auto-update applied — services restarting")
		}
	}
}
