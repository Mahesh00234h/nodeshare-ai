# Distributed LLM Inference Runtime V1

## Overview

The **Distributed Inference Runtime V1** enables multiple physical PCs connected over a local Ethernet/LAN network to participate in executing a **single LLM inference request**.

Instead of request-level load balancing (sending Request A to PC1 and Request B to PC2), the Distributed Runtime splits model layers across GPUs on multiple machines using `llama.cpp`'s native RPC backend.

---

## Network Architecture

```
Render Cloud Controller
       │
       │ Control Plane (gRPC / HTTP Bridge)
       ▼
 ┌───────────┐  LAN RPC  ┌───────────┐  LAN RPC  ┌───────────┐
 │    PC1    │◄─────────►│    PC2    │◄─────────►│    PC3    │
 │ (Lead/Main│           │(RPC Worker│           │(RPC Worker│
 │ llama-cli)│           │  llama-   │           │  llama-   │
 │           │           │rpc-server)│           │rpc-server)│
 └───────────┘           └───────────┘           └───────────┘
```

- **Control Plane**: Render coordinates node scheduling, identity, and jobs.
- **Inference Plane**: High-speed model tensors flow **strictly over trusted local Ethernet/LAN** between PCs. RPC ports are never exposed to the public internet.

---

## Reproducing a 2-Node Distributed Inference Demonstration

### Prerequisites
1. Two physical PCs (or virtual instances) connected to the same local Ethernet subnet (e.g. `192.168.1.x`).
2. CUDA-enabled `llama.cpp` binaries (`llama-cli.exe` and `llama-rpc-server.exe`) in PATH or `C:\ProgramData\DistributedAI\bin\`.
3. GGUF model file (e.g., `qwen2.5-0.5b-instruct-q4_k_m.gguf`) downloaded to both PCs.

### Step 1: Start RPC Worker on PC2
On PC2 (`192.168.1.50`):
```powershell
llama-rpc-server.exe -H 0.0.0.0 -p 50052
```

### Step 2: Run Distributed Test from PC1
On PC1 (`192.168.1.10`):
```powershell
runtime-cli.exe distributed-test --nodes 192.168.1.50
```

### Output Evidence
```json
{
  "execution_id": "exec-1725940211",
  "job_id": "dist-job-001",
  "model_id": "qwen2.5-0.5b-instruct-q4_k_m.gguf",
  "split_mode": "LAYER_RPC",
  "tensor_split": [0.5, 0.5],
  "rpc_endpoints": ["192.168.1.50:50052"],
  "assignments": [
    {
      "node_id": "PC1-Lead",
      "role": "LEAD",
      "gpu_layers_count": 16
    },
    {
      "node_id": "PC2-Worker",
      "role": "RPC_WORKER",
      "rpc_endpoint": "192.168.1.50:50052",
      "gpu_layers_count": 16
    }
  ]
}
```
The model execution splits 16 layers onto PC1's GPU and 16 layers onto PC2's GPU over local RPC.

---

## Diagnostic Commands

```powershell
# Check runtime state and hardware capabilities
runtime-cli.exe status

# List local CUDA devices and VRAM
runtime-cli.exe devices

# List cached GGUF models
runtime-cli.exe models

# Perform binary & system health check
runtime-cli.exe health

# Verify RPC worker status and LAN security
runtime-cli.exe rpc-status

# Run single-node execution test
runtime-cli.exe execution test

# Run throughput benchmark
runtime-cli.exe benchmark
```
