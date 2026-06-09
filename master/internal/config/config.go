package config

import "os"

type Config struct {
	DatabaseURL    string
	RedisURL       string
	JWTSecret      string
	AgentToken     string
	ListenAddr     string
	Environment    string
	AdminEmail     string
	AdminPassword  string
	InstallDir     string
	GitHubRepo     string
	AllowedOrigins string // comma-separated; empty = localhost only
}

func Load() *Config {
	return &Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://cpanel:cpanel_dev_password@localhost:5432/cpanel?sslmode=disable"),
		RedisURL:       getEnv("REDIS_URL", "redis://:cpanel_dev_redis@localhost:6379/0"),
		JWTSecret:      getEnv("JWT_SECRET", "dev_secret_change_in_production_32c"),
		AgentToken:     getEnv("AGENT_TOKEN", "dev_agent_token"),
		ListenAddr:     getEnv("LISTEN_ADDR", ":8080"),
		Environment:    getEnv("ENVIRONMENT", "development"),
		AdminEmail:     getEnv("ADMIN_EMAIL", "admin@panel.local"),
		AdminPassword:  getEnv("ADMIN_PASSWORD", ""),
		InstallDir:     getEnv("INSTALL_DIR", "/opt/controlpanel"),
		GitHubRepo:     getEnv("GITHUB_REPO", "Sirbuschi2003/ControlPanelVPS-"),
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
