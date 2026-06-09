#!/bin/bash
# Deploy ControlPanelVPS (master binary + frontend)
set -e

GOBIN="/home/claudedev/go/bin/go"
BASE="/opt/controlpanel"

echo "==> Build agent binary..."
cd "$BASE/agent"
"$GOBIN" build -o "$BASE/bin/agent" ./cmd/agent/

echo "==> Build master binary..."
cd "$BASE/master"
"$GOBIN" build -o "$BASE/bin/master" ./cmd/server/

echo "==> Build frontend..."
cd "$BASE/frontend"
npm run build --quiet

echo "==> Stop services..."
sudo systemctl stop cpanel-agent cpanel-master cpanel-frontend

echo "==> Deploy frontend (clean)..."
sudo rm -rf "$BASE/frontend-standalone"
sudo mkdir -p "$BASE/frontend-standalone"
sudo cp -r "$BASE/frontend/.next/standalone/." "$BASE/frontend-standalone/"
sudo cp -r "$BASE/frontend/.next/static" "$BASE/frontend-standalone/.next/static"
sudo chown -R claudedev:claudedev "$BASE/frontend-standalone"

echo "==> Start services..."
sudo systemctl start cpanel-agent cpanel-master cpanel-frontend
sleep 2
sudo systemctl is-active cpanel-agent cpanel-master cpanel-frontend
echo "==> Done."
