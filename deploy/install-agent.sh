#!/usr/bin/env bash
# ControlPanelVPS — Agent-only Installation (for additional servers)
# Usage: bash install-agent.sh --master https://panel.example.com --token SECRET
set -euo pipefail

INSTALL_DIR="/opt/controlpanel-agent"
LOG_DIR="/var/log/controlpanel"
GO_VERSION="1.22.4"
MASTER_URL=""
AGENT_TOKEN=""
NODE_ID=$(hostname)

RED='\033[0;31m'; GREEN='\033[0;32m'; BLUE='\033[0;34m'; NC='\033[0m'
info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
success() { echo -e "${GREEN}[OK]${NC}    $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --master) MASTER_URL="$2"; shift 2 ;;
    --token)  AGENT_TOKEN="$2"; shift 2 ;;
    --node-id) NODE_ID="$2"; shift 2 ;;
    *) error "Unknown argument: $1" ;;
  esac
done

[[ $EUID -eq 0 ]] || error "Run as root: sudo bash install-agent.sh ..."
[[ -n "$MASTER_URL" ]] || error "--master is required"
[[ -n "$AGENT_TOKEN" ]] || error "--token is required"

. /etc/os-release
info "Detected: $PRETTY_NAME"

# Install Go
if ! command -v go &>/dev/null; then
  info "Installing Go ${GO_VERSION}..."
  ARCH=$(dpkg --print-architecture 2>/dev/null || uname -m)
  [[ "$ARCH" =~ (amd64|x86_64) ]] && GOARCH="amd64" || GOARCH="arm64"
  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz" -o /tmp/go.tar.gz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tar.gz
  export PATH=$PATH:/usr/local/go/bin
  success "Go installed"
fi

# Install git
apt-get update -qq && apt-get install -y -qq git curl

# Clone / update
mkdir -p "$INSTALL_DIR"
REPO="https://github.com/Sirbuschi2003/ControlPanelVPS-"
if [[ -d "$INSTALL_DIR/.git" ]]; then
  git -C "$INSTALL_DIR" pull --rebase
else
  git clone "$REPO" "$INSTALL_DIR"
fi

# Build agent
info "Building agent..."
cd "$INSTALL_DIR/agent"
/usr/local/go/bin/go mod download
/usr/local/go/bin/go build -ldflags="-w -s" -o /usr/local/bin/cpanel-agent ./cmd/agent
success "Agent built"

mkdir -p "$LOG_DIR"

# Systemd service
cat > /etc/systemd/system/cpanel-agent.service <<EOF
[Unit]
Description=ControlPanelVPS Agent
After=network.target

[Service]
Type=simple
User=root
Environment=LISTEN_ADDR=:8087
Environment=AGENT_TOKEN=${AGENT_TOKEN}
Environment=MASTER_URL=${MASTER_URL}
Environment=NODE_ID=${NODE_ID}
ExecStart=/usr/local/bin/cpanel-agent
Restart=on-failure
RestartSec=5
StandardOutput=append:${LOG_DIR}/agent.log
StandardError=append:${LOG_DIR}/agent-error.log

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable cpanel-agent
systemctl start cpanel-agent
success "Agent service started"

# Firewall: only allow agent port from master
if command -v ufw &>/dev/null; then
  MASTER_IP=$(echo "$MASTER_URL" | grep -oP '(?<=://)[^:/]+')
  ufw allow from "$MASTER_IP" to any port 8087 comment "ControlPanel Agent" 2>/dev/null || true
fi

echo ""
success "Agent installed on $(hostname) ($(hostname -I | awk '{print $1}'))"
echo ""
echo "Now add this server in your panel:"
echo "  Agent URL:   http://$(hostname -I | awk '{print $1}'):8087"
echo "  Agent Token: ${AGENT_TOKEN}"
echo ""
