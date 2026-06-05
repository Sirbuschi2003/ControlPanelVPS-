package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PackageService manages package updates on remote servers via the agent.
type PackageService struct {
	db *pgxpool.Pool
}

// NewPackageService creates a new PackageService.
func NewPackageService(db *pgxpool.Pool) *PackageService {
	return &PackageService{db: db}
}

func (s *PackageService) agentFor(ctx context.Context, serverID string) (*agent.AgentClient, error) {
	var agentURL, agentToken string
	err := s.db.QueryRow(ctx, `SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID).
		Scan(&agentURL, &agentToken)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	return agent.NewAgentClient(agentURL, agentToken), nil
}

// ListUpdates returns available package updates from the agent.
func (s *PackageService) ListUpdates(ctx context.Context, serverID string) ([]models.PackageUpdate, error) {
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return nil, err
	}

	data, err := ac.Get(ctx, "/packages/updates")
	if err != nil {
		return nil, fmt.Errorf("agent list package updates: %w", err)
	}

	var updates []models.PackageUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse package updates response: %w", err)
	}
	if updates == nil {
		updates = []models.PackageUpdate{}
	}
	return updates, nil
}

// ApplyUpdates instructs the agent to install the given packages.
// If packages is empty, all available updates are applied.
func (s *PackageService) ApplyUpdates(ctx context.Context, serverID string, packages []string) error {
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return err
	}

	if packages == nil {
		packages = []string{}
	}

	_, err = ac.Post(ctx, "/packages/update", map[string]any{
		"packages": packages,
	})
	if err != nil {
		return fmt.Errorf("agent apply package updates: %w", err)
	}
	return nil
}
