package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ServerService struct {
	db         *pgxpool.Pool
	httpClient *http.Client
}

func NewServerService(db *pgxpool.Pool) *ServerService {
	return &ServerService{
		db: db,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *ServerService) List(ctx context.Context) ([]models.Server, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, hostname, ip_address, agent_url, role, status, last_seen, created_at
		FROM servers ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query servers: %w", err)
	}
	defer rows.Close()

	var servers []models.Server
	for rows.Next() {
		var srv models.Server
		if err := rows.Scan(&srv.ID, &srv.Name, &srv.Hostname, &srv.IPAddress, &srv.AgentURL, &srv.Role, &srv.Status, &srv.LastSeen, &srv.CreatedAt); err != nil {
			return nil, err
		}
		servers = append(servers, srv)
	}
	if servers == nil {
		servers = []models.Server{}
	}
	return servers, nil
}

func (s *ServerService) Create(ctx context.Context, name, hostname, ip, agentURL, agentToken, role string) (*models.Server, error) {
	var srv models.Server
	err := s.db.QueryRow(ctx, `
		INSERT INTO servers (name, hostname, ip_address, agent_url, agent_token, role)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, hostname, ip_address, agent_url, role, status, last_seen, created_at
	`, name, hostname, ip, agentURL, agentToken, role).Scan(
		&srv.ID, &srv.Name, &srv.Hostname, &srv.IPAddress, &srv.AgentURL,
		&srv.Role, &srv.Status, &srv.LastSeen, &srv.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert server: %w", err)
	}
	return &srv, nil
}

func (s *ServerService) GetMetrics(ctx context.Context, serverID string) (*models.ServerMetrics, error) {
	var agentURL, agentToken string
	err := s.db.QueryRow(ctx, `SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID).
		Scan(&agentURL, &agentToken)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, agentURL+"/metrics", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+agentToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		_, _ = s.db.Exec(ctx, `UPDATE servers SET status = 'offline' WHERE id = $1`, serverID)
		return nil, fmt.Errorf("agent unreachable: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var metrics models.ServerMetrics
	if err := json.Unmarshal(body, &metrics); err != nil {
		return nil, fmt.Errorf("parse metrics: %w", err)
	}
	metrics.ServerID = serverID

	_, _ = s.db.Exec(ctx, `UPDATE servers SET status = 'online', last_seen = NOW() WHERE id = $1`, serverID)
	return &metrics, nil
}
