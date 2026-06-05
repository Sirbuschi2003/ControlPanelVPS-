package executor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// VhostConfig holds all configuration for an Nginx virtual host.
type VhostConfig struct {
	Domain        string   `json:"domain"`
	Aliases       []string `json:"aliases"`
	PHPVersion    string   `json:"php_version"`
	DocumentRoot  string   `json:"document_root"`
	IndexFile     string   `json:"index_file"`
	SSLEnabled    bool     `json:"ssl_enabled"`
	SSLForceHTTPS bool     `json:"ssl_force_https"`
	SSLCertPath   string   `json:"ssl_cert_path"`
	SSLKeyPath    string   `json:"ssl_key_path"`
}

const (
	nginxSitesAvailable = "/etc/nginx/sites-available"
	nginxSitesEnabled   = "/etc/nginx/sites-enabled"
	nginxWebRoot        = "/var/www"
)

// buildNginxConfig generates a complete Nginx server block for the given config.
func buildNginxConfig(cfg VhostConfig) string {
	docRoot := cfg.DocumentRoot
	if docRoot == "" {
		docRoot = fmt.Sprintf("/var/www/%s/public_html", cfg.Domain)
	}
	indexFile := cfg.IndexFile
	if indexFile == "" {
		indexFile = "index.php index.html index.htm"
	}

	phpVersion := cfg.PHPVersion
	if phpVersion == "" {
		phpVersion = "8.2"
	}

	serverNames := cfg.Domain
	if len(cfg.Aliases) > 0 {
		serverNames = cfg.Domain + " " + strings.Join(cfg.Aliases, " ")
	}

	var b strings.Builder

	// Upstream block for PHP-FPM
	fmt.Fprintf(&b, "upstream php%s {\n", strings.ReplaceAll(phpVersion, ".", ""))
	fmt.Fprintf(&b, "    server unix:/run/php/php%s-fpm.sock;\n", phpVersion)
	b.WriteString("}\n\n")

	if cfg.SSLEnabled && cfg.SSLForceHTTPS {
		// HTTP block that redirects to HTTPS
		b.WriteString("server {\n")
		b.WriteString("    listen 80;\n")
		b.WriteString("    listen [::]:80;\n")
		fmt.Fprintf(&b, "    server_name %s;\n", serverNames)
		b.WriteString("\n")
		b.WriteString("    # Redirect HTTP to HTTPS\n")
		b.WriteString("    return 301 https://$host$request_uri;\n")
		b.WriteString("}\n\n")
	}

	if cfg.SSLEnabled {
		// HTTPS block
		b.WriteString("server {\n")
		b.WriteString("    listen 443 ssl http2;\n")
		b.WriteString("    listen [::]:443 ssl http2;\n")
		fmt.Fprintf(&b, "    server_name %s;\n", serverNames)
		b.WriteString("\n")
		fmt.Fprintf(&b, "    ssl_certificate     %s;\n", cfg.SSLCertPath)
		fmt.Fprintf(&b, "    ssl_certificate_key %s;\n", cfg.SSLKeyPath)
		b.WriteString("    ssl_protocols       TLSv1.2 TLSv1.3;\n")
		b.WriteString("    ssl_ciphers         HIGH:!aNULL:!MD5;\n")
		b.WriteString("    ssl_prefer_server_ciphers on;\n")
		b.WriteString("    ssl_session_cache   shared:SSL:10m;\n")
		b.WriteString("    ssl_session_timeout 10m;\n")
		b.WriteString("\n")
		writeNginxLocationBlock(&b, docRoot, indexFile, phpVersion)
		b.WriteString("}\n")

		if !cfg.SSLForceHTTPS {
			// Also serve HTTP (no redirect)
			b.WriteString("\n")
			b.WriteString("server {\n")
			b.WriteString("    listen 80;\n")
			b.WriteString("    listen [::]:80;\n")
			fmt.Fprintf(&b, "    server_name %s;\n", serverNames)
			b.WriteString("\n")
			writeNginxLocationBlock(&b, docRoot, indexFile, phpVersion)
			b.WriteString("}\n")
		}
	} else {
		// HTTP only block
		b.WriteString("server {\n")
		b.WriteString("    listen 80;\n")
		b.WriteString("    listen [::]:80;\n")
		fmt.Fprintf(&b, "    server_name %s;\n", serverNames)
		b.WriteString("\n")
		writeNginxLocationBlock(&b, docRoot, indexFile, phpVersion)
		b.WriteString("}\n")
	}

	return b.String()
}

