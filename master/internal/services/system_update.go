package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SystemUpdateService struct {
	db    *pgxpool.Pool
	store *db.Store
}

func NewSystemUpdateService(pool *pgxpool.Pool) *SystemUpdateService {
	return &SystemUpdateService{db: pool, store: db.NewStore(pool)}
}

type SystemInfo struct {
	Commit       string `json:"commit"`
	Branch       string `json:"branch"`
	CommitDate   string `json:"commit_date"`
	NodeID       string `json:"node_id"`
	Hostname     string `json:"hostname"`
	OS           string `json:"os"`
}

type UpdateCheckResult struct {
	Available    bool   `json:"available"`
	CurrentCommit string `json:"current_commit"`
	LatestCommit string `json:"latest_commit"`
}

type UpdateRunResult struct {
	PreviousCommit string `json:"previous_commit"`
	NewCommit      string `json:"new_commit"`
	ChangedFiles   int    `json:"changed_files"`
	Output         string `json:"output"`
	Duration       string `json:"duration"`
}

func (s *SystemUpdateService) GetInfo(ctx context.Context, serverID string) (*SystemInfo, error) {
	sid, err := s.resolveServer(ctx, serverID)
	if err != nil {
		return nil, err
	}
	agentURL, agentToken, err := s.store.GetServerAgentInfo(ctx, sid)
	if err != nil {
		return nil, err
	}
	c := agent.NewAgentClient(agentURL, agentToken)
	body, err := c.Get(ctx, "/system/info")
	if err != nil {
		return nil, fmt.Errorf("agent unreachable: %w", err)
	}
	var info SystemInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (s *SystemUpdateService) CheckUpdates(ctx context.Context, serverID string) (*UpdateCheckResult, error) {
	sid, err := s.resolveServer(ctx, serverID)
	if err != nil {
		return nil, err
	}
	agentURL, agentToken, err := s.store.GetServerAgentInfo(ctx, sid)
	if err != nil {
		return nil, err
	}
	c := agent.NewAgentClient(agentURL, agentToken)
	body, err := c.Get(ctx, "/system/check-updates")
	if err != nil {
		return nil, fmt.Errorf("agent unreachable: %w", err)
	}
	var result UpdateCheckResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *SystemUpdateService) RunUpdate(ctx context.Context, serverID string) (*UpdateRunResult, error) {
	sid, err := s.resolveServer(ctx, serverID)
	if err != nil {
		return nil, err
	}
	agentURL, agentToken, err := s.store.GetServerAgentInfo(ctx, sid)
	if err != nil {
		return nil, err
	}
	c := agent.NewAgentClient(agentURL, agentToken)
	// Long timeout — update can take several minutes
	body, err := c.Post(ctx, "/system/update", map[string]string{})
	if err != nil {
		return nil, fmt.Errorf("update failed: %w", err)
	}
	var result UpdateRunResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *SystemUpdateService) resolveServer(ctx context.Context, serverID string) (string, error) {
	if serverID != "" {
		return serverID, nil
	}
	return s.store.GetFirstServerID(ctx)
}
