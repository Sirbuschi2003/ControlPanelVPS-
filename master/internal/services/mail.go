package services

import (
	"context"
	"fmt"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// MailService manages mail domains, accounts and aliases via the remote agent.
type MailService struct {
	db *pgxpool.Pool
}

// NewMailService creates a new MailService.
func NewMailService(db *pgxpool.Pool) *MailService {
	return &MailService{db: db}
}

func (s *MailService) agentFor(ctx context.Context, serverID string) (*agent.AgentClient, error) {
	var agentURL, agentToken string
	err := s.db.QueryRow(ctx, `SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID).
		Scan(&agentURL, &agentToken)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	return agent.NewAgentClient(agentURL, agentToken), nil
}

// ---- Mail Domains ----

// ListDomains returns all mail domains for a server.
func (s *MailService) ListDomains(ctx context.Context, serverID string) ([]models.MailDomain, error) {
	query := `SELECT id, server_id, domain, enabled, created_at
		FROM mail_domains ORDER BY created_at DESC`
	args := []interface{}{}
	if serverID != "" {
		query = `SELECT id, server_id, domain, enabled, created_at
		FROM mail_domains WHERE server_id = $1 ORDER BY created_at DESC`
		args = append(args, serverID)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query mail domains: %w", err)
	}
	defer rows.Close()

	var domains []models.MailDomain
	for rows.Next() {
		var d models.MailDomain
		if err := rows.Scan(&d.ID, &d.ServerID, &d.Domain, &d.Enabled, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan mail domain: %w", err)
		}
		domains = append(domains, d)
	}
	if domains == nil {
		domains = []models.MailDomain{}
	}
	return domains, nil
}

// CreateDomain creates a new mail domain on the agent and stores it in the database.
func (s *MailService) CreateDomain(ctx context.Context, serverID, domain string) (*models.MailDomain, error) {
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return nil, err
	}

	_, err = ac.Post(ctx, "/mail/domains", map[string]string{"domain": domain})
	if err != nil {
		return nil, fmt.Errorf("agent create mail domain: %w", err)
	}

	var d models.MailDomain
	err = s.db.QueryRow(ctx, `
		INSERT INTO mail_domains (server_id, domain) VALUES ($1, $2)
		RETURNING id, server_id, domain, enabled, created_at
	`, serverID, domain).Scan(&d.ID, &d.ServerID, &d.Domain, &d.Enabled, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert mail domain: %w", err)
	}
	return &d, nil
}

// DeleteDomain removes a mail domain from the agent and the database.
func (s *MailService) DeleteDomain(ctx context.Context, id string) error {
	var domain, serverID string
	err := s.db.QueryRow(ctx, `SELECT server_id, domain FROM mail_domains WHERE id = $1`, id).
		Scan(&serverID, &domain)
	if err != nil {
		return fmt.Errorf("mail domain not found: %w", err)
	}

	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return err
	}

	if err := ac.Delete(ctx, "/mail/domains/"+domain); err != nil {
		return fmt.Errorf("agent delete mail domain: %w", err)
	}

	_, err = s.db.Exec(ctx, `DELETE FROM mail_domains WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete mail domain from db: %w", err)
	}
	return nil
}

// ---- Mail Accounts ----

// ListAccounts returns all mail accounts for a domain.
func (s *MailService) ListAccounts(ctx context.Context, domainID string) ([]models.MailAccount, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, domain_id, username, quota_mb, enabled, created_at
		FROM mail_accounts WHERE domain_id = $1 ORDER BY created_at DESC
	`, domainID)
	if err != nil {
		return nil, fmt.Errorf("query mail accounts: %w", err)
	}
	defer rows.Close()

	var accounts []models.MailAccount
	for rows.Next() {
		var a models.MailAccount
		if err := rows.Scan(&a.ID, &a.DomainID, &a.Username, &a.QuotaMB, &a.Enabled, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan mail account: %w", err)
		}
		accounts = append(accounts, a)
	}
	if accounts == nil {
		accounts = []models.MailAccount{}
	}
	return accounts, nil
}

// CreateAccount creates a new mail account on the agent and stores it in the database.
func (s *MailService) CreateAccount(ctx context.Context, domainID, username, password string, quotaMB int) (*models.MailAccount, error) {
	var domain, serverID string
	err := s.db.QueryRow(ctx, `SELECT server_id, domain FROM mail_domains WHERE id = $1`, domainID).
		Scan(&serverID, &domain)
	if err != nil {
		return nil, fmt.Errorf("mail domain not found: %w", err)
	}

	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return nil, err
	}

	email := username + "@" + domain
	_, err = ac.Post(ctx, "/mail/accounts", map[string]any{
		"email":    email,
		"password": password,
		"quota_mb": quotaMB,
	})
	if err != nil {
		return nil, fmt.Errorf("agent create mail account: %w", err)
	}

	hashedPw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	var a models.MailAccount
	err = s.db.QueryRow(ctx, `
		INSERT INTO mail_accounts (domain_id, username, password, quota_mb)
		VALUES ($1, $2, $3, $4)
		RETURNING id, domain_id, username, quota_mb, enabled, created_at
	`, domainID, username, string(hashedPw), quotaMB).Scan(
		&a.ID, &a.DomainID, &a.Username, &a.QuotaMB, &a.Enabled, &a.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert mail account: %w", err)
	}
	return &a, nil
}

// DeleteAccount removes a mail account from the agent and the database.
func (s *MailService) DeleteAccount(ctx context.Context, id string) error {
	var username, domainID string
	err := s.db.QueryRow(ctx, `SELECT domain_id, username FROM mail_accounts WHERE id = $1`, id).
		Scan(&domainID, &username)
	if err != nil {
		return fmt.Errorf("mail account not found: %w", err)
	}

	var domain, serverID string
	err = s.db.QueryRow(ctx, `SELECT server_id, domain FROM mail_domains WHERE id = $1`, domainID).
		Scan(&serverID, &domain)
	if err != nil {
		return fmt.Errorf("mail domain not found: %w", err)
	}

	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return err
	}

	email := username + "@" + domain
	if err := ac.Delete(ctx, "/mail/accounts/"+email); err != nil {
		return fmt.Errorf("agent delete mail account: %w", err)
	}

	_, err = s.db.Exec(ctx, `DELETE FROM mail_accounts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete mail account from db: %w", err)
	}
	return nil
}

