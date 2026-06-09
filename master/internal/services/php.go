package services

import (
	"context"
	"fmt"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PHPService struct{ db *pgxpool.Pool }

func NewPHPService(db *pgxpool.Pool) *PHPService { return &PHPService{db: db} }

func (s *PHPService) Get(ctx context.Context, domainID string) (*models.PHPSettings, error) {
	var p models.PHPSettings
	err := s.db.QueryRow(ctx, `
		SELECT id, domain_id, memory_limit, max_execution_time, upload_max_filesize,
		       post_max_size, max_input_vars, display_errors, created_at
		FROM php_settings WHERE domain_id = $1`, domainID,
	).Scan(&p.ID, &p.DomainID, &p.MemoryLimit, &p.MaxExecutionTime, &p.UploadMaxFilesize,
		&p.PostMaxSize, &p.MaxInputVars, &p.DisplayErrors, &p.CreatedAt)
	if err != nil {
		// Return defaults if not configured yet
		return &models.PHPSettings{
			DomainID:          domainID,
			MemoryLimit:       256,
			MaxExecutionTime:  60,
			UploadMaxFilesize: 64,
			PostMaxSize:       64,
			MaxInputVars:      1000,
			DisplayErrors:     false,
		}, nil
	}
	return &p, nil
}

func (s *PHPService) Save(ctx context.Context, p models.PHPSettings) (*models.PHPSettings, error) {
	var domainName, serverID, phpVersion string
	err := s.db.QueryRow(ctx, `SELECT name, server_id, php_version FROM domains WHERE id = $1`, p.DomainID).
		Scan(&domainName, &serverID, &phpVersion)
	if err != nil {
		return nil, fmt.Errorf("domain not found: %w", err)
	}

	// Upsert
	err = s.db.QueryRow(ctx, `
		INSERT INTO php_settings (domain_id, memory_limit, max_execution_time, upload_max_filesize, post_max_size, max_input_vars, display_errors)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (domain_id) DO UPDATE SET
			memory_limit = EXCLUDED.memory_limit,
			max_execution_time = EXCLUDED.max_execution_time,
			upload_max_filesize = EXCLUDED.upload_max_filesize,
			post_max_size = EXCLUDED.post_max_size,
			max_input_vars = EXCLUDED.max_input_vars,
			display_errors = EXCLUDED.display_errors,
			updated_at = NOW()
		RETURNING id, domain_id, memory_limit, max_execution_time, upload_max_filesize,
		          post_max_size, max_input_vars, display_errors, created_at`,
		p.DomainID, p.MemoryLimit, p.MaxExecutionTime, p.UploadMaxFilesize,
		p.PostMaxSize, p.MaxInputVars, p.DisplayErrors,
	).Scan(&p.ID, &p.DomainID, &p.MemoryLimit, &p.MaxExecutionTime, &p.UploadMaxFilesize,
		&p.PostMaxSize, &p.MaxInputVars, &p.DisplayErrors, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("upsert php settings: %w", err)
	}

	var agentURL, agentToken string
	_ = s.db.QueryRow(ctx, `SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID).Scan(&agentURL, &agentToken)
	ac := agent.NewAgentClient(agentURL, agentToken)
	_, err = ac.Put(ctx, "/php/settings/"+domainName, map[string]any{
		"php_version":          phpVersion,
		"memory_limit":         p.MemoryLimit,
		"max_execution_time":   p.MaxExecutionTime,
		"upload_max_filesize":  p.UploadMaxFilesize,
		"post_max_size":        p.PostMaxSize,
		"max_input_vars":       p.MaxInputVars,
		"display_errors":       p.DisplayErrors,
	})
	if err != nil {
		return &p, fmt.Errorf("agent update php settings: %w", err)
	}
	return &p, nil
}
