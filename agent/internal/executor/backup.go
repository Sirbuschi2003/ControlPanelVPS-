package executor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BackupRequest holds parameters for running a backup job.
type BackupRequest struct {
	Name          string            `json:"name"`
	StorageType   string            `json:"storage_type"`
	IncludePaths  []string          `json:"include_paths"`
	StorageConfig map[string]string `json:"storage_config"`
	Encrypt       bool              `json:"encrypt"`
}

// BackupResult contains information about the completed backup.
type BackupResult struct {
	FilePath  string `json:"file_path"`
	SizeBytes int64  `json:"size_bytes"`
}

const backupDir = "/var/backups/cpanel"

// RunBackup creates a tar.gz backup of the specified paths, optionally encrypts it,
// and uploads it to the configured storage backend.
func RunBackup(req BackupRequest) (*BackupResult, error) {
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return nil, fmt.Errorf("creating backup directory: %w", err)
	}

	timestamp := time.Now().UTC().Format("20060102-150405")
	archiveName := fmt.Sprintf("%s-%s.tar.gz", req.Name, timestamp)
	archivePath := filepath.Join(backupDir, archiveName)

	// Build tar command
	tarArgs := []string{"-czf", archivePath}
	tarArgs = append(tarArgs, req.IncludePaths...)

	tarCmd := exec.Command("tar", tarArgs...)
	tarOut, err := tarCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tar failed: %s: %w", string(tarOut), err)
	}

	finalPath := archivePath

	// Encrypt if requested
	if req.Encrypt {
		encryptedPath := archivePath + ".enc"
		backupKey := os.Getenv("BACKUP_KEY")
		if backupKey == "" {
			return nil, fmt.Errorf("BACKUP_KEY environment variable not set")
		}

		encCmd := exec.Command("openssl", "enc",
			"-aes-256-cbc",
			"-pbkdf2",
			"-in", archivePath,
			"-out", encryptedPath,
			"-k", backupKey,
		)
		encOut, err := encCmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("encryption failed: %s: %w", string(encOut), err)
		}

		// Remove the unencrypted archive
		_ = os.Remove(archivePath)
		finalPath = encryptedPath
	}

	// Upload to remote storage
	switch strings.ToLower(req.StorageType) {
	case "s3":
		if err := uploadToS3(finalPath, req.StorageConfig); err != nil {
			return nil, fmt.Errorf("S3 upload failed: %w", err)
		}
	case "sftp":
		if err := uploadToSFTP(finalPath, req.StorageConfig); err != nil {
			return nil, fmt.Errorf("SFTP upload failed: %w", err)
		}
	case "local", "":
		// Already stored locally, nothing to do
	default:
		return nil, fmt.Errorf("unknown storage type: %s", req.StorageType)
	}

	// Get file size
	fi, err := os.Stat(finalPath)
	if err != nil {
		return nil, fmt.Errorf("stat backup file: %w", err)
	}

	return &BackupResult{
		FilePath:  finalPath,
		SizeBytes: fi.Size(),
	}, nil
}

// ListBackups returns a list of backup filenames in the backup directory.
func ListBackups() ([]string, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("reading backup directory: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	return files, nil
}

// DeleteBackup removes a backup file by filename.
func DeleteBackup(filename string) error {
	// Sanitize: only allow the filename, not a path
	if strings.Contains(filename, "/") || strings.Contains(filename, "..") {
		return fmt.Errorf("invalid filename: %s", filename)
	}

	path := filepath.Join(backupDir, filename)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("backup file not found: %s", filename)
		}
		return fmt.Errorf("deleting backup: %w", err)
	}
	return nil
}

// uploadToS3 uploads a file to an S3 bucket using the aws CLI.
func uploadToS3(filePath string, config map[string]string) error {
	bucket := config["bucket"]
	if bucket == "" {
		return fmt.Errorf("S3 bucket not specified in storage_config")
	}

	prefix := config["prefix"]
	key := filepath.Base(filePath)
	if prefix != "" {
		key = strings.TrimSuffix(prefix, "/") + "/" + key
	}

	s3URL := fmt.Sprintf("s3://%s/%s", bucket, key)

	cmdArgs := []string{"s3", "cp", filePath, s3URL}

	// Allow custom endpoint (e.g. for MinIO or Wasabi)
	if endpoint := config["endpoint"]; endpoint != "" {
		cmdArgs = append(cmdArgs, "--endpoint-url", endpoint)
	}

	cmd := exec.Command("aws", cmdArgs...)

	// Set region if provided
	if region := config["region"]; region != "" {
		cmd.Env = append(os.Environ(), "AWS_DEFAULT_REGION="+region)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("aws s3 cp failed: %s: %w", string(out), err)
	}
	return nil
}

// uploadToSFTP uploads a file via SFTP using the sftp CLI client.
func uploadToSFTP(filePath string, config map[string]string) error {
	host := config["host"]
	if host == "" {
		return fmt.Errorf("SFTP host not specified in storage_config")
	}

	user := config["user"]
	if user == "" {
		user = "root"
	}

	port := config["port"]
	if port == "" {
		port = "22"
	}

	remotePath := config["remote_path"]
	if remotePath == "" {
		remotePath = "/backups"
	}

	remoteFile := strings.TrimSuffix(remotePath, "/") + "/" + filepath.Base(filePath)

	// Build sftp batch commands
	batchContent := fmt.Sprintf("put %s %s\nbye\n", filePath, remoteFile)

	// Write batch file to a temp location
	tmpFile, err := os.CreateTemp("", "sftp-batch-*")
	if err != nil {
		return fmt.Errorf("creating sftp batch file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(batchContent); err != nil {
		return fmt.Errorf("writing sftp batch file: %w", err)
	}
	tmpFile.Close()

	cmd := exec.Command("sftp",
		"-P", port,
		"-o", "StrictHostKeyChecking=no",
		"-b", tmpFile.Name(),
		fmt.Sprintf("%s@%s", user, host),
	)

	// If identity key is specified
	if keyPath := config["identity_file"]; keyPath != "" {
		cmd.Args = append(cmd.Args[:len(cmd.Args)-1],
			"-i", keyPath,
			cmd.Args[len(cmd.Args)-1],
		)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sftp upload failed: %s: %w", string(out), err)
	}
	return nil
}
