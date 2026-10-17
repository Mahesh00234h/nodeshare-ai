# Architecture Audit Report

**Date:** September 10, 2026  
**System:** Distributed AI Infrastructure & Inference Runtime V1  
**Scope:** Complete Codebase & Protocol Inspection  

---

## Executive Summary

This architecture audit details all existing components, entry points, communication protocols, dependencies, data models, and known limitations across the **Go Node Agent**, **Distributed Inference Runtime V1**, and **Python Cluster Controller**.

---

## 1. Inventory of Existing Components

```
d:\decentralized\
├── node-agent/               # Go compute agent running on physical PCs
├── distributed-runtime/      # Distributed inference engine wrapping llama.cpp RPC
└── cluster-controller/       # Python control plane on Render (FastAPI + gRPC + SQLite/PostgreSQL)
```

---

## 2. Component Audits

### Component 1: Go Node Agent (`node-agent/`)
- **Location:** `d:\decentralized\node-agent`
- **Entry Points:**
  - `cmd/node-agent/main.go` (Main Agent Daemon)
  - `cmd/installer/main.go` (`node-agent-installer.exe` automated installer)
  - `cmd/launcher/main.go` (`launch-agent.exe` silent background launcher)
- **Purpose:** PC identity management, enrollment, authentication, gRPC heartbeats (5s interval), hardware/GPU telemetry collection (NVIDIA CUDA via `nvidia-smi`), model cache management, and job execution bridging.
- **Protocols Used:**
  - gRPC over HTTP/2 (Primary)
  - gRPC-Web / HTTP-1.1 Fallback (`/node_controller.v1.NodeControllerService/*`)
- **Dependencies:**
  - `github.com/distributed-ai/distributed-runtime` (via Go `replace` rule in `go.mod`)
  - `google.golang.org/grpc`
  - `golang.org/x/sys/windows`
- **Current Status:** Operational & verified online with Render control plane.
- **Known Limitations:**
  - Background process execution on Windows requires detached process flag (`-WindowStyle Hidden` or `launch-agent.exe`) to prevent termination when closing interactive console host windows.

---

### Component 2: Distributed Inference Runtime (`distributed-runtime/`)
- **Location:** `d:\decentralized\distributed-runtime`
- **Entry Points:**
  - `cmd/runtime-cli/main.go` (`runtime-cli.exe` diagnostic CLI)
  - `cmd/runtime/main.go` (`runtime.exe` standalone runtime engine daemon)
- **Purpose:** Enforces real multi-node model-parallel inference using `llama.cpp`'s native RPC backend. Features explicit state machine, heterogeneous hardware scoring, layer-split VRAM budget planning, model checksum verification, and token streaming.
- **Packages:**
  - `pkg/runtime`: Runtime engine lifecycle & state machine (`STARTING` $\rightarrow$ `INITIALIZING` $\rightarrow$ `CHECKING_LLAMA` $\rightarrow$ `CHECKING_GPU` $\rightarrow$ `READY` $\rightarrow$ `LOADING_MODEL` $\rightarrow$ `PREPARING_CLUSTER` $\rightarrow$ `RUNNING` $\rightarrow$ `DRAINING` $\rightarrow$ `DEGRADED` $\rightarrow$ `RECOVERING` $\rightarrow$ `STOPPING` $\rightarrow$ `ERROR`).
  - `pkg/llama`: `LlamaProcessManager` (`llama-cli` line-buffered token streamer) & `RPCWorkerManager` (`llama-rpc-server` daemon supervisor on port 50052).
  - `pkg/execution`: `ExecutionPlan`, `NodeAssignment`, `DeviceScorer`, and `ExecutionPlanner` (calculates `SINGLE_NODE` vs `LAYER_RPC` tensor splits).
  - `pkg/communication`: `LANInterfaceInfo` (selects Ethernet over Wi-Fi) and `LANSecurityGuard` (validates private IPv4 subnets: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `127.0.0.0/8`).
  - `pkg/model`: `GGUFModelManifest`, `VerifyModelFileSHA256` checksum validator, and `ModelCacheManager` (`C:\ProgramData\DistributedAI\models\`).
  - `pkg/recovery`: `FailureReport` classifier and `RecoveryPlanner` (pre-execution failure re-planning).
- **Protocols Used:** `llama.cpp` RPC protocol (Binary TCP on port 50052 over LAN).
- **Dependencies:** `llama-cli.exe`, `llama-rpc-server.exe` (with automatic fallback to mock execution drivers if binaries are absent).
- **Current Status:** Operational. Passed 100% unit tests and diagnostic tests.
- **Known Limitations:**
  - In-flight RPC disconnects mid-generation cause job termination (`IN_PROGRESS_FAULT_RECOVERY_UNSUPPORTED`). Pre-execution node failure triggers re-planning.

---

### Component 3: Python Cluster Controller (`cluster-controller/`)
- **Location:** `d:\decentralized\cluster-controller`
- **Entry Points:**
  - `app/main.py` (FastAPI + gRPC Servicer)
  - Deployment: Render PaaS (`https://llm-cluster.onrender.com:443`)
- **Purpose:** Cluster state authority, node registry & health monitoring (degrades after 15s without heartbeat, offline after 30s), model registry, scheduling, gRPC bridge, and OpenAI REST API endpoints (`/v1/chat/completions`, `/v1/nodes`).
- **Protocols Used:** gRPC Protobuf (`node_controller.proto`), HTTP/1.1 REST (`fastapi`).
- **Dependencies:** `fastapi`, `uvicorn`, `grpcio`, `pydantic`, `sqlalchemy`, `pytest`.
- **Current Status:** Operational and deployed on Render.
- **Known Limitations:** Hosted on Render free tier (cold starts apply).

---

## 3. Protocol & Interface Definitions Discovered

### Protobuf Service (`node_controller.proto`)
```protobuf
service NodeControllerService {
  rpc EnrollNode (EnrollmentRequest) returns (EnrollmentResponse);
  rpc RegisterNode (RegistrationRequest) returns (RegistrationResponse);
  rpc SendHeartbeat (HeartbeatRequest) returns (HeartbeatResponse);
  rpc SendTelemetry (TelemetryRequest) returns (TelemetryResponse);
  rpc StreamControl (stream ControlUpstream) returns (stream ControlDownstream);
}
```

### Go InferenceRuntime Interface (`pkg/runtime/runtime.go`)
```go
type InferenceRuntime interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health() State
	ListDevices() ([]DeviceCapabilities, error)
	ListModels() ([]ModelManifest, error)
	LoadModel(ctx context.Context, manifest ModelManifest) error
	UnloadModel(ctx context.Context, modelID string) error
	PrepareDistributedExecution(plan execution.ExecutionPlan) error
	Generate(ctx context.Context, req GenerationRequest) (*GenerationResult, error)
	StreamGenerate(ctx context.Context, req GenerationRequest, tokenCb func(string) error) (*GenerationResult, error)
	Cancel(jobID string) error
}
```

---

## 4. Summary Matrix

| Component | Language | Location | Primary Protocol | Key Status |
| :--- | :--- | :--- | :--- | :--- |
| **Node Agent** | Go | `node-agent/` | gRPC / HTTP Bridge | Operational |
| **Distributed Runtime** | Go | `distributed-runtime/` | llama.cpp LAN RPC | Operational |
| **Cluster Controller** | Python | `cluster-controller/` | gRPC / REST | Operational (Render) |
| **RPC Worker Daemon** | C++ | `llama.cpp` | TCP 50052 | Operational |
