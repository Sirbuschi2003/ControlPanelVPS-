#!/usr/bin/env bash
# ControlPanelVPS — Full Installation Script
# Supports: Ubuntu 22.04, Ubuntu 24.04, Ubuntu 26.04, Debian 12
set -euo pipefail

REPO="https://github.com/Sirbuschi2003/ControlPanelVPS-"
INSTALL_DIR="/opt/controlpanel"
DATA_DIR="/var/lib/controlpanel"
LOG_DIR="/var/log/controlpanel"
GO_VERSION="1.22.4"
NODE_VERSION="22"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
success() { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }
step()    { echo -e "\n${CYAN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; \
            echo -e "${CYAN}${BOLD}  $*${NC}"; \
            echo -e "${CYAN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; }

# ── Root check ──────────────────────────────────────────────────────────────
[[ $EUID -eq 0 ]] || error "Als root ausführen: sudo bash install.sh"

# ── OS check ────────────────────────────────────────────────────────────────
. /etc/os-release
info "Betriebssystem erkannt: $PRETTY_NAME"
[[ "$ID" =~ ^(ubuntu|debian)$ ]] || error "Nicht unterstütztes OS. Nutze Ubuntu 22.04/24.04/26.04 oder Debian 12."

# ── Interactive setup ────────────────────────────────────────────────────────
# Bei "curl | bash" liest bash das Script von stdin (der Pipe).
# exec </dev/tty würde bash dazu bringen, die nächsten Script-Zeilen vom
# Terminal zu lesen → Script bricht. Stattdessen: </dev/tty pro read-Aufruf.

echo ""
echo -e "${BLUE}${BOLD}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}${BOLD}║     ControlPanelVPS — Installation     ║${NC}"
echo -e "${BLUE}${BOLD}╚════════════════════════════════════════╝${NC}"
echo ""

while true; do
  read -rp "Panel-Domain (z.B. panel.example.com): " PANEL_DOMAIN </dev/tty
  [[ -n "$PANEL_DOMAIN" ]] && break
  echo -e "${RED}Fehler:${NC} Domain darf nicht leer sein."
done


# Generate secrets
JWT_SECRET=$(openssl rand -hex 32)
AGENT_TOKEN=$(openssl rand -hex 24)
DB_PASSWORD=$(openssl rand -hex 16)

echo ""
info "Starte Installation für: ${BOLD}$PANEL_DOMAIN${NC}"
echo ""

# ── SWAP (prevent OOM on small VPS) ─────────────────────────────────────────
step "Swap-Speicher prüfen / anlegen"
TOTAL_RAM_MB=$(free -m | awk '/^Mem:/{print $2}')
CURRENT_SWAP=$(free -m | awk '/^Swap:/{print $2}')
info "RAM: ${TOTAL_RAM_MB} MB  |  Swap: ${CURRENT_SWAP} MB"

if [[ $CURRENT_SWAP -lt 512 ]]; then
  if [[ ! -f /swapfile ]]; then
    info "Lege 2 GB Swap-Datei an (verhindert OOM beim Kompilieren)..."
    fallocate -l 2G /swapfile 2>/dev/null || dd if=/dev/zero of=/swapfile bs=1M count=2048 status=progress
    chmod 600 /swapfile
    mkswap /swapfile
    swapon /swapfile
    echo '/swapfile none swap sw 0 0' >> /etc/fstab
    success "Swap angelegt: 2 GB"
  else
    swapon /swapfile 2>/dev/null || true
    success "Swap bereits vorhanden, aktiviert"
  fi
else
  success "Ausreichend Swap vorhanden (${CURRENT_SWAP} MB)"
fi

# ── System packages ──────────────────────────────────────────────────────────
step "Systempakete installieren"

export DEBIAN_FRONTEND=noninteractive

# Pre-seed Postfix so it doesn't ask interactive questions
info "Paketinstallation vorkonfigurieren..."
{
  echo "postfix postfix/main_mailer_type select Internet Site"
  echo "postfix postfix/mailname string $PANEL_DOMAIN"
} | debconf-set-selections

info "Paketlisten aktualisieren..."
# Cached pages freigeben bevor apt lädt (wichtig bei kleinem RAM)
sync
echo 1 > /proc/sys/vm/drop_caches
apt-get update
success "Paketlisten aktualisiert"

# pkg_install "Beschreibung" [--flags] paket1 paket2 ...
# Flags (z.B. --no-install-recommends) werden vor den Paketnamen übergeben.
pkg_install() {
  local desc="$1"; shift
  echo -e "\n${BLUE}[APT]${NC} ${BOLD}$desc${NC}"
  apt-get install -y "$@"
  apt-get clean    # nur .deb-Cache löschen, Paketlisten bleiben erhalten
  success "$desc installiert"
}

pkg_install "Basis-Tools" \
  --no-install-recommends \
  curl wget git ca-certificates gnupg lsb-release \
  htop net-tools unzip jq openssl software-properties-common

pkg_install "PostgreSQL & Redis" \
  --no-install-recommends \
  postgresql postgresql-contrib redis-server

pkg_install "Nginx & Certbot" \
  --no-install-recommends \
  nginx certbot python3-certbot-nginx

pkg_install "Sicherheits-Tools (Fail2ban, UFW, unattended-upgrades)" \
  --no-install-recommends \
  fail2ban ufw unattended-upgrades apt-listchanges

pkg_install "Postfix & Dovecot (Mailserver)" \
  postfix postfix-mysql \
  dovecot-core dovecot-imapd dovecot-pop3d dovecot-lmtpd \
  libsasl2-modules libsasl2-modules-db sasl2-bin

pkg_install "DNS-Server (BIND9)" \
  --no-install-recommends \
  bind9 bind9utils dnsutils

# Rspamd: hat eigenes Repository, da es oft nicht in Standard-Repos ist
info "Rspamd-Repository einrichten..."
if ! apt-cache show rspamd >/dev/null 2>&1 || \
   [[ "$(apt-cache policy rspamd 2>/dev/null | grep Candidate | awk '{print $2}')" == "(none)" ]]; then
  curl -fsSL https://rspamd.com/apt-stable/gpg.key \
    | gpg --dearmor -o /usr/share/keyrings/rspamd-archive-keyring.gpg
  echo "deb [signed-by=/usr/share/keyrings/rspamd-archive-keyring.gpg] \
https://rspamd.com/apt-stable/ $(lsb_release -cs) main" \
    > /etc/apt/sources.list.d/rspamd.list
  apt-get update
fi
pkg_install "Rspamd (Spam-Filter)" --no-install-recommends rspamd

success "Alle Systempakete installiert"

# ── Node.js ──────────────────────────────────────────────────────────────────
step "Node.js ${NODE_VERSION} installieren"
if command -v node &>/dev/null && node --version | grep -q "^v${NODE_VERSION}"; then
  success "Node.js bereits installiert: $(node --version)"
else
  info "NodeSource-Repository einrichten..."
  curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash -
  apt-get install -y nodejs
  apt-get clean
  success "Node.js $(node --version) installiert"
fi

# ── Directories & User ───────────────────────────────────────────────────────
step "Verzeichnisse anlegen"
mkdir -p "$INSTALL_DIR/bin" "$DATA_DIR" "$LOG_DIR"
useradd -r -s /bin/false -d "$INSTALL_DIR" cpanel 2>/dev/null || true
# cpanel user must own bin/ so the self-update can replace the binary at runtime
chown cpanel:cpanel "$INSTALL_DIR/bin" "$LOG_DIR"
success "Verzeichnisse angelegt"

# ── Clone repo ────────────────────────────────────────────────────────────────
step "Repository klonen / aktualisieren"
if [[ -d "$INSTALL_DIR/.git" ]]; then
  info "Existierendes Repository wird aktualisiert..."
  git -C "$INSTALL_DIR" reset --hard HEAD
  # go mod tidy generates go.sum (untracked) — remove before pull to avoid conflicts
  rm -f "$INSTALL_DIR/master/go.sum" "$INSTALL_DIR/agent/go.sum"
  git -C "$INSTALL_DIR" pull
  success "Repository aktualisiert"
else
  if [[ -d "$INSTALL_DIR" ]]; then
    info "Altes Verzeichnis $INSTALL_DIR (kein Git-Repo) wird entfernt..."
    rm -rf "$INSTALL_DIR"
  fi
  info "Klone $REPO ..."
  git clone "$REPO" "$INSTALL_DIR"
  success "Repository geklont nach $INSTALL_DIR"
fi

# ── PostgreSQL ────────────────────────────────────────────────────────────────
step "PostgreSQL konfigurieren"
systemctl start postgresql
systemctl enable postgresql
info "Datenbankbenutzer 'cpanel' anlegen..."
sudo -u postgres psql -c "CREATE USER cpanel WITH PASSWORD '$DB_PASSWORD';" 2>/dev/null \
  || sudo -u postgres psql -c "ALTER USER cpanel WITH PASSWORD '$DB_PASSWORD';"
sudo -u postgres psql -c "CREATE DATABASE cpanel OWNER cpanel;" 2>/dev/null || true
success "PostgreSQL konfiguriert"

# ── Redis ─────────────────────────────────────────────────────────────────────
step "Redis konfigurieren"
REDIS_PASSWORD=$(openssl rand -hex 16)
# Idempotent: replace existing requirepass or add new one
if grep -q "^requirepass " /etc/redis/redis.conf; then
  sed -i "s|^requirepass .*|requirepass $REDIS_PASSWORD|" /etc/redis/redis.conf
elif grep -q "^# requirepass foobared" /etc/redis/redis.conf; then
  sed -i "s|^# requirepass foobared|requirepass $REDIS_PASSWORD|" /etc/redis/redis.conf
else
  echo "requirepass $REDIS_PASSWORD" >> /etc/redis/redis.conf
fi
systemctl restart redis-server
systemctl enable redis-server
success "Redis konfiguriert"

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
INSTALL_DIR=${INSTALL_DIR}
GITHUB_REPO=Sirbuschi2003/ControlPanelVPS-
EOF
chmod 600 "$INSTALL_DIR/.env"
success ".env geschrieben"

# ── Download pre-built binaries from GitHub Releases ─────────────────────────
RELEASE_BASE="https://github.com/Sirbuschi2003/ControlPanelVPS-/releases/download/latest"
step "Binaries herunterladen (GitHub Release)"
mkdir -p "$INSTALL_DIR/bin"
info "Master-API herunterladen..."
curl -fL --progress-bar "${RELEASE_BASE}/master" -o "$INSTALL_DIR/bin/master"
chmod +x "$INSTALL_DIR/bin/master"
success "Master-API: $INSTALL_DIR/bin/master"

info "Agent herunterladen..."
curl -fL --progress-bar "${RELEASE_BASE}/agent" -o "$INSTALL_DIR/bin/agent"
chmod +x "$INSTALL_DIR/bin/agent"
success "Agent: $INSTALL_DIR/bin/agent"

info "Frontend herunterladen und entpacken..."
mkdir -p "$INSTALL_DIR/frontend-standalone"
curl -fL --progress-bar "${RELEASE_BASE}/frontend.tar.gz" \
  | tar -xz -C "$INSTALL_DIR/frontend-standalone"
success "Frontend: $INSTALL_DIR/frontend-standalone"

# ── Systemd: Master ───────────────────────────────────────────────────────────
step "Systemd-Dienste einrichten"

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
NoNewPrivileges=yes

[Install]
WantedBy=multi-user.target
EOF

# ── Systemd: Agent ────────────────────────────────────────────────────────────
NODE_ID_VAL=$(hostname)
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
Environment=NODE_ID=${NODE_ID_VAL}
ExecStart=${INSTALL_DIR}/bin/agent
Restart=on-failure
RestartSec=5
StandardOutput=append:${LOG_DIR}/agent.log
StandardError=append:${LOG_DIR}/agent-error.log

[Install]
WantedBy=multi-user.target
EOF

# ── Systemd: Frontend (Next.js standalone) ───────────────────────────────────
cat > /etc/systemd/system/cpanel-frontend.service <<EOF
[Unit]
Description=ControlPanelVPS Frontend (Next.js)
After=network.target cpanel-master.service

[Service]
Type=simple
User=cpanel
Group=cpanel
WorkingDirectory=${INSTALL_DIR}/frontend-standalone
Environment=PORT=3000
Environment=HOSTNAME=127.0.0.1
Environment=NODE_ENV=production
ExecStart=/usr/bin/node ${INSTALL_DIR}/frontend-standalone/server.js
Restart=on-failure
RestartSec=5
StandardOutput=append:${LOG_DIR}/frontend.log
StandardError=append:${LOG_DIR}/frontend-error.log

[Install]
WantedBy=multi-user.target
EOF

# Fix ownership so cpanel user can write logs
chown -R cpanel:cpanel "$LOG_DIR"
chown -R cpanel:cpanel "$INSTALL_DIR/frontend-standalone" 2>/dev/null || true

info "Systemd neu laden..."
systemctl daemon-reload
systemctl enable cpanel-master cpanel-agent cpanel-frontend
systemctl start cpanel-master cpanel-agent cpanel-frontend
success "Dienste gestartet"

# Dienststatus kurz zeigen
sleep 2
for svc in cpanel-master cpanel-agent cpanel-frontend; do
  STATUS=$(systemctl is-active "$svc" 2>/dev/null || echo "unknown")
  if [[ "$STATUS" == "active" ]]; then
    success "  $svc: ${GREEN}aktiv${NC}"
  else
    warn "  $svc: $STATUS (prüfe: journalctl -u $svc -n 20)"
  fi
done

# ── Nginx ─────────────────────────────────────────────────────────────────────
step "Nginx konfigurieren"
cat > /etc/nginx/sites-available/controlpanel <<EOF
server {
    listen 80;
    server_name ${PANEL_DOMAIN};

    # Frontend (Next.js)
    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;
    }

    # Master API
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

ln -sf /etc/nginx/sites-available/controlpanel /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default
nginx -t
systemctl enable nginx
systemctl reload nginx
success "Nginx konfiguriert"

# ── SSL (Let's Encrypt) ───────────────────────────────────────────────────────
step "SSL-Zertifikat ausstellen (Let's Encrypt)"
info "Certbot für $PANEL_DOMAIN starten..."
if certbot --nginx -d "$PANEL_DOMAIN" --non-interactive --agree-tos -m "admin@${PANEL_DOMAIN}"; then
  CERT_PATH="/etc/letsencrypt/live/${PANEL_DOMAIN}/fullchain.pem"
  KEY_PATH="/etc/letsencrypt/live/${PANEL_DOMAIN}/privkey.pem"

  # Postfix: TLS-Zertifikat eintragen
  postconf -e "smtpd_tls_cert_file = ${CERT_PATH}"
  postconf -e "smtpd_tls_key_file  = ${KEY_PATH}"
  postconf -e "smtp_tls_cert_file  = ${CERT_PATH}"
  postconf -e "smtp_tls_key_file   = ${KEY_PATH}"

  # Dovecot: ssl=required aktivieren + Zertifikat setzen
  sed -i "s|^ssl = yes|ssl = required|" /etc/dovecot/conf.d/10-ssl.conf
  grep -q "ssl_cert = " /etc/dovecot/conf.d/10-ssl.conf \
    || cat >> /etc/dovecot/conf.d/10-ssl.conf <<DVCERTEOF
ssl_cert = <${CERT_PATH}
ssl_key = <${KEY_PATH}
DVCERTEOF

  systemctl restart postfix dovecot 2>/dev/null || true
  success "SSL-Zertifikat ausgestellt und eingebunden"
else
  warn "SSL fehlgeschlagen (DNS noch nicht propagiert?)"
  warn "Später ausführen: certbot --nginx -d $PANEL_DOMAIN"
fi

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
success "Automatische Sicherheitsupdates aktiviert (täglich)"

# ── Mail TLS & Rspamd ────────────────────────────────────────────────────────
step "Mailserver-Sicherheit konfigurieren"

info "Virtuelles Mailbox-Verzeichnis anlegen..."
mkdir -p /var/mail/vhosts
groupadd -g 5000 vmail 2>/dev/null || true
useradd -g vmail -u 5000 vmail -d /var/mail/vhosts -s /sbin/nologin 2>/dev/null || true
chown -R vmail:vmail /var/mail/vhosts

info "Rspamd konfigurieren..."
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
systemctl restart rspamd 2>/dev/null || true

info "Postfix TLS und Submission-Ports konfigurieren..."
postconf -e "smtpd_tls_security_level       = may"
postconf -e "smtpd_tls_mandatory_protocols  = !SSLv2,!SSLv3,!TLSv1,!TLSv1.1"
postconf -e "smtpd_tls_protocols            = !SSLv2,!SSLv3,!TLSv1,!TLSv1.1"
postconf -e "smtp_tls_security_level        = may"
postconf -e "smtp_tls_mandatory_protocols   = !SSLv2,!SSLv3,!TLSv1,!TLSv1.1"
postconf -e "smtp_tls_protocols             = !SSLv2,!SSLv3,!TLSv1,!TLSv1.1"
postconf -e "smtpd_sasl_auth_enable         = yes"
postconf -e "smtpd_sasl_type                = dovecot"
postconf -e "smtpd_sasl_path                = private/auth"
postconf -e "smtpd_sasl_security_options    = noanonymous"
postconf -e "milter_protocol                = 6"
postconf -e "milter_default_action          = accept"
postconf -e "smtpd_milters                  = inet:127.0.0.1:11332"
postconf -e "non_smtpd_milters              = inet:127.0.0.1:11332"

# Submission (587) und SMTPS (465) aktivieren
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

info "Dovecot TLS konfigurieren..."
cat > /etc/dovecot/conf.d/10-ssl.conf <<'DVEOF'
ssl = yes
ssl_min_protocol = TLSv1.2
ssl_cipher_list = ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES128-GCM-SHA256
ssl_prefer_server_ciphers = yes
DVEOF

systemctl enable postfix dovecot 2>/dev/null || true
systemctl restart postfix dovecot 2>/dev/null || true
success "Mailserver-Sicherheit konfiguriert"

# ── Firewall ──────────────────────────────────────────────────────────────────
step "Firewall konfigurieren (UFW)"
ufw --force reset
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh        comment "SSH"
ufw allow 80/tcp     comment "HTTP"
ufw allow 443/tcp    comment "HTTPS"
ufw allow 25/tcp     comment "SMTP MTA-to-MTA"
ufw allow 587/tcp    comment "SMTP Submission"
ufw allow 465/tcp    comment "SMTPS"
ufw allow 993/tcp    comment "IMAPS"
ufw allow 143/tcp    comment "IMAP STARTTLS"
ufw --force enable
success "Firewall aktiv — Ports: 22, 80, 443, 25, 587, 465, 993, 143"

# ── Register local server ─────────────────────────────────────────────────────
step "Server im Panel registrieren"
info "Warte auf API-Start..."
for i in $(seq 1 15); do
  curl -sf http://localhost:8080/api/auth/login -o /dev/null 2>/dev/null && break
  sleep 2
done

LOGIN_TOKEN=$(curl -sf -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@panel.local","password":"ControlPanel2024!"}' \
  | jq -r '.token // empty' 2>/dev/null || echo "")

if [[ -n "$LOGIN_TOKEN" ]]; then
  SERVER_IP=$(hostname -I | awk '{print $1}')
  curl -sf -X POST http://localhost:8080/api/servers \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $LOGIN_TOKEN" \
    -d "{
      \"name\": \"$(hostname)\",
      \"hostname\": \"$(hostname)\",
      \"ip_address\": \"${SERVER_IP}\",
      \"agent_url\": \"http://127.0.0.1:8087\",
      \"agent_token\": \"${AGENT_TOKEN}\",
      \"role\": \"primary\"
    }" > /dev/null
  success "Server '$(hostname)' (${SERVER_IP}) im Panel registriert"
else
  warn "Auto-Registrierung fehlgeschlagen — über Panel → Server → Hinzufügen nachholen"
fi

# ── Fertig ────────────────────────────────────────────────────────────────────
echo ""
echo -e "${GREEN}${BOLD}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}${BOLD}║   ✓  Installation erfolgreich abgeschlossen!         ║${NC}"
echo -e "${GREEN}${BOLD}╚══════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  ${CYAN}Panel-URL:${NC}      ${BOLD}https://${PANEL_DOMAIN}${NC}"
echo -e "  ${CYAN}Admin-E-Mail:${NC}   ${BOLD}admin@panel.local${NC}"
echo -e "  ${CYAN}Admin-Passwort:${NC} ${BOLD}ControlPanel2024!${NC}  ← nach Login ändern!"
echo -e "  ${CYAN}Agent-Token:${NC}    ${YELLOW}${AGENT_TOKEN}${NC}  ← sicher notieren!"
echo ""
echo -e "  ${CYAN}Konfiguration:${NC}  ${INSTALL_DIR}/.env"
echo -e "  ${CYAN}Logs:${NC}           ${LOG_DIR}/"
echo ""
echo -e "  ${CYAN}Dienststatus:${NC}"
echo -e "    systemctl status cpanel-master"
echo -e "    systemctl status cpanel-agent"
echo -e "    systemctl status cpanel-frontend"
echo ""
echo -e "  ${CYAN}Mail-Ports:${NC}  25 (MTA), 587 (Submission+STARTTLS), 465 (SMTPS), 993 (IMAPS)"
echo -e "  ${CYAN}Spam-Filter:${NC} Rspamd  |  ${CYAN}Sicherheit:${NC} unattended-upgrades aktiv"
echo ""
echo -e "  ${YELLOW}HINWEIS:${NC} Prüfe bei deinem Anbieter, ob Port 25 (ausgehend) freigeschaltet ist."
echo -e "  ${YELLOW}HINWEIS:${NC} DKIM-DNS-Eintrag: Panel → E-Mail → Domain → DKIM einrichten."
echo ""
