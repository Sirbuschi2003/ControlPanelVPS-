package services

import (
	"context"
	"fmt"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DomainAliasService struct{ db *pgxpool.Pool }

func NewDomainAliasService(db *pgxpool.Pool) *DomainAliasService {
	return &DomainAliasService{db: db}
}

func (s *DomainAliasService) List(ctx context.Context, domainID string) ([]models.DomainAlias, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, domain_id, alias, created_at FROM domain_aliases WHERE domain_id = $1 ORDER BY alias`, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.DomainAlias
	for rows.Next() {
		var a models.DomainAlias
		if err := rows.Scan(&a.ID, &a.DomainID, &a.Alias, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if out == nil {
		out = []models.DomainAlias{}
	}
	return out, nil
}

func (s *DomainAliasService) Create(ctx context.Context, domainID, alias string) (*models.DomainAlias, error) {
	var a models.DomainAlias
	err := s.db.QueryRow(ctx, `
		INSERT INTO domain_aliases (domain_id, alias) VALUES ($1, $2)
		RETURNING id, domain_id, alias, created_at`, domainID, alias,
	).Scan(&a.ID, &a.DomainID, &a.Alias, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert alias: %w", err)
	}
	_ = s.rebuildVhost(ctx, domainID)
	return &a, nil
}

func (s *DomainAliasService) Delete(ctx context.Context, id string) error {
	var domainID string
	_ = s.db.QueryRow(ctx, `SELECT domain_id FROM domain_aliases WHERE id = $1`, id).Scan(&domainID)
	if _, err := s.db.Exec(ctx, `DELETE FROM domain_aliases WHERE id = $1`, id); err != nil {
		return err
	}
	_ = s.rebuildVhost(ctx, domainID)
	return nil
}

// rebuildVhost updates the Nginx vhost with the current alias list.
func (s *DomainAliasService) rebuildVhost(ctx context.Context, domainID string) error {
	rows, err := s.db.Query(ctx, `SELECT alias FROM domain_aliases WHERE domain_id = $1`, domainID)
	if err != nil {
		return err
	}
	var aliases []string
	for rows.Next() {
		var a string
		_ = rows.Scan(&a)
		aliases = append(aliases, a)
	}
	rows.Close()

	var serverID, domain, phpVersion, docRoot, customDirectives string
	var sslEnabled bool
	var sslCertPath, sslKeyPath *string
	err = s.db.QueryRow(ctx, `
		SELECT w.server_id, w.domain, w.php_version, w.document_root, w.ssl_enabled,
		       COALESCE(w.custom_directives, ''),
		       sc.cert_path, sc.key_path
		FROM websites w
		LEFT JOIN ssl_certs sc ON sc.id = w.ssl_cert_id
		WHERE w.domain = (SELECT name FROM domains WHERE id = $1)
		LIMIT 1`, domainID,
	).Scan(&serverID, &domain, &phpVersion, &docRoot, &sslEnabled, &customDirectives, &sslCertPath, &sslKeyPath)
	if err != nil {
		return err
	}

	var agentURL, agentToken string
	_ = s.db.QueryRow(ctx, `SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID).Scan(&agentURL, &agentToken)

	payload := map[string]any{
		"domain":            domain,
		"aliases":           aliases,
		"php_version":       phpVersion,
		"document_root":     docRoot,
		"ssl_enabled":       sslEnabled,
		"custom_directives": customDirectives,
	}
	if sslCertPath != nil {
		payload["ssl_cert_path"] = *sslCertPath
	}
	if sslKeyPath != nil {
		payload["ssl_key_path"] = *sslKeyPath
	}

	ac := agent.NewAgentClient(agentURL, agentToken)
	_, err = ac.Put(ctx, "/nginx/vhosts/"+domain, payload)
	return err
}
