package executor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// PackageInfo holds information about an upgradeable package.
type PackageInfo struct {
	Name           string `json:"name"`
	CurrentVersion string `json:"current_version"`
	NewVersion     string `json:"new_version"`
	Priority       string `json:"priority"`
}

// ListUpdates runs apt-get update and returns a list of upgradeable packages.
func ListUpdates() ([]PackageInfo, error) {
	// Update the package index quietly
	updateCmd := exec.Command("apt-get", "update", "-qq")
	updateOut, err := updateCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("apt-get update failed: %s: %w", string(updateOut), err)
	}

	// List upgradeable packages
	listCmd := exec.Command("apt", "list", "--upgradeable")
	listOut, err := listCmd.CombinedOutput()
	if err != nil {
		// apt list --upgradeable exits 0 on success; a non-zero exit is a real error
		return nil, fmt.Errorf("apt list --upgradeable failed: %s: %w", string(listOut), err)
	}

	return parseAptUpgradeableOutput(string(listOut)), nil
}

// parseAptUpgradeableOutput parses the output of `apt list --upgradeable`.
// Line format example:
//   nginx/jammy-updates 1.18.0-7ubuntu1 amd64 [upgradable from: 1.18.0-6ubuntu1]
func parseAptUpgradeableOutput(output string) []PackageInfo {
	var packages []PackageInfo
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip the "Listing..." header and empty lines
		if line == "" || strings.HasPrefix(line, "Listing...") {
			continue
		}

		// Split on whitespace
		// Format: name/source version arch [upgradable from: old_version]
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		// name/source -> extract name
		namePart := parts[0]
		name := namePart
		if idx := strings.Index(namePart, "/"); idx != -1 {
			name = namePart[:idx]
		}

		newVersion := parts[1]

		// Extract current version from "[upgradable from: X.Y.Z]"
		currentVersion := ""
		joined := strings.Join(parts, " ")
		if idx := strings.Index(joined, "upgradable from: "); idx != -1 {
			rest := joined[idx+len("upgradable from: "):]
			rest = strings.TrimSuffix(rest, "]")
			currentVersion = strings.TrimSpace(rest)
		}

		packages = append(packages, PackageInfo{
			Name:           name,
			CurrentVersion: currentVersion,
			NewVersion:     newVersion,
		})
	}

	return packages
}

// ApplyUpdates upgrades the specified packages (or all upgradeable packages if none specified).
// The operation has a 10-minute timeout.
func ApplyUpdates(packages []string) error {
	var args []string

	env := append([]string{}, "DEBIAN_FRONTEND=noninteractive")

	if len(packages) == 0 {
		// Upgrade all
		args = []string{"apt-get", "upgrade", "-y",
			"-o", "Dpkg::Options::=--force-confnew",
		}
	} else {
		// Upgrade only specified packages
		args = append([]string{"apt-get", "install", "-y",
			"--only-upgrade",
			"-o", "Dpkg::Options::=--force-confnew",
		}, packages...)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = append(os.Environ(), env...)

	// 10-minute timeout
	done := make(chan error, 1)
	go func() {
		out, err := cmd.CombinedOutput()
		if err != nil {
			done <- fmt.Errorf("apt-get failed: %s: %w", string(out), err)
			return
		}
		done <- nil
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(10 * time.Minute):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return fmt.Errorf("apt-get timed out after 10 minutes")
	}
}
