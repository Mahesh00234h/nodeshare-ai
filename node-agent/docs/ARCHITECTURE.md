# System Architecture - Distributed AI Cluster Node Agent

## Overview

The Node Agent is a headless, production-oriented background infrastructure service designed to run on heterogeneous compute nodes within a private distributed LLM cluster.

```
              Lovable Control Dashboard
                        |
                        v
               Render Cluster Controller
                        |
            Secure Node Protocol (gRPC)
                        |
    +-------------------+-------------------+
    |                   |                   |
    v                   v                   v
  Node 1              Node 2              Node 3
  Agent               Agent               Agent
    |                   |                   |
    +-------------------+-------------------+
                        |
            Distributed LLM Runtime (Mock / Ollama)
```

## Key Architectural Principles

1. **Replaceable Worker Model:** Node Agent does not make central cluster scheduling decisions. It reports hardware capacity (`NodeCapabilities`) and executes workloads assigned by the central controller.
2. **OS Service Integration:** Isolated platform implementation (`internal/service/windows.go` and `linux.go`) allowing native Windows Service and Linux systemd auto-start on boot without terminal session requirement.
3. **Dynamic NVML Fallback:** Hardware Abstraction Layer uses dynamic library loading (`nvml.dll` / `libnvidia-ml.so`). If NVIDIA GPU is missing, agent gracefully continues with `gpu_available = false`.
4. **Persistent Node Identity:** Secure random UUID generated upon initial setup and stored in `identity/node_id.json`. Identity persists across PC reboots.
5. **Inference Runtime Abstraction:** `InferenceRuntime` interface cleanly isolates execution engines (e.g. `MockInferenceRuntime`, `OllamaRuntime`). Core codebase remains runtime-agnostic.
6. **Safety Guard Subsystem:** Monitors RAM usage, GPU temperature, and disk space. Transitions node to `DEGRADED` state when thresholds are violated to prevent hardware damage.
