import datetime
from typing import Optional, List
from sqlalchemy import (
    String, Integer, BigInteger, Float, Boolean, DateTime, Text, JSON, ForeignKey, UniqueConstraint, Index
)
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column, relationship

class Base(DeclarativeBase):
    pass

class NodeModel(Base):
    __tablename__ = "nodes"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    node_id: Mapped[str] = mapped_column(String(64), unique=True, index=True, nullable=False)
    auth_token: Mapped[str] = mapped_column(String(128), nullable=False)
    hostname: Mapped[str] = mapped_column(String(128), nullable=False, default="")
    status: Mapped[str] = mapped_column(String(32), index=True, nullable=False, default="CONNECTING")
    os: Mapped[str] = mapped_column(String(64), default="")
    os_version: Mapped[str] = mapped_column(String(64), default="")
    architecture: Mapped[str] = mapped_column(String(32), default="")
    agent_version: Mapped[str] = mapped_column(String(32), default="1.0.0")
    protocol_version: Mapped[str] = mapped_column(String(16), default="v1")
    
    first_seen_at: Mapped[datetime.datetime] = mapped_column(DateTime, default=datetime.datetime.utcnow)
    last_seen_at: Mapped[datetime.datetime] = mapped_column(DateTime, default=datetime.datetime.utcnow)
    last_heartbeat_at: Mapped[datetime.datetime] = mapped_column(DateTime, index=True, default=datetime.datetime.utcnow)
    registered_at: Mapped[Optional[datetime.datetime]] = mapped_column(DateTime, nullable=True)
    updated_at: Mapped[datetime.datetime] = mapped_column(DateTime, default=datetime.datetime.utcnow, onupdate=datetime.datetime.utcnow)

    # Relationships
    capabilities: Mapped[Optional["NodeCapabilitiesModel"]] = relationship("NodeCapabilitiesModel", back_populates="node", cascade="all, delete-orphan", uselist=False)
    gpus: Mapped[List["NodeGPUModel"]] = relationship("NodeGPUModel", back_populates="node", cascade="all, delete-orphan")
    telemetry: Mapped[List["NodeTelemetryModel"]] = relationship("NodeTelemetryModel", back_populates="node", cascade="all, delete-orphan")
    installed_models: Mapped[List["NodeInstalledModel"]] = relationship("NodeInstalledModel", back_populates="node", cascade="all, delete-orphan")
    assignments: Mapped[List["JobAssignmentModel"]] = relationship("JobAssignmentModel", back_populates="node")

class NodeCapabilitiesModel(Base):
    __tablename__ = "node_capabilities"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    node_id: Mapped[str] = mapped_column(String(64), ForeignKey("nodes.node_id", ondelete="CASCADE"), unique=True, nullable=False)
    
    cpu_model: Mapped[str] = mapped_column(String(128), default="")
    cpu_vendor: Mapped[str] = mapped_column(String(64), default="")
    cpu_cores: Mapped[int] = mapped_column(Integer, default=1)
    cpu_threads: Mapped[int] = mapped_column(Integer, default=1)
    
    ram_total_bytes: Mapped[int] = mapped_column(BigInteger, default=0)
    storage_total_bytes: Mapped[int] = mapped_column(BigInteger, default=0)
    storage_available_bytes: Mapped[int] = mapped_column(BigInteger, default=0)
    estimated_bandwidth: Mapped[int] = mapped_column(BigInteger, default=0)
    runtime_capabilities: Mapped[list] = mapped_column(JSON, default=list)

    node: Mapped["NodeModel"] = relationship("NodeModel", back_populates="capabilities")

class NodeGPUModel(Base):
    __tablename__ = "node_gpus"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    node_id: Mapped[str] = mapped_column(String(64), ForeignKey("nodes.node_id", ondelete="CASCADE"), nullable=False)
    
    vendor: Mapped[str] = mapped_column(String(64), default="NVIDIA")
    name: Mapped[str] = mapped_column(String(128), default="")
    vram_total_bytes: Mapped[int] = mapped_column(BigInteger, default=0)
    vram_used_bytes: Mapped[int] = mapped_column(BigInteger, default=0)
    vram_avail_bytes: Mapped[int] = mapped_column(BigInteger, default=0)
    utilization_percent: Mapped[float] = mapped_column(Float, default=0.0)
    temperature_c: Mapped[int] = mapped_column(Integer, default=0)
    driver_version: Mapped[str] = mapped_column(String(64), default="")
    compute_capability: Mapped[str] = mapped_column(String(32), default="")

    node: Mapped["NodeModel"] = relationship("NodeModel", back_populates="gpus")

class NodeTelemetryModel(Base):
    __tablename__ = "node_telemetry"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    node_id: Mapped[str] = mapped_column(String(64), ForeignKey("nodes.node_id", ondelete="CASCADE"), index=True, nullable=False)
    timestamp: Mapped[datetime.datetime] = mapped_column(DateTime, index=True, default=datetime.datetime.utcnow)
    
    cpu_utilization: Mapped[float] = mapped_column(Float, default=0.0)
    ram_utilization: Mapped[float] = mapped_column(Float, default=0.0)
    ram_used_bytes: Mapped[int] = mapped_column(BigInteger, default=0)
    gpu_utilization: Mapped[float] = mapped_column(Float, default=0.0)
    vram_utilization: Mapped[float] = mapped_column(Float, default=0.0)
    gpu_temperature: Mapped[int] = mapped_column(Integer, default=0)
    active_jobs: Mapped[int] = mapped_column(Integer, default=0)
    queued_jobs: Mapped[int] = mapped_column(Integer, default=0)
    uptime_seconds: Mapped[int] = mapped_column(BigInteger, default=0)

    node: Mapped["NodeModel"] = relationship("NodeModel", back_populates="telemetry")

