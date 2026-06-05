package executor

import (
	"fmt"
	"os/exec"
	"strings"
)

// FirewallRule describes a UFW firewall rule.
type FirewallRule struct {
	Action    string  `json:"action"`
	Direction string  `json:"direction"`
	Protocol  string  `json:"protocol"`
	Source    string  `json:"source"`
	DestPort  *string `json:"dest_port"`
	Comment   *string `json:"comment"`
}

// AddRule adds a UFW firewall rule.
func AddRule(rule FirewallRule) error {
	args := buildUFWArgs(rule, false)
	cmd := exec.Command("ufw", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ufw add rule failed: %s: %w", string(out), err)
	}
	return nil
}

// DeleteRule deletes a UFW firewall rule by its description.
func DeleteRule(rule FirewallRule) error {
	// Build the same argument list as add, but prefix with "delete"
	ruleArgs := buildUFWArgs(rule, false)
	args := append([]string{"--force", "delete"}, ruleArgs...)
	cmd := exec.Command("ufw", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ufw delete rule failed: %s: %w", string(out), err)
	}
	return nil
}

// EnableFirewall enables UFW (non-interactively).
func EnableFirewall() error {
	cmd := exec.Command("ufw", "--force", "enable")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ufw enable failed: %s: %w", string(out), err)
	}
	return nil
}

// DisableFirewall disables UFW.
func DisableFirewall() error {
	cmd := exec.Command("ufw", "disable")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ufw disable failed: %s: %w", string(out), err)
	}
	return nil
}

// ReloadFirewall reloads the UFW rules.
func ReloadFirewall() error {
	cmd := exec.Command("ufw", "reload")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ufw reload failed: %s: %w", string(out), err)
	}
	return nil
}

// GetStatus returns the verbose UFW status output.
func GetStatus() (string, error) {
	cmd := exec.Command("ufw", "status", "verbose")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ufw status failed: %s: %w", string(out), err)
	}
	return string(out), nil
}

// buildUFWArgs constructs the ufw command arguments from a FirewallRule.
//
// UFW syntax examples:
//   ufw allow 80/tcp
//   ufw allow from 1.2.3.4 to any port 80 proto tcp
//   ufw deny in from any to any port 22 proto tcp
//   ufw allow from 192.168.1.0/24
func buildUFWArgs(rule FirewallRule, forDelete bool) []string {
	var args []string

	action := strings.ToLower(rule.Action)
	if action == "" {
		action = "allow"
	}

	direction := strings.ToLower(rule.Direction)

	// Start with action and optional direction
	if direction == "in" || direction == "out" {
		args = append(args, action, direction)
	} else {
		args = append(args, action)
	}

	// Source
	if rule.Source != "" && rule.Source != "any" {
		args = append(args, "from", rule.Source)
	} else {
		args = append(args, "from", "any")
	}

	// Destination port
	if rule.DestPort != nil && *rule.DestPort != "" {
		args = append(args, "to", "any", "port", *rule.DestPort)
	} else {
		args = append(args, "to", "any")
	}

	// Protocol
	proto := strings.ToLower(rule.Protocol)
	if proto == "tcp" || proto == "udp" {
		args = append(args, "proto", proto)
	}

	// Comment (only supported on add, not delete)
	if !forDelete && rule.Comment != nil && *rule.Comment != "" {
		args = append(args, "comment", *rule.Comment)
	}

	return args
}
