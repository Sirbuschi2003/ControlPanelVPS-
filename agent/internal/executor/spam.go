package executor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const rspamdAPIBase = "http://127.0.0.1:11334"

// SpamConfig holds the rspamd action thresholds and service state.
type SpamConfig struct {
	Enabled   bool    `json:"enabled"`
	Reject    float64 `json:"reject"`
	AddHeader float64 `json:"add_header"`
	Greylist  float64 `json:"greylist"`
}

// SetupRspamd writes the base Rspamd configuration and enables the service.
func SetupRspamd() error {
	if err := writeFileSafe("/etc/rspamd/local.d/worker-proxy.inc", []byte(`milter_servers = "127.0.0.1:11332";
`)); err != nil {
		return fmt.Errorf("write proxy config: %w", err)
	}

	// local.d files must NOT wrap keys in a section block
	if err := writeFileSafe("/etc/rspamd/local.d/actions.conf", []byte("reject = 15;\nadd_header = 6;\ngreylist = 4;\n")); err != nil {
		return fmt.Errorf("write actions config: %w", err)
	}

	if err := writeFileSafe("/etc/rspamd/local.d/dkim_signing.conf", []byte("enabled = true;\n")); err != nil {
		return fmt.Errorf("write dkim config: %w", err)
	}

	for _, args := range [][]string{{"enable", "rspamd"}, {"restart", "rspamd"}} {
		if out, err := runCmdOutput("systemctl", args...); err != nil {
			return fmt.Errorf("rspamd setup systemctl %v: %w\n%s", args, err, out)
		}
	}
	return nil
}

// GetSpamConfig reads the current rspamd thresholds and service state.
func GetSpamConfig() (*SpamConfig, error) {
	out, _ := runCmdOutput("systemctl", "is-active", "rspamd")
	cfg := &SpamConfig{
		Enabled:   strings.TrimSpace(out) == "active",
		Reject:    15,
		AddHeader: 6,
		Greylist:  4,
	}

	data, err := readFileSafe("/etc/rspamd/local.d/actions.conf")
	if err != nil {
		return cfg, nil
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), ";"))
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		val, err := strconv.ParseFloat(strings.TrimSpace(kv[1]), 64)
		if err != nil {
			continue
		}
		switch strings.TrimSpace(kv[0]) {
		case "reject":
			cfg.Reject = val
		case "add_header":
			cfg.AddHeader = val
		case "greylist":
			cfg.Greylist = val
		}
	}
	return cfg, nil
}

// SetSpamConfig writes rspamd thresholds and enables/disables the service.
func SetSpamConfig(cfg SpamConfig) error {
	content := fmt.Sprintf("reject = %.1f;\nadd_header = %.1f;\ngreylist = %.1f;\n",
		cfg.Reject, cfg.AddHeader, cfg.Greylist)
	if err := writeFileSafe("/etc/rspamd/local.d/actions.conf", []byte(content)); err != nil {
		return fmt.Errorf("write actions config: %w", err)
	}

	if cfg.Enabled {
		if out, err := runCmdOutput("systemctl", "enable", "--now", "rspamd"); err != nil {
			return fmt.Errorf("enable rspamd: %w\n%s", err, out)
		}
		if out, err := runCmdOutput("systemctl", "reload-or-restart", "rspamd"); err != nil {
			return fmt.Errorf("reload rspamd: %w\n%s", err, out)
		}
	} else {
		if out, err := runCmdOutput("systemctl", "disable", "--now", "rspamd"); err != nil {
			return fmt.Errorf("disable rspamd: %w\n%s", err, out)
		}
	}
	return nil
}

// GetRspamdStats returns scan statistics via the rspamd HTTP API.
func GetRspamdStats() (map[string]any, error) {
	resp, err := http.Get(rspamdAPIBase + "/stat")
	if err == nil {
		defer resp.Body.Close()
		body, readErr := io.ReadAll(resp.Body)
		if readErr == nil {
			var stats map[string]any
			if json.Unmarshal(body, &stats) == nil {
				return stats, nil
			}
		}
	}
	// Fallback: rspamc CLI
	out, err := runCmdOutput("rspamc", "stat")
	if err != nil {
		return map[string]any{"error": "rspamd not running"}, nil
	}
	stats := map[string]any{"raw": out}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		val := strings.TrimSpace(parts[1])
		switch {
		case strings.Contains(parts[0], "Messages scanned"):
			stats["messages_scanned"] = val
		case strings.Contains(parts[0], "action reject"):
			stats["rejected"] = val
		case strings.Contains(parts[0], "action add header"):
			stats["tagged"] = val
		case strings.Contains(parts[0], "action greylist"):
			stats["greylisted"] = val
		}
	}
	return stats, nil
}

// GetRspamdStatus is kept for backwards compatibility.
func GetRspamdStatus() (map[string]any, error) {
	return GetRspamdStats()
}
