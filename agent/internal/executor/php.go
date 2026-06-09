package executor

import (
	"fmt"
	"os"
	"path/filepath"
)

// PHPPoolConfig holds PHP-FPM pool settings for a domain.
type PHPPoolConfig struct {
	Domain            string
	PHPVersion        string
	MemoryLimit       int
	MaxExecutionTime  int
	UploadMaxFilesize int
	PostMaxSize       int
	MaxInputVars      int
	DisplayErrors     bool
}

// WritePHPPool creates or updates a PHP-FPM pool config for the given domain.
func WritePHPPool(cfg PHPPoolConfig) error {
	phpVersion := cfg.PHPVersion
	if phpVersion == "" {
		phpVersion = "8.2"
	}
	poolDir := fmt.Sprintf("/etc/php/%s/fpm/pool.d", phpVersion)
	if err := os.MkdirAll(poolDir, 0755); err != nil {
		return fmt.Errorf("create pool dir: %w", err)
	}

	displayErrors := "Off"
	if cfg.DisplayErrors {
		displayErrors = "On"
	}
	memLimit := cfg.MemoryLimit
	if memLimit <= 0 {
		memLimit = 256
	}
	maxExec := cfg.MaxExecutionTime
	if maxExec <= 0 {
		maxExec = 60
	}
	upload := cfg.UploadMaxFilesize
	if upload <= 0 {
		upload = 64
	}
	post := cfg.PostMaxSize
	if post <= 0 {
		post = 64
	}
	inputVars := cfg.MaxInputVars
	if inputVars <= 0 {
		inputVars = 1000
	}

	content := fmt.Sprintf(`[%s]
user = www-data
group = www-data
listen = /run/php/php%s-fpm-%s.sock
listen.owner = www-data
listen.group = www-data
pm = dynamic
pm.max_children = 10
pm.start_servers = 2
pm.min_spare_servers = 1
pm.max_spare_servers = 5
php_admin_value[memory_limit] = %dM
php_admin_value[max_execution_time] = %d
php_admin_value[upload_max_filesize] = %dM
php_admin_value[post_max_size] = %dM
php_admin_value[max_input_vars] = %d
php_flag[display_errors] = %s
`, cfg.Domain, phpVersion, cfg.Domain, memLimit, maxExec, upload, post, inputVars, displayErrors)

	poolFile := filepath.Join(poolDir, cfg.Domain+".conf")
	if err := os.WriteFile(poolFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("write pool config: %w", err)
	}

	if out, err := runCmdOutput("systemctl", "reload", fmt.Sprintf("php%s-fpm", phpVersion)); err != nil {
		return fmt.Errorf("reload php-fpm: %w\n%s", err, out)
	}
	return nil
}

// DeletePHPPool removes a domain's PHP-FPM pool config.
func DeletePHPPool(domain, phpVersion string) error {
	if phpVersion == "" {
		phpVersion = "8.2"
	}
	poolFile := fmt.Sprintf("/etc/php/%s/fpm/pool.d/%s.conf", phpVersion, domain)
	_ = os.Remove(poolFile)
	out, err := runCmdOutput("systemctl", "reload", fmt.Sprintf("php%s-fpm", phpVersion))
	if err != nil {
		return fmt.Errorf("reload php-fpm: %w\n%s", err, out)
	}
	return nil
}
