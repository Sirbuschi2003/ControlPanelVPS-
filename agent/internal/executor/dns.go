package executor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ZoneConfig holds the metadata for a DNS zone.
type ZoneConfig struct {
	Name       string `json:"name"`
	Nameserver string `json:"nameserver"`
	AdminEmail string `json:"admin_email"`
	Serial     int    `json:"serial"`
}

// RecordRequest holds a DNS record to be added to a zone.
type RecordRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
}

const (
	bindZonesDir     = "/etc/bind/zones"
	namedConfLocal   = "/etc/bind/named.conf.local"
)

// buildZoneFile generates the content for a BIND zone file.
func buildZoneFile(cfg ZoneConfig) string {
	serial := cfg.Serial
	if serial == 0 {
		serial = int(time.Now().Unix())
	}
	ns := cfg.Nameserver
	if ns == "" {
		ns = "ns1." + cfg.Name + "."
	}
	email := cfg.AdminEmail
	if email == "" {
		email = "hostmaster." + cfg.Name + "."
	}
	// Convert email address to BIND format: user@domain.tld -> user.domain.tld.
	email = strings.ReplaceAll(email, "@", ".") + "."
	if !strings.HasSuffix(email, "..") {
		// already has trailing dot from above
	}

	return fmt.Sprintf(`$ORIGIN %s.
$TTL 3600
@	IN	SOA	%s	%s (
		%d	; Serial
		3600	; Refresh
		1800	; Retry
		604800	; Expire
		300	; Minimum TTL
)

@	IN	NS	%s
`, cfg.Name, ns, email, serial, ns)
}

// CreateZone creates a BIND zone file and adds it to named.conf.local.
func CreateZone(cfg ZoneConfig) error {
	if err := os.MkdirAll(bindZonesDir, 0755); err != nil {
		return fmt.Errorf("creating zones dir: %w", err)
	}

	zoneFile := filepath.Join(bindZonesDir, cfg.Name+".zone")
	content := buildZoneFile(cfg)

	if err := os.WriteFile(zoneFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing zone file: %w", err)
	}

	// Add zone entry to named.conf.local
	entry := fmt.Sprintf(`
zone "%s" {
    type master;
    file "%s";
};
`, cfg.Name, zoneFile)

	confData, err := os.ReadFile(namedConfLocal)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading named.conf.local: %w", err)
	}

	// Only add if not already present
	if !strings.Contains(string(confData), fmt.Sprintf(`zone "%s"`, cfg.Name)) {
		f, err := os.OpenFile(namedConfLocal, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("opening named.conf.local: %w", err)
		}
		defer f.Close()
		if _, err := f.WriteString(entry); err != nil {
			return fmt.Errorf("writing to named.conf.local: %w", err)
		}
	}

	return ReloadDNS()
}

// DeleteZone removes the zone file and its entry from named.conf.local.
func DeleteZone(name string) error {
	zoneFile := filepath.Join(bindZonesDir, name+".zone")
	if err := os.Remove(zoneFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing zone file: %w", err)
	}

	// Remove zone entry from named.conf.local
	confData, err := os.ReadFile(namedConfLocal)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading named.conf.local: %w", err)
	}

	// Remove the zone block using a regex
	pattern := fmt.Sprintf(`\s*zone\s+"%s"\s*\{[^}]*\};\s*`, regexp.QuoteMeta(name))
	re := regexp.MustCompile(pattern)
	updated := re.ReplaceAllString(string(confData), "\n")

	if err := os.WriteFile(namedConfLocal, []byte(updated), 0644); err != nil {
		return fmt.Errorf("writing named.conf.local: %w", err)
	}

	return ReloadDNS()
}

// AddRecord appends a DNS record to a zone file and increments the SOA serial.
func AddRecord(zoneName string, rec RecordRequest) error {
	zoneFile := filepath.Join(bindZonesDir, zoneName+".zone")

	data, err := os.ReadFile(zoneFile)
	if err != nil {
		return fmt.Errorf("reading zone file: %w", err)
	}

	content := string(data)
	content, err = incrementSerial(content)
	if err != nil {
		return fmt.Errorf("incrementing serial: %w", err)
	}

	ttl := rec.TTL
	if ttl == 0 {
		ttl = 3600
	}

	var recordLine string
	if rec.Type == "MX" {
		recordLine = fmt.Sprintf("%s\t%d\tIN\t%s\t%d\t%s\n", rec.Name, ttl, rec.Type, rec.Priority, rec.Content)
	} else {
		recordLine = fmt.Sprintf("%s\t%d\tIN\t%s\t%s\n", rec.Name, ttl, rec.Type, rec.Content)
	}

	content += recordLine

	if err := os.WriteFile(zoneFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing zone file: %w", err)
	}

	return ReloadDNS()
}

