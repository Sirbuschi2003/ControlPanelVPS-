package executor

import (
	"fmt"
	"strings"
)

// SetupRspamd writes the base Rspamd configuration and enables the service.
func SetupRspamd() error {
	// Worker proxy (milter on port 11332)
	proxyConf := `
# Rspamd milter proxy for Postfix
milter_servers = "127.0.0.1:11332";
`
	if err := writeFileSafe("/etc/rspamd/local.d/worker-proxy.inc", []byte(proxyConf)); err != nil {
		return fmt.Errorf("write proxy config: %w", err)
	}

	// Actions / thresholds
	actionsConf := `
# Spam action thresholds
actions {
  reject = 15;       # Definitive spam → reject
  add_header = 6;    # Probable spam → add X-Spam header
  greylist = 4;      # Possible spam → greylist
}
`
	if err := writeFileSafe("/etc/rspamd/local.d/actions.conf", []byte(actionsConf)); err != nil {
		return fmt.Errorf("write actions config: %w", err)
	}

	// Enable DKIM signing module
	dkimConf := `# DKIM signing enabled
enabled = true;
`
	if err := writeFileSafe("/etc/rspamd/local.d/dkim_signing.conf", []byte(dkimConf)); err != nil {
		return fmt.Errorf("write dkim config: %w", err)
	}

	// Anti-phishing
	phishConf := `
# Phishing detection
openphish_enabled = true;
phishtank_enabled = true;
`
	if err := writeFileSafe("/etc/rspamd/local.d/phishing.conf", []byte(phishConf)); err != nil {
		return fmt.Errorf("write phishing config: %w", err)
	}

	// Enable and start Rspamd
	for _, cmd := range [][]string{
		{"systemctl", "enable", "rspamd"},
		{"systemctl", "restart", "rspamd"},
	} {
		if out, err := runCmdOutput(cmd[0], cmd[1:]...); err != nil {
			return fmt.Errorf("rspamd setup %v: %w\n%s", cmd, err, out)
		}
	}
	return nil
}

// GetRspamdStatus returns current spam filter statistics.
func GetRspamdStatus() (map[string]any, error) {
	out, err := runCmdOutput("rspamc", "stat")
	if err != nil {
		return nil, fmt.Errorf("rspamc stat: %w", err)
	}

	stats := map[string]any{"raw": out}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Messages scanned:") {
			stats["messages_scanned"] = strings.TrimSpace(strings.Split(line, ":")[1])
		}
		if strings.Contains(line, "Messages with action reject:") {
			stats["rejected"] = strings.TrimSpace(strings.Split(line, ":")[1])
		}
		if strings.Contains(line, "Messages with action add header:") {
			stats["tagged"] = strings.TrimSpace(strings.Split(line, ":")[1])
		}
	}
	return stats, nil
}

// TrainSpam teaches Rspamd that a message is spam (via rspamc).
func TrainSpam(emlFile string) error {
	_, err := runCmdOutput("rspamc", "learn_spam", emlFile)
	return err
}

// TrainHam teaches Rspamd that a message is NOT spam.
func TrainHam(emlFile string) error {
	_, err := runCmdOutput("rspamc", "learn_ham", emlFile)
	return err
}
