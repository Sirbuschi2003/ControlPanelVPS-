package db

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"

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

// WriteAuditLog inserts a structured audit entry. It never blocks the caller on error.
// ipAddress is anonymized (last IPv4 octet zeroed) to minimise PII storage per DSGVO.
func WriteAuditLog(ctx context.Context, pool *pgxpool.Pool, userID, action, resource, remoteAddr string, details map[string]any) {
	anonIP := anonymizeIP(remoteAddr)

	var detailsJSON []byte
	if details != nil {
		detailsJSON, _ = json.Marshal(details)
	}

	// userID is nullable (UUID); use nil when empty so the FK constraint is satisfied.
	var uid any
	if userID != "" {
		uid = userID
	}

	_, _ = pool.Exec(ctx,
		`INSERT INTO audit_log (user_id, action, resource, details, ip_address)
		 VALUES ($1, $2, $3, $4, $5)`,
		uid, action, resource, detailsJSON, anonIP,
	)
}

// anonymizeIP removes the last octet of an IPv4 address (e.g. 192.168.1.42 → 192.168.1.0)
// and strips ports. For IPv6 the last 64 bits are zeroed. Unknown formats are returned as-is.
func anonymizeIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return addr
	}
	if v4 := ip.To4(); v4 != nil {
		v4[3] = 0
		return v4.String()
	}
	// IPv6: zero last 64 bits
	v6 := ip.To16()
	for i := 8; i < 16; i++ {
		v6[i] = 0
	}
	return strings.Replace(v6.String(), "::", ":0::", 1)
}