// UpdateAccount changes password and/or quota of a mail account.
// Pass password="" to keep the current one.
func (s *MailService) UpdateAccount(ctx context.Context, id, password string, quotaMB int) (*models.MailAccount, error) {
	var username, domainID string
	err := s.db.QueryRow(ctx, `SELECT domain_id, username FROM mail_accounts WHERE id = $1`, id).
		Scan(&domainID, &username)
	if err != nil {
		return nil, fmt.Errorf("mail account not found: %w", err)
	}

	var domain, serverID string
	err = s.db.QueryRow(ctx, `SELECT server_id, domain FROM mail_domains WHERE id = $1`, domainID).
		Scan(&serverID, &domain)
	if err != nil {
		return nil, fmt.Errorf("mail domain not found: %w", err)
	}

	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return nil, err
	}

	email := username + "@" + domain
	_, err = ac.Put(ctx, "/mail/accounts/"+email, map[string]any{
		"password": password,
		"quota_mb": quotaMB,
	})
	if err != nil {
		return nil, fmt.Errorf("agent update mail account: %w", err)
	}

	// Update password hash in DB if new password provided
	if password != "" {
		hashedPw, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if hashErr != nil {
			return nil, fmt.Errorf("hash password: %w", hashErr)
		}
		_, err = s.db.Exec(ctx, `UPDATE mail_accounts SET password = $1, quota_mb = $2, updated_at = NOW() WHERE id = $3`,
			string(hashedPw), quotaMB, id)
	} else {
		_, err = s.db.Exec(ctx, `UPDATE mail_accounts SET quota_mb = $1, updated_at = NOW() WHERE id = $2`,
			quotaMB, id)
	}
	if err != nil {
		return nil, fmt.Errorf("update mail account in db: %w", err)
	}

	var a models.MailAccount
	err = s.db.QueryRow(ctx, `
		SELECT id, domain_id, username, quota_mb, enabled, created_at
		FROM mail_accounts WHERE id = $1
	`, id).Scan(&a.ID, &a.DomainID, &a.Username, &a.QuotaMB, &a.Enabled, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("fetch updated account: %w", err)
	}
	return &a, nil
}

// ---- Mail Aliases ----

// ListAliases returns all mail aliases for a domain.
func (s *MailService) ListAliases(ctx context.Context, domainID string) ([]models.MailAlias, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, domain_id, source, destination, created_at
		FROM mail_aliases WHERE domain_id = $1 ORDER BY created_at DESC
	`, domainID)
	if err != nil {
		return nil, fmt.Errorf("query mail aliases: %w", err)
	}
	defer rows.Close()

	var aliases []models.MailAlias
	for rows.Next() {
		var a models.MailAlias
		if err := rows.Scan(&a.ID, &a.DomainID, &a.Source, &a.Destination, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan mail alias: %w", err)
		}
		aliases = append(aliases, a)
	}
	if aliases == nil {
		aliases = []models.MailAlias{}
	}
	return aliases, nil
}

// CreateAlias creates a new mail alias in the database and on the agent.
func (s *MailService) CreateAlias(ctx context.Context, domainID, source, destination string) (*models.MailAlias, error) {
	var domain, serverID string
	err := s.db.QueryRow(ctx, `SELECT server_id, domain FROM mail_domains WHERE id = $1`, domainID).
		Scan(&serverID, &domain)
	if err != nil {
		return nil, fmt.Errorf("mail domain not found: %w", err)
	}

	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return nil, err
	}

	_, err = ac.Post(ctx, "/mail/aliases", map[string]string{
		"source":      source,
		"destination": destination,
		"domain":      domain,
	})
	if err != nil {
		return nil, fmt.Errorf("agent create mail alias: %w", err)
	}

	var a models.MailAlias
	err = s.db.QueryRow(ctx, `
		INSERT INTO mail_aliases (domain_id, source, destination) VALUES ($1, $2, $3)
		RETURNING id, domain_id, source, destination, created_at
	`, domainID, source, destination).Scan(&a.ID, &a.DomainID, &a.Source, &a.Destination, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert mail alias: %w", err)
	}
	return &a, nil
}

// DeleteAlias removes a mail alias from the database.
func (s *MailService) DeleteAlias(ctx context.Context, id string) error {
	var domainID, source string
	err := s.db.QueryRow(ctx, `SELECT domain_id, source FROM mail_aliases WHERE id = $1`, id).
		Scan(&domainID, &source)
	if err != nil {
		return fmt.Errorf("mail alias not found: %w", err)
	}

	var domain, serverID string
	err = s.db.QueryRow(ctx, `SELECT server_id, domain FROM mail_domains WHERE id = $1`, domainID).
		Scan(&serverID, &domain)
	if err != nil {
		return fmt.Errorf("mail domain not found: %w", err)
	}

	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return err
	}

	if err := ac.Delete(ctx, "/mail/aliases/"+source); err != nil {
		return fmt.Errorf("agent delete mail alias: %w", err)
	}

	_, err = s.db.Exec(ctx, `DELETE FROM mail_aliases WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete mail alias from db: %w", err)
	}
	return nil
}