class NodeInstalledModel(Base):
    __tablename__ = "node_models"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    node_id: Mapped[str] = mapped_column(String(64), ForeignKey("nodes.node_id", ondelete="CASCADE"), nullable=False)
    model_id: Mapped[str] = mapped_column(String(128), index=True, nullable=False)
    version: Mapped[str] = mapped_column(String(32), default="1.0.0")
    status: Mapped[str] = mapped_column(String(32), default="READY")
    checksum: Mapped[str] = mapped_column(String(128), default="")
    size_bytes: Mapped[int] = mapped_column(BigInteger, default=0)
    last_verified_at: Mapped[datetime.datetime] = mapped_column(DateTime, default=datetime.datetime.utcnow)

    node: Mapped["NodeModel"] = relationship("NodeModel", back_populates="installed_models")

class JobModel(Base):
    __tablename__ = "jobs"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    job_id: Mapped[str] = mapped_column(String(64), unique=True, index=True, nullable=False)
    request_id: Mapped[str] = mapped_column(String(64), default="")
    model_id: Mapped[str] = mapped_column(String(128), index=True, nullable=False)
    status: Mapped[str] = mapped_column(String(32), index=True, default="QUEUED")
    priority: Mapped[int] = mapped_column(Integer, default=1)
    payload: Mapped[str] = mapped_column(Text, default="")
    execution_id: Mapped[str] = mapped_column(String(64), default="")
    error_message: Mapped[str] = mapped_column(Text, default="")
    
    created_at: Mapped[datetime.datetime] = mapped_column(DateTime, index=True, default=datetime.datetime.utcnow)
    started_at: Mapped[Optional[datetime.datetime]] = mapped_column(DateTime, nullable=True)
    completed_at: Mapped[Optional[datetime.datetime]] = mapped_column(DateTime, nullable=True)

    assignments: Mapped[List["JobAssignmentModel"]] = relationship("JobAssignmentModel", back_populates="job", cascade="all, delete-orphan")

class JobAssignmentModel(Base):
    __tablename__ = "job_assignments"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    job_id: Mapped[str] = mapped_column(String(64), ForeignKey("jobs.job_id", ondelete="CASCADE"), index=True, nullable=False)
    node_id: Mapped[str] = mapped_column(String(64), ForeignKey("nodes.node_id", ondelete="CASCADE"), index=True, nullable=False)
    
    role: Mapped[str] = mapped_column(String(32), default="worker")
    partition_id: Mapped[str] = mapped_column(String(64), default="partition-0")
    status: Mapped[str] = mapped_column(String(32), default="ASSIGNED")
    failure_reason: Mapped[str] = mapped_column(Text, default="")
    
    assigned_at: Mapped[datetime.datetime] = mapped_column(DateTime, default=datetime.datetime.utcnow)
    started_at: Mapped[Optional[datetime.datetime]] = mapped_column(DateTime, nullable=True)
    completed_at: Mapped[Optional[datetime.datetime]] = mapped_column(DateTime, nullable=True)

    job: Mapped["JobModel"] = relationship("JobModel", back_populates="assignments")
    node: Mapped["NodeModel"] = relationship("NodeModel", back_populates="assignments")

class RegisteredModel(Base):
    __tablename__ = "models"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    model_id: Mapped[str] = mapped_column(String(128), unique=True, index=True, nullable=False)
    name: Mapped[str] = mapped_column(String(128), nullable=False)
    version: Mapped[str] = mapped_column(String(32), default="1.0.0")
    parameter_count: Mapped[str] = mapped_column(String(32), default="7B")
    quantization: Mapped[str] = mapped_column(String(32), default="q4_0")
    size_bytes: Mapped[int] = mapped_column(BigInteger, default=0)
    sha256_checksum: Mapped[str] = mapped_column(String(128), default="")
    created_at: Mapped[datetime.datetime] = mapped_column(DateTime, default=datetime.datetime.utcnow)

class ClusterEventModel(Base):
    __tablename__ = "cluster_events"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    event_type: Mapped[str] = mapped_column(String(64), index=True, nullable=False)
    node_id: Mapped[Optional[str]] = mapped_column(String(64), nullable=True)
    job_id: Mapped[Optional[str]] = mapped_column(String(64), nullable=True)
    message: Mapped[str] = mapped_column(Text, default="")
    metadata_json: Mapped[dict] = mapped_column(JSON, default=dict)
    timestamp: Mapped[datetime.datetime] = mapped_column(DateTime, index=True, default=datetime.datetime.utcnow)

class AuditLogModel(Base):
    __tablename__ = "audit_logs"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    action: Mapped[str] = mapped_column(String(64), nullable=False)
    actor: Mapped[str] = mapped_column(String(64), default="system")
    details: Mapped[dict] = mapped_column(JSON, default=dict)
    timestamp: Mapped[datetime.datetime] = mapped_column(DateTime, default=datetime.datetime.utcnow)
