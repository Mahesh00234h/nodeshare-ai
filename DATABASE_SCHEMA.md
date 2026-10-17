# Database Schema Specification

**Engine:** PostgreSQL 15+ / SQLite3  
**ORM:** SQLAlchemy 2.x  
**Migration Tool:** Alembic  

---

## 1. Relational Schema Overview

```
               ┌────────────────┐
               │     nodes      │
               └───────┬────────┘
                       │
       ┌───────────────┼───────────────┐
       ▼               ▼               ▼
┌──────────────┐┌──────────────┐┌──────────────┐
│  node_gpus   ││node_telemetry││  node_models │
└──────────────┘└──────────────┘└──────────────┘

               ┌────────────────┐
               │     models     │
               └────────────────┘

               ┌────────────────┐
               │      jobs      │
               └───────┬────────┘
                       │
                       ▼
               ┌────────────────┐
               │   job_events   │
               └────────────────┘
```

---

## 2. Table Definitions

### Table: `nodes`
- `node_id` (VARCHAR(64), Primary Key)
- `hostname` (VARCHAR(128))
- `ip_address` (VARCHAR(64))
- `agent_version` (VARCHAR(32))
- `os_info` (VARCHAR(128))
- `status` (VARCHAR(32)) — `READY`, `BUSY`, `DEGRADED`, `OFFLINE`, etc.
- `cpu_cores` (INTEGER)
- `total_ram_mb` (BIGINT)
- `registered_at` (TIMESTAMP UTC)
- `last_seen_at` (TIMESTAMP UTC)

### Table: `node_gpus`
- `id` (INTEGER, Primary Key, Auto-increment)
- `node_id` (VARCHAR(64), Foreign Key $\rightarrow$ `nodes.node_id`)
- `gpu_index` (INTEGER)
- `gpu_name` (VARCHAR(128))
- `vram_total_mb` (BIGINT)
- `cuda_capability` (VARCHAR(32))

### Table: `models`
- `model_id` (VARCHAR(128), Primary Key)
- `name` (VARCHAR(128))
- `architecture` (VARCHAR(64))
- `quantization` (VARCHAR(32))
- `file_size_bytes` (BIGINT)
- `required_vram_mb` (BIGINT)

### Table: `jobs`
- `job_id` (VARCHAR(64), Primary Key)
- `model_id` (VARCHAR(128))
- `status` (VARCHAR(32)) — `QUEUED`, `PLANNING`, `RUNNING`, `COMPLETED`, `FAILED`
- `priority` (INTEGER)
- `created_at` (TIMESTAMP UTC)
- `completed_at` (TIMESTAMP UTC)
