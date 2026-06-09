package executor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	postfixVirtualDomains = "/etc/postfix/virtual_mailbox_domains"
	postfixVirtualMaps    = "/etc/postfix/virtual_mailbox_maps"
	postfixAliasMaps      = "/etc/postfix/virtual_alias_maps"
	dovecotUsersFile      = "/etc/dovecot/users"
	mailVhostsBase        = "/var/mail/vhosts"
)

// AddMailDomain adds a domain to the Postfix virtual mailbox domains list.
func AddMailDomain(domain string) error {
	// Append domain to virtual_mailbox_domains if not already present
	if err := appendLineIfMissing(postfixVirtualDomains, domain); err != nil {
		return fmt.Errorf("adding mail domain to domains file: %w", err)
	}

	// Create the mail directory for this domain
	domainDir := filepath.Join(mailVhostsBase, domain)
	if err := os.MkdirAll(domainDir, 0750); err != nil {
		return fmt.Errorf("creating mail directory %s: %w", domainDir, err)
	}

	if err := runPostmap(postfixVirtualDomains); err != nil {
		return err
	}
	return reloadPostfix()
}

// RemoveMailDomain removes a domain and all its accounts from Postfix virtual mailbox config.
func RemoveMailDomain(domain string) error {
	// Remove domain from virtual_mailbox_domains
	if err := removeLineContaining(postfixVirtualDomains, domain); err != nil {
		return fmt.Errorf("removing mail domain from domains file: %w", err)
	}

	// Remove all accounts for this domain from virtual_mailbox_maps
	if err := removeLinesContaining(postfixVirtualMaps, "@"+domain); err != nil {
		return fmt.Errorf("removing accounts from virtual maps: %w", err)
	}

	// Remove all accounts for this domain from dovecot users
	if err := removeLinesContaining(dovecotUsersFile, "@"+domain); err != nil {
		return fmt.Errorf("removing accounts from dovecot users: %w", err)
	}

	// Remove all aliases for this domain
	if err := removeLinesContaining(postfixAliasMaps, "@"+domain); err != nil {
		return fmt.Errorf("removing aliases for domain: %w", err)
	}

	if err := runPostmap(postfixVirtualDomains); err != nil {
		return err
	}
	if err := runPostmap(postfixVirtualMaps); err != nil {
		return err
	}
	if err := runPostmap(postfixAliasMaps); err != nil {
		return err
	}
	return reloadPostfix()
}

// CreateMailAccount creates a new virtual mailbox account.
// hashedPassword should already be in SHA512-CRYPT format (from HashPassword).
func CreateMailAccount(email, hashedPassword string, quotaMB int) error {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid email address: %s", email)
	}
	user := parts[0]
	domain := parts[1]

	// Add to virtual_mailbox_maps: email -> domain/user/
	mapEntry := fmt.Sprintf("%s\t%s/%s/", email, domain, user)
	if err := appendLineIfMissing(postfixVirtualMaps, mapEntry); err != nil {
		return fmt.Errorf("adding to virtual mailbox maps: %w", err)
	}

	// Build dovecot passwd-file entry; quota_mb=0 means unlimited (no quota rule)
	var dovecotEntry string
	if quotaMB <= 0 {
		dovecotEntry = fmt.Sprintf("%s:{SHA512-CRYPT}%s::::::", email, hashedPassword)
	} else {
		dovecotEntry = fmt.Sprintf("%s:{SHA512-CRYPT}%s::::::userdb_quota_rule=*:storage=%dM",
			email, hashedPassword, quotaMB)
	}
	if err := appendLineIfMissing(dovecotUsersFile, dovecotEntry); err != nil {
		return fmt.Errorf("adding to dovecot users: %w", err)
	}

	// Create maildir structure
	mailDir := filepath.Join(mailVhostsBase, domain, user)
	for _, subdir := range []string{"", "cur", "new", "tmp"} {
		dir := filepath.Join(mailDir, subdir)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("creating maildir %s: %w", dir, err)
		}
	}

	if err := runPostmap(postfixVirtualMaps); err != nil {
		return err
	}
	return reloadDovecot()
}

