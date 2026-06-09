package services

import (
	"context"
	"fmt"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WebsiteService manages virtual-host websites on the remote agent.
type WebsiteService struct {
	db *pgxpool.Pool
}

// NewWebsiteService creates a new WebsiteService.
func NewWebsiteService(db *pgxpool.Pool) *WebsiteService {
	return &WebsiteService{db: db}
}

func (s *WebsiteService) agentFor(ctx context.Context, serverID string) (*agent.AgentClient, error) {
	var agentURL, agentToken string
	err := s.db.QueryRow(ctx, `SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID).
		Scan(&agentURL, &agentToken)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	return agent.NewAgentClient(agentURL, agentToken), nil
}

// List returns all websites. If serverID is empty all websites are returned.
func (s *WebsiteService) List(ctx context.Context, serverID string) ([]models.Website, error) {
	var query string
	var args []any

	if serverID == "" {
		query = `
			SELECT id, server_id, domain, aliases, php_version, document_root,
			       index_file, ssl_enabled, ssl_force_https, ssl_cert_id,
			       enabled, notes, created_at
			FROM websites ORDER BY created_at DESC
		`
	} else {
		query = `
			SELECT id, server_id, domain, aliases, php_version, document_root,
			       index_file, ssl_enabled, ssl_force_https, ssl_cert_id,
			       enabled, notes, created_at
			FROM websites WHERE server_id = $1 ORDER BY created_at DESC
		`
		args = []any{serverID}
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query websites: %w", err)
	}
	defer rows.Close()
	return scanWebsites(rows)
}

func scanWebsites(rows pgx.Rows) ([]models.Website, error) {
	var websites []models.Website
	for rows.Next() {
		var w models.Website
		var certID *string
		if err := rows.Scan(
			&w.ID, &w.ServerID, &w.Domain, &w.Aliases, &w.PHPVersion, &w.DocumentRoot,
			&w.IndexFile, &w.SSLEnabled, &w.SSLForceHTTPS, &certID,
			&w.Enabled, &w.Notes, &w.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan website: %w", err)
		}
		w.SSLCertID = certID
		websites = append(websites, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if websites == nil {
		websites = []models.Website{}
	}
	return websites, nil
}

// Get returns a single website by ID.
func (s *WebsiteService) Get(ctx context.Context, id string) (*models.Website, error) {
	var w models.Website
	var certID *string
	err := s.db.QueryRow(ctx, `
		SELECT id, server_id, domain, aliases, php_version, document_root,
		       index_file, ssl_enabled, ssl_force_https, ssl_cert_id,
		       enabled, notes, created_at
		FROM websites WHERE id = $1
	`, id).Scan(
		&w.ID, &w.ServerID, &w.Domain, &w.Aliases, &w.PHPVersion, &w.DocumentRoot,
		&w.IndexFile, &w.SSLEnabled, &w.SSLForceHTTPS, &certID,
		&w.Enabled, &w.Notes, &w.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get website %s: %w", id, err)
	}
	w.SSLCertID = certID
	return &w, nil
}

type nginxVhostPayload struct {
	Domain           string   `json:"domain"`
	Aliases          []string `json:"aliases"`
	PHPVersion       string   `json:"php_version"`
	DocumentRoot     string   `json:"document_root"`
	IndexFile        string   `json:"index_file"`
	SSLEnabled       bool     `json:"ssl_enabled"`
	SSLForceHTTPS    bool     `json:"ssl_force_https"`
	SSLCertPath      string   `json:"ssl_cert_path"`
	SSLKeyPath       string   `json:"ssl_key_path"`
	CustomDirectives string   `json:"custom_directives"`
}

// Create creates a new website on the agent and stores it in the database.
func (s *WebsiteService) Create(ctx context.Context, serverID, domain, phpVersion, docRoot string, aliases []string) (*models.Website, error) {
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return nil, err
	}

	if aliases == nil {
		aliases = []string{}
	}

	payload := nginxVhostPayload{
		Domain:       domain,
		Aliases:      aliases,
		PHPVersion:   phpVersion,
		DocumentRoot: docRoot,
		IndexFile:    "index.php index.html",
	}

	_, err = ac.Post(ctx, "/nginx/vhosts", payload)
	if err != nil {
		return nil, fmt.Errorf("agent create vhost: %w", err)
	}

	var w models.Website
	var certID *string
	err = s.db.QueryRow(ctx, `
		INSERT INTO websites (server_id, domain, aliases, php_version, document_root, index_file)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, server_id, domain, aliases, php_version, document_root,
		          index_file, ssl_enabled, ssl_force_https, ssl_cert_id,
		          enabled, notes, created_at
	`, serverID, domain, aliases, phpVersion, docRoot, payload.IndexFile).Scan(
		&w.ID, &w.ServerID, &w.Domain, &w.Aliases, &w.PHPVersion, &w.DocumentRoot,
		&w.IndexFile, &w.SSLEnabled, &w.SSLForceHTTPS, &certID,
		&w.Enabled, &w.Notes, &w.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert website: %w", err)
	}
	w.SSLCertID = certID
	return &w, nil
}

// Update updates a website on the agent and in the database.
func (s *WebsiteService) Update(ctx context.Context, id string, updates map[string]any) (*models.Website, error) {
	w, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	ac, err := s.agentFor(ctx, w.ServerID)
	if err != nil {
		return nil, err
	}

	// Apply updates to local model
	if v, ok := updates["php_version"].(string); ok {
		w.PHPVersion = v
	}
	if v, ok := updates["document_root"].(string); ok {
		w.DocumentRoot = v
	}
	if v, ok := updates["ssl_enabled"].(bool); ok {
		w.SSLEnabled = v
	}
	if v, ok := updates["ssl_force_https"].(bool); ok {
		w.SSLForceHTTPS = v
	}
	customDirectives := ""
	if v, ok := updates["custom_directives"].(string); ok {
		customDirectives = v
	}

	payload := nginxVhostPayload{
		Domain:           w.Domain,
		Aliases:          w.Aliases,
		PHPVersion:       w.PHPVersion,
		DocumentRoot:     w.DocumentRoot,
		IndexFile:        w.IndexFile,
		SSLEnabled:       w.SSLEnabled,
		SSLForceHTTPS:    w.SSLForceHTTPS,
		CustomDirectives: customDirectives,
	}

	_, err = ac.Put(ctx, "/nginx/vhosts/"+w.Domain, payload)
	if err != nil {
		return nil, fmt.Errorf("agent update vhost: %w", err)
	}

	var certID *string
	err = s.db.QueryRow(ctx, `
		UPDATE websites SET
			php_version       = $1,
			document_root     = $2,
			ssl_enabled       = $3,
			ssl_force_https   = $4,
			custom_directives = $5,
			updated_at        = NOW()
		WHERE id = $6
		RETURNING id, server_id, domain, aliases, php_version, document_root,
		          index_file, ssl_enabled, ssl_force_https, ssl_cert_id,
		          enabled, notes, created_at
	`, w.PHPVersion, w.DocumentRoot, w.SSLEnabled, w.SSLForceHTTPS, customDirectives, id).Scan(
		&w.ID, &w.ServerID, &w.Domain, &w.Aliases, &w.PHPVersion, &w.DocumentRoot,
		&w.IndexFile, &w.SSLEnabled, &w.SSLForceHTTPS, &certID,
		&w.Enabled, &w.Notes, &w.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update website: %w", err)
	}
	w.SSLCertID = certID
	return w, nil
}

// Delete removes a website from the agent and the database.
func (s *WebsiteService) Delete(ctx context.Context, id string) error {
	w, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	ac, err := s.agentFor(ctx, w.ServerID)
	if err != nil {
		return err
	}

	if err := ac.Delete(ctx, "/nginx/vhosts/"+w.Domain); err != nil {
		return fmt.Errorf("agent delete vhost: %w", err)
	}

	_, err = s.db.Exec(ctx, `DELETE FROM websites WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete website from db: %w", err)
	}
	return nil
}

// Toggle enables or disables a website on the agent.
func (s *WebsiteService) Toggle(ctx context.Context, id string, enabled bool) error {
	w, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	ac, err := s.agentFor(ctx, w.ServerID)
	if err != nil {
		return err
	}

	_, err = ac.Post(ctx, "/nginx/vhosts/"+w.Domain+"/toggle", map[string]bool{"enabled": enabled})
	if err != nil {
		return fmt.Errorf("agent toggle vhost: %w", err)
	}

	_, err = s.db.Exec(ctx, `UPDATE websites SET enabled = $1, updated_at = NOW() WHERE id = $2`, enabled, id)
	if err != nil {
		return fmt.Errorf("update website enabled: %w", err)
	}
	return nil
}

// EnableSSL links an SSL certificate to a website and updates the agent.
func (s *WebsiteService) EnableSSL(ctx context.Context, websiteID, certID string) error {
	w, err := s.Get(ctx, websiteID)
	if err != nil {
		return err
	}

	// Fetch cert details for paths
	var certDomain string
	err = s.db.QueryRow(ctx, `SELECT domain FROM ssl_certs WHERE id = $1`, certID).Scan(&certDomain)
	if err != nil {
		return fmt.Errorf("ssl cert not found: %w", err)
	}

	ac, err := s.agentFor(ctx, w.ServerID)
	if err != nil {
		return err
	}

	payload := nginxVhostPayload{
		Domain:        w.Domain,
		Aliases:       w.Aliases,
		PHPVersion:    w.PHPVersion,
		DocumentRoot:  w.DocumentRoot,
		IndexFile:     w.IndexFile,
		SSLEnabled:    true,
		SSLForceHTTPS: true,
		SSLCertPath:   fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", certDomain),
		SSLKeyPath:    fmt.Sprintf("/etc/letsencrypt/live/%s/privkey.pem", certDomain),
	}

	_, err = ac.Put(ctx, "/nginx/vhosts/"+w.Domain, payload)
	if err != nil {
		return fmt.Errorf("agent enable ssl: %w", err)
	}

	_, err = s.db.Exec(ctx, `
		UPDATE websites SET ssl_enabled = TRUE, ssl_force_https = TRUE,
		       ssl_cert_id = $1, updated_at = NOW()
		WHERE id = $2
	`, certID, websiteID)
	if err != nil {
		return fmt.Errorf("update website ssl: %w", err)
	}
	return nil
}