func writeNginxLocationBlock(b *strings.Builder, docRoot, indexFile, phpVersion string) {
	phpUpstream := fmt.Sprintf("php%s", strings.ReplaceAll(phpVersion, ".", ""))
	fmt.Fprintf(b, "    root %s;\n", docRoot)
	fmt.Fprintf(b, "    index %s;\n", indexFile)
	b.WriteString("\n")
	b.WriteString("    access_log /var/log/nginx/$host-access.log;\n")
	b.WriteString("    error_log  /var/log/nginx/$host-error.log;\n")
	b.WriteString("\n")
	b.WriteString("    location / {\n")
	b.WriteString("        try_files $uri $uri/ /index.php?$query_string;\n")
	b.WriteString("    }\n")
	b.WriteString("\n")
	b.WriteString("    location ~ \\.php$ {\n")
	b.WriteString("        include snippets/fastcgi-php.conf;\n")
	fmt.Fprintf(b, "        fastcgi_pass %s;\n", phpUpstream)
	b.WriteString("        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;\n")
	b.WriteString("        include fastcgi_params;\n")
	b.WriteString("    }\n")
	b.WriteString("\n")
	b.WriteString("    location ~ /\\.ht {\n")
	b.WriteString("        deny all;\n")
	b.WriteString("    }\n")
}

// CreateVhost creates a new Nginx virtual host config, enables it, and reloads Nginx.
func CreateVhost(cfg VhostConfig) error {
	docRoot := cfg.DocumentRoot
	if docRoot == "" {
		docRoot = fmt.Sprintf("%s/%s/public_html", nginxWebRoot, cfg.Domain)
	}

	// Create web root directory if it doesn't exist
	if err := os.MkdirAll(docRoot, 0755); err != nil {
		return fmt.Errorf("creating document root %s: %w", docRoot, err)
	}

	confPath := filepath.Join(nginxSitesAvailable, cfg.Domain+".conf")
	content := buildNginxConfig(cfg)

	if err := os.WriteFile(confPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing nginx config %s: %w", confPath, err)
	}

	// Create symlink in sites-enabled
	enabledPath := filepath.Join(nginxSitesEnabled, cfg.Domain+".conf")
	// Remove stale symlink if present
	_ = os.Remove(enabledPath)
	if err := os.Symlink(confPath, enabledPath); err != nil {
		return fmt.Errorf("creating symlink %s -> %s: %w", enabledPath, confPath, err)
	}

	return ReloadNginx()
}

// UpdateVhost overwrites an existing virtual host config and reloads Nginx.
func UpdateVhost(domain string, cfg VhostConfig) error {
	cfg.Domain = domain
	confPath := filepath.Join(nginxSitesAvailable, domain+".conf")
	content := buildNginxConfig(cfg)

	if err := os.WriteFile(confPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing nginx config %s: %w", confPath, err)
	}

	return ReloadNginx()
}

// DeleteVhost removes the Nginx virtual host config and reloads Nginx.
func DeleteVhost(domain string) error {
	enabledPath := filepath.Join(nginxSitesEnabled, domain+".conf")
	availablePath := filepath.Join(nginxSitesAvailable, domain+".conf")

	// Remove symlink (ignore error if not exists)
	_ = os.Remove(enabledPath)

	// Remove config file
	if err := os.Remove(availablePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing nginx config %s: %w", availablePath, err)
	}

	return ReloadNginx()
}

// ToggleVhost creates or removes the symlink in sites-enabled to enable/disable a vhost.
func ToggleVhost(domain string, enabled bool) error {
	confPath := filepath.Join(nginxSitesAvailable, domain+".conf")
	enabledPath := filepath.Join(nginxSitesEnabled, domain+".conf")

	if enabled {
		// Ensure the source config exists
		if _, err := os.Stat(confPath); os.IsNotExist(err) {
			return fmt.Errorf("nginx config not found: %s", confPath)
		}
		_ = os.Remove(enabledPath)
		if err := os.Symlink(confPath, enabledPath); err != nil {
			return fmt.Errorf("creating symlink: %w", err)
		}
	} else {
		if err := os.Remove(enabledPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing symlink: %w", err)
		}
	}

	return ReloadNginx()
}

// ListVhosts returns all .conf filenames in /etc/nginx/sites-available.
func ListVhosts() ([]string, error) {
	entries, err := os.ReadDir(nginxSitesAvailable)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", nginxSitesAvailable, err)
	}

	var vhosts []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".conf") {
			// Strip .conf suffix to return domain names
			vhosts = append(vhosts, strings.TrimSuffix(e.Name(), ".conf"))
		}
	}
	return vhosts, nil
}

// ReloadNginx tests the Nginx configuration and reloads the service.
func ReloadNginx() error {
	// Test config first
	testCmd := exec.Command("nginx", "-t")
	testOut, err := testCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nginx config test failed: %s", string(testOut))
	}

	// Reload nginx
	reloadCmd := exec.Command("systemctl", "reload", "nginx")
	reloadOut, err := reloadCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nginx reload failed: %s", string(reloadOut))
	}

	return nil
}
