package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Alert struct {
	Level     string    `json:"level"`
	Category  string    `json:"category"`
	Message   string    `json:"message"`
	Value     string    `json:"value"`
	Threshold string    `json:"threshold"`
	Time      time.Time `json:"time"`
}

type HealthReport struct {
	Healthy  bool    `json:"healthy"`
	Alerts   []Alert `json:"alerts"`
	Score    int     `json:"score"`
	ServerID string  `json:"server_id"`
}

type MonitoringService struct{ store *db.Store }

func NewMonitoringService(pool *pgxpool.Pool) *MonitoringService {
	return &MonitoringService{store: db.NewStore(pool)}
}

func (s *MonitoringService) HealthCheck(ctx context.Context, serverID string) (*HealthReport, error) {
	agentURL, token, err := s.store.GetServerAgentInfo(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	c := agent.NewAgentClient(agentURL, token)
	data, err := c.Get(ctx, "/monitoring/health")
	if err != nil {
		return nil, fmt.Errorf("agent unreachable: %w", err)
	}
	var report HealthReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	report.ServerID = serverID
	return &report, nil
}

func (s *MonitoringService) SetupMailTLS(ctx context.Context, serverID, hostname, certPath, keyPath string) error {
	agentURL, token, err := s.store.GetServerAgentInfo(ctx, serverID)
	if err != nil {
		return fmt.Errorf("server not found: %w", err)
	}
	c := agent.NewAgentClient(agentURL, token)
	body := map[string]string{
		"hostname":  hostname,
		"cert_path": certPath,
		"key_path":  keyPath,
	}
	_, err = c.Post(ctx, "/mail/setup-tls", body)
	if err != nil {
		return fmt.Errorf("agent error: %w", err)
	}
	return nil
}

func (s *MonitoringService) SetupRspamd(ctx context.Context, serverID string) error {
	agentURL, token, err := s.store.GetServerAgentInfo(ctx, serverID)
	if err != nil {
		return fmt.Errorf("server not found: %w", err)
	}
	c := agent.NewAgentClient(agentURL, token)
	_, err = c.Post(ctx, "/mail/setup-rspamd", nil)
	if err != nil {
		return fmt.Errorf("agent error: %w", err)
	}
	return nil
}

func (s *MonitoringService) SetupDKIM(ctx context.Context, serverID, domain string) (map[string]string, error) {
	agentURL, token, err := s.store.GetServerAgentInfo(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	c := agent.NewAgentClient(agentURL, token)
	data, err := c.Post(ctx, "/mail/dkim/"+domain, nil)
	if err != nil {
		return nil, fmt.Errorf("agent error: %w", err)
	}
	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return result, nil
}

func (s *MonitoringService) GetRspamdStatus(ctx context.Context, serverID string) (map[string]any, error) {
	agentURL, token, err := s.store.GetServerAgentInfo(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	c := agent.NewAgentClient(agentURL, token)
	data, err := c.Get(ctx, "/mail/rspamd/status")
	if err != nil {
		return nil, fmt.Errorf("agent error: %w", err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return result, nil
}

// SpamConfig mirrors the agent's SpamConfig struct.
type SpamConfig struct {
	Enabled   bool    `json:"enabled"`
	Reject    float64 `json:"reject"`
	AddHeader float64 `json:"add_header"`
	Greylist  float64 `json:"greylist"`
}

func (s *MonitoringService) GetSpamConfig(ctx context.Context, serverID string) (*SpamConfig, error) {
	agentURL, token, err := s.store.GetServerAgentInfo(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	c := agent.NewAgentClient(agentURL, token)
	data, err := c.Get(ctx, "/mail/spam/config")
	if err != nil {
		return nil, fmt.Errorf("agent error: %w", err)
	}
	var cfg SpamConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &cfg, nil
}

func (s *MonitoringService) SetSpamConfig(ctx context.Context, serverID string, cfg SpamConfig) error {
	agentURL, token, err := s.store.GetServerAgentInfo(ctx, serverID)
	if err != nil {
		return fmt.Errorf("server not found: %w", err)
	}
	c := agent.NewAgentClient(agentURL, token)
	_, err = c.Put(ctx, "/mail/spam/config", cfg)
	if err != nil {
		return fmt.Errorf("agent error: %w", err)
	}
	return nil
}
