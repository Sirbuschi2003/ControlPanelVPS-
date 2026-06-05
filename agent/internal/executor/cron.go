package executor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CronEntry describes a cron job managed by the control panel.
type CronEntry struct {
	ID       string `json:"id"`
	Schedule string `json:"schedule"`
	User     string `json:"user"`
	Command  string `json:"command"`
	Name     string `json:"name"`
}

const cronDir = "/etc/cron.d"

// cronFilePath returns the cron.d file path for a given job ID.
func cronFilePath(id string) string {
	return filepath.Join(cronDir, "cpanel-"+id)
}

// CreateCron writes a new cron job to /etc/cron.d/cpanel-{id}.
func CreateCron(entry CronEntry) error {
	if err := validateCronEntry(entry); err != nil {
		return err
	}

	content := buildCronContent(entry)
	path := cronFilePath(entry.ID)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing cron file %s: %w", path, err)
	}
	return nil
}

// DeleteCron removes the cron job file for the given ID.
func DeleteCron(id string) error {
	path := cronFilePath(id)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("cron job not found: %s", id)
		}
		return fmt.Errorf("removing cron file %s: %w", path, err)
	}
	return nil
}

// UpdateCron overwrites an existing cron job file.
func UpdateCron(entry CronEntry) error {
	if err := validateCronEntry(entry); err != nil {
		return err
	}

	content := buildCronContent(entry)
	path := cronFilePath(entry.ID)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing cron file %s: %w", path, err)
	}
	return nil
}

// ListCrons reads and parses all /etc/cron.d/cpanel-* files.
func ListCrons() ([]CronEntry, error) {
	entries, err := os.ReadDir(cronDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []CronEntry{}, nil
		}
		return nil, fmt.Errorf("reading cron directory: %w", err)
	}

	var crons []CronEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "cpanel-") {
			continue
		}

		id := strings.TrimPrefix(e.Name(), "cpanel-")
		path := filepath.Join(cronDir, e.Name())

		entry, err := parseCronFile(path, id)
		if err != nil {
			// Skip files that can't be parsed
			continue
		}
		crons = append(crons, *entry)
	}
	return crons, nil
}

// buildCronContent generates the cron.d file content for a job entry.
func buildCronContent(entry CronEntry) string {
	// cron.d file format:
	// # Job name comment
	// schedule user command
	// (requires a trailing newline)
	name := entry.Name
	if name == "" {
		name = entry.ID
	}
	user := entry.User
	if user == "" {
		user = "root"
	}
	return fmt.Sprintf("# %s\nSHELL=/bin/sh\nPATH=/usr/local/sbin:/usr/local/bin:/sbin:/bin:/usr/sbin:/usr/bin\n%s %s %s\n",
		name, entry.Schedule, user, entry.Command)
}

// parseCronFile reads a cron.d file and extracts the CronEntry fields.
func parseCronFile(path, id string) (*CronEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	entry := &CronEntry{ID: id}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Comment line — treat as job name
		if strings.HasPrefix(trimmed, "#") {
			if entry.Name == "" {
				entry.Name = strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
			}
			continue
		}

		// Skip SHELL and PATH declarations
		if strings.HasPrefix(trimmed, "SHELL=") || strings.HasPrefix(trimmed, "PATH=") {
			continue
		}

		// Parse the cron schedule line: "minute hour dom month dow user command..."
		// A standard cron.d line has 6 time fields + 1 user field + command
		fields := strings.Fields(trimmed)
		if len(fields) < 7 {
			continue
		}

		// Schedule is the first 5 fields (min hour dom month dow)
		entry.Schedule = strings.Join(fields[:5], " ")
		entry.User = fields[5]
		entry.Command = strings.Join(fields[6:], " ")
		break
	}

	if entry.Schedule == "" {
		return nil, fmt.Errorf("no valid cron line found in %s", path)
	}

	return entry, nil
}

// validateCronEntry checks for required fields.
func validateCronEntry(entry CronEntry) error {
	if entry.ID == "" {
		return fmt.Errorf("cron entry ID is required")
	}
	if strings.Contains(entry.ID, "/") || strings.Contains(entry.ID, "..") {
		return fmt.Errorf("invalid cron ID: %s", entry.ID)
	}
	if entry.Schedule == "" {
		return fmt.Errorf("cron schedule is required")
	}
	if entry.Command == "" {
		return fmt.Errorf("cron command is required")
	}
	return nil
}
