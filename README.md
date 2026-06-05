# ControlPanelVPS

A modern, self-hosted Linux server control panel with multi-server management.

## Architecture

- **Master** — Go REST API + PostgreSQL + Redis. Runs the web UI backend and coordinates agents.
- **Agent** — Lightweight Go binary on every managed server. Collects metrics and executes commands.
- **Frontend** — Next.js 15 with TypeScript and Tailwind CSS.

## Quick Start (Development)

### Prerequisites
- Docker + Docker Compose
- Go 1.22+
- Node.js 22+

### Run locally

```bash
# Start PostgreSQL + Redis
docker compose up postgres redis -d

# Run the master API
cd master && go run ./cmd/server

# Run the agent (separate terminal)
cd agent && go run ./cmd/agent

# Run the frontend (separate terminal)
cd frontend && npm install && npm run dev
```

Open [http://localhost:3000](http://localhost:3000)

Default credentials: `admin@panel.local` / `changeme`

## Production Installation

Single server (recommended for starting out):

```bash
curl -fsSL https://raw.githubusercontent.com/Sirbuschi2003/ControlPanelVPS-/master/deploy/install.sh | bash
```

Add a second server later:

```bash
curl -fsSL https://raw.githubusercontent.com/Sirbuschi2003/ControlPanelVPS-/master/deploy/install-agent.sh | bash -s -- \
  --master https://panel.yourdomain.com \
  --token YOUR_AGENT_TOKEN
```

## Features (Phase 1 — In Progress)

- [x] User authentication (JWT + 2FA)
- [x] Server inventory & registration
- [x] Real-time system metrics (CPU, RAM, Disk, Network)
- [x] Web terminal (SSH in browser)
- [ ] Nginx / web server management
- [ ] Database management (MySQL, PostgreSQL)
- [ ] DNS management (PowerDNS)
- [ ] Email server (Postfix + Dovecot)
- [ ] SSL/TLS (Let's Encrypt)
- [ ] Firewall (nftables)
- [ ] Backup management
- [ ] Multi-server load balancing
- [ ] Docker management

## Tech Stack

| Layer | Technology |
|---|---|
| Frontend | Next.js 15, TypeScript, Tailwind CSS, shadcn/ui |
| Backend | Go 1.22, Chi router, JWT auth |
| Database | PostgreSQL 16, Redis 7 |
| Agent | Go binary, gopsutil |
| Reverse Proxy | Caddy (auto HTTPS) |

## Security

- JWT authentication with short-lived tokens + refresh
- TOTP 2FA (Google Authenticator compatible)
- All agent communication authenticated with shared tokens (mTLS in roadmap)
- Full audit log of every action
- Rate limiting on all auth endpoints
- nftables firewall management

## License

MIT
