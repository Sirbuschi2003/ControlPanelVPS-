package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BackupService manages backup configurations and jobs.
type BackupService struct {
	db *pgxpool.Pool
}

// NewBackupService creates a new BackupService.
func NewBackupService(db *pgxpool.Pool) *BackupService {
	return &BackupService{db: db}
}

func (s *BackupService) agentFor(ctx context.Context, serverID string) (*agent.AgentClient, error) {
	var agentURL, agentToken string
	err := s.db.QueryRow(ctx, `SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID).
		Scan(&agentURL, &agentToken)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	return agent.NewAgentClient(agentURL, agentToken), nil
}

// ListConfigs returns all backup configurations for a server.
func (s *BackupService) ListConfigs(ctx context.Context, serverID string) ([]models.BackupConfig, error) {
	query := `SELECT id, server_id, name, storage_type, schedule, retention_days,
		       include_paths, storage_config, encrypt, enabled, created_at
		FROM backup_configs ORDER BY created_at DESC`
	args := []interface{}{}
	if serverID != "" {
		query = `SELECT id, server_id, name, storage_type, schedule, retention_days,
		       include_paths, storage_config, encrypt, enabled, created_at
		FROM backup_configs WHERE server_id = $1 ORDER BY created_at DESC`
		args = append(args, serverID)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query backup configs: %w", err)
	}
	defer rows.Close()

	var configs []models.BackupConfig
	for rows.Next() {
		var c models.BackupConfig
		var storageConfigJSON []byte
		if err := rows.Scan(
			&c.ID, &c.ServerID, &c.Name, &c.StorageType, &c.Schedule, &c.RetentionDays,
			&c.IncludePaths, &storageConfigJSON, &c.Encrypt, &c.Enabled, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan backup config: %w", err)
		}
		if err := json.Unmarshal(storageConfigJSON, &c.StorageConfig); err != nil {
			c.StorageConfig = map[string]string{}
		}
		configs = append(configs, c)
	}
	if configs == nil {
		configs = []models.BackupConfig{}
	}
	return configs, nil
}

// CreateConfig creates a new backup configuration.
func (s *BackupService) CreateConfig(ctx context.Context, serverID, name, storageType, schedule string, retentionDays int, includePaths []string, storageConfig map[string]string, encrypt bool) (*models.BackupConfig, error) {
	if includePaths == nil {
		includePaths = []string{}
	}
	if storageConfig == nil {
		storageConfig = map[string]string{}
	}

	storageConfigJSON, err := json.Marshal(storageConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal storage config: %w", err)
	}

	var c models.BackupConfig
	var scJSON []byte
	err = s.db.QueryRow(ctx, `
		INSERT INTO backup_configs (server_id, name, storage_type, schedule, retention_days,
		                            include_paths, storage_config, encrypt)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, server_id, name, storage_type, schedule, retention_days,
		          include_paths, storage_config, encrypt, enabled, created_at
	`, serverID, name, storageType, schedule, retentionDays, includePaths, storageConfigJSON, encrypt).Scan(
		&c.ID, &c.ServerID, &c.Name, &c.StorageType, &c.Schedule, &c.RetentionDays,
		&c.IncludePaths, &scJSON, &c.Encrypt, &c.Enabled, &c.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert backup config: %w", err)
	}
	if err := json.Unmarshal(scJSON, &c.StorageConfig); err != nil {
		c.StorageConfig = storageConfig
	}
	return &c, nil
}

// UpdateConfig updates an existing backup configuration.
func (s *BackupService) UpdateConfig(ctx context.Context, id, name, storageType, schedule string, retentionDays int, includePaths []string, storageConfig map[string]string, encrypt bool) error {
	if includePaths == nil {
		includePaths = []string{}
	}
	if storageConfig == nil {
		storageConfig = map[string]string{}
	}

	storageConfigJSON, err := json.Marshal(storageConfig)
	if err != nil {
		return fmt.Errorf("marshal storage config: %w", err)
	}

	_, err = s.db.Exec(ctx, `
		UPDATE backup_configs
		SET name           = $1,
		    storage_type   = $2,
		    schedule       = $3,
		    retention_days = $4,
		    include_paths  = $5,
		    storage_config = $6,
		    encrypt        = $7,
		    updated_at     = NOW()
		WHERE id = $8
	`, name, storageType, schedule, retentionDays, includePaths, storageConfigJSON, encrypt, id)
	if err != nil {
		return fmt.Errorf("update backup config: %w", err)
	}
	return nil
}

// DeleteConfig removes a backup configuration.
func (s *BackupService) ToggleConfig(ctx context.Context, id string, enabled bool) error {
	_, err := s.db.Exec(ctx, `UPDATE backup_configs SET enabled = $1, updated_at = NOW() WHERE id = $2`, enabled, id)
	if err != nil {
		return fmt.Errorf("toggle backup config: %w", err)
	}
	return nil
}

func (s *BackupService) DeleteConfig(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM backup_configs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete backup config: %w", err)
	}
	return nil
}

type agentRunBackupPayload struct {
	ConfigID      string            `json:"config_id"`
	StorageType   string            `json:"storage_type"`
	Schedule      string            `json:"schedule"`
	RetentionDays int               `json:"retention_days"`
	IncludePaths  []string          `json:"include_paths"`
	StorageConfig map[string]string `json:"storage_config"`
	Encrypt       bool              `json:"encrypt"`
}

type agentBackupJobResponse struct {
	Status    string `json:"status"`
	SizeBytes int64  `json:"size_bytes"`
	FilePath  string `json:"file_path"`
	Error     string `json:"error"`
}

// RunBackup triggers a backup job on the agent and records it in the database.
func (s *BackupService) RunBackup(ctx context.Context, configID string) (*models.BackupJob, error) {
	var c models.BackupConfig
	var scJSON []byte
	err := s.db.QueryRow(ctx, `
		SELECT id, server_id, name, storage_type, schedule, retention_days,
		       include_paths, storage_config, encrypt, enabled, created_at
		FROM backup_configs WHERE id = $1
	`, configID).Scan(
		&c.ID, &c.ServerID, &c.Name, &c.StorageType, &c.Schedule, &c.RetentionDays,
		&c.IncludePaths, &scJSON, &c.Encrypt, &c.Enabled, &c.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("backup config not found: %w", err)
	}
	if err := json.Unmarshal(scJSON, &c.StorageConfig); err != nil {
		c.StorageConfig = map[string]string{}
	}

	// Create job record with running status
	var job models.BackupJob
	err = s.db.QueryRow(ctx, `
		INSERT INTO backup_jobs (config_id, status) VALUES ($1, 'running')
		RETURNING id, config_id, status, size_bytes, file_path, error_message, started_at, finished_at
	`, configID).Scan(
		&job.ID, &job.ConfigID, &job.Status, &job.SizeBytes,
		&job.FilePath, &job.ErrorMessage, &job.StartedAt, &job.FinishedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert backup job: %w", err)
	}

	ac, err := s.agentFor(ctx, c.ServerID)
	if err != nil {
		s.failJob(context.Background(), job.ID, err.Error())
		return nil, err
	}

	payload := agentRunBackupPayload{
		ConfigID:      configID,
		StorageType:   c.StorageType,
		Schedule:      c.Schedule,
		RetentionDays: c.RetentionDays,
		IncludePaths:  c.IncludePaths,
		StorageConfig: c.StorageConfig,
		Encrypt:       c.Encrypt,
	}

	respBody, err := ac.Post(ctx, "/backups", payload)
	if err != nil {
		errMsg := err.Error()
		s.failJob(context.Background(), job.ID, errMsg)
		job.Status = "failed"
		job.ErrorMessage = &errMsg
		return &job, fmt.Errorf("agent run backup: %w", err)
	}

	var agentResp agentBackupJobResponse
	status := "completed"
	var sizeBytes int64
	var filePath *string
	var errMsg *string

	if jsonErr := json.Unmarshal(respBody, &agentResp); jsonErr == nil {
		if agentResp.Status != "" {
			status = agentResp.Status
		}
		sizeBytes = agentResp.SizeBytes
		if agentResp.FilePath != "" {
			fp := agentResp.FilePath
			filePath = &fp
		}
		if agentResp.Error != "" {
			em := agentResp.Error
			errMsg = &em
			status = "failed"
		}
	}

	now := time.Now()
	err = s.db.QueryRow(ctx, `
		UPDATE backup_jobs
		SET status = $1, size_bytes = $2, file_path = $3, error_message = $4, finished_at = $5
		WHERE id = $6
		RETURNING id, config_id, status, size_bytes, file_path, error_message, started_at, finished_at
	`, status, sizeBytes, filePath, errMsg, now, job.ID).Scan(
		&job.ID, &job.ConfigID, &job.Status, &job.SizeBytes,
		&job.FilePath, &job.ErrorMessage, &job.StartedAt, &job.FinishedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update backup job: %w", err)
	}
	return &job, nil
}

func (s *BackupService) failJob(ctx context.Context, jobID, errMsg string) {
	now := time.Now()
	_, _ = s.db.Exec(ctx, `
		UPDATE backup_jobs SET status = 'failed', error_message = $1, finished_at = $2 WHERE id = $3
	`, errMsg, now, jobID)
}

// ListJobs returns backup jobs. If configID is empty, all jobs are returned.
func (s *BackupService) ListJobs(ctx context.Context, configID string) ([]models.BackupJob, error) {
	query := `SELECT id, config_id, status, size_bytes, file_path, error_message, started_at, finished_at
		FROM backup_jobs ORDER BY started_at DESC LIMIT 200`
	args := []interface{}{}
	if configID != "" {
		query = `SELECT id, config_id, status, size_bytes, file_path, error_message, started_at, finished_at
			FROM backup_jobs WHERE config_id = $1 ORDER BY started_at DESC LIMIT 200`
		args = append(args, configID)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query backup jobs: %w", err)
	}
	defer rows.Close()

	var jobs []models.BackupJob
	for rows.Next() {
		var j models.BackupJob
		if err := rows.Scan(
			&j.ID, &j.ConfigID, &j.Status, &j.SizeBytes,
			&j.FilePath, &j.ErrorMessage, &j.StartedAt, &j.FinishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan backup job: %w", err)
		}
		jobs = append(jobs, j)
	}
	if jobs == nil {
		jobs = []models.BackupJob{}
	}
	return jobs, nil
}
