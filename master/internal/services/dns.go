package services

import (
	"context"
	"fmt"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DNSService manages DNS zones and records via the remote agent.
type DNSService struct {
	db *pgxpool.Pool
}

// NewDNSService creates a new DNSService.
func NewDNSService(db *pgxpool.Pool) *DNSService {
	return &DNSService{db: db}
}

func (s *DNSService) agentFor(ctx context.Context, serverID string) (*agent.AgentClient, error) {
	var agentURL, agentToken string
	err := s.db.QueryRow(ctx, `SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID).
		Scan(&agentURL, &agentToken)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	return agent.NewAgentClient(agentURL, agentToken), nil
}

// ListZones returns all DNS zones for a server.
func (s *DNSService) ListZones(ctx context.Context, serverID string) ([]models.DNSZone, error) {
	query := `SELECT id, server_id, name, serial, refresh, retry, expire, minimum,
		       nameserver, admin_email, created_at
		FROM dns_zones ORDER BY created_at DESC`
	args := []interface{}{}
	if serverID != "" {
		query = `SELECT id, server_id, name, serial, refresh, retry, expire, minimum,
		       nameserver, admin_email, created_at
		FROM dns_zones WHERE server_id = $1 ORDER BY created_at DESC`
		args = append(args, serverID)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query dns zones: %w", err)
	}
	defer rows.Close()

	var zones []models.DNSZone
	for rows.Next() {
		var z models.DNSZone
		if err := rows.Scan(
			&z.ID, &z.ServerID, &z.Name, &z.Serial, &z.Refresh, &z.Retry, &z.Expire, &z.Minimum,
			&z.Nameserver, &z.AdminEmail, &z.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan dns zone: %w", err)
		}
		zones = append(zones, z)
	}
	if zones == nil {
		zones = []models.DNSZone{}
	}
	return zones, nil
}

// GetZone returns a single DNS zone with its records.
func (s *DNSService) GetZone(ctx context.Context, id string) (*models.DNSZone, error) {
	var z models.DNSZone
	err := s.db.QueryRow(ctx, `
		SELECT id, server_id, name, serial, refresh, retry, expire, minimum,
		       nameserver, admin_email, created_at
		FROM dns_zones WHERE id = $1
	`, id).Scan(
		&z.ID, &z.ServerID, &z.Name, &z.Serial, &z.Refresh, &z.Retry, &z.Expire, &z.Minimum,
		&z.Nameserver, &z.AdminEmail, &z.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get dns zone %s: %w", id, err)
	}

	records, err := s.recordsByZone(ctx, z.ID)
	if err != nil {
		return nil, err
	}
	z.Records = records
	return &z, nil
}

func (s *DNSService) recordsByZone(ctx context.Context, zoneID string) ([]models.DNSRecord, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, zone_id, name, type, content, ttl, priority, created_at
		FROM dns_records WHERE zone_id = $1 ORDER BY type, name
	`, zoneID)
	if err != nil {
		return nil, fmt.Errorf("query dns records: %w", err)
	}
	defer rows.Close()

	var records []models.DNSRecord
	for rows.Next() {
		var r models.DNSRecord
		if err := rows.Scan(&r.ID, &r.ZoneID, &r.Name, &r.Type, &r.Content, &r.TTL, &r.Priority, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan dns record: %w", err)
		}
		records = append(records, r)
	}
	if records == nil {
		records = []models.DNSRecord{}
	}
	return records, nil
}

type agentDNSZonePayload struct {
	Name       string `json:"name"`
	Nameserver string `json:"nameserver"`
	AdminEmail string `json:"admin_email"`
}

// CreateZone creates a new DNS zone on the agent and stores it in the database.
func (s *DNSService) CreateZone(ctx context.Context, serverID, name, nameserver, adminEmail string) (*models.DNSZone, error) {
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return nil, err
	}

	payload := agentDNSZonePayload{
		Name:       name,
		Nameserver: nameserver,
		AdminEmail: adminEmail,
	}

	_, err = ac.Post(ctx, "/dns/zones", payload)
	if err != nil {
		return nil, fmt.Errorf("agent create dns zone: %w", err)
	}

	var z models.DNSZone
	err = s.db.QueryRow(ctx, `
		INSERT INTO dns_zones (server_id, name, nameserver, admin_email)
		VALUES ($1, $2, $3, $4)
		RETURNING id, server_id, name, serial, refresh, retry, expire, minimum,
		          nameserver, admin_email, created_at
	`, serverID, name, nameserver, adminEmail).Scan(
		&z.ID, &z.ServerID, &z.Name, &z.Serial, &z.Refresh, &z.Retry, &z.Expire, &z.Minimum,
		&z.Nameserver, &z.AdminEmail, &z.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert dns zone: %w", err)
	}
	z.Records = []models.DNSRecord{}
	return &z, nil
}

