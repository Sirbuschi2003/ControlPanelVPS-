# ControlPanelVPS

A modern, self-hosted Linux server control panel — similar to Plesk, but fully open-source.

## Features

- **Dashboard** — Server overview with real-time health alerts
- **Server Management** — Register and monitor multiple servers with live metrics (CPU, RAM, Disk, Network)
- **Website Management** — Create and manage websites with Nginx, PHP version selection, aliases
- **SSL/TLS** — Let's Encrypt certificate issuance, renewal, and auto-renew
- **Databases** — Create and manage MySQL/PostgreSQL databases with password retrieval
- **DNS Management** — Full zone and record management (A, AAAA, CNAME, MX, TXT, SRV, CAA)
- **Mail Server** — Domains, mailboxes, aliases via Postfix + Dovecot; DKIM setup; Rspamd spam filter
- **Firewall** — UFW rule management with toggle and live reload
- **Backups** — Scheduled backups to local storage, S3, or SFTP with retention policies
- **System Services** — Start/stop/restart/enable/disable systemd services
- **Cron Jobs** — Create and manage cron jobs per server
- **Log Viewer** — Real-time log viewer with auto-refresh (Nginx, Syslog, Auth, Mail, MySQL, Fail2ban)
- **File Manager** — Browse, read, write, delete files and create directories
- **Web Terminal** — SSH-in-browser terminal
- **Package Updates** — View and apply pending system package updates
- **Monitoring** — Health check scoring with alerts (CPU, RAM, Disk thresholds)
- **User Management** — Multi-user RBAC with TOTP 2FA (Google Authenticator compatible)
- **Settings** — SMTP configuration, panel-wide settings
- **Self-Update** — One-click panel update from GitHub Releases with auto-update toggle

## Architecture

```
┌─────────────┐     ┌────────────────────────────────┐
│   Browser   │────▶│  Nginx (reverse proxy, SSL)    │
└─────────────┘     └───────────┬────────────────────┘
                                │
                    ┌───────────▼────────────┐
                    │  Next.js Frontend      │  :3000
                    │  (TypeScript, Tailwind)│
                    └───────────┬────────────┘
                                │ /api/*
                    ┌───────────▼────────────┐
                    │  Master API (Go)       │  :8080
                    │  Chi • JWT • PostgreSQL│
                    └───────────┬────────────┘
                                │ agent calls
              ┌─────────────────▼──────────────────┐
              │         Agent (Go, per server)      │  :8087
              │  gopsutil • Nginx • Certbot • BIND  │
              └────────────────────────────────────┘
```

## Production Installation

Requires Ubuntu 22.04 / 24.04 or Debian 12. Run as root:

```bash
curl -fsSL https://raw.githubusercontent.com/Sirbuschi2003/ControlPanelVPS-/master/deploy/install.sh | bash
```

The installer will ask for your panel domain (e.g. `panel.example.com`), then automatically:
- Installs PostgreSQL, Redis, Nginx, Postfix, Dovecot, BIND9, Rspamd, Fail2ban
- Downloads pre-built binaries from GitHub Releases
- Issues a Let's Encrypt TLS certificate
- Configures UFW firewall (ports 22, 80, 443, 25, 587, 465, 993, 143)
- Creates systemd services (`cpanel-master`, `cpanel-agent`, `cpanel-frontend`)

**Default credentials** (change immediately after login):
- Email: `admin@panel.local`
- Password: `ControlPanel2024!`

### Add a second server

```bash
curl -fsSL https://raw.githubusercontent.com/Sirbuschi2003/ControlPanelVPS-/master/deploy/install-agent.sh | bash -s -- \
  --master https://panel.yourdomain.com \
  --token YOUR_AGENT_TOKEN
```

## Local Development

### Prerequisites
- Docker + Docker Compose
- Go 1.22+
- Node.js 22+

```bash
# Start PostgreSQL + Redis
docker compose up postgres redis -d

# Run master API (http://localhost:8080)
cd master && go run ./cmd/server

# Run agent (separate terminal)
cd agent && go run ./cmd/agent

# Run frontend (http://localhost:3000)
cd frontend && npm install && npm run dev
```

Default dev credentials: `admin@panel.local` / `ControlPanel2024!`

## Tech Stack

| Component | Technology |
|---|---|
| Frontend | Next.js 15, TypeScript, Tailwind CSS |
| Backend | Go 1.22, Chi router, JWT, bcrypt |
| Database | PostgreSQL 16 |
| Cache/Session | Redis 7 |
| Agent | Go, gopsutil |
| Reverse Proxy | Nginx (auto-configured with Let's Encrypt) |
| Mail | Postfix + Dovecot + Rspamd |
| DNS | BIND9 |

## Security

- JWT authentication with rate-limited login (10 req/min)
- TOTP 2FA (Google Authenticator / Authy compatible)
- Admin-only routes for user and settings management
- Agent communication secured with shared token
- GDPR/DSGVO: IPs anonymized in audit log (last octet zeroed)
- NIS2: full audit trail for all mutations
- `Content-Type: application/json` enforced; no CORS wildcards
- Request body limited to 10 MB

## License

MIT