// DeleteRecord removes a DNS record from a zone file.
// recordID is in the format "{zone}:{name}:{type}" or just the record line prefix.
func DeleteRecord(zoneName, recordID string) error {
	zoneFile := filepath.Join(bindZonesDir, zoneName+".zone")

	data, err := os.ReadFile(zoneFile)
	if err != nil {
		return fmt.Errorf("reading zone file: %w", err)
	}

	// recordID format: "name:type" or "name:type:content"
	parts := strings.SplitN(recordID, ":", 3)
	if len(parts) < 2 {
		return fmt.Errorf("invalid recordID format, expected name:type[:content]")
	}
	recName := parts[0]
	recType := parts[1]
	recContent := ""
	if len(parts) == 3 {
		recContent = parts[2]
	}

	lines := strings.Split(string(data), "\n")
	var newLines []string
	removed := false

	for _, line := range lines {
		// Skip comment lines and empty lines during matching
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			newLines = append(newLines, line)
			continue
		}

		fields := strings.Fields(trimmed)
		// A record line has at least 4 fields: name TTL class type [content...]
		if len(fields) >= 4 {
			// Fields could be: name ttl IN type content
			//              or: name IN type content (no explicit TTL)
			var name, typ, content string
			if fields[2] == "IN" {
				name = fields[0]
				typ = fields[3]
				if len(fields) > 4 {
					content = strings.Join(fields[4:], " ")
				}
			} else if fields[1] == "IN" {
				name = fields[0]
				typ = fields[2]
				if len(fields) > 3 {
					content = strings.Join(fields[3:], " ")
				}
			}

			if name == recName && typ == recType && !removed {
				if recContent == "" || strings.Contains(content, recContent) {
					removed = true
					continue
				}
			}
		}
		newLines = append(newLines, line)
	}

	if !removed {
		return fmt.Errorf("record not found: %s in zone %s", recordID, zoneName)
	}

	newContent, err := incrementSerial(strings.Join(newLines, "\n"))
	if err != nil {
		return fmt.Errorf("incrementing serial: %w", err)
	}

	if err := os.WriteFile(zoneFile, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("writing zone file: %w", err)
	}

	return ReloadDNS()
}

// ReloadDNS reloads the BIND DNS server using rndc.
func ReloadDNS() error {
	cmd := exec.Command("rndc", "reload")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rndc reload failed: %s: %w", string(out), err)
	}
	return nil
}

// incrementSerial finds the SOA serial in a zone file and increments it.
func incrementSerial(zoneContent string) (string, error) {
	// SOA serial is the first number after the opening parenthesis of the SOA record.
	// Pattern: matches the serial number (first standalone integer after the SOA opening paren)
	re := regexp.MustCompile(`(?m)(^\s*\$ORIGIN.*\n(?:.*\n)*?.*SOA.*\(\s*\n\s*)(\d+)(\s*;\s*Serial)`)
	found := false
	result := re.ReplaceAllStringFunc(zoneContent, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 4 {
			return match
		}
		serial, err := strconv.Atoi(submatches[2])
		if err != nil {
			return match
		}
		found = true
		return submatches[1] + strconv.Itoa(serial+1) + submatches[3]
	})

	if !found {
		// Try a simpler pattern: just find the first number followed by "; Serial"
		re2 := regexp.MustCompile(`(\s*)(\d+)(\s*;\s*Serial)`)
		found2 := false
		result = re2.ReplaceAllStringFunc(zoneContent, func(match string) string {
			if found2 {
				return match
			}
			submatches := re2.FindStringSubmatch(match)
			if len(submatches) < 4 {
				return match
			}
			serial, err := strconv.Atoi(submatches[2])
			if err != nil {
				return match
			}
			found2 = true
			return submatches[1] + strconv.Itoa(serial+1) + submatches[3]
		})
		if !found2 {
			return zoneContent, fmt.Errorf("SOA serial not found in zone file")
		}
		return result, nil
	}
	return result, nil
}
