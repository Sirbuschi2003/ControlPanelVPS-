package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	vsftpdUserConfDir = "/etc/vsftpd/user_conf"
	vsftpdConf        = "/etc/vsftpd.conf"
)

// SetupVsftpd ensures vsftpd is configured for virtual/chrooted users (run once).
func SetupVsftpd() error {
	additions := `
# ControlPanelVPS additions
chroot_local_user=YES
allow_writeable_chroot=YES
user_sub_token=$USER
local_root=/var/www/$USER
pasv_enable=YES
pasv_min_port=40000
pasv_max_port=50000
`
	data, _ := os.ReadFile(vsftpdConf)
	if !strings.Contains(string(data), "ControlPanelVPS") {
		f, err := os.OpenFile(vsftpdConf, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("open vsftpd.conf: %w", err)
		}
		defer f.Close()
		if _, err := f.WriteString(additions); err != nil {
			return fmt.Errorf("write vsftpd.conf: %w", err)
		}
	}
	_ = os.MkdirAll(vsftpdUserConfDir, 0755)
	if out, err := runCmdOutput("systemctl", "enable", "--now", "vsftpd"); err != nil {
		return fmt.Errorf("enable vsftpd: %w\n%s", err, out)
	}
	return nil
}

// CreateFTPAccount creates a system user for FTP access chrooted to homeDir.
func CreateFTPAccount(username, password, homeDir string) error {
	_ = os.MkdirAll(homeDir, 0755)

	// Add /usr/sbin/nologin to /etc/shells if missing (vsftpd requires it)
	shells, _ := os.ReadFile("/etc/shells")
	if !strings.Contains(string(shells), "/usr/sbin/nologin") {
		f, _ := os.OpenFile("/etc/shells", os.O_APPEND|os.O_WRONLY, 0644)
		if f != nil {
			_, _ = f.WriteString("/usr/sbin/nologin\n")
			f.Close()
		}
	}

	// Create system user (ignore error if already exists)
	runCmdOutput("adduser", "--no-create-home", "--home", homeDir,
		"--shell", "/usr/sbin/nologin", "--ingroup", "www-data",
		"--disabled-password", "--gecos", "", username)

	// Set password
	input := fmt.Sprintf("%s:%s", username, password)
	if out, err := runCmdInputOutput(input, "chpasswd"); err != nil {
		return fmt.Errorf("set ftp password: %w\n%s", err, out)
	}

	// Per-user vsftpd config: set local_root to the domain's document root
	userConf := filepath.Join(vsftpdUserConfDir, username)
	content := fmt.Sprintf("local_root=%s\n", homeDir)
	if err := os.WriteFile(userConf, []byte(content), 0644); err != nil {
		return fmt.Errorf("write user conf: %w", err)
	}

	return reloadVsftpd()
}

// DeleteFTPAccount removes the FTP user and its vsftpd config.
func DeleteFTPAccount(username string) error {
	runCmdOutput("deluser", "--remove-home", username)
	_ = os.Remove(filepath.Join(vsftpdUserConfDir, username))
	return reloadVsftpd()
}

// UpdateFTPPassword changes the password for an existing FTP user.
func UpdateFTPPassword(username, newPassword string) error {
	input := fmt.Sprintf("%s:%s", username, newPassword)
	if out, err := runCmdInputOutput(input, "chpasswd"); err != nil {
		return fmt.Errorf("update ftp password: %w\n%s", err, out)
	}
	return nil
}

func reloadVsftpd() error {
	out, err := runCmdOutput("systemctl", "restart", "vsftpd")
	if err != nil {
		return fmt.Errorf("restart vsftpd: %w\n%s", err, out)
	}
	return nil
}
