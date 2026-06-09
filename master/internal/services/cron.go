package services

import (
	"context"
	"fmt"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CronService manages cron jobs on remote servers.
type CronService struct {
	db *pgxpool.Pool
}

// NewCronService creates a new CronService.
func NewCronService(db *pgxpool.Pool) *CronService {
	return &CronService{db: db}
}

func (s *CronService) agentFor(ctx context.Context, serverID string) (*agent.AgentClient, error) {
	var agentURL, agentToken string
	err := s.db.QueryRow(ctx, `SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID).
		Scan(&agentURL, &agentToken)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	return agent.NewAgentClient(agentURL, agentToken), nil
}

// List returns all cron jobs for a server.
func (s *CronService) List(ctx context.Context, serverID string) ([]models.CronJob, error) {
	query := `SELECT id, server_id, name, command, schedule, run_as_user, enabled, last_run, last_status, created_at
		FROM cron_jobs ORDER BY created_at DESC`
	args := []interface{}{}
	if serverID != "" {
		query = `SELECT id, server_id, name, command, schedule, run_as_user, enabled, last_run, last_status, created_at
		FROM cron_jobs WHERE server_id = $1 ORDER BY created_at DESC`
		args = append(args, serverID)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query cron jobs: %w", err)
	}
	defer rows.Close()

	var jobs []models.CronJob
	for rows.Next() {
		var j models.CronJob
		if err := rows.Scan(
			&j.ID, &j.ServerID, &j.Name, &j.Command, &j.Schedule, &j.RunAsUser,
			&j.Enabled, &j.LastRun, &j.LastStatus, &j.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan cron job: %w", err)
		}
		jobs = append(jobs, j)
	}
	if jobs == nil {
		jobs = []models.CronJob{}
	}
	return jobs, nil
}

type agentCronPayload struct {
	Name      string `json:"name"`
	Command   string `json:"command"`
	Schedule  string `json:"schedule"`
	RunAsUser string `json:"run_as_user"`
}

// Create creates a new cron job on the agent and stores it in the database.
func (s *CronService) Create(ctx context.Context, serverID, name, command, schedule, runAsUser string) (*models.CronJob, error) {
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return nil, err
	}

	payload := agentCronPayload{
		Name:      name,
		Command:   command,
		Schedule:  schedule,
		RunAsUser: runAsUser,
	}

	_, err = ac.Post(ctx, "/crons", payload)
	if err != nil {
		return nil, fmt.Errorf("agent create cron job: %w", err)
	}

	var j models.CronJob
	err = s.db.QueryRow(ctx, `
		INSERT INTO cron_jobs (server_id, name, command, schedule, run_as_user)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, server_id, name, command, schedule, run_as_user, enabled, last_run, last_status, created_at
	`, serverID, name, command, schedule, runAsUser).Scan(
		&j.ID, &j.ServerID, &j.Name, &j.Command, &j.Schedule, &j.RunAsUser,
		&j.Enabled, &j.LastRun, &j.LastStatus, &j.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert cron job: %w", err)
	}
	return &j, nil
}

// Update updates a cron job on the agent and in the database.
func (s *CronService) Update(ctx context.Context, id, command, schedule string, enabled bool) error {
	var serverID string
	err := s.db.QueryRow(ctx, `SELECT server_id FROM cron_jobs WHERE id = $1`, id).Scan(&serverID)
	if err != nil {
		return fmt.Errorf("cron job not found: %w", err)
	}

	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return err
	}

	_, err = ac.Put(ctx, "/crons/"+id, map[string]any{
		"command":  command,
		"schedule": schedule,
		"enabled":  enabled,
	})
	if err != nil {
		return fmt.Errorf("agent update cron job: %w", err)
	}

	_, err = s.db.Exec(ctx, `
		UPDATE cron_jobs SET command = $1, schedule = $2, enabled = $3, updated_at = NOW()
		WHERE id = $4
	`, command, schedule, enabled, id)
	if err != nil {
		return fmt.Errorf("update cron job: %w", err)
	}
	return nil
}

// Delete removes a cron job from the agent and the database.
func (s *CronService) Delete(ctx context.Context, id string) error {
	var serverID string
	err := s.db.QueryRow(ctx, `SELECT server_id FROM cron_jobs WHERE id = $1`, id).Scan(&serverID)
	if err != nil {
		return fmt.Errorf("cron job not found: %w", err)
	}

	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return err
	}

	if err := ac.Delete(ctx, "/crons/"+id); err != nil {
		return fmt.Errorf("agent delete cron job: %w", err)
	}

	_, err = s.db.Exec(ctx, `DELETE FROM cron_jobs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete cron job from db: %w", err)
	}
	return nil
}
