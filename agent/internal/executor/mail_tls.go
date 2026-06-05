package executor

// ConfigureMailTLS configures Postfix and Dovecot for TLS/SSL.
// Must be called after AddMailDomain and before first use.
// Requires a valid TLS certificate for the hostname.
func ConfigureMailTLS(hostname, certPath, keyPath string) error {
	if err := configurePostfixTLS(hostname, certPath, keyPath); err != nil {
		return err
	}
	return configureDovecotTLS(certPath, keyPath)
}

func configurePostfixTLS(hostname, certPath, keyPath string) error {
	settings := map[string]string{
		// Hostname
		"myhostname": hostname,

		// Incoming TLS (from other mail servers and clients)
		"smtpd_tls_cert_file":              certPath,
		"smtpd_tls_key_file":               keyPath,
		"smtpd_tls_security_level":         "may",     // offer TLS, don't require (other servers may not support it)
		"smtpd_tls_mandatory_protocols":    "!SSLv2,!SSLv3,!TLSv1,!TLSv1.1",
		"smtpd_tls_protocols":              "!SSLv2,!SSLv3,!TLSv1,!TLSv1.1",
		"smtpd_tls_mandatory_ciphers":      "medium",
		"smtpd_tls_loglevel":               "1",
		"smtpd_tls_received_header":        "yes",
		"smtpd_tls_session_cache_database": "btree:${data_directory}/smtpd_scache",

		// Outgoing TLS (to other mail servers)
		"smtp_tls_cert_file":              certPath,
		"smtp_tls_key_file":               keyPath,
		"smtp_tls_security_level":         "may",
		"smtp_tls_mandatory_protocols":    "!SSLv2,!SSLv3,!TLSv1,!TLSv1.1",
		"smtp_tls_protocols":              "!SSLv2,!SSLv3,!TLSv1,!TLSv1.1",
		"smtp_tls_loglevel":               "1",
		"smtp_tls_session_cache_database": "btree:${data_directory}/smtp_scache",

		// Submission port 587 authentication
		"smtpd_sasl_auth_enable":          "yes",
		"smtpd_sasl_type":                 "dovecot",
		"smtpd_sasl_path":                 "private/auth",
		"smtpd_sasl_security_options":     "noanonymous",
		"smtpd_sasl_local_domain":         "$myhostname",
		"broken_sasl_auth_clients":        "yes",

		// Relay restrictions
		"smtpd_relay_restrictions": "permit_mynetworks permit_sasl_authenticated defer_unauth_destination",
		"smtpd_recipient_restrictions": "permit_mynetworks permit_sasl_authenticated reject_unauth_destination",

		// Milter for Rspamd
		"milter_protocol":        "6",
		"milter_default_action":  "accept",
		"smtpd_milters":          "inet:127.0.0.1:11332",
		"non_smtpd_milters":      "inet:127.0.0.1:11332",
	}

	for k, v := range settings {
		if _, err := runPostfixCommand("postconf", "-e", k+"="+v); err != nil {
			return err
		}
	}

	// Enable submission (587) and smtps (465) in master.cf
	if err := enablePostfixSubmissionPorts(); err != nil {
		return err
	}

	return reloadPostfix()
}

func enablePostfixSubmissionPorts() error {
	masterCF := "/etc/postfix/master.cf"
	content, err := readFileContents(masterCF)
	if err != nil {
		return err
	}

	submissionBlock := `
submission inet n       -       y       -       -       smtpd
  -o syslog_name=postfix/submission
  -o smtpd_tls_security_level=encrypt
  -o smtpd_sasl_auth_enable=yes
  -o smtpd_tls_auth_only=yes
  -o smtpd_reject_unlisted_recipient=no
  -o smtpd_recipient_restrictions=
  -o smtpd_relay_restrictions=permit_sasl_authenticated,reject
  -o milter_macro_daemon_name=ORIGINATING

smtps     inet  n       -       y       -       -       smtpd
  -o syslog_name=postfix/smtps
  -o smtpd_tls_wrappermode=yes
  -o smtpd_sasl_auth_enable=yes
  -o smtpd_reject_unlisted_recipient=no
  -o smtpd_recipient_restrictions=
  -o smtpd_relay_restrictions=permit_sasl_authenticated,reject
  -o milter_macro_daemon_name=ORIGINATING
`

	if !containsString(content, "submission inet") {
		content += submissionBlock
		if err := writeFileContents(masterCF, content); err != nil {
			return err
		}
	}
	return nil
}

