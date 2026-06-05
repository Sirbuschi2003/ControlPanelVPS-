package models

import "time"

type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	Role        string    `json:"role"`
	TOTPEnabled bool      `json:"totp_enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

type Server struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Hostname   string     `json:"hostname"`
	IPAddress  string     `json:"ip_address"`
	AgentURL   string     `json:"agent_url"`
	Role       string     `json:"role"`
	Status     string     `json:"status"`
	LastSeen   *time.Time `json:"last_seen"`
	CreatedAt  time.Time  `json:"created_at"`
}

type ServerMetrics struct {
	ServerID  string  `json:"server_id"`
	CPUUsage  float64 `json:"cpu_usage"`
	MemTotal  uint64  `json:"mem_total"`
	MemUsed   uint64  `json:"mem_used"`
	DiskTotal uint64  `json:"disk_total"`
	DiskUsed  uint64  `json:"disk_used"`
	Uptime    uint64  `json:"uptime"`
	LoadAvg   float64 `json:"load_avg"`
}

type Website struct {
	ID            string     `json:"id"`
	ServerID      string     `json:"server_id"`
	Domain        string     `json:"domain"`
	Aliases       []string   `json:"aliases"`
	PHPVersion    string     `json:"php_version"`
	DocumentRoot  string     `json:"document_root"`
	IndexFile     string     `json:"index_file"`
	SSLEnabled    bool       `json:"ssl_enabled"`
	SSLForceHTTPS bool       `json:"ssl_force_https"`
	SSLCertID     *string    `json:"ssl_cert_id"`
	Enabled       bool       `json:"enabled"`
	Notes         *string    `json:"notes"`
	CreatedAt     time.Time  `json:"created_at"`
}

type SSLCert struct {
	ID         string     `json:"id"`
	ServerID   string     `json:"server_id"`
	Domain     string     `json:"domain"`
	SANDomains []string   `json:"san_domains"`
	Status     string     `json:"status"`
	Issuer     *string    `json:"issuer"`
	IssuedAt   *time.Time `json:"issued_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	AutoRenew  bool       `json:"auto_renew"`
	CreatedAt  time.Time  `json:"created_at"`
}

type ManagedDatabase struct {
	ID         string    `json:"id"`
	ServerID   string    `json:"server_id"`
	Name       string    `json:"name"`
	DBType     string    `json:"db_type"`
	DBUser     string    `json:"db_user"`
	DBPassword string    `json:"db_password,omitempty"`
	Charset    string    `json:"charset"`
	Collation  string    `json:"collation"`
	SizeBytes  int64     `json:"size_bytes"`
	Notes      *string   `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
}

type DNSZone struct {
	ID         string      `json:"id"`
	ServerID   string      `json:"server_id"`
	Name       string      `json:"name"`
	Serial     int         `json:"serial"`
	Refresh    int         `json:"refresh"`
	Retry      int         `json:"retry"`
	Expire     int         `json:"expire"`
	Minimum    int         `json:"minimum"`
	Nameserver string      `json:"nameserver"`
	AdminEmail string      `json:"admin_email"`
	Records    []DNSRecord `json:"records,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
}

type DNSRecord struct {
	ID        string    `json:"id"`
	ZoneID    string    `json:"zone_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	TTL       int       `json:"ttl"`
	Priority  int       `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
}

type MailDomain struct {
	ID        string    `json:"id"`
	ServerID  string    `json:"server_id"`
	Domain    string    `json:"domain"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type MailAccount struct {
	ID        string    `json:"id"`
	DomainID  string    `json:"domain_id"`
	Username  string    `json:"username"`
	QuotaMB   int       `json:"quota_mb"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type MailAlias struct {
	ID          string    `json:"id"`
	DomainID    string    `json:"domain_id"`
	Source      string    `json:"source"`
	Destination string    `json:"destination"`
	CreatedAt   time.Time `json:"created_at"`
}

type FirewallRule struct {
	ID        string    `json:"id"`
	ServerID  string    `json:"server_id"`
	RuleOrder int       `json:"rule_order"`
	Action    string    `json:"action"`
	Direction string    `json:"direction"`
	Protocol  string    `json:"protocol"`
	Source    string    `json:"source"`
	DestPort  *string   `json:"dest_port"`
	Comment   *string   `json:"comment"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type BackupConfig struct {
	ID            string            `json:"id"`
	ServerID      string            `json:"server_id"`
	Name          string            `json:"name"`
	StorageType   string            `json:"storage_type"`
	Schedule      string            `json:"schedule"`
	RetentionDays int               `json:"retention_days"`
	IncludePaths  []string          `json:"include_paths"`
	StorageConfig map[string]string `json:"storage_config"`
	Encrypt       bool              `json:"encrypt"`
	Enabled       bool              `json:"enabled"`
	CreatedAt     time.Time         `json:"created_at"`
}

type BackupJob struct {
	ID           string     `json:"id"`
	ConfigID     string     `json:"config_id"`
	Status       string     `json:"status"`
	SizeBytes    int64      `json:"size_bytes"`
	FilePath     *string    `json:"file_path"`
	ErrorMessage *string    `json:"error_message"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
}

type CronJob struct {
	ID          string     `json:"id"`
	ServerID    string     `json:"server_id"`
	Name        string     `json:"name"`
	Command     string     `json:"command"`
	Schedule    string     `json:"schedule"`
	RunAsUser   string     `json:"run_as_user"`
	Enabled     bool       `json:"enabled"`
	LastRun     *time.Time `json:"last_run"`
	LastStatus  *string    `json:"last_status"`
	CreatedAt   time.Time  `json:"created_at"`
}

type SystemService struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Active      string `json:"active"`
	Enabled     string `json:"enabled"`
	LoadState   string `json:"load_state"`
}

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Raw       string `json:"raw"`
}

type FileEntry struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	IsDir      bool      `json:"is_dir"`
	Mode       string    `json:"mode"`
	ModifiedAt time.Time `json:"modified_at"`
}

type PackageUpdate struct {
	Name           string `json:"name"`
	CurrentVersion string `json:"current_version"`
	NewVersion     string `json:"new_version"`
	Priority       string `json:"priority"`
}
