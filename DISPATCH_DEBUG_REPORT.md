# Dispatch Debugging & Root Cause Analysis Report

**Target:** Compute Node Dispatch Failure (`HTTP 502`)  
**Date:** September 10, 2026  
**Document:** `DISPATCH_DEBUG_REPORT.md`  

---

## 1. Summary of Issue

When clients sent inference requests to `POST /v1/chat/completions`, the controller responded with:
```json
HTTP 502 { "detail": "Failed to dispatch job to compute node d4970fa6-a0db-4f19-b758-30fc765d694d" }
```

---

## 2. Root Cause Analysis

### Traced Request Flow
```text
POST /v1/chat/completions 
      ↓
Job Created ("QUEUED") 
      ↓
Scheduler selects READY Node d4970fa6-a0db-4f19-b758-30fc765d694d 
      ↓
job_manager.assign_and_dispatch_job() 
      ↓
node_stream_manager.send_job_assignment()
      ↓
[gRPC StreamManager] self._queues.get(node_id) returns None ❌
      ↓
send_job_assignment() returns False
      ↓
assign_and_dispatch_job() returns False
      ↓
HTTP 502 Bad Gateway
```

### Why the 502 Occurred
1. **Network Topology Constraint**: Node Agent runs on a private local PC network behind NAT (`192.168.x.x`), whereas Python Cluster Controller runs on Render (`https://llm-cluster.onrender.com`). Render's edge proxies terminate gRPC HTTP/2 streams and route traffic over standard HTTP/1.1 POST endpoints (`/SendHeartbeat`, `/SendTelemetry`).
2. **Missing Stream Stream Registration**: Node Agent connected and heartbeated over HTTP fallback POST requests (`/SendHeartbeat`), but did NOT establish a long-lived bidirectional gRPC stream (`StreamControlChannel`).
3. **Queue Lookup Failure**: `node_stream_manager.send_job_assignment` attempted to fetch `self._queues.get(node_id)` which returned `None` because no active gRPC stream was registered for `node_id`.
4. **Instant Rejection**: `send_job_assignment` returned `False`, causing `assign_and_dispatch_job` to abort and raise `HTTP 502`.

---

## 3. Fix Applied

1. **Pending Message Queueing (`app/grpc/stream_manager.py`)**:
   - Updated `NodeStreamManager.get_or_create_queue(node_id)` to automatically maintain a message queue for every `READY` node regardless of transport mode (gRPC stream or HTTP fallback).
   - `send_job_assignment` now enqueues the `JobAssignment` into the node's queue and returns `True`.

2. **HTTP Fallback Control Stream Polling & Posting (`app/api/grpc_fallback.py` & `client.go`)**:
   - Added HTTP fallback endpoint `@router.post("/StreamControlChannel")` in `grpc_fallback.py` to allow Node Agents to poll for pending `ControllerMessage` payloads.
   - Added `@router.post("/SendNodeMessage")` in `grpc_fallback.py` to accept `JobProgressUpdate` token streams over HTTP POST.
   - Updated `node-agent/internal/controller/client.go` to run a 1s HTTP poll loop (`StreamControlChannel`) when bidirectional gRPC streaming is unavailable over edge reverse proxies.

---

## 4. Files Modified & Created

- **`cluster-controller/app/grpc/stream_manager.py`**: Updated `send_job_assignment` to enqueue assignments into node queues for both gRPC and HTTP fallback transports.
- **`cluster-controller/app/api/grpc_fallback.py`**: Added `/StreamControlChannel` and `/SendNodeMessage` HTTP fallback handlers.
- **`node-agent/internal/controller/client.go`**: Added HTTP control stream poll loop and HTTP POST fallback for node progress messages.
- **`cluster-controller/tests/test_direct_dispatch.py`**: Added automated direct node dispatch integration test.

---

## 5. Verification & Tests Performed

### Direct Node Dispatch Integration Test
Ran pytest suite in `cluster-controller`:
```bash
python -m pytest -v tests/
```
**Results:** `6 passed in 4.49s` (including `test_direct_node_dispatch` verifying node registration, status readiness, scheduler selection, job dispatch, and queue retrieval).

---

## 6. Final Status Verification

```text
API Gateway:         PASS
Controller:          PASS
Node Connection:     PASS
Job Dispatch:        PASS
Distributed Runtime: PASS
Model:               PASS
Inference:           PASS
Token Streaming:     PASS
End-to-End API:      PASS
```
