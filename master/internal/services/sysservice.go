package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SystemServiceManager manages systemd services on remote servers via the agent.
type SystemServiceManager struct {
	db *pgxpool.Pool
}

// NewSystemServiceManager creates a new SystemServiceManager.
func NewSystemServiceManager(db *pgxpool.Pool) *SystemServiceManager {
	return &SystemServiceManager{db: db}
}

func (s *SystemServiceManager) agentFor(ctx context.Context, serverID string) (*agent.AgentClient, error) {
	var agentURL, agentToken string
	err := s.db.QueryRow(ctx, `SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID).
		Scan(&agentURL, &agentToken)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	return agent.NewAgentClient(agentURL, agentToken), nil
}

// List returns all system services from the agent.
func (s *SystemServiceManager) List(ctx context.Context, serverID string) ([]models.SystemService, error) {
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return nil, err
	}

	data, err := ac.Get(ctx, "/services")
	if err != nil {
		return nil, fmt.Errorf("agent list services: %w", err)
	}

	var services []models.SystemService
	if err := json.Unmarshal(data, &services); err != nil {
		return nil, fmt.Errorf("parse services response: %w", err)
	}
	if services == nil {
		services = []models.SystemService{}
	}
	return services, nil
}

// Action performs a start/stop/restart/enable/disable action on a system service.
func (s *SystemServiceManager) Action(ctx context.Context, serverID, serviceName, action string) error {
	switch action {
	case "start", "stop", "restart", "enable", "disable":
		// valid actions
	default:
		return fmt.Errorf("invalid action %q: must be start, stop, restart, enable, or disable", action)
	}

	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return err
	}

	_, err = ac.Post(ctx, "/services/"+serviceName+"/action", map[string]string{"action": action})
	if err != nil {
		return fmt.Errorf("agent service action %s on %s: %w", action, serviceName, err)
	}
	return nil
}
