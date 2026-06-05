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
