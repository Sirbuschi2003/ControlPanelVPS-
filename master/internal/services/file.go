package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FileService manages files on remote servers via the agent.
type FileService struct {
	db *pgxpool.Pool
}

// NewFileService creates a new FileService.
func NewFileService(db *pgxpool.Pool) *FileService {
	return &FileService{db: db}
}

func (s *FileService) agentFor(ctx context.Context, serverID string) (*agent.AgentClient, error) {
	var agentURL, agentToken string
	err := s.db.QueryRow(ctx, `SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID).
		Scan(&agentURL, &agentToken)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	return agent.NewAgentClient(agentURL, agentToken), nil
}

// List returns directory entries at the given path.
func (s *FileService) List(ctx context.Context, serverID, path string) ([]models.FileEntry, error) {
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return nil, err
	}

	data, err := ac.Get(ctx, "/files?path="+url.QueryEscape(path))
	if err != nil {
		return nil, fmt.Errorf("agent list files: %w", err)
	}

	var entries []models.FileEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse file list response: %w", err)
	}
	if entries == nil {
		entries = []models.FileEntry{}
	}
	return entries, nil
}

// Read returns the content of the file at path.
func (s *FileService) Read(ctx context.Context, serverID, path string) (string, error) {
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return "", err
	}

	data, err := ac.Get(ctx, "/files/content?path="+url.QueryEscape(path))
	if err != nil {
		return "", fmt.Errorf("agent read file: %w", err)
	}

	// Agent may return the content as a JSON string or raw bytes.
	var content string
	if jsonErr := json.Unmarshal(data, &content); jsonErr == nil {
		return content, nil
	}
	return string(data), nil
}

// Write writes content to the file at path on the remote server.
func (s *FileService) Write(ctx context.Context, serverID, path, content string) error {
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return err
	}

	_, err = ac.Post(ctx, "/files/content", map[string]string{
		"path":    path,
		"content": content,
	})
	if err != nil {
		return fmt.Errorf("agent write file: %w", err)
	}
	return nil
}

// Delete removes the file or directory at path.
func (s *FileService) Delete(ctx context.Context, serverID, path string) error {
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return err
	}

	if err := ac.Delete(ctx, "/files?path="+url.QueryEscape(path)); err != nil {
		return fmt.Errorf("agent delete file: %w", err)
	}
	return nil
}

// Mkdir creates a directory at path.
func (s *FileService) Mkdir(ctx context.Context, serverID, path string) error {
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return err
	}

	_, err = ac.Post(ctx, "/files/mkdir", map[string]string{"path": path})
	if err != nil {
		return fmt.Errorf("agent mkdir: %w", err)
	}
	return nil
}
