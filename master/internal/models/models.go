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

// ServerMetrics matches the nested JSON format returned by the agent collector.
type ServerMetrics struct {
	ServerID      string        `json:"server_id"`
	Timestamp     time.Time     `json:"timestamp"`
	CPU           CPUMetrics    `json:"cpu"`
	Memory        MemoryMetrics `json:"memory"`
	Disk          DiskMetrics   `json:"disk"`
	Network       NetMetrics    `json:"network"`
	Uptime        uint64        `json:"uptime"`
	LoadAvg       LoadMetrics   `json:"load_avg"`
	Hostname      string        `json:"hostname"`
	OS            string        `json:"os"`
	KernelVersion string        `json:"kernel_version"`
}

type CPUMetrics struct {
	UsagePercent float64 `json:"usage_percent"`
	Cores        int     `json:"cores"`
}

type MemoryMetrics struct {
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	FreeBytes    uint64  `json:"free_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

type DiskMetrics struct {
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	FreeBytes    uint64  `json:"free_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

type NetMetrics struct {
	BytesSent uint64 `json:"bytes_sent"`
	BytesRecv uint64 `json:"bytes_recv"`
}

type LoadMetrics struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
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
	ZoneType   string      `json:"zone_type"`
	MasterIP   *string     `json:"master_ip,omitempty"`
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

// Domain is the central entity — one domain = one Plesk subscription.
type Domain struct {
	ID           string    `json:"id"`
	ServerID     string    `json:"server_id"`
	ServerName   string    `json:"server_name,omitempty"`
	ServerIP     string    `json:"server_ip,omitempty"`
	Name         string    `json:"name"`
	OwnerUserID  *string   `json:"owner_user_id,omitempty"`
	OwnerName    *string   `json:"owner_name,omitempty"`
	DocumentRoot string    `json:"document_root"`
	PHPVersion   string    `json:"php_version"`
	Status       string    `json:"status"`
	WebsiteID    *string   `json:"website_id,omitempty"`
	DNSZoneID    *string   `json:"dns_zone_id,omitempty"`
	MailDomainID *string   `json:"mail_domain_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// DomainUser represents a user who has been granted access to a domain.
type DomainUser struct {
	DomainID  string    `json:"domain_id"`
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name,omitempty"`
	UserEmail string    `json:"user_email,omitempty"`
	GrantedAt time.Time `json:"granted_at"`
}

// DomainResources bundles all resources associated with a domain.
type DomainResources struct {
	Domain    Domain            `json:"domain"`
	Website   *Website          `json:"website,omitempty"`
	DNSZone   *DNSZone          `json:"dns_zone,omitempty"`
	MailDomain *MailDomain      `json:"mail_domain,omitempty"`
	SSLCerts  []SSLCert         `json:"ssl_certs"`
	Databases []ManagedDatabase `json:"databases"`
	CronJobs  []CronJob         `json:"cron_jobs"`
}
