# Controller Integration Contract

**Document Version:** 1.0.0  
**Target:** Python Cluster Controller $\leftrightarrow$ Go Node Agent & Distributed Runtime  
**Protocol:** gRPC Protobuf (`node_controller.v1.NodeControllerService`) & HTTP gRPC-Web Fallback  

---

## 1. Protocol Architecture & Ingress

The Python Cluster Controller exposes gRPC endpoints over HTTP/2 (standard gRPC) and HTTP/1.1 gRPC-Web fallback endpoints for edge environments (Render PaaS):

```
gRPC Standard Ingress:  port 50051 (HTTP/2)
gRPC Fallback Ingress:  port 8000/443 (HTTP/1.1 POST /node_controller.v1.NodeControllerService/{RPC})
```

---

## 2. Authentication & Enrollment

### Enrollment Token Flow
1. Node Agent connects with pre-configured `--enrollment-token` (Default: `dev-enrollment-token`).
2. `EnrollNode(EnrollmentRequest)` RPC validates the enrollment token.
3. Controller responds with `EnrollmentResponse`:
   - `success`: boolean
   - `node_id`: UUIDv4 string (persisted by agent to `C:\ProgramData\DistributedAI\node_identity.json`)
   - `auth_token`: Cryptographic session token for subsequent RPC requests.

---

## 3. Node Registration & Lifecycle Messages

### `RegisterNode(RegistrationRequest)`
Node Agent sends hardware and capabilities payload upon startup:

```protobuf
message RegistrationRequest {
  string node_id = 1;
  string hostname = 2;
  string ip_address = 3;
  string agent_version = 4;
  string os_info = 5;
  string cpu_info = 6;
  int64 total_ram_mb = 7;
  repeated GPUInfo gpus = 8;
  repeated string supported_models = 9;
}
```

Controller responds with `RegistrationResponse`:
- `registered`: boolean
- `heartbeat_interval_seconds`: uint32 (default: 5)
- `assigned_role`: string (`PRIMARY` or `WORKER`)

---

## 4. Heartbeat & Telemetry Streams

### Heartbeat Loop (`SendHeartbeat`)
- **Interval:** 5 seconds
- **Request:** `HeartbeatRequest { string node_id = 1; NodeState state = 2; int64 timestamp = 3; }`
- **Response:** `HeartbeatResponse { bool acknowledged = 1; int64 timestamp = 2; }`
- **Thresholds:**
  - $> 15s$ missing heartbeats $\rightarrow$ `NODE_DEGRADED`
  - $> 30s$ missing heartbeats $\rightarrow$ `NODE_OFFLINE`

### Telemetry Payload (`SendTelemetry`)
- **Interval:** 10 seconds (or on state change)
- **Request:**
  ```protobuf
  message TelemetryRequest {
    string node_id = 1;
    float cpu_usage_percent = 2;
    int64 free_ram_mb = 3;
    repeated GPUTelemetry gpu_telemetry = 4; // utilization %, VRAM used/free, temp
    string current_runtime_state = 5;
  }
  ```

---

## 5. Control & Job Execution Stream (`StreamControl`)

Bidirectional gRPC stream: `rpc StreamControl(stream ControlUpstream) returns (stream ControlDownstream)`

### Upstream Messages (Agent $\rightarrow$ Controller):
1. `Heartbeat`
2. `JobStatusUpdate` (`QUEUED`, `RUNNING`, `STREAMING`, `COMPLETED`, `FAILED`, `CANCELLED`)
3. `TokenChunk` (`job_id`, `chunk`, `index`)
4. `FailureReport` (`error_code`, `message`, `failing_node_id`)

### Downstream Messages (Controller $\rightarrow$ Agent):
1. `ExecutionPlanAssignment`:
   ```protobuf
   message ExecutionPlanAssignment {
     string job_id = 1;
     string execution_id = 2;
     string model_id = 3;
     string prompt = 4;
     repeated NodeAssignment assignments = 5; // node_id, rpc_endpoint, vram_budget_mb, layer_split
     int32 max_tokens = 6;
     float temperature = 7;
   }
   ```
2. `CancelJob`: `CancelJobCommand { string job_id = 1; }`
3. `DrainNode`: `DrainNodeCommand { string node_id = 1; }`

---

## 6. Model Metadata Management

The controller tracks model manifests without storing heavy GGUF binary files.
- **Model Metadata Schema:** `model_id`, `name`, `architecture`, `quantization`, `file_size_bytes`, `required_vram_mb`, `sha256_checksum`.
- **Node Placement State:** `AVAILABLE`, `DOWNLOADING`, `VERIFYING`, `MISSING`.

---

## 7. Token Streaming Contract (Client API)

`POST /v1/chat/completions` with `"stream": true` returns OpenAI-compatible Server-Sent Events (SSE):

```http
HTTP/1.1 200 OK
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive

data: {"id":"chatcmpl-9912","object":"chat.completion.chunk","created":1725998400,"model":"qwen2.5-0.5b","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-9912","object":"chat.completion.chunk","created":1725998405,"model":"qwen2.5-0.5b","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":"stop"}]}

data: [DONE]
```

---

## 8. Failure Handling & Recovery Contract

- **Pre-execution Failure:** If a assigned node fails before inference starts, the controller's `RecoveryManager` re-invokes the `ExecutionPlanner` to assign remaining ready nodes.
- **In-flight Mid-generation Failure:** If a participating node loses connection during active matrix multiplication, the job transitions to `FAILED` with `RPC_WORKER_DISCONNECTED` error code and client receives HTTP 500 / SSE error chunk.
