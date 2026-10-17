# Execution Flow & Request Lifecycle

**System:** Distributed AI Infrastructure  
**Document:** End-to-End Inference Trace (`EXECUTION_FLOW.md`)  

---

## 1. Complete Inference Lifecycle Trace

```text
1. Client HTTP POST Request (/v1/chat/completions)
        │
        ▼
2. Controller Job Queue & Scheduler (app/scheduler)
        │
        ▼
3. Execution Planner Invoked (pkg/execution)
   (Evaluates node VRAM budgets, scoring, LAN IPs)
        │
        ▼
4. Target Nodes Selected (PC1 Primary + PC2 RPC Worker)
        │
        ▼
5. Model SHA256 Checksum Verification & Load (pkg/model)
        │
        ▼
6. Distributed Runtime Prepared (pkg/runtime)
   - PC2 starts llama-rpc-server on 0.0.0.0:50052
   - PC1 starts llama-cli with -rpc 192.168.1.50:50052 --tensor-split 0.5,0.5
        │
        ▼
7. Distributed Tensor Execution (Local LAN Ethernet)
        │
        ▼
8. Line-Buffered Token Streaming (pkg/llama/process.go)
        │
        ▼
9. Response Streamed Back to Client (OpenAI Server-Sent Events format)
```

---

## 2. Mermaid Sequence Diagram

```mermaid
sequenceDiagram
    autonumber
    actor Client as HTTP Client / User
    participant Controller as Python Controller (Render)
    participant Agent1 as PC 1 Node Agent (Primary)
    participant Runtime1 as PC 1 Distributed Runtime
    participant Llama1 as PC 1 llama-cli (GPU0)
    participant Agent2 as PC 2 Node Agent (Worker)
    participant Llama2 as PC 2 llama-rpc-server (0.0.0.0:50052)

    Client->>Controller: POST /v1/chat/completions (model="qwen2.5-0.5b", prompt="Hello", stream=true)
    Controller->>Controller: Register Job & Query Node Registry (PC1, PC2 READY)
    Controller->>Agent1: Assign Job & Execution Plan (LAYER_RPC split)
    
    rect rgb(235, 245, 255)
        note over Agent1,Agent2: 1. Setup Phase & Model Checksum Validation
        Agent1->>Runtime1: PrepareDistributedExecution(Plan)
        Runtime1->>Runtime1: Verify GGUF Model SHA256 Checksum
        Agent1->>Agent2: Signal RPC Worker Ready Check
        Agent2->>Llama2: Start llama-rpc-server -H 0.0.0.0 -p 50052 (LAN only)
        Llama2-->>Agent2: Listening on 192.168.1.50:50052
        Agent2-->>Agent1: Worker RPC Ready
    end

    rect rgb(240, 255, 240)
        note over Runtime1,Llama2: 2. Distributed Inference Execution (LAN Tensors)
        Runtime1->>Llama1: Launch llama-cli -m model.gguf -p "Hello" -rpc 192.168.1.50:50052 --tensor-split 0.5,0.5
        
        loop Tensor Execution over LAN
            Llama1->>Llama2: Remote Layer Offload & KV-Cache Synch (Port 50052)
            Llama2-->>Llama1: Computed Hidden States
        end
    end

    rect rgb(255, 245, 235)
        note over Llama1,Client: 3. Line-Buffered Token Streaming
        loop Token Stream Generation
            Llama1-->>Runtime1: Generated Token Chunk ("Hello")
            Runtime1-->>Agent1: Stream Token
            Agent1-->>Controller: gRPC Stream / Chunk
            Controller-->>Client: HTTP SSE data: {"choices":[{"delta":{"content":"Hello"}}]}
        end
    end

    Llama1-->>Runtime1: Execution Complete (Tokens: 42, Speed: 38.4 T/s)
    Runtime1-->>Agent1: Job State: COMPLETED
    Agent1-->>Controller: JobStatus (COMPLETED, metrics)
    Controller-->>Client: HTTP SSE data: [DONE]
```

---

## 3. Detailed Component Interaction Breakdown

| Step | Action | Executing Component | Source File / Handler |
| :--- | :--- | :--- | :--- |
| **1. Request Entry** | HTTP POST `/v1/chat/completions` | Python Controller | `cluster-controller/app/api/cluster.py` |
| **2. Job Scheduling** | Query active nodes, compute VRAM capacity | Controller Scheduler | `cluster-controller/app/scheduler/scorer.py` |
| **3. Plan Calculation** | Calculate `SINGLE_NODE` vs `LAYER_RPC` split ratio | Distributed Planner | `distributed-runtime/pkg/execution/plan.go` |
| **4. Security Validation** | Verify node IP belongs to private CIDR | LAN Security Guard | `distributed-runtime/pkg/communication/security.go` |
| **5. Model Verification** | Compute SHA256 checksum against manifest | Model Manager | `distributed-runtime/pkg/model/loader.go` |
| **6. RPC Startup** | Bind `llama-rpc-server` to LAN interface | RPC Supervisor | `distributed-runtime/pkg/llama/rpc.go` |
| **7. Engine Invocation** | Launch `llama-cli` process with `-rpc` flags | Process Supervisor | `distributed-runtime/pkg/llama/process.go` |
| **8. Token Streaming** | Line-buffer stdout, callback each token | Llama Manager | `distributed-runtime/pkg/llama/process.go` |