// DeleteMailAccount removes a virtual mailbox account.
func DeleteMailAccount(email string) error {
	// Remove from virtual_mailbox_maps (match the email prefix)
	if err := removeLinesContaining(postfixVirtualMaps, email+"\t"); err != nil {
		return fmt.Errorf("removing from virtual mailbox maps: %w", err)
	}

	// Remove from dovecot users
	if err := removeLinesContaining(dovecotUsersFile, email+":"); err != nil {
		return fmt.Errorf("removing from dovecot users: %w", err)
	}

	if err := runPostmap(postfixVirtualMaps); err != nil {
		return err
	}
	return reloadDovecot()
}

// AddAlias adds a virtual alias mapping in Postfix.
func AddAlias(source, destination string) error {
	entry := fmt.Sprintf("%s\t%s", source, destination)
	if err := appendLineIfMissing(postfixAliasMaps, entry); err != nil {
		return fmt.Errorf("adding alias: %w", err)
	}

	if err := runPostmap(postfixAliasMaps); err != nil {
		return err
	}
	return reloadPostfix()
}

// RemoveAlias removes a virtual alias from Postfix.
func RemoveAlias(source string) error {
	if err := removeLinesContaining(postfixAliasMaps, source+"\t"); err != nil {
		return fmt.Errorf("removing alias: %w", err)
	}

	if err := runPostmap(postfixAliasMaps); err != nil {
		return err
	}
	return reloadPostfix()
}

// HashPassword generates a SHA512-CRYPT password hash using doveadm.
func HashPassword(password string) string {
	cmd := exec.Command("doveadm", "pw", "-s", "SHA512-CRYPT", "-p", password)
	out, err := cmd.Output()
	if err == nil {
		hash := strings.TrimSpace(string(out))
		// doveadm pw output includes the scheme prefix like {SHA512-CRYPT}...
		// Strip the scheme prefix if present since we add it ourselves in CreateMailAccount.
		hash = strings.TrimPrefix(hash, "{SHA512-CRYPT}")
		return hash
	}

	// Fallback: use openssl passwd if doveadm is unavailable
	cmd2 := exec.Command("openssl", "passwd", "-6", password)
	out2, err2 := cmd2.Output()
	if err2 == nil {
		return strings.TrimSpace(string(out2))
	}

	// Last resort: return a placeholder indicating hashing failed.
	// The caller should handle this case.
	return ""
}

// --- Helper functions ---

// appendLineIfMissing appends a line to a file only if it's not already present.
func appendLineIfMissing(filePath, line string) error {
	data, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", filePath, err)
	}

	if strings.Contains(string(data), line) {
		return nil // already exists
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", filePath, err)
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening %s: %w", filePath, err)
	}
	defer f.Close()

	content := line
	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		content = "\n" + content
	}
	_, err = f.WriteString(content + "\n")
	return err
}

// removeLineContaining removes the first line containing the given substring.
func removeLineContaining(filePath, substr string) error {
	return removeLinesContaining(filePath, substr)
}

// removeLinesContaining removes all lines containing the given substring.
func removeLinesContaining(filePath, substr string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading %s: %w", filePath, err)
	}

	var newLines []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, substr) {
			newLines = append(newLines, line)
		}
	}

	return os.WriteFile(filePath, []byte(strings.Join(newLines, "\n")+"\n"), 0644)
}

// runPostmap runs postmap on a Postfix database file to rebuild the lookup table.
func runPostmap(filePath string) error {
	cmd := exec.Command("postmap", filePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("postmap %s failed: %s: %w", filePath, string(out), err)
	}
	return nil
}

// reloadPostfix reloads the Postfix mail server.
func reloadPostfix() error {
	cmd := exec.Command("postfix", "reload")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("postfix reload failed: %s: %w", string(out), err)
	}
	return nil
}

// reloadDovecot restarts Dovecot (restart works whether active or inactive).
func reloadDovecot() error {
	cmd := exec.Command("systemctl", "restart", "dovecot")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dovecot reload failed: %s: %w", string(out), err)
	}
	return nil
}
