#!/bin/bash
# Linux Systemd Unit Installer for Distributed AI Cluster Node Agent
set -e

CONTROLLER_URL="${CONTROLLER_URL:-localhost:50051}"
ENROLLMENT_TOKEN="${ENROLLMENT_TOKEN:-}"
INSTALL_DIR="/usr/local/bin"
DATA_DIR="/var/lib/distributed-ai"

echo "=== Distributed AI Node Agent Linux Installer ==="

# 1. Create Directories
mkdir -p "$DATA_DIR/config"
mkdir -p "$DATA_DIR/identity"
mkdir -p "$DATA_DIR/logs"
mkdir -p "$DATA_DIR/models"

# 2. Copy Executable
if [ -f "node-agent" ]; then
    cp "node-agent" "$INSTALL_DIR/node-agent"
    chmod +x "$INSTALL_DIR/node-agent"
fi

# 3. Write Config
cat <<EOF > "$DATA_DIR/config/config.json"
{
  "controller_url": "$CONTROLLER_URL",
  "enrollment_token": "$ENROLLMENT_TOKEN",
  "node_data_dir": "$DATA_DIR",
  "log_level": "INFO",
  "telemetry_interval_seconds": 5,
  "heartbeat_interval_seconds": 5,
  "max_concurrent_tasks": 4
}
EOF

# 4. Systemd Service Unit
cat <<EOF > /etc/systemd/system/distributed-ai-node.service
[Unit]
Description=Distributed AI Compute Node Agent
After=network.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/node-agent --config $DATA_DIR/config/config.json
Restart=always
RestartSec=5s
User=root
WorkingDirectory=$DATA_DIR

[Install]
WantedBy=multi-user.target
EOF

# 5. Reload & Start Service
systemctl daemon-reload
systemctl enable distributed-ai-node
systemctl restart distributed-ai-node

echo "=== Installation Complete! Node Agent is running as a systemd service. ==="
