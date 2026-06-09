package services

import (
	"context"
	"fmt"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubdomainService struct{ db *pgxpool.Pool }

func NewSubdomainService(db *pgxpool.Pool) *SubdomainService { return &SubdomainService{db: db} }

func (s *SubdomainService) List(ctx context.Context, domainID string) ([]models.Subdomain, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, domain_id, server_id, name, document_root, php_version, enabled, created_at
		FROM subdomains WHERE domain_id = $1 ORDER BY name`, domainID)
	if err != nil {
		return nil, fmt.Errorf("query subdomains: %w", err)
	}
	defer rows.Close()
	var out []models.Subdomain
	for rows.Next() {
		var sub models.Subdomain
		if err := rows.Scan(&sub.ID, &sub.DomainID, &sub.ServerID, &sub.Name, &sub.DocumentRoot, &sub.PHPVersion, &sub.Enabled, &sub.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	if out == nil {
		out = []models.Subdomain{}
	}
	return out, nil
}

func (s *SubdomainService) Create(ctx context.Context, domainID, name, documentRoot, phpVersion string) (*models.Subdomain, error) {
	var parentDomain, serverID string
	err := s.db.QueryRow(ctx, `SELECT name, server_id FROM domains WHERE id = $1`, domainID).Scan(&parentDomain, &serverID)
	if err != nil {
		return nil, fmt.Errorf("domain not found: %w", err)
	}
	if documentRoot == "" {
		documentRoot = fmt.Sprintf("/var/www/%s/subdomains/%s/public_html", parentDomain, name)
	}
	if phpVersion == "" {
		phpVersion = "8.2"
	}

	var agentURL, agentToken string
	_ = s.db.QueryRow(ctx, `SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID).Scan(&agentURL, &agentToken)
	ac := agent.NewAgentClient(agentURL, agentToken)
	_, err = ac.Post(ctx, "/subdomains", map[string]string{
		"name":          name,
		"domain":        parentDomain,
		"document_root": documentRoot,
		"php_version":   phpVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("agent create subdomain: %w", err)
	}

	var sub models.Subdomain
	err = s.db.QueryRow(ctx, `
		INSERT INTO subdomains (domain_id, server_id, name, document_root, php_version)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, domain_id, server_id, name, document_root, php_version, enabled, created_at`,
		domainID, serverID, name, documentRoot, phpVersion,
	).Scan(&sub.ID, &sub.DomainID, &sub.ServerID, &sub.Name, &sub.DocumentRoot, &sub.PHPVersion, &sub.Enabled, &sub.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert subdomain: %w", err)
	}
	return &sub, nil
}

func (s *SubdomainService) Delete(ctx context.Context, id string) error {
	var name, domainID string
	err := s.db.QueryRow(ctx, `SELECT name, domain_id FROM subdomains WHERE id = $1`, id).Scan(&name, &domainID)
	if err != nil {
		return fmt.Errorf("subdomain not found: %w", err)
	}
	var parentDomain, serverID string
	_ = s.db.QueryRow(ctx, `SELECT name, server_id FROM domains WHERE id = $1`, domainID).Scan(&parentDomain, &serverID)
	var agentURL, agentToken string
	_ = s.db.QueryRow(ctx, `SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID).Scan(&agentURL, &agentToken)
	ac := agent.NewAgentClient(agentURL, agentToken)
	_ = ac.Delete(ctx, fmt.Sprintf("/subdomains/%s?domain=%s", name, parentDomain))
	_, err = s.db.Exec(ctx, `DELETE FROM subdomains WHERE id = $1`, id)
	return err
}