// DeleteZone removes a DNS zone and its records from the agent and the database.
func (s *DNSService) DeleteZone(ctx context.Context, id string) error {
	var name, serverID string
	err := s.db.QueryRow(ctx, `SELECT server_id, name FROM dns_zones WHERE id = $1`, id).
		Scan(&serverID, &name)
	if err != nil {
		return fmt.Errorf("dns zone not found: %w", err)
	}

	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return err
	}

	if err := ac.Delete(ctx, "/dns/zones/"+name); err != nil {
		return fmt.Errorf("agent delete dns zone: %w", err)
	}

	// Cascade deletes records via FK
	_, err = s.db.Exec(ctx, `DELETE FROM dns_zones WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete dns zone from db: %w", err)
	}
	return nil
}

type agentDNSRecordPayload struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
}

// AddRecord adds a DNS record to the agent and the database.
func (s *DNSService) AddRecord(ctx context.Context, zoneID, name, recType, content string, ttl, priority int) (*models.DNSRecord, error) {
	var zoneName, serverID string
	err := s.db.QueryRow(ctx, `SELECT server_id, name FROM dns_zones WHERE id = $1`, zoneID).
		Scan(&serverID, &zoneName)
	if err != nil {
		return nil, fmt.Errorf("dns zone not found: %w", err)
	}

	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return nil, err
	}

	payload := agentDNSRecordPayload{
		Name:     name,
		Type:     recType,
		Content:  content,
		TTL:      ttl,
		Priority: priority,
	}

	_, err = ac.Post(ctx, "/dns/zones/"+zoneName+"/records", payload)
	if err != nil {
		return nil, fmt.Errorf("agent add dns record: %w", err)
	}

	var r models.DNSRecord
	err = s.db.QueryRow(ctx, `
		INSERT INTO dns_records (zone_id, name, type, content, ttl, priority)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, zone_id, name, type, content, ttl, priority, created_at
	`, zoneID, name, recType, content, ttl, priority).Scan(
		&r.ID, &r.ZoneID, &r.Name, &r.Type, &r.Content, &r.TTL, &r.Priority, &r.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert dns record: %w", err)
	}
	return &r, nil
}

// DeleteRecord removes a DNS record from the agent and the database.
func (s *DNSService) DeleteRecord(ctx context.Context, id string) error {
	var zoneID, name, recType string
	err := s.db.QueryRow(ctx, `SELECT zone_id, name, type FROM dns_records WHERE id = $1`, id).
		Scan(&zoneID, &name, &recType)
	if err != nil {
		return fmt.Errorf("dns record not found: %w", err)
	}

	var zoneName, serverID string
	err = s.db.QueryRow(ctx, `SELECT server_id, name FROM dns_zones WHERE id = $1`, zoneID).
		Scan(&serverID, &zoneName)
	if err != nil {
		return fmt.Errorf("dns zone not found: %w", err)
	}

	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return err
	}

	if err := ac.Delete(ctx, "/dns/records/"+id); err != nil {
		return fmt.Errorf("agent delete dns record: %w", err)
	}

	_, err = s.db.Exec(ctx, `DELETE FROM dns_records WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete dns record from db: %w", err)
	}
	return nil
}