func configureDovecotTLS(certPath, keyPath string) error {
	sslConf := `/etc/dovecot/conf.d/10-ssl.conf`

	content := `## SSL/TLS settings for Dovecot
ssl = required
ssl_cert = <` + certPath + `
ssl_key = <` + keyPath + `
ssl_min_protocol = TLSv1.2
ssl_cipher_list = ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384
ssl_prefer_server_ciphers = yes
`
	if err := writeFileContents(sslConf, content); err != nil {
		return err
	}

	// SASL auth socket for Postfix
	authConf := `/etc/dovecot/conf.d/10-master.conf`
	saslSocket := `
service auth {
  unix_listener /var/spool/postfix/private/auth {
    mode = 0660
    user = postfix
    group = postfix
  }
}
`
	existing, _ := readFileContents(authConf)
	if !containsString(existing, "postfix/private/auth") {
		if err := appendToFile(authConf, saslSocket); err != nil {
			return err
		}
	}

	return reloadDovecot()
}

// ConfigureDKIM generates a DKIM key pair for a domain and configures Rspamd.
func ConfigureDKIM(domain string) (publicKey string, err error) {
	keyDir := "/etc/rspamd/dkim"
	if err := mkdirIfNotExists(keyDir); err != nil {
		return "", err
	}

	keyFile := keyDir + "/" + domain + ".key"
	pubFile := keyDir + "/" + domain + ".pub"

	// Generate RSA key pair
	if _, err := runPostfixCommand("openssl", "genrsa", "-out", keyFile, "2048"); err != nil {
		return "", err
	}
	if _, err := runPostfixCommand("openssl", "rsa", "-in", keyFile, "-pubout", "-out", pubFile); err != nil {
		return "", err
	}

	// Write Rspamd DKIM config
	rspamdDKIM := "/etc/rspamd/local.d/dkim_signing.conf"
	dkimConf := `# DKIM signing for ` + domain + `
domain {
  ` + domain + ` {
    path = "` + keyFile + `";
    selector = "mail";
  }
}
`
	if err := appendToFile(rspamdDKIM, dkimConf); err != nil {
		return "", err
	}

	pubContent, err := readFileContents(pubFile)
	if err != nil {
		return "", err
	}
	return pubContent, nil
}

// Helper: run a command and return combined output
func runPostfixCommand(name string, args ...string) (string, error) {
	return runCmdOutput(name, args...)
}

// GetDKIMPublicKey returns the DNS TXT record value for DKIM.
// The caller should add this as: mail._domainkey.{domain} TXT "v=DKIM1; k=rsa; p=..."
func GetDKIMPublicKey(domain string) (string, error) {
	pubFile := "/etc/rspamd/dkim/" + domain + ".pub"
	content, err := readFileContents(pubFile)
	if err != nil {
		return "", err
	}
	// Strip PEM headers, join lines
	var key string
	for _, line := range splitLines(content) {
		if line == "-----BEGIN PUBLIC KEY-----" || line == "-----END PUBLIC KEY-----" {
			continue
		}
		key += line
	}
	return "v=DKIM1; k=rsa; p=" + key, nil
}

// Helpers reused from other executors in the package
func readFileContents(path string) (string, error) {
	data, err := readFileSafe(path)
	return string(data), err
}

func writeFileContents(path, content string) error {
	return writeFileSafe(path, []byte(content))
}

func appendToFile(path, content string) error {
	existing, _ := readFileContents(path)
	return writeFileContents(path, existing+content)
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func mkdirIfNotExists(path string) error {
	_, err := runCmdOutput("mkdir", "-p", path)
	return err
}
