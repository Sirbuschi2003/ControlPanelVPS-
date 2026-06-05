package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LogService retrieves log data from remote servers via the agent.
type LogService struct {
	db *pgxpool.Pool
}

// NewLogService creates a new LogService.
func NewLogService(db *pgxpool.Pool) *LogService {
	return &LogService{db: db}
}

func (s *LogService) agentFor(ctx context.Context, serverID string) (*agent.AgentClient, error) {
	var agentURL, agentToken string
	err := s.db.QueryRow(ctx, `SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID).
		Scan(&agentURL, &agentToken)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	return agent.NewAgentClient(agentURL, agentToken), nil
}

// GetLog fetches the last `lines` lines of the named log from the agent.
func (s *LogService) GetLog(ctx context.Context, serverID, logName string, lines int) ([]string, error) {
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/logs/%s?lines=%d", logName, lines)
	data, err := ac.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("agent get log %s: %w", logName, err)
	}

	// Agent may return a JSON array of strings or a plain text body.
	var result []string
	if err := json.Unmarshal(data, &result); err == nil {
		return result, nil
	}

	// Fall back to splitting on newlines.
	text := strings.TrimRight(string(data), "\n")
	if text == "" {
		return []string{}, nil
	}
	return strings.Split(text, "\n"), nil
}

// ListAvailable returns the names of logs available on the server.
func (s *LogService) ListAvailable(ctx context.Context, serverID string) ([]string, error) {
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return nil, err
	}

	data, err := ac.Get(ctx, "/logs")
	if err != nil {
		return nil, fmt.Errorf("agent list logs: %w", err)
	}

	var logs []string
	if err := json.Unmarshal(data, &logs); err != nil {
		return nil, fmt.Errorf("parse logs response: %w", err)
	}
	if logs == nil {
		logs = []string{}
	}
	return logs, nil
}
