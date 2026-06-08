package services

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DatabaseService manages databases on remote servers via the agent.
type DatabaseService struct {
	db *pgxpool.Pool
}

// NewDatabaseService creates a new DatabaseService.
func NewDatabaseService(db *pgxpool.Pool) *DatabaseService {
	return &DatabaseService{db: db}
}

func (s *DatabaseService) agentFor(ctx context.Context, serverID string) (*agent.AgentClient, error) {
	var agentURL, agentToken string
	err := s.db.QueryRow(ctx, `SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID).
		Scan(&agentURL, &agentToken)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	return agent.NewAgentClient(agentURL, agentToken), nil
}

// xorKey is a simple XOR key used for basic password obfuscation.
// TODO: use proper AES-GCM encryption with a key from panel_settings.
const xorKey = "ControlPanelVPS-XOR-Key-2024"

func obfuscatePassword(password string) string {
	key := []byte(xorKey)
	data := []byte(password)
	for i := range data {
		data[i] ^= key[i%len(key)]
	}
	return base64.StdEncoding.EncodeToString(data)
}

func deobfuscatePassword(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	key := []byte(xorKey)
	for i := range data {
		data[i] ^= key[i%len(key)]
	}
	return string(data), nil
}

// List returns all managed databases, optionally filtered by serverID.
func (s *DatabaseService) List(ctx context.Context, serverID string) ([]models.ManagedDatabase, error) {
	query := `SELECT id, server_id, name, db_type, db_user, charset, db_collation, size_bytes, notes, created_at
		FROM managed_databases ORDER BY created_at DESC`
	args := []interface{}{}
	if serverID != "" {
		query = `SELECT id, server_id, name, db_type, db_user, charset, db_collation, size_bytes, notes, created_at
			FROM managed_databases WHERE server_id = $1 ORDER BY created_at DESC`
		args = append(args, serverID)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query databases: %w", err)
	}
	defer rows.Close()

	var dbs []models.ManagedDatabase
	for rows.Next() {
		var db models.ManagedDatabase
		if err := rows.Scan(
			&db.ID, &db.ServerID, &db.Name, &db.DBType, &db.DBUser,
			&db.Charset, &db.Collation, &db.SizeBytes, &db.Notes, &db.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan database: %w", err)
		}
		dbs = append(dbs, db)
	}
	if dbs == nil {
		dbs = []models.ManagedDatabase{}
	}
	return dbs, nil
}

type agentCreateDBPayload struct {
	Name       string `json:"name"`
	DBType     string `json:"db_type"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
}

// Create creates a new database on the agent and stores the record in the database.
func (s *DatabaseService) Create(ctx context.Context, serverID, name, dbType, dbUser, dbPassword string) (*models.ManagedDatabase, error) {
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return nil, err
	}

	payload := agentCreateDBPayload{
		Name:       name,
		DBType:     dbType,
		DBUser:     dbUser,
		DBPassword: dbPassword,
	}

	_, err = ac.Post(ctx, "/databases", payload)
	if err != nil {
		return nil, fmt.Errorf("agent create database: %w", err)
	}

	encryptedPw := obfuscatePassword(dbPassword)

	var db models.ManagedDatabase
	err = s.db.QueryRow(ctx, `
		INSERT INTO managed_databases (server_id, name, db_type, db_user, db_password)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, server_id, name, db_type, db_user, charset, db_collation, size_bytes, notes, created_at
	`, serverID, name, dbType, dbUser, encryptedPw).Scan(
		&db.ID, &db.ServerID, &db.Name, &db.DBType, &db.DBUser,
		&db.Charset, &db.Collation, &db.SizeBytes, &db.Notes, &db.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert database: %w", err)
	}
	return &db, nil
}

// Delete removes a database from the agent and the database.
func (s *DatabaseService) Delete(ctx context.Context, id string) error {
	var name, dbType, serverID string
	err := s.db.QueryRow(ctx, `SELECT server_id, name, db_type FROM managed_databases WHERE id = $1`, id).
		Scan(&serverID, &name, &dbType)
	if err != nil {
		return fmt.Errorf("database not found: %w", err)
	}

	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return err
	}

	if err := ac.Delete(ctx, fmt.Sprintf("/databases/%s?db_type=%s", name, dbType)); err != nil {
		return fmt.Errorf("agent delete database: %w", err)
	}

	_, err = s.db.Exec(ctx, `DELETE FROM managed_databases WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete database from db: %w", err)
	}
	return nil
}

// GetPassword returns the decrypted password for a managed database.
func (s *DatabaseService) GetPassword(ctx context.Context, id string) (string, error) {
	var encryptedPw string
	err := s.db.QueryRow(ctx, `SELECT db_password FROM managed_databases WHERE id = $1`, id).
		Scan(&encryptedPw)
	if err != nil {
		return "", fmt.Errorf("database not found: %w", err)
	}
	return deobfuscatePassword(encryptedPw)
}
