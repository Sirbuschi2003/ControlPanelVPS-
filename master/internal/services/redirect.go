package services

import (
	"context"
	"fmt"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RedirectService struct{ db *pgxpool.Pool }

func NewRedirectService(db *pgxpool.Pool) *RedirectService { return &RedirectService{db: db} }

func (s *RedirectService) List(ctx context.Context, domainID string) ([]models.Redirect, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, domain_id, source_path, target_url, redirect_type, enabled, created_at
		FROM redirects WHERE domain_id = $1 ORDER BY created_at DESC`, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Redirect
	for rows.Next() {
		var r models.Redirect
		if err := rows.Scan(&r.ID, &r.DomainID, &r.SourcePath, &r.TargetURL, &r.RedirectType, &r.Enabled, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if out == nil {
		out = []models.Redirect{}
	}
	return out, nil
}

func (s *RedirectService) Create(ctx context.Context, domainID, sourcePath, targetURL string, redirectType int) (*models.Redirect, error) {
	if redirectType != 301 && redirectType != 302 {
		redirectType = 301
	}
	var r models.Redirect
	err := s.db.QueryRow(ctx, `
		INSERT INTO redirects (domain_id, source_path, target_url, redirect_type)
		VALUES ($1, $2, $3, $4)
		RETURNING id, domain_id, source_path, target_url, redirect_type, enabled, created_at`,
		domainID, sourcePath, targetURL, redirectType,
	).Scan(&r.ID, &r.DomainID, &r.SourcePath, &r.TargetURL, &r.RedirectType, &r.Enabled, &r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert redirect: %w", err)
	}
	_ = s.applyToNginx(ctx, domainID)
	return &r, nil
}

func (s *RedirectService) Delete(ctx context.Context, id string) error {
	var domainID string
	_ = s.db.QueryRow(ctx, `SELECT domain_id FROM redirects WHERE id = $1`, id).Scan(&domainID)
	if _, err := s.db.Exec(ctx, `DELETE FROM redirects WHERE id = $1`, id); err != nil {
		return err
	}
	_ = s.applyToNginx(ctx, domainID)
	return nil
}

// applyToNginx pushes all active redirects into the Nginx vhost config.
func (s *RedirectService) applyToNginx(ctx context.Context, domainID string) error {
	rows, err := s.db.Query(ctx, `
		SELECT source_path, target_url, redirect_type FROM redirects
		WHERE domain_id = $1 AND enabled = true`, domainID)
	if err != nil {
		return err
	}
	type redRule struct {
		SourcePath   string `json:"source_path"`
		TargetURL    string `json:"target_url"`
		RedirectType int    `json:"redirect_type"`
	}
	var rules []redRule
	for rows.Next() {
		var rr redRule
		_ = rows.Scan(&rr.SourcePath, &rr.TargetURL, &rr.RedirectType)
		rules = append(rules, rr)
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
		"php_version":       phpVersion,
		"document_root":     docRoot,
		"ssl_enabled":       sslEnabled,
		"custom_directives": customDirectives,
		"redirects":         rules,
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
