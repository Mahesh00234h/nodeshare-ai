# Distributed AI Node Agent V1

# DistributedAINodeAgent

This repository contains the standalone **Node Agent V1** for private distributed LLM compute nodes.


The central control plane (Python Cluster Controller V1) is located separately in `d:\cluster-controller\` for independent git version control and cloud deployment (e.g. Render).

---

## Repository Structure

```
d:\decentralized\                # Go Node Agent Repository
├── cmd/                         # Node Agent main CLI & binary entrypoints
│   ├── node-agent/              # Background service daemon entrypoint
│   ├── mock-controller/         # Mock gRPC controller for isolated agent testing
│   └── node-cli/                # Diagnostic CLI tool
│
├── internal/                    # Core Go Agent Modules
│   ├── agent/                   # Agent state machine (CONNECTING, READY, BUSY, etc.)
│   ├── hardware/                # CPU, RAM, GPU/VRAM, Storage, Network detection
│   ├── controller/              # gRPC client, heartbeats, telemetry & job loop
│   ├── jobs/                    # Inference execution engine & reservation
│   ├── models/                  # Local model file inventory manager
│   ├── security/                # Safety thresholds & credentials management
│   └── identity/                # Persistent Node UUID persistence
│
├── api/                         # Protobuf definitions & generated Go code
├── installer/                   # Windows PowerShell & Linux bash service installers
├── docs/                        # Architecture & protocol documentation
├── configs/                     # Default configuration JSON files
└── bin/                         # Compiled executables (node-agent.exe, etc.)
```

---

## Quick Start

### 1. Build Node Agent
```bash
go build -o bin\node-agent.exe cmd\node-agent\main.go
```

### 2. Run Node Agent in Foreground Mode
```bash
.\bin\node-agent.exe --foreground --controller-url localhost:50051 --enrollment-token dev-enrollment-token
```

---

## Standalone Cluster Controller Location

The Python Cluster Controller is located in `d:\cluster-controller\`:
```bash
cd d:\cluster-controller
python -m uvicorn app.main:app --host 0.0.0.0 --port 8000
```
- **REST & OpenAI API:** `http://localhost:8000`
- **Node gRPC Server:** `localhost:50051`
