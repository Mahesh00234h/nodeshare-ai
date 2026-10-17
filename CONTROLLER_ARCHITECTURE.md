# Controller Architecture Specification

**System:** Python Distributed AI Cluster Controller  
**Location:** `d:\decentralized\cluster-controller`  
**Deployment Target:** Render PaaS (`https://llm-cluster.onrender.com:443`)  

---

## 1. High-Level Architecture

The Python Cluster Controller serves as the central orchestration brain for the private distributed compute network. It manages node enrollment, health state reconciliation, heterogeneous multi-node scheduling, and provides an OpenAI-compatible REST API interface.

```
                           ┌──────────────────────┐
                           │     PROJECT SCOUT    │
                           │   OpenAI API Client  │
                           └──────────┬───────────┘
                                      │ HTTP REST (/v1/chat/completions)
                                      ▼
                 ┌────────────────────────────────────────┐
                 │       PYTHON CLUSTER CONTROLLER        │
                 │                (Render)                │
                 │                                        │
                 │  ┌──────────────────────────────────┐  │
                 │  │ API Gateway & Security Router    │  │
                 │  └────────────────┬─────────────────┘  │
                 │                   │                    │
                 │  ┌────────────────▼─────────────────┐  │
                 │  │ Job Manager & Token Multiplexer  │  │
                 │  └────────────────┬─────────────────┘  │
                 │                   │                    │
                 │  ┌────────────────▼─────────────────┐  │
                 │  │ Heterogeneous Capacity Scheduler │  │
                 │  └────────────────┬─────────────────┘  │
                 │                   │                    │
                 │  ┌────────────────▼─────────────────┐  │
                 │  │ Distributed Execution            │  │
                 │  │ Orchestrator                     │  │
                 │  └────────────────┬─────────────────┘  │
                 │                   │                    │
                 │  ┌────────────────▼─────────────────┐  │
                 │  │ Health Worker & Recovery Manager │  │
                 │  └──────────────────────────────────┘  │
                 │                                        │
                 │          PostgreSQL Database           │
                 └───────────────────┬────────────────────┘
                                     │ gRPC / HTTP Bridge Stream
                     ┌───────────────┼───────────────┐
                     ▼               ▼               ▼
               ┌───────────┐   ┌───────────┐   ┌───────────┐
               │  PC NODE 1│   │  PC NODE 2│   │  PC NODE 3│
               │ Go Agent  │   │ Go Agent  │   │ Go Agent  │
               │ llama.cpp │   │ llama.cpp │   │ llama.cpp │
               └───────────┘   └───────────┘   └───────────┘
```

---

## 2. Core Service Modules

| Module | File | Purpose |
| :--- | :--- | :--- |
| **Node Registry** | `app/nodes/registry.py` | Tracks active nodes, RAM/VRAM capacity, OS, and hardware metrics. |
| **Heterogeneous Scheduler** | `app/scheduler/scheduler.py` | Scores candidate compute nodes based on available VRAM headroom, GPU model, and current load. |
| **Job Manager** | `app/jobs/manager.py` | Manages job lifecycle (`CREATED` $\rightarrow$ `QUEUED` $\rightarrow$ `RUNNING` $\rightarrow$ `STREAMING` $\rightarrow$ `COMPLETED`). |
| **Execution Orchestrator** | `app/services/execution_orchestrator.py` | Calculates single-node vs multi-node layer-split execution plans. |
| **gRPC Fallback Bridge** | `app/api/grpc_fallback.py` | Handles gRPC binary frame parsing over HTTP/1.1 for edge proxies. |
| **Health Monitor Worker** | `app/workers/health_worker.py` | Background task checking missed heartbeats (15s Warning, 30s Offline). |

---

## 3. Deployment Topology

- **Render Service**: Web Service (Python 3.12 + FastAPI + Uvicorn)
- **Database**: PostgreSQL Persistent Database (`DATABASE_URL`)
- **Node Storage**: Compute PC local storage (`C:\ProgramData\DistributedAI\models\`). Zero heavy model binaries stored on Render.
