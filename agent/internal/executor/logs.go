package executor

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LogFiles maps log names to their file paths on the server.
var LogFiles = map[string]string{
	"nginx-access": "/var/log/nginx/access.log",
	"nginx-error":  "/var/log/nginx/error.log",
	"syslog":       "/var/log/syslog",
	"auth":         "/var/log/auth.log",
	"mail":         "/var/log/mail.log",
	"mysql":        "/var/log/mysql/error.log",
	"fail2ban":     "/var/log/fail2ban.log",
	"dpkg":         "/var/log/dpkg.log",
}

// GetLog returns the last N lines of the named log file.
func GetLog(logName string, lines int) ([]string, error) {
	path, ok := LogFiles[logName]
	if !ok {
		return nil, fmt.Errorf("unknown log: %s", logName)
	}

	if lines <= 0 {
		lines = 200
	}

	return readLastNLines(path, lines)
}

// ListLogs returns the names of log files that actually exist on this server.
func ListLogs() []string {
	var available []string
	for name, path := range LogFiles {
		if _, err := os.Stat(path); err == nil {
			available = append(available, name)
		}
	}
	return available
}

// TailLog returns the last N lines of the named log as a single newline-separated string.
func TailLog(logName string, lines int) (string, error) {
	logLines, err := GetLog(logName, lines)
	if err != nil {
		return "", err
	}
	return strings.Join(logLines, "\n"), nil
}

// readLastNLines reads the last n lines from a file using a buffered reverse-scan approach.
// This avoids loading the entire file into memory for large log files.
func readLastNLines(filePath string, n int) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening log file %s: %w", filePath, err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat log file: %w", err)
	}

	size := fi.Size()
	if size == 0 {
		return []string{}, nil
	}

	// Read chunks from the end of the file until we have enough lines.
	const chunkSize = 4096
	var buf []byte
	offset := size
	linesFound := 0

	for offset > 0 && linesFound <= n {
		readSize := int64(chunkSize)
		if readSize > offset {
			readSize = offset
		}
		offset -= readSize

		chunk := make([]byte, readSize)
		_, err := f.ReadAt(chunk, offset)
		if err != nil {
			return nil, fmt.Errorf("reading log file: %w", err)
		}

		buf = append(chunk, buf...)

		// Count newlines in what we have so far
		linesFound = strings.Count(string(buf), "\n")
	}

	// Now split the buffer into lines and return the last n
	scanner := bufio.NewScanner(strings.NewReader(string(buf)))
	var allLines []string
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if len(allLines) <= n {
		return allLines, nil
	}
	return allLines[len(allLines)-n:], nil
}
