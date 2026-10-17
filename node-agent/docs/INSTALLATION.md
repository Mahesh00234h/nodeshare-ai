# Installation & Setup Guide

## Windows Installation (Windows Service)

1. Build the Node Agent executable:
   ```cmd
   go build -o node-agent.exe ./cmd/node-agent
   ```
2. Open PowerShell as Administrator and execute the installer:
   ```powershell
   .\installer\windows\install.ps1 -ControllerURL "https://llm-cluster.onrender.com:443" -EnrollmentToken "YOUR_ENROLLMENT_TOKEN"
   ```
3. The installation script will:
   - Create data directories under `C:\ProgramData\DistributedAI\`
   - Copy binary to `C:\Program Files\DistributedAI\NodeAgent\`
   - Generate persistent UUID node ID
   - Register and start the `DistributedAINodeAgent` Windows Service
   - Enable auto-start on boot

4. Service Status Check:
   ```cmd
   sc query DistributedAINodeAgent
   ```

---

## Linux Installation (systemd)

1. Build Linux binary:
   ```bash
   go build -o node-agent ./cmd/node-agent
   ```
2. Run installation script as root:
   ```bash
   sudo CONTROLLER_URL="controller.example.com:50051" ENROLLMENT_TOKEN="YOUR_TOKEN" ./installer/linux/install.sh
   ```
3. Verify service state:
   ```bash
   sudo systemctl status distributed-ai-node
   ```
