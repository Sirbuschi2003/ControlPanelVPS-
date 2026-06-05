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

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; CYAN='\033[0;36m'; NC='\033[0m'
info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
success() { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }
step()    { echo -e "\n${CYAN}━━━ $* ━━━${NC}"; }

# apt wrapper: shows a spinner + package name, no suppressed output
apt_install() {
  local desc="$1"; shift
  echo -e "${BLUE}[APT]${NC}   $desc"
  DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends "$@"
  echo -e "${GREEN}[OK]${NC}    $desc installiert"
  # Free apt cache after each group to save RAM
  apt-get clean
  rm -rf /var/lib/apt/lists/*
}

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

# ── System packages (in groups to avoid OOM on small VPS) ────────────────────
step "Systempakete installieren"
info "Paketlisten aktualisieren (apt-get update)..."
apt-get update -o Dpkg::Progress-Fancy="1"
success "Paketlisten aktualisiert"

apt_install "Basis-Tools (curl, git, build-essential, openssl)" \
  curl wget git build-essential ca-certificates gnupg lsb-release \
  htop net-tools unzip jq openssl

apt_install "Datenbank & Cache (PostgreSQL, Redis)" \
  postgresql postgresql-contrib redis-server

apt_install "Webserver & SSL (Nginx, Certbot)" \
  nginx certbot python3-certbot-nginx

apt_install "Sicherheit (Fail2ban, UFW, unattended-upgrades)" \
  fail2ban ufw unattended-upgrades apt-listchanges

apt_install "Mailserver (Postfix, Dovecot)" \
  postfix postfix-mysql dovecot-core dovecot-imapd dovecot-pop3d dovecot-lmtpd

# Rspamd: needs own repo on older systems
if apt-cache show rspamd &>/dev/null 2>&1; then
  apt_install "Spam-Filter (Rspamd)" rspamd
else
  info "Rspamd-Repository einrichten..."
  curl -fsSL https://rspamd.com/apt-stable/gpg.key | gpg --dearmor -o /usr/share/keyrings/rspamd-archive-keyring.gpg
  echo "deb [signed-by=/usr/share/keyrings/rspamd-archive-keyring.gpg] https://rspamd.com/apt-stable/ $(lsb_release -cs) main" \
    > /etc/apt/sources.list.d/rspamd.list
  apt-get update
  apt_install "Spam-Filter (Rspamd)" rspamd
fi

apt_install "DNS-Server (BIND9)" bind9 bind9utils

success "Alle Systempakete installiert"

# ── Go ───────────────────────────────────────────────────────────────────────
step "Go installieren"
if ! command -v go &>/dev/null || [[ "$(go version | awk '{print $3}')" != "go${GO_VERSION}" ]]; then
  info "Go ${GO_VERSION} herunterladen..."
  ARCH=$(dpkg --print-architecture)
  [[ "$ARCH" == "amd64" ]] && GOARCH="amd64" || GOARCH="arm64"
  curl -fsSL --progress-bar "https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz" -o /tmp/go.tar.gz
  info "Go entpacken..."
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tar.gz
  rm /tmp/go.tar.gz
  echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
  export PATH=$PATH:/usr/local/go/bin
  success "Go $(go version | awk '{print $3}') installiert"
else
  export PATH=$PATH:/usr/local/go/bin
  success "Go bereits installiert: $(go version | awk '{print $3}')"
fi

# ── Node.js ──────────────────────────────────────────────────────────────────
step "Node.js installieren"
if ! command -v node &>/dev/null; then
  info "NodeSource-Repository einrichten..."
  curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash -
  info "Node.js installieren..."
  DEBIAN_FRONTEND=noninteractive apt-get install -y nodejs
  success "Node.js $(node --version) / npm $(npm --version) installiert"
else
  success "Node.js bereits installiert: $(node --version)"
fi

# ── Directories ───────────────────────────────────────────────────────────────
step "Verzeichnisse und Benutzer anlegen"
mkdir -p "$INSTALL_DIR/bin" "$DATA_DIR" "$LOG_DIR"
useradd -r -s /bin/false -d "$INSTALL_DIR" cpanel 2>/dev/null || true
success "Verzeichnisse angelegt: $INSTALL_DIR"

# ── Clone repo ────────────────────────────────────────────────────────────────
step "Repository klonen"
if [[ -d "$INSTALL_DIR/.git" ]]; then
  info "Repository bereits vorhanden — aktualisiere..."
  git -C "$INSTALL_DIR" pull --rebase
  success "Repository aktualisiert"
else
  info "Klone $REPO nach $INSTALL_DIR ..."
  git clone "$REPO" "$INSTALL_DIR"
  success "Repository geklont"
fi

# ── PostgreSQL ────────────────────────────────────────────────────────────────
step "PostgreSQL konfigurieren"
systemctl start postgresql
systemctl enable postgresql
info "Datenbankbenutzer und Datenbank anlegen..."
sudo -u postgres psql -c "CREATE USER cpanel WITH PASSWORD '$DB_PASSWORD';" 2>/dev/null || \
  sudo -u postgres psql -c "ALTER USER cpanel WITH PASSWORD '$DB_PASSWORD';"
sudo -u postgres psql -c "CREATE DATABASE cpanel OWNER cpanel;" 2>/dev/null || true
success "PostgreSQL konfiguriert"

# ── Redis ─────────────────────────────────────────────────────────────────────
step "Redis konfigurieren"
REDIS_PASSWORD=$(openssl rand -hex 16)
sed -i "s/# requirepass foobared/requirepass $REDIS_PASSWORD/" /etc/redis/redis.conf
sed -i "s/requirepass .*/requirepass $REDIS_PASSWORD/" /etc/redis/redis.conf
systemctl restart redis-server
systemctl enable redis-server
success "Redis konfiguriert (Passwort gesetzt)"

# ── Environment file ──────────────────────────────────────────────────────────
step "Konfigurationsdatei schreiben"
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
success ".env geschrieben: $INSTALL_DIR/.env"

# ── Build Master ──────────────────────────────────────────────────────────────
step "Master-API kompilieren"
info "Go-Abhängigkeiten herunterladen..."
cd "$INSTALL_DIR/master"
/usr/local/go/bin/go mod download
info "Master-Binary kompilieren (das kann 1-2 Minuten dauern)..."
/usr/local/go/bin/go build -ldflags="-w -s" -o "$INSTALL_DIR/bin/master" ./cmd/server
success "Master-API kompiliert → $INSTALL_DIR/bin/master"

# ── Build Agent ───────────────────────────────────────────────────────────────
step "Agent kompilieren"
info "Go-Abhängigkeiten herunterladen..."
cd "$INSTALL_DIR/agent"
/usr/local/go/bin/go mod download
info "Agent-Binary kompilieren..."
/usr/local/go/bin/go build -ldflags="-w -s" -o "$INSTALL_DIR/bin/agent" ./cmd/agent
success "Agent kompiliert → $INSTALL_DIR/bin/agent"

# ── Build Frontend ────────────────────────────────────────────────────────────
step "Frontend bauen (Next.js)"
cd "$INSTALL_DIR/frontend"
info "npm-Pakete installieren..."
npm ci
info "Next.js-Build starten (2-5 Minuten)..."
NEXT_PUBLIC_API_URL="https://${PANEL_DOMAIN}" npm run build
success "Frontend gebaut"

# ── Systemd: Master ───────────────────────────────────────────────────────────
step "Systemd-Dienste einrichten"
info "Service-Dateien schreiben..."
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

info "Systemd neu laden und Dienste starten..."
systemctl daemon-reload
systemctl enable cpanel-master cpanel-agent
systemctl start cpanel-master cpanel-agent
success "Dienste gestartet (cpanel-master, cpanel-agent)"

# ── Nginx ─────────────────────────────────────────────────────────────────────
step "Nginx konfigurieren"
info "Virtual-Host schreiben..."
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
success "Nginx konfiguriert"

# ── SSL ───────────────────────────────────────────────────────────────────────
step "SSL-Zertifikat (Let's Encrypt)"
info "Certbot für $PANEL_DOMAIN starten..."
certbot --nginx -d "$PANEL_DOMAIN" --non-interactive --agree-tos -m "$ADMIN_EMAIL" && {
  CERT_PATH="/etc/letsencrypt/live/${PANEL_DOMAIN}/fullchain.pem"
  KEY_PATH="/etc/letsencrypt/live/${PANEL_DOMAIN}/privkey.pem"
  # Wire certificate into Postfix
  postconf -e "smtpd_tls_cert_file = ${CERT_PATH}"
  postconf -e "smtpd_tls_key_file = ${KEY_PATH}"
  postconf -e "smtp_tls_cert_file = ${CERT_PATH}"
  postconf -e "smtp_tls_key_file = ${KEY_PATH}"
  # Wire certificate into Dovecot
  sed -i "s|^ssl = yes|ssl = required|" /etc/dovecot/conf.d/10-ssl.conf
  cat >> /etc/dovecot/conf.d/10-ssl.conf <<DVCERTEOF
ssl_cert = <${CERT_PATH}
ssl_key = <${KEY_PATH}
DVCERTEOF
  systemctl restart postfix dovecot 2>/dev/null || true
  success "SSL-Zertifikat ausgestellt und in Postfix/Dovecot eingebunden"
} || warn "SSL fehlgeschlagen — manuell ausführen: certbot --nginx -d $PANEL_DOMAIN"

# ── Unattended Security Updates ──────────────────────────────────────────────
step "Automatische Sicherheitsupdates konfigurieren"
cat > /etc/apt/apt.conf.d/50unattended-upgrades <<'UUEOF'
Unattended-Upgrade::Allowed-Origins {
    "${distro_id}:${distro_codename}-security";
    "${distro_id}ESMApps:${distro_codename}-apps-security";
    "${distro_id}ESM:${distro_codename}-infra-security";
};
Unattended-Upgrade::AutoFixInterruptedDpkg "true";
Unattended-Upgrade::MinimalSteps "true";
Unattended-Upgrade::Remove-Unused-Dependencies "true";
Unattended-Upgrade::Automatic-Reboot "false";
Unattended-Upgrade::Mail "root";
UUEOF

cat > /etc/apt/apt.conf.d/20auto-upgrades <<'AUEOF'
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
APT::Periodic::Download-Upgradeable-Packages "1";
APT::Periodic::AutocleanInterval "7";
AUEOF
success "Automatische Sicherheitsupdates aktiviert"

# ── Mail TLS & Rspamd ────────────────────────────────────────────────────────
step "Mailserver-Sicherheit konfigurieren (TLS + Rspamd)"

# Create virtual mailboxes directory
mkdir -p /var/mail/vhosts
groupadd -g 5000 vmail 2>/dev/null || true
useradd -g vmail -u 5000 vmail -d /var/mail/vhosts 2>/dev/null || true
chown -R vmail:vmail /var/mail/vhosts

# Rspamd milter configuration
mkdir -p /etc/rspamd/local.d
cat > /etc/rspamd/local.d/worker-proxy.inc <<'RSEOF'
milter_servers = "127.0.0.1:11332";
RSEOF
cat > /etc/rspamd/local.d/actions.conf <<'RSAEOF'
actions {
  reject = 15;
  add_header = 6;
  greylist = 4;
}
RSAEOF
systemctl enable rspamd 2>/dev/null || true
systemctl start rspamd 2>/dev/null || true

# Postfix: basic TLS config (certs configured after certbot runs)
postconf -e "smtpd_tls_security_level = may"
postconf -e "smtpd_tls_mandatory_protocols = !SSLv2,!SSLv3,!TLSv1,!TLSv1.1"
postconf -e "smtpd_tls_protocols = !SSLv2,!SSLv3,!TLSv1,!TLSv1.1"
postconf -e "smtp_tls_security_level = may"
postconf -e "smtp_tls_mandatory_protocols = !SSLv2,!SSLv3,!TLSv1,!TLSv1.1"
postconf -e "smtp_tls_protocols = !SSLv2,!SSLv3,!TLSv1,!TLSv1.1"
postconf -e "smtpd_sasl_auth_enable = yes"
postconf -e "smtpd_sasl_type = dovecot"
postconf -e "smtpd_sasl_path = private/auth"
postconf -e "smtpd_sasl_security_options = noanonymous"
postconf -e "milter_protocol = 6"
postconf -e "milter_default_action = accept"
postconf -e "smtpd_milters = inet:127.0.0.1:11332"
postconf -e "non_smtpd_milters = inet:127.0.0.1:11332"

# Enable submission (587) and smtps (465) if not already active
if ! grep -q "^submission " /etc/postfix/master.cf; then
cat >> /etc/postfix/master.cf <<'MCEOF'

submission inet n       -       y       -       -       smtpd
  -o syslog_name=postfix/submission
  -o smtpd_tls_security_level=encrypt
  -o smtpd_sasl_auth_enable=yes
  -o smtpd_tls_auth_only=yes
  -o smtpd_relay_restrictions=permit_sasl_authenticated,reject
  -o milter_macro_daemon_name=ORIGINATING

smtps     inet  n       -       y       -       -       smtpd
  -o syslog_name=postfix/smtps
  -o smtpd_tls_wrappermode=yes
  -o smtpd_sasl_auth_enable=yes
  -o smtpd_relay_restrictions=permit_sasl_authenticated,reject
  -o milter_macro_daemon_name=ORIGINATING
MCEOF
fi

# Dovecot: require TLS (will be activated after certbot cert is available)
cat > /etc/dovecot/conf.d/10-ssl.conf <<'DVEOF'
ssl = yes
ssl_min_protocol = TLSv1.2
ssl_cipher_list = ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES128-GCM-SHA256
ssl_prefer_server_ciphers = yes
DVEOF

info "Postfix und Dovecot aktivieren..."
systemctl enable postfix dovecot 2>/dev/null || true
success "Mailserver-Sicherheit konfiguriert"

# ── Firewall (with mail ports) ────────────────────────────────────────────────
step "Firewall konfigurieren (UFW)"
info "Regeln setzen..."
ufw --force reset
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow http
ufw allow https
ufw allow 25/tcp   comment "SMTP MTA-to-MTA"
ufw allow 587/tcp  comment "SMTP Submission (clients)"
ufw allow 465/tcp  comment "SMTPS (clients)"
ufw allow 993/tcp  comment "IMAPS"
ufw allow 143/tcp  comment "IMAP STARTTLS"
ufw --force enable
success "Firewall aktiv — offene Ports: 22, 80, 443, 25, 587, 465, 993, 143"

# ── Register local server in panel ───────────────────────────────────────────
step "Server im Panel registrieren"
info "Warte 5 Sekunden auf Dienst-Start..."
sleep 5
info "Login-Token anfordern..."
LOGIN_TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"${ADMIN_EMAIL}\",\"password\":\"${ADMIN_PASSWORD}\"}" | jq -r .token 2>/dev/null || echo "")

