package executor

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

// SSLIssueRequest holds parameters for issuing an SSL certificate via certbot.
type SSLIssueRequest struct {
	Domain     string   `json:"domain"`
	SANDomains []string `json:"san_domains"`
	Email      string   `json:"email"`
}

// SSLCertInfo holds metadata about an installed SSL certificate.
type SSLCertInfo struct {
	Domain    string `json:"domain"`
	CertPath  string `json:"cert_path"`
	KeyPath   string `json:"key_path"`
	ExpiresAt string `json:"expires_at"`
	Issuer    string `json:"issuer"`
}

// IssueSSL requests a new Let's Encrypt certificate via certbot and returns cert info.
func IssueSSL(req SSLIssueRequest) (*SSLCertInfo, error) {
	args := []string{
		"certonly",
		"--nginx",
		"--non-interactive",
		"--agree-tos",
		"-m", req.Email,
		"-d", req.Domain,
	}
	for _, san := range req.SANDomains {
		args = append(args, "-d", san)
	}

	cmd := exec.Command("certbot", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("certbot issue failed: %s: %w", string(out), err)
	}

	certPath := fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", req.Domain)
	keyPath := fmt.Sprintf("/etc/letsencrypt/live/%s/privkey.pem", req.Domain)

	expiry, _ := GetCertExpiry(req.Domain)

	return &SSLCertInfo{
		Domain:    req.Domain,
		CertPath:  certPath,
		KeyPath:   keyPath,
		ExpiresAt: expiry,
	}, nil
}

// RenewSSL renews the certificate for the given domain using certbot.
func RenewSSL(domain string) error {
	cmd := exec.Command("certbot", "renew", "--cert-name", domain, "--non-interactive")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("certbot renew failed: %s: %w", string(out), err)
	}
	return nil
}

// DeleteSSL deletes the certificate for the given domain using certbot.
func DeleteSSL(domain string) error {
	cmd := exec.Command("certbot", "delete", "--cert-name", domain, "--non-interactive")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("certbot delete failed: %s: %w", string(out), err)
	}
	return nil
}

// ListSSL parses `certbot certificates` output and returns installed cert info.
func ListSSL() ([]SSLCertInfo, error) {
	cmd := exec.Command("certbot", "certificates")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("certbot certificates failed: %s: %w", string(out), err)
	}

	var certs []SSLCertInfo
	var current *SSLCertInfo

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "Certificate Name:") {
			if current != nil {
				certs = append(certs, *current)
			}
			name := strings.TrimSpace(strings.TrimPrefix(line, "Certificate Name:"))
			current = &SSLCertInfo{
				Domain:   name,
				CertPath: fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", name),
				KeyPath:  fmt.Sprintf("/etc/letsencrypt/live/%s/privkey.pem", name),
			}
			continue
		}

		if current == nil {
			continue
		}

		if strings.HasPrefix(line, "Expiry Date:") {
			// e.g. "Expiry Date: 2024-09-01 12:00:00+00:00 (VALID: 89 days)"
			parts := strings.Fields(strings.TrimPrefix(line, "Expiry Date:"))
			if len(parts) >= 2 {
				current.ExpiresAt = parts[0] + " " + parts[1]
			}
		} else if strings.HasPrefix(line, "Issuer:") {
			current.Issuer = strings.TrimSpace(strings.TrimPrefix(line, "Issuer:"))
		} else if strings.HasPrefix(line, "Certificate Path:") {
			current.CertPath = strings.TrimSpace(strings.TrimPrefix(line, "Certificate Path:"))
		} else if strings.HasPrefix(line, "Private Key Path:") {
			current.KeyPath = strings.TrimSpace(strings.TrimPrefix(line, "Private Key Path:"))
		}
	}

	if current != nil {
		certs = append(certs, *current)
	}

	return certs, nil
}

// GetCertExpiry extracts the expiry date from a domain's certificate via openssl.
func GetCertExpiry(domain string) (string, error) {
	certPath := fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", domain)
	cmd := exec.Command("openssl", "x509", "-enddate", "-noout", "-in", certPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("openssl failed: %s: %w", string(out), err)
	}

	// Output format: "notAfter=Jun  1 12:00:00 2025 GMT"
	result := strings.TrimSpace(string(out))
	result = strings.TrimPrefix(result, "notAfter=")
	return result, nil
}
