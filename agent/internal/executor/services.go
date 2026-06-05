package executor

import (
	"fmt"
	"os/exec"
	"strings"
)

// ServiceInfo holds status information for a systemd service.
type ServiceInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Active      string `json:"active"`
	Enabled     string `json:"enabled"`
	LoadState   string `json:"load_state"`
}

// WellKnownServices is the list of services the control panel monitors.
var WellKnownServices = []string{
	"nginx", "apache2", "mysql", "mariadb", "postgresql",
	"redis-server", "postfix", "dovecot", "fail2ban",
	"ufw", "ssh", "cron", "php8.2-fpm", "php8.1-fpm", "php8.3-fpm",
}

// validActions is the set of allowed systemctl actions for security.
var validActions = map[string]bool{
	"start":   true,
	"stop":    true,
	"restart": true,
	"enable":  true,
	"disable": true,
	"reload":  true,
}

// ListServices queries systemctl for the status of all well-known services.
func ListServices() ([]ServiceInfo, error) {
	var services []ServiceInfo

	for _, name := range WellKnownServices {
		info, err := GetServiceStatus(name)
		if err != nil {
			// Service not installed or not found — skip it
			continue
		}
		// Skip services that are not loaded (not installed)
		if info.LoadState == "not-found" || info.LoadState == "masked" {
			continue
		}
		services = append(services, *info)
	}

	return services, nil
}

// ServiceAction performs a systemctl action on a named service.
func ServiceAction(name, action string) error {
	if !validActions[strings.ToLower(action)] {
		return fmt.Errorf("invalid action %q: must be one of start, stop, restart, enable, disable, reload", action)
	}

	cmd := exec.Command("systemctl", action, name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s %s failed: %s: %w", action, name, string(out), err)
	}
	return nil
}

// GetServiceStatus queries systemctl for detailed status of a single service.
func GetServiceStatus(name string) (*ServiceInfo, error) {
	cmd := exec.Command("systemctl", "show", name,
		"--property=ActiveState,UnitFileState,LoadState,Description",
		"--no-pager",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("systemctl show %s failed: %s: %w", name, string(out), err)
	}

	info := &ServiceInfo{Name: name}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		switch key {
		case "ActiveState":
			info.Active = value
		case "UnitFileState":
			info.Enabled = value
		case "LoadState":
			info.LoadState = value
		case "Description":
			info.Description = value
		}
	}

	return info, nil
}
