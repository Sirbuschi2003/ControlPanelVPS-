package services

import (
	"context"
	"fmt"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FTPService struct{ db *pgxpool.Pool }

func NewFTPService(db *pgxpool.Pool) *FTPService { return &FTPService{db: db} }

func (s *FTPService) agentFor(ctx context.Context, serverID string) (*agent.AgentClient, error) {
	var agentURL, agentToken string
	err := s.db.QueryRow(ctx, `SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID).
		Scan(&agentURL, &agentToken)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	return agent.NewAgentClient(agentURL, agentToken), nil
}

func (s *FTPService) List(ctx context.Context, domainID string) ([]models.FTPAccount, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, domain_id, server_id, username, home_dir, enabled, created_at
		FROM ftp_accounts WHERE domain_id = $1 ORDER BY username`, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.FTPAccount
	for rows.Next() {
		var f models.FTPAccount
		if err := rows.Scan(&f.ID, &f.DomainID, &f.ServerID, &f.Username, &f.HomeDir, &f.Enabled, &f.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	if out == nil {
		out = []models.FTPAccount{}
	}
	return out, nil
}

func (s *FTPService) Create(ctx context.Context, domainID, username, password, homeDir string) (*models.FTPAccount, error) {
	var domainName, serverID, docRoot string
	err := s.db.QueryRow(ctx, `SELECT name, server_id, document_root FROM domains WHERE id = $1`, domainID).
		Scan(&domainName, &serverID, &docRoot)
	if err != nil {
		return nil, fmt.Errorf("domain not found: %w", err)
	}
	if homeDir == "" {
		homeDir = docRoot + "/public_html"
	}

	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	_, err = ac.Post(ctx, "/ftp/accounts", map[string]string{
		"username": username,
		"password": password,
		"home_dir": homeDir,
	})
	if err != nil {
		return nil, fmt.Errorf("agent create ftp: %w", err)
	}

	var f models.FTPAccount
	err = s.db.QueryRow(ctx, `
		INSERT INTO ftp_accounts (domain_id, server_id, username, home_dir)
		VALUES ($1, $2, $3, $4)
		RETURNING id, domain_id, server_id, username, home_dir, enabled, created_at`,
		domainID, serverID, username, homeDir,
	).Scan(&f.ID, &f.DomainID, &f.ServerID, &f.Username, &f.HomeDir, &f.Enabled, &f.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert ftp account: %w", err)
	}
	return &f, nil
}

func (s *FTPService) Delete(ctx context.Context, id string) error {
	var username, serverID string
	err := s.db.QueryRow(ctx, `SELECT username, server_id FROM ftp_accounts WHERE id = $1`, id).Scan(&username, &serverID)
	if err != nil {
		return fmt.Errorf("ftp account not found: %w", err)
	}
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return err
	}
	_ = ac.Delete(ctx, "/ftp/accounts/"+username)
	_, err = s.db.Exec(ctx, `DELETE FROM ftp_accounts WHERE id = $1`, id)
	return err
}

func (s *FTPService) UpdatePassword(ctx context.Context, id, password string) error {
	var username, serverID string
	err := s.db.QueryRow(ctx, `SELECT username, server_id FROM ftp_accounts WHERE id = $1`, id).Scan(&username, &serverID)
	if err != nil {
		return fmt.Errorf("ftp account not found: %w", err)
	}
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return err
	}
	_, err = ac.Put(ctx, "/ftp/accounts/"+username+"/password", map[string]string{"password": password})
	return err
}
