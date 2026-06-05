#!/usr/bin/env bash
# ControlPanelVPS — Update script
set -euo pipefail

INSTALL_DIR="/opt/controlpanel"
GREEN='\033[0;32m'; BLUE='\033[0;34m'; NC='\033[0m'
info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
success() { echo -e "${GREEN}[OK]${NC}    $*"; }

[[ $EUID -eq 0 ]] || { echo "Run as root"; exit 1; }

info "Pulling latest code..."
git -C "$INSTALL_DIR" pull --rebase

info "Rebuilding master..."
cd "$INSTALL_DIR/master"
/usr/local/go/bin/go mod download
/usr/local/go/bin/go build -ldflags="-w -s" -o "$INSTALL_DIR/bin/master" ./cmd/server

info "Rebuilding agent..."
cd "$INSTALL_DIR/agent"
/usr/local/go/bin/go mod download
/usr/local/go/bin/go build -ldflags="-w -s" -o "$INSTALL_DIR/bin/agent" ./cmd/agent

info "Rebuilding frontend..."
cd "$INSTALL_DIR/frontend"
npm ci --silent && npm run build

info "Restarting services..."
systemctl restart cpanel-master cpanel-agent

success "Update complete!"
