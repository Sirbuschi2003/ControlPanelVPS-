#!/usr/bin/env bash
# ControlPanelVPS — Full Installation Script (Master + Agent on same server)
# Supports: Ubuntu 22.04, Ubuntu 24.04, Debian 12
set -euo pipefail

REPO="https://github.com/Sirbuschi2003/ControlPanelVPS-"
INSTALL_DIR="/opt/controlpanel"
DATA_DIR="/var/lib/controlpanel"
LOG_DIR="/var/log/controlpanel"
GO_VERSION="1.22.4"
NODE_VERSION="22"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
success() { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

# ── Root check ──────────────────────────────────────────────────────────────
[[ $EUID -eq 0 ]] || error "Run as root: sudo bash install.sh"

# ── OS check ────────────────────────────────────────────────────────────────
. /etc/os-release
info "Detected: $PRETTY_NAME"
[[ "$ID" =~ ^(ubuntu|debian)$ ]] || error "Unsupported OS. Use Ubuntu 22.04/24.04 or Debian 12."

# ── Interactive setup ────────────────────────────────────────────────────────
echo ""
echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     ControlPanelVPS — Installation     ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""

read -rp "Panel domain (e.g. panel.example.com): " PANEL_DOMAIN
[[ -n "$PANEL_DOMAIN" ]] || error "Domain cannot be empty"

read -rp "Admin email: " ADMIN_EMAIL
[[ -n "$ADMIN_EMAIL" ]] || error "Email cannot be empty"

read -rsp "Admin password (min 12 chars): " ADMIN_PASSWORD; echo
[[ ${#ADMIN_PASSWORD} -ge 12 ]] || error "Password too short (min 12 chars)"

# Generate secrets
JWT_SECRET=$(openssl rand -hex 32)
AGENT_TOKEN=$(openssl rand -hex 24)
DB_PASSWORD=$(openssl rand -hex 16)

echo ""
info "Starting installation..."
echo ""

# ── System packages ──────────────────────────────────────────────────────────
info "Installing system packages..."
apt-get update -qq
apt-get install -y -qq \
  curl wget git build-essential ca-certificates gnupg lsb-release \
  postgresql postgresql-contrib redis-server \
  nginx certbot python3-certbot-nginx \
  fail2ban ufw \
  htop net-tools unzip jq

success "System packages installed"

# ── Go ───────────────────────────────────────────────────────────────────────
if ! command -v go &>/dev/null || [[ "$(go version | awk '{print $3}')" != "go${GO_VERSION}" ]]; then
  info "Installing Go ${GO_VERSION}..."
  ARCH=$(dpkg --print-architecture)
  [[ "$ARCH" == "amd64" ]] && GOARCH="amd64" || GOARCH="arm64"
  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz" -o /tmp/go.tar.gz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tar.gz
  rm /tmp/go.tar.gz
  echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
  export PATH=$PATH:/usr/local/go/bin
  success "Go ${GO_VERSION} installed"
fi

# ── Node.js ──────────────────────────────────────────────────────────────────
if ! command -v node &>/dev/null; then
  info "Installing Node.js ${NODE_VERSION}..."
  curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash -
  apt-get install -y -qq nodejs
  success "Node.js $(node --version) installed"
fi

# ── Directories ───────────────────────────────────────────────────────────────
info "Creating directories..."
mkdir -p "$INSTALL_DIR" "$DATA_DIR" "$LOG_DIR"
useradd -r -s /bin/false -d "$INSTALL_DIR" cpanel 2>/dev/null || true

# ── Clone repo ────────────────────────────────────────────────────────────────
info "Cloning repository..."
if [[ -d "$INSTALL_DIR/.git" ]]; then
  git -C "$INSTALL_DIR" pull --rebase
else
  git clone "$REPO" "$INSTALL_DIR"
fi

# ── PostgreSQL ────────────────────────────────────────────────────────────────
info "Configuring PostgreSQL..."
systemctl start postgresql
systemctl enable postgresql
sudo -u postgres psql -c "CREATE USER cpanel WITH PASSWORD '$DB_PASSWORD';" 2>/dev/null || \
  sudo -u postgres psql -c "ALTER USER cpanel WITH PASSWORD '$DB_PASSWORD';"
sudo -u postgres psql -c "CREATE DATABASE cpanel OWNER cpanel;" 2>/dev/null || true
success "PostgreSQL configured"

# ── Redis ─────────────────────────────────────────────────────────────────────
info "Configuring Redis..."
REDIS_PASSWORD=$(openssl rand -hex 16)
sed -i "s/# requirepass foobared/requirepass $REDIS_PASSWORD/" /etc/redis/redis.conf
sed -i "s/requirepass .*/requirepass $REDIS_PASSWORD/" /etc/redis/redis.conf
systemctl restart redis-server
systemctl enable redis-server
success "Redis configured"

# ── Environment file ──────────────────────────────────────────────────────────
info "Writing configuration..."
cat > "$INSTALL_DIR/.env" <<EOF
DATABASE_URL=postgres://cpanel:${DB_PASSWORD}@localhost:5432/cpanel?sslmode=disable
REDIS_URL=redis://:${REDIS_PASSWORD}@localhost:6379/0
JWT_SECRET=${JWT_SECRET}
AGENT_TOKEN=${AGENT_TOKEN}
LISTEN_ADDR=:8080
ENVIRONMENT=production
PANEL_DOMAIN=${PANEL_DOMAIN}
ADMIN_EMAIL=${ADMIN_EMAIL}
EOF
chmod 600 "$INSTALL_DIR/.env"

# ── Build Master ──────────────────────────────────────────────────────────────
info "Building master API..."
cd "$INSTALL_DIR/master"
/usr/local/go/bin/go mod download
/usr/local/go/bin/go build -ldflags="-w -s" -o "$INSTALL_DIR/bin/master" ./cmd/server
success "Master API built"

# ── Build Agent ───────────────────────────────────────────────────────────────
info "Building agent..."
cd "$INSTALL_DIR/agent"
/usr/local/go/bin/go mod download
/usr/local/go/bin/go build -ldflags="-w -s" -o "$INSTALL_DIR/bin/agent" ./cmd/agent
success "Agent built"

# ── Build Frontend ────────────────────────────────────────────────────────────
info "Building frontend..."
cd "$INSTALL_DIR/frontend"
npm ci --silent
NEXT_PUBLIC_API_URL="https://${PANEL_DOMAIN}" npm run build
success "Frontend built"

# ── Systemd: Master ───────────────────────────────────────────────────────────
info "Installing systemd services..."
cat > /etc/systemd/system/cpanel-master.service <<EOF
[Unit]
Description=ControlPanelVPS Master API
After=network.target postgresql.service redis-server.service
Requires=postgresql.service redis-server.service

[Service]
Type=simple
User=cpanel
Group=cpanel
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=${INSTALL_DIR}/.env
ExecStart=${INSTALL_DIR}/bin/master
Restart=on-failure
RestartSec=5
StandardOutput=append:${LOG_DIR}/master.log
StandardError=append:${LOG_DIR}/master-error.log

[Install]
WantedBy=multi-user.target
EOF

# ── Systemd: Agent ────────────────────────────────────────────────────────────
cat > /etc/systemd/system/cpanel-agent.service <<EOF
[Unit]
Description=ControlPanelVPS Agent
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
Environment=LISTEN_ADDR=:8087
Environment=AGENT_TOKEN=${AGENT_TOKEN}
Environment=NODE_ID=$(hostname)
ExecStart=${INSTALL_DIR}/bin/agent
Restart=on-failure
RestartSec=5
StandardOutput=append:${LOG_DIR}/agent.log
StandardError=append:${LOG_DIR}/agent-error.log

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable cpanel-master cpanel-agent
systemctl start cpanel-master cpanel-agent
success "Services started"

# ── Nginx ─────────────────────────────────────────────────────────────────────
info "Configuring Nginx..."
cat > /etc/nginx/sites-available/controlpanel <<EOF
server {
    listen 80;
    server_name ${PANEL_DOMAIN};

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
    }
}
EOF

ln -sf /etc/nginx/sites-available/controlpanel /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default
nginx -t && systemctl reload nginx

# ── SSL ───────────────────────────────────────────────────────────────────────
info "Requesting SSL certificate..."
certbot --nginx -d "$PANEL_DOMAIN" --non-interactive --agree-tos -m "$ADMIN_EMAIL" || \
  warn "SSL setup failed — run: certbot --nginx -d $PANEL_DOMAIN"

# ── Firewall ─────────────────────────────────────────────────────────────────
info "Configuring firewall..."
ufw --force reset
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow http
ufw allow https
ufw --force enable
success "Firewall configured"

# ── Register local server in panel ───────────────────────────────────────────
info "Registering local server..."
sleep 3
curl -s -X POST http://localhost:8080/api/servers \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $(curl -s -X POST http://localhost:8080/api/auth/login \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"${ADMIN_EMAIL}\",\"password\":\"${ADMIN_PASSWORD}\"}" | jq -r .token)" \
  -d "{
    \"name\": \"$(hostname)\",
    \"hostname\": \"$(hostname)\",
    \"ip_address\": \"$(hostname -I | awk '{print $1}')\",
    \"agent_url\": \"http://127.0.0.1:8087\",
    \"agent_token\": \"${AGENT_TOKEN}\",
    \"role\": \"general\"
  }" > /dev/null && success "Local server registered" || warn "Could not auto-register server"

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo -e "${GREEN}╔════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║     Installation Complete!                 ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  Panel URL:   ${BLUE}https://${PANEL_DOMAIN}${NC}"
echo -e "  Admin Email: ${BLUE}${ADMIN_EMAIL}${NC}"
echo -e "  Agent Token: ${YELLOW}${AGENT_TOKEN}${NC} (save this!)"
echo ""
echo -e "  Config:  ${INSTALL_DIR}/.env"
echo -e "  Logs:    ${LOG_DIR}/"
echo ""
echo -e "  Services:"
echo -e "    systemctl status cpanel-master"
echo -e "    systemctl status cpanel-agent"
echo ""
