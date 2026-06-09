package services

import (
	"context"
	"fmt"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DomainService manages the central domain entity and its provisioning.
type DomainService struct {
	db         *pgxpool.Pool
	websiteSvc *WebsiteService
	dnsSvc     *DNSService
	mailSvc    *MailService
}

// NewDomainService creates a new DomainService.
func NewDomainService(db *pgxpool.Pool, websiteSvc *WebsiteService, dnsSvc *DNSService, mailSvc *MailService) *DomainService {
	return &DomainService{db: db, websiteSvc: websiteSvc, dnsSvc: dnsSvc, mailSvc: mailSvc}
}

// AccessibleDomainIDs returns nil for admins (no filter applied), or the list of
// domain IDs accessible to the given non-admin user.
func (s *DomainService) AccessibleDomainIDs(ctx context.Context, userID, role string) ([]string, error) {
	if role == "admin" {
		return nil, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT id FROM domains WHERE owner_user_id = $1
		UNION
		SELECT domain_id FROM domain_users WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query accessible domains: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if ids == nil {
		ids = []string{}
	}
	return ids, nil
}

// List returns domains, filtered to accessible ones for non-admin users.
func (s *DomainService) List(ctx context.Context, userID, role string) ([]models.Domain, error) {
	var rows interface {
		Next() bool
		Scan(dest ...any) error
		Close()
		Err() error
	}
	var err error

	if role == "admin" {
		rows, err = s.db.Query(ctx, `
			SELECT d.id, d.server_id, s.name, s.ip_address,
			       d.name, d.owner_user_id, u.name,
			       d.document_root, d.php_version, d.status,
			       d.website_id, d.dns_zone_id, d.mail_domain_id, d.created_at
			FROM domains d
			JOIN servers s ON s.id = d.server_id
			LEFT JOIN users u ON u.id = d.owner_user_id
			ORDER BY d.created_at DESC
		`)
	} else {
		rows, err = s.db.Query(ctx, `
			SELECT d.id, d.server_id, s.name, s.ip_address,
			       d.name, d.owner_user_id, u.name,
			       d.document_root, d.php_version, d.status,
			       d.website_id, d.dns_zone_id, d.mail_domain_id, d.created_at
			FROM domains d
			JOIN servers s ON s.id = d.server_id
			LEFT JOIN users u ON u.id = d.owner_user_id
			WHERE d.owner_user_id = $1
			   OR d.id IN (SELECT domain_id FROM domain_users WHERE user_id = $1)
			ORDER BY d.created_at DESC
		`, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("query domains: %w", err)
	}
	defer rows.Close()

	return scanDomains(rows)
}

// Get returns a single domain by ID.
func (s *DomainService) Get(ctx context.Context, id string) (*models.Domain, error) {
	var d models.Domain
	err := s.db.QueryRow(ctx, `
		SELECT d.id, d.server_id, s.name, s.ip_address,
		       d.name, d.owner_user_id, u.name,
		       d.document_root, d.php_version, d.status,
		       d.website_id, d.dns_zone_id, d.mail_domain_id, d.created_at
		FROM domains d
		JOIN servers s ON s.id = d.server_id
		LEFT JOIN users u ON u.id = d.owner_user_id
		WHERE d.id = $1
	`, id).Scan(
		&d.ID, &d.ServerID, &d.ServerName, &d.ServerIP,
		&d.Name, &d.OwnerUserID, &d.OwnerName,
		&d.DocumentRoot, &d.PHPVersion, &d.Status,
		&d.WebsiteID, &d.DNSZoneID, &d.MailDomainID, &d.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get domain %s: %w", id, err)
	}
	return &d, nil
}

// Create provisions a new domain. Depending on flags it auto-creates:
// a website (Nginx vhost), a DNS zone with default records, and a mail domain.
func (s *DomainService) Create(ctx context.Context, serverID, name, ownerUserID, phpVersion string, provisionWeb, provisionDNS, provisionMail bool) (*models.Domain, error) {
	docRoot := fmt.Sprintf("/var/www/%s/public_html", name)

	// Insert domain record (status: provisioning)
	var domainID string
	err := s.db.QueryRow(ctx, `
		INSERT INTO domains (server_id, name, owner_user_id, document_root, php_version, status)
		VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, 'provisioning')
		RETURNING id
	`, serverID, name, ownerUserID, docRoot, phpVersion).Scan(&domainID)
	if err != nil {
		return nil, fmt.Errorf("insert domain: %w", err)
	}

	var websiteID, dnsZoneID, mailDomainID *string
	var provErrors []string

	// Provision website (Nginx vhost + document root)
	if provisionWeb {
		website, err := s.websiteSvc.Create(ctx, serverID, name, phpVersion, docRoot, []string{})
		if err != nil {
			provErrors = append(provErrors, fmt.Sprintf("website: %v", err))
		} else {
			// Link website to domain
			_, _ = s.db.Exec(ctx, `UPDATE websites SET domain_id = $1 WHERE id = $2`, domainID, website.ID)
			websiteID = &website.ID
		}
	}

	// Provision DNS zone with default records (A, www, MX, SPF)
	if provisionDNS {
		nameserver := "ns1." + name
		adminEmail := "admin@" + name
		zone, err := s.dnsSvc.CreateZone(ctx, serverID, name, nameserver, adminEmail, "master", "")
		if err != nil {
			provErrors = append(provErrors, fmt.Sprintf("dns: %v", err))
		} else {
			_, _ = s.db.Exec(ctx, `UPDATE dns_zones SET domain_id = $1 WHERE id = $2`, domainID, zone.ID)
			dnsZoneID = &zone.ID
		}
	}

	// Provision mail domain (Postfix/Dovecot)
	if provisionMail {
		mailDomain, err := s.mailSvc.CreateDomain(ctx, serverID, name)
		if err != nil {
			provErrors = append(provErrors, fmt.Sprintf("mail: %v", err))
		} else {
			_, _ = s.db.Exec(ctx, `UPDATE mail_domains SET domain_id = $1 WHERE id = $2`, domainID, mailDomain.ID)
			mdID := mailDomain.ID
			mailDomainID = &mdID
		}
	}

	// Update domain with provisioned resource IDs
	status := "active"
	if len(provErrors) > 0 && websiteID == nil && dnsZoneID == nil && mailDomainID == nil {
		status = "error"
	} else if len(provErrors) > 0 {
		status = "partial"
	}

	_, err = s.db.Exec(ctx, `
		UPDATE domains
		SET website_id = $1, dns_zone_id = $2, mail_domain_id = $3, status = $4, updated_at = NOW()
		WHERE id = $5
	`, websiteID, dnsZoneID, mailDomainID, status, domainID)
	if err != nil {
		return nil, fmt.Errorf("update domain: %w", err)
	}

	// Grant owner access
	if ownerUserID != "" {
		_, _ = s.db.Exec(ctx, `
			INSERT INTO domain_users (domain_id, user_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, domainID, ownerUserID)
	}

	return s.Get(ctx, domainID)
}

// Delete removes a domain and all its provisioned resources.
func (s *DomainService) Delete(ctx context.Context, id string) error {
	d, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// Remove website
	if d.WebsiteID != nil {
		_ = s.websiteSvc.Delete(ctx, *d.WebsiteID)
	}
	// Remove DNS zone
	if d.DNSZoneID != nil {
		_ = s.dnsSvc.DeleteZone(ctx, *d.DNSZoneID)
	}
	// Remove mail domain
	if d.MailDomainID != nil {
		_ = s.mailSvc.DeleteDomain(ctx, *d.MailDomainID)
	}

	// Delete domain — cascades to domain_users
	_, err = s.db.Exec(ctx, `DELETE FROM domains WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete domain: %w", err)
	}
	return nil
}

// AssignUser grants a user access to a domain.
func (s *DomainService) AssignUser(ctx context.Context, domainID, userID string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO domain_users (domain_id, user_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, domainID, userID)
	if err != nil {
		return fmt.Errorf("assign user to domain: %w", err)
	}
	return nil
}

// RemoveUser revokes a user's access to a domain.
func (s *DomainService) RemoveUser(ctx context.Context, domainID, userID string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM domain_users WHERE domain_id = $1 AND user_id = $2`, domainID, userID)
	if err != nil {
		return fmt.Errorf("remove user from domain: %w", err)
	}
	return nil
}

// ListUsers returns all users with access to a domain.
func (s *DomainService) ListUsers(ctx context.Context, domainID string) ([]models.DomainUser, error) {
	rows, err := s.db.Query(ctx, `
		SELECT du.domain_id, du.user_id, u.name, u.email, du.granted_at
		FROM domain_users du
		JOIN users u ON u.id = du.user_id
		WHERE du.domain_id = $1
		ORDER BY du.granted_at DESC
	`, domainID)
	if err != nil {
		return nil, fmt.Errorf("list domain users: %w", err)
	}
	defer rows.Close()

	var users []models.DomainUser
	for rows.Next() {
		var u models.DomainUser
		if err := rows.Scan(&u.DomainID, &u.UserID, &u.UserName, &u.UserEmail, &u.GrantedAt); err != nil {
			return nil, fmt.Errorf("scan domain user: %w", err)
		}
		users = append(users, u)
	}
	if users == nil {
		users = []models.DomainUser{}
	}
	return users, nil
}

// GetResources returns all resources linked to a domain.
func (s *DomainService) GetResources(ctx context.Context, id string) (*models.DomainResources, error) {
	d, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	res := &models.DomainResources{
		Domain:    *d,
		SSLCerts:  []models.SSLCert{},
		Databases: []models.ManagedDatabase{},
		CronJobs:  []models.CronJob{},
	}

	// Website
	if d.WebsiteID != nil {
		w, err := s.websiteSvc.Get(ctx, *d.WebsiteID)
		if err == nil {
			res.Website = w
		}
	}

	// DNS zone
	if d.DNSZoneID != nil {
		z, err := s.dnsSvc.GetZone(ctx, *d.DNSZoneID)
		if err == nil {
			res.DNSZone = z
		}
	}

	// Mail domain
	if d.MailDomainID != nil {
		rows, err := s.db.Query(ctx, `SELECT id, server_id, domain, enabled, created_at FROM mail_domains WHERE id = $1`, *d.MailDomainID)
		if err == nil {
			defer rows.Close()
			if rows.Next() {
				var md models.MailDomain
				if err := rows.Scan(&md.ID, &md.ServerID, &md.Domain, &md.Enabled, &md.CreatedAt); err == nil {
					res.MailDomain = &md
				}
			}
		}
	}

	// SSL certs
	sslRows, err := s.db.Query(ctx, `
		SELECT id, server_id, domain, san_domains, status, issuer, issued_at, expires_at, auto_renew, created_at
		FROM ssl_certs WHERE domain_id = $1 ORDER BY created_at DESC
	`, id)
	if err == nil {
		defer sslRows.Close()
		for sslRows.Next() {
			var c models.SSLCert
			_ = sslRows.Scan(&c.ID, &c.ServerID, &c.Domain, &c.SANDomains, &c.Status, &c.Issuer, &c.IssuedAt, &c.ExpiresAt, &c.AutoRenew, &c.CreatedAt)
			res.SSLCerts = append(res.SSLCerts, c)
		}
	}

	// Databases
	dbRows, err := s.db.Query(ctx, `
		SELECT id, server_id, name, db_type, db_user, charset, collation, size_bytes, notes, created_at
		FROM managed_databases WHERE domain_id = $1 ORDER BY created_at DESC
	`, id)
	if err == nil {
		defer dbRows.Close()
		for dbRows.Next() {
			var db models.ManagedDatabase
			_ = dbRows.Scan(&db.ID, &db.ServerID, &db.Name, &db.DBType, &db.DBUser, &db.Charset, &db.Collation, &db.SizeBytes, &db.Notes, &db.CreatedAt)
			res.Databases = append(res.Databases, db)
		}
	}

	// Cron jobs
	cronRows, err := s.db.Query(ctx, `
		SELECT id, server_id, name, command, schedule, run_as_user, enabled, last_run, last_status, created_at
		FROM cron_jobs WHERE domain_id = $1 ORDER BY created_at DESC
	`, id)
	if err == nil {
		defer cronRows.Close()
		for cronRows.Next() {
			var c models.CronJob
			_ = cronRows.Scan(&c.ID, &c.ServerID, &c.Name, &c.Command, &c.Schedule, &c.RunAsUser, &c.Enabled, &c.LastRun, &c.LastStatus, &c.CreatedAt)
			res.CronJobs = append(res.CronJobs, c)
		}
	}

	return res, nil
}

type domainScanner interface {
	Next() bool
	Scan(dest ...any) error
	Close()
	Err() error
}

func scanDomains(rows domainScanner) ([]models.Domain, error) {
	defer rows.Close()
	var domains []models.Domain
	for rows.Next() {
		var d models.Domain
		if err := rows.Scan(
			&d.ID, &d.ServerID, &d.ServerName, &d.ServerIP,
			&d.Name, &d.OwnerUserID, &d.OwnerName,
			&d.DocumentRoot, &d.PHPVersion, &d.Status,
			&d.WebsiteID, &d.DNSZoneID, &d.MailDomainID, &d.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan domain: %w", err)
		}
		domains = append(domains, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if domains == nil {
		domains = []models.Domain{}
	}
	return domains, nil
}