if [[ -n "$LOGIN_TOKEN" && "$LOGIN_TOKEN" != "null" ]]; then
  info "Server-Eintrag erstellen..."
  curl -s -X POST http://localhost:8080/api/servers \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $LOGIN_TOKEN" \
    -d "{
      \"name\": \"$(hostname)\",
      \"hostname\": \"$(hostname)\",
      \"ip_address\": \"$(hostname -I | awk '{print $1}')\",
      \"agent_url\": \"http://127.0.0.1:8087\",
      \"agent_token\": \"${AGENT_TOKEN}\",
      \"role\": \"general\"
    }" > /dev/null
  success "Lokaler Server '$(hostname)' im Panel registriert"
else
  warn "Automatische Registrierung fehlgeschlagen — bitte manuell über das Panel hinzufügen"
fi

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║        Installation erfolgreich abgeschlossen!   ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  ${CYAN}Panel-URL:${NC}     ${BLUE}https://${PANEL_DOMAIN}${NC}"
echo -e "  ${CYAN}Admin-E-Mail:${NC}  ${BLUE}${ADMIN_EMAIL}${NC}"
echo -e "  ${CYAN}Agent-Token:${NC}   ${YELLOW}${AGENT_TOKEN}${NC}  ← sicher aufbewahren!"
echo ""
echo -e "  ${CYAN}Konfiguration:${NC}  $INSTALL_DIR/.env"
echo -e "  ${CYAN}Logs:${NC}           $LOG_DIR/"
echo ""
echo -e "  ${CYAN}Dienste prüfen:${NC}"
echo -e "    systemctl status cpanel-master"
echo -e "    systemctl status cpanel-agent"
echo ""
echo -e "  ${CYAN}Mail-Ports:${NC}  25 (MTA), 587 (Submission), 465 (SMTPS), 993 (IMAPS), 143 (IMAP)"
echo -e "  ${CYAN}Spam-Filter:${NC} Rspamd Milter auf 127.0.0.1:11332"
echo -e "  ${CYAN}Sicherheit:${NC}  unattended-upgrades aktiv (täglich, nur Security-Patches)"
echo ""
echo -e "  ${YELLOW}HINWEIS:${NC} Prüfe bei Dogado, ob ausgehender Port 25 freigeschaltet ist."
echo -e "  ${YELLOW}HINWEIS:${NC} DKIM-DNS-Eintrag über Panel → E-Mail → DKIM einrichten hinzufügen."
echo ""
