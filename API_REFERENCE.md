# API Reference Specification

**System:** Python Cluster Controller API  
**Base URL:** `https://llm-cluster.onrender.com`  

---

## 1. Public OpenAI-Compatible Inference APIs

### `POST /v1/chat/completions`
Generates chat completions using active cluster compute nodes.

**Headers:**
- `Authorization`: `Bearer dev-secret-key` (or configured `AI_API_KEY`)
- `Content-Type`: `application/json`

**Request Body:**
```json
{
  "model": "qwen2.5-0.5b-instruct-q4_k_m.gguf",
  "messages": [
    {"role": "user", "content": "Explain quantum computing in one sentence."}
  ],
  "temperature": 0.7,
  "max_tokens": 150,
  "stream": true
}
```

**Response Stream (SSE):**
```text
data: {"id":"chatcmpl-9912","object":"chat.completion.chunk","created":1725998400,"model":"qwen2.5-0.5b","choices":[{"index":0,"delta":{"content":"Quantum"},"finish_reason":null}]}

data: {"id":"chatcmpl-9912","object":"chat.completion.chunk","created":1725998401,"model":"qwen2.5-0.5b","choices":[{"index":0,"delta":{"content":" computing..."},"finish_reason":"stop"}]}

data: [DONE]
```

---

### `GET /v1/models`
Lists all GGUF models registered across active cluster nodes.

---

### `POST /v1/embeddings`
Generates text embedding vectors.

---

## 2. Admin Management APIs (`/api/admin/*`)

### Nodes Management
- `GET /api/admin/nodes`: List all enrolled nodes and status.
- `GET /api/admin/nodes/{node_id}`: Inspect node hardware & telemetry.
- `POST /api/admin/nodes/{node_id}/drain`: Mark node `DRAINING`.

### Jobs Management
- `GET /api/admin/jobs`: List active and completed jobs.
- `POST /api/admin/jobs/{job_id}/cancel`: Cancel active inference job.

### Cluster Health
- `GET /api/admin/cluster/health`: Summary of cluster capacity & node count.

---

## 3. System Health Check APIs

- `GET /health`: Controller health status (`200 OK`).
- `GET /ready`: Readiness check (`200 OK` when database connected).
- `GET /version`: Version information.
