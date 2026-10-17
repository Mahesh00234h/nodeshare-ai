# Deployment Guide — Render & Docker

---

## 1. Local Docker Compose Setup

Run full local stack with PostgreSQL and FastAPI Controller:

```bash
cd d:\decentralized\cluster-controller
docker-compose up --build -d
```

### `docker-compose.yml` Configuration:
```yaml
version: '3.8'

services:
  controller:
    build: .
    ports:
      - "8000:8000"
      - "50051:50051"
    environment:
      - DATABASE_URL=postgresql+asyncpg://postgres:postgres@postgres:5432/cluster_db
      - AI_API_KEY=dev-secret-key
      - NODE_ENROLLMENT_SECRET=dev-enrollment-token
    depends_on:
      - postgres

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=cluster_db
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=postgres
    ports:
      - "5432:5432"
```

---

## 2. Render Cloud Deployment

### Environment Variables on Render:
- `DATABASE_URL`: Connection string for Render PostgreSQL database.
- `AI_API_KEY`: Secret Bearer token for client HTTP REST access.
- `NODE_ENROLLMENT_SECRET`: Token required for physical PCs to enroll into the cluster.
- `LOG_LEVEL`: `INFO`

### Health Check Endpoint:
- Path: `/health`
- Expected Status: `200 OK`
