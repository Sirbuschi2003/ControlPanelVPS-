package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store wraps a pgxpool.Pool and provides helper queries used by the services layer.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a new Store backed by the given connection pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// GetServerAgentInfo returns the agent URL and token for a server by ID.
func (s *Store) GetServerAgentInfo(ctx context.Context, serverID string) (agentURL, agentToken string, err error) {
	err = s.pool.QueryRow(ctx,
		`SELECT agent_url, agent_token FROM servers WHERE id = $1`,
		serverID,
	).Scan(&agentURL, &agentToken)
	if err != nil {
		return "", "", fmt.Errorf("get server agent info for %s: %w", serverID, err)
	}
	return agentURL, agentToken, nil
}

// GetFirstServerID returns the ID of the oldest server record. Useful for single-server
// setups where a server_id is not explicitly provided.
func (s *Store) GetFirstServerID(ctx context.Context) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx,
		`SELECT id FROM servers ORDER BY created_at ASC LIMIT 1`,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("get first server id: %w", err)
	}
	return id, nil
}

// Pool returns the underlying connection pool for direct queries.
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}
