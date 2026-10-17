# Controller Protocol Specification (v1)

## Communication Protocol
- **Transport:** gRPC / HTTP2
- **Serialization:** Protocol Buffers (`api/proto/node_controller.proto`)
- **Protocol Version:** `v1`

## RPC Methods

### 1. `EnrollNode(EnrollmentRequest) returns (EnrollmentResponse)`
Performs initial one-time node enrollment using an `enrollment_token`. Returns node-specific `auth_token` and `node_secret` for subsequent connections.

### 2. `RegisterNode(RegistrationRequest) returns (RegistrationResponse)`
Registers full hardware capabilities (CPU model, physical cores, threads, RAM total, GPUs, VRAM, storage, network interfaces, agent version, runtime capabilities).

### 3. `SendHeartbeat(HeartbeatRequest) returns (HeartbeatResponse)`
Regular persistent heartbeat (default 5s). Reports node state (`READY`, `BUSY`, `DRAINING`, `DEGRADED`), uptime, active jobs, CPU %, RAM %, GPU %, VRAM %, and GPU temperature.

### 4. `SendTelemetry(TelemetryRequest) returns (TelemetryResponse)`
Periodic resource telemetry report (default 5s).

### 5. `StreamControlChannel(stream NodeMessage) returns (stream ControllerMessage)`
Bidirectional control stream for job assignments (`JobAssignment`), token streaming (`JobProgressUpdate`), job cancellation (`JobCancel`), model distribution commands (`ModelCommand`), and node draining (`DrainCommand`).
