package executor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileInfo holds metadata about a file or directory.
type FileInfo struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	IsDir      bool      `json:"is_dir"`
	Mode       string    `json:"mode"`
	ModifiedAt time.Time `json:"modified_at"`
}

// AllowedRoots is the set of path prefixes the file manager is permitted to access.
var AllowedRoots = []string{
	"/var/www",
	"/etc/nginx",
	"/etc/postfix",
	"/var/log",
	"/home",
	"/tmp",
}

// deleteAllowedRoots is a narrower set of roots where deletion is permitted.
var deleteAllowedRoots = []string{
	"/var/www",
	"/home",
	"/tmp",
}

// maxReadSize is the maximum file size the agent will read into memory (1 MB).
const maxReadSize = 1 * 1024 * 1024

// validatePath checks that the path is under an allowed root and contains no
// path traversal sequences.
func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}

	// Clean the path to resolve any . or .. components
	clean := filepath.Clean(path)

	// Reject any remaining traversal (should be eliminated by Clean, but be explicit)
	if strings.Contains(clean, "/../") || strings.HasSuffix(clean, "/..") {
		return fmt.Errorf("path traversal not allowed")
	}

	for _, root := range AllowedRoots {
		if clean == root || strings.HasPrefix(clean, root+"/") {
			return nil
		}
	}

	return fmt.Errorf("path %q is not under an allowed root (%s)", path, strings.Join(AllowedRoots, ", "))
}

// validateDeletePath checks that a path is under the narrower delete-allowed roots.
func validateDeletePath(path string) error {
	clean := filepath.Clean(path)

	for _, root := range deleteAllowedRoots {
		if clean == root || strings.HasPrefix(clean, root+"/") {
			return nil
		}
	}

	return fmt.Errorf("deletion not allowed for path %q: only allowed under /var/www, /home, /tmp", path)
}

// ListDirectory returns metadata for all entries in a directory.
func ListDirectory(path string) ([]FileInfo, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", path, err)
	}

	var files []FileInfo
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}

		fullPath := filepath.Join(path, e.Name())
		files = append(files, FileInfo{
			Name:       e.Name(),
			Path:       fullPath,
			Size:       info.Size(),
			IsDir:      e.IsDir(),
			Mode:       info.Mode().String(),
			ModifiedAt: info.ModTime(),
		})
	}
	return files, nil
}

// ReadFile reads and returns the contents of a file (max 1 MB).
func ReadFile(path string) (string, error) {
	if err := validatePath(path); err != nil {
		return "", err
	}

	fi, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat file %s: %w", path, err)
	}
	if fi.IsDir() {
		return "", fmt.Errorf("%s is a directory", path)
	}
	if fi.Size() > maxReadSize {
		return "", fmt.Errorf("file too large: %d bytes (max %d bytes)", fi.Size(), maxReadSize)
	}

	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening file %s: %w", path, err)
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxReadSize))
	if err != nil {
		return "", fmt.Errorf("reading file %s: %w", path, err)
	}
	return string(data), nil
}

// WriteFile writes content to a file, creating parent directories as needed.
func WriteFile(path, content string) error {
	if err := validatePath(path); err != nil {
		return err
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating parent directory for %s: %w", path, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", path, err)
	}
	return nil
}

// DeletePath removes a file or directory (recursively). Only allowed under
// /var/www, /home, and /tmp.
func DeletePath(path string) error {
	if err := validatePath(path); err != nil {
		return err
	}
	if err := validateDeletePath(path); err != nil {
		return err
	}

	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("deleting %s: %w", path, err)
	}
	return nil
}

// MakeDir creates a directory (and any necessary parents) under an allowed root.
func MakeDir(path string) error {
	if err := validatePath(path); err != nil {
		return err
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", path, err)
	}
	return nil
}
