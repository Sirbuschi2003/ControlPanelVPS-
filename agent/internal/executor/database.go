package executor

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// DBCreateRequest holds parameters for creating a database and associated user.
type DBCreateRequest struct {
	Name     string `json:"name"`
	DBType   string `json:"db_type"`
	DBUser   string `json:"db_user"`
	Password string `json:"db_password"`
}

// systemDatabases are databases that should be excluded from user listings.
var systemDatabases = map[string]bool{
	"information_schema": true,
	"performance_schema": true,
	"mysql":              true,
	"sys":                true,
}

// CreateDatabase creates a MySQL/MariaDB database, user, and grants privileges.
func CreateDatabase(req DBCreateRequest) error {
	query := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s`; "+
			"CREATE USER IF NOT EXISTS `%s`@`localhost` IDENTIFIED BY '%s'; "+
			"GRANT ALL PRIVILEGES ON `%s`.* TO `%s`@`localhost`; "+
			"FLUSH PRIVILEGES;",
		req.Name, req.DBUser, req.Password, req.Name, req.DBUser,
	)

	_, err := ExecMySQL(query)
	if err != nil {
		return fmt.Errorf("creating database: %w", err)
	}
	return nil
}

// DropDatabase drops the database and removes the associated user.
func DropDatabase(name, dbType, dbUser string) error {
	query := fmt.Sprintf(
		"DROP DATABASE IF EXISTS `%s`; "+
			"DROP USER IF EXISTS `%s`@`localhost`; "+
			"FLUSH PRIVILEGES;",
		name, dbUser,
	)

	_, err := ExecMySQL(query)
	if err != nil {
		return fmt.Errorf("dropping database: %w", err)
	}
	return nil
}

// ListDatabases returns all non-system databases.
func ListDatabases(dbType string) ([]string, error) {
	out, err := ExecMySQL("SHOW DATABASES;")
	if err != nil {
		return nil, fmt.Errorf("listing databases: %w", err)
	}

	var dbs []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		db := strings.TrimSpace(line)
		if db == "" || systemDatabases[db] {
			continue
		}
		dbs = append(dbs, db)
	}
	return dbs, nil
}

// GetDatabaseSize returns the total size in bytes of a database.
func GetDatabaseSize(name string) (int64, error) {
	query := fmt.Sprintf(
		"SELECT SUM(data_length + index_length) FROM information_schema.tables WHERE table_schema = '%s';",
		name,
	)
	out, err := ExecMySQL(query)
	if err != nil {
		return 0, fmt.Errorf("getting database size: %w", err)
	}

	sizeStr := strings.TrimSpace(out)
	if sizeStr == "" || sizeStr == "NULL" {
		return 0, nil
	}

	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing database size %q: %w", sizeStr, err)
	}
	return size, nil
}

// ExecMySQL runs a MySQL query using the root credentials and returns stdout.
// It uses the Debian maintenance credentials file when available, falling back
// to a plain root login (socket authentication on Ubuntu/Debian).
func ExecMySQL(query string) (string, error) {
	// Try Debian maintenance credentials first (available on Ubuntu/Debian)
	args := []string{
		"--defaults-extra-file=/etc/mysql/debian.cnf",
		"--batch",
		"--skip-column-names",
		"-e", query,
	}

	cmd := exec.Command("mysql", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Fall back to root with socket authentication (no password)
		args2 := []string{
			"-u", "root",
			"--batch",
			"--skip-column-names",
			"-e", query,
		}
		cmd2 := exec.Command("mysql", args2...)
		out2, err2 := cmd2.CombinedOutput()
		if err2 != nil {
			return "", fmt.Errorf("mysql exec failed: %s: %w", string(out2), err2)
		}
		return string(out2), nil
	}
	return string(out), nil
}
