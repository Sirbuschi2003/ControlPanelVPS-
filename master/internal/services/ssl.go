package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SSLService manages SSL/TLS certificates via the remote agent.
type SSLService struct {
	db *pgxpool.Pool
}

// NewSSLService creates a new SSLService.
func NewSSLService(db *pgxpool.Pool) *SSLService {
	return &SSLService{db: db}
}

func (s *SSLService) agentFor(ctx context.Context, serverID string) (*agent.AgentClient, error) {
	var agentURL, agentToken string
	err := s.db.QueryRow(ctx, `SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID).
		Scan(&agentURL, &agentToken)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	return agent.NewAgentClient(agentURL, agentToken), nil
}

// List returns all SSL certificates for a server.
func (s *SSLService) List(ctx context.Context, serverID string) ([]models.SSLCert, error) {
	query := `SELECT id, server_id, domain, san_domains, status, issuer,
		       issued_at, expires_at, auto_renew, created_at
		FROM ssl_certs ORDER BY created_at DESC`
	args := []interface{}{}
	if serverID != "" {
		query = `SELECT id, server_id, domain, san_domains, status, issuer,
		       issued_at, expires_at, auto_renew, created_at
		FROM ssl_certs WHERE server_id = $1 ORDER BY created_at DESC`
		args = append(args, serverID)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query ssl certs: %w", err)
	}
	defer rows.Close()

	var certs []models.SSLCert
	for rows.Next() {
		var c models.SSLCert
		if err := rows.Scan(
			&c.ID, &c.ServerID, &c.Domain, &c.SANDomains, &c.Status, &c.Issuer,
			&c.IssuedAt, &c.ExpiresAt, &c.AutoRenew, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan ssl cert: %w", err)
		}
		certs = append(certs, c)
	}
	if certs == nil {
		certs = []models.SSLCert{}
	}
	return certs, nil
}

type agentSSLIssuePayload struct {
	Domain     string   `json:"domain"`
	SANDomains []string `json:"san_domains"`
	Email      string   `json:"email"`
}

type agentSSLIssueResponse struct {
	Status    string `json:"status"`
	Issuer    string `json:"issuer"`
	IssuedAt  string `json:"issued_at"`
	ExpiresAt string `json:"expires_at"`
}

// Issue requests a new certificate from the agent and records it in the database.
func (s *SSLService) Issue(ctx context.Context, serverID, domain string, sanDomains []string, email string) (*models.SSLCert, error) {
	if sanDomains == nil {
		sanDomains = []string{}
	}

	// Insert pending record first
	var cert models.SSLCert
	err := s.db.QueryRow(ctx, `
		INSERT INTO ssl_certs (server_id, domain, san_domains, status)
		VALUES ($1, $2, $3, 'pending')
		RETURNING id, server_id, domain, san_domains, status, issuer,
		          issued_at, expires_at, auto_renew, created_at
	`, serverID, domain, sanDomains).Scan(
		&cert.ID, &cert.ServerID, &cert.Domain, &cert.SANDomains, &cert.Status, &cert.Issuer,
		&cert.IssuedAt, &cert.ExpiresAt, &cert.AutoRenew, &cert.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert ssl cert: %w", err)
	}

	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return &cert, err
	}

	payload := agentSSLIssuePayload{
		Domain:     domain,
		SANDomains: sanDomains,
		Email:      email,
	}

	respBody, err := ac.Post(ctx, "/ssl/issue", payload)
	if err != nil {
		// Mark as failed
		_, _ = s.db.Exec(ctx, `UPDATE ssl_certs SET status = 'failed', updated_at = NOW() WHERE id = $1`, cert.ID)
		return nil, fmt.Errorf("agent issue ssl: %w", err)
	}

	var agentResp agentSSLIssueResponse
	if err := json.Unmarshal(respBody, &agentResp); err == nil && agentResp.Status != "" {
		status := agentResp.Status
		if status == "" {
			status = "issued"
		}
		var issuedAt, expiresAt *time.Time
		if agentResp.IssuedAt != "" {
			t, err := time.Parse(time.RFC3339, agentResp.IssuedAt)
			if err == nil {
				issuedAt = &t
			}
		}
		if agentResp.ExpiresAt != "" {
			t, err := time.Parse(time.RFC3339, agentResp.ExpiresAt)
			if err == nil {
				expiresAt = &t
			}
		}
		var issuer *string
		if agentResp.Issuer != "" {
			issuer = &agentResp.Issuer
		}
		now := time.Now()
		if issuedAt == nil {
			issuedAt = &now
		}

		err = s.db.QueryRow(ctx, `
			UPDATE ssl_certs
			SET status = $1, issuer = $2, issued_at = $3, expires_at = $4, updated_at = NOW()
			WHERE id = $5
			RETURNING id, server_id, domain, san_domains, status, issuer,
			          issued_at, expires_at, auto_renew, created_at
		`, status, issuer, issuedAt, expiresAt, cert.ID).Scan(
			&cert.ID, &cert.ServerID, &cert.Domain, &cert.SANDomains, &cert.Status, &cert.Issuer,
			&cert.IssuedAt, &cert.ExpiresAt, &cert.AutoRenew, &cert.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("update ssl cert: %w", err)
		}
	} else {
		// Agent responded but we could not parse details – mark as issued
		now := time.Now()
		issued := "issued"
		err = s.db.QueryRow(ctx, `
			UPDATE ssl_certs
			SET status = $1, issued_at = $2, updated_at = NOW()
			WHERE id = $3
			RETURNING id, server_id, domain, san_domains, status, issuer,
			          issued_at, expires_at, auto_renew, created_at
		`, issued, now, cert.ID).Scan(
			&cert.ID, &cert.ServerID, &cert.Domain, &cert.SANDomains, &cert.Status, &cert.Issuer,
			&cert.IssuedAt, &cert.ExpiresAt, &cert.AutoRenew, &cert.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("update ssl cert: %w", err)
		}
	}

	return &cert, nil
}

// Renew requests renewal of an existing certificate from the agent.
func (s *SSLService) Renew(ctx context.Context, id string) error {
	var domain, serverID string
	err := s.db.QueryRow(ctx, `SELECT server_id, domain FROM ssl_certs WHERE id = $1`, id).
		Scan(&serverID, &domain)
	if err != nil {
		return fmt.Errorf("ssl cert not found: %w", err)
	}

	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return err
	}

	_, err = ac.Post(ctx, "/ssl/renew/"+domain, nil)
	if err != nil {
		return fmt.Errorf("agent renew ssl: %w", err)
	}

	_, err = s.db.Exec(ctx, `UPDATE ssl_certs SET status = 'renewing', updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("update ssl cert status: %w", err)
	}
	return nil
}

// Delete removes an SSL certificate from the agent and the database.
func (s *SSLService) Delete(ctx context.Context, id string) error {
	var domain, serverID string
	err := s.db.QueryRow(ctx, `SELECT server_id, domain FROM ssl_certs WHERE id = $1`, id).
		Scan(&serverID, &domain)
	if err != nil {
		return fmt.Errorf("ssl cert not found: %w", err)
	}

	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return err
	}

	if err := ac.Delete(ctx, "/ssl/certs/"+domain); err != nil {
		return fmt.Errorf("agent delete ssl cert: %w", err)
	}

	_, err = s.db.Exec(ctx, `DELETE FROM ssl_certs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete ssl cert from db: %w", err)
	}
	return nil
}
