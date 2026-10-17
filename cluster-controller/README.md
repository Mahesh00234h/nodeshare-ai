# Distributed AI Cluster Controller V1

Central Control Plane for heterogeneous compute node discovery, gRPC orchestration, resource monitoring, dynamic model scheduling, and OpenAI-compatible API serving.

## Features
- **Node Auto-Provisioning & Health Monitoring**: gRPC bidirectional streams with heartbeat timeouts and dynamic health state recovery (`READY`, `DEGRADED`, `OFFLINE`).
- **OpenAI-Compatible Inference API**: `/v1/chat/completions`, `/v1/models`, `/v1/embeddings` with real-time SSE token streaming.
- **Custom Dual Codec**: Dual JSON/Protobuf codec for zero-friction cross-language integration with Go Node Agents.
- **Deployable on Render & Docker**: Dockerfile & docker-compose included for PostgreSQL + Redis + FastAPI stack.

## Getting Started

### Local Execution
```bash
python -m uvicorn app.main:app --host 0.0.0.0 --port 8000
```
- REST API: `http://localhost:8000`
- gRPC Port: `50051`

### Running Tests
```bash
python -m pytest -v tests/
```
