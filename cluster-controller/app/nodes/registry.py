import datetime
import uuid
import logging
from typing import Optional, List, Dict, Any
from sqlalchemy import select, delete
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.database.models import NodeModel, NodeCapabilitiesModel, NodeGPUModel, NodeInstalledModel
from app.nodes.capabilities import NodeCapabilityReport
from app.events.publisher import event_publisher
from app.events.types import ClusterEventType
from app.core.security import verify_enrollment_token

logger = logging.getLogger(__name__)

class NodeRegistry:
    async def enroll_node(self, session: AsyncSession, node_id: str, enrollment_token: str, hostname: str, agent_version: str) -> Dict[str, Any]:
        if not verify_enrollment_token(enrollment_token):
            return {"success": False, "message": "Invalid enrollment token"}

        stmt = select(NodeModel).where(NodeModel.node_id == node_id)
        res = await session.execute(stmt)
        existing_node = res.scalar_one_or_none()

        if existing_node:
            # Re-enrollment for existing node
            existing_node.last_seen_at = datetime.datetime.utcnow()
            existing_node.updated_at = datetime.datetime.utcnow()
            await session.commit()
            return {
                "success": True,
                "auth_token": existing_node.auth_token,
                "node_secret": "secret-token-key",
                "message": "Node re-enrolled successfully"
            }

        new_auth_token = f"node-auth-{uuid.uuid4().hex}"
        new_node = NodeModel(
            node_id=node_id,
            auth_token=new_auth_token,
            hostname=hostname,
            status="CONNECTING",
            agent_version=agent_version,
            first_seen_at=datetime.datetime.utcnow(),
            last_seen_at=datetime.datetime.utcnow(),
            last_heartbeat_at=datetime.datetime.utcnow(),
        )
        session.add(new_node)
        await session.commit()

        await event_publisher.publish(
            ClusterEventType.NODE_ENROLLED,
            f"Node {node_id} enrolled successfully",
            node_id=node_id
        )

        return {
            "success": True,
            "auth_token": new_auth_token,
            "node_secret": "secret-token-key",
            "message": "Node enrolled successfully"
        }

    async def register_node(self, session: AsyncSession, report: NodeCapabilityReport, auth_token: str) -> Dict[str, Any]:
        stmt = select(NodeModel).where(NodeModel.node_id == report.node_id).options(
            selectinload(NodeModel.capabilities),
            selectinload(NodeModel.gpus)
        )
        res = await session.execute(stmt)
        node = res.scalar_one_or_none()

        now = datetime.datetime.utcnow()
        if not node:
            node = NodeModel(
                node_id=report.node_id,
                auth_token=auth_token,
                hostname=report.hostname,
                status="READY",
                agent_version=report.agent_version,
                first_seen_at=now,
                last_seen_at=now,
                last_heartbeat_at=now,
            )
            session.add(node)
            await session.flush()
        elif node.auth_token != auth_token:
            node.auth_token = auth_token
        node.hostname = report.hostname
        node.os = report.os
        node.os_version = report.os_version
        node.architecture = report.architecture
        node.agent_version = report.agent_version
        node.protocol_version = report.protocol_version
        node.status = "READY"
        node.last_seen_at = now
        node.last_heartbeat_at = now
        node.registered_at = now
        node.updated_at = now

        # Update Capabilities
        caps_stmt = select(NodeCapabilitiesModel).where(NodeCapabilitiesModel.node_id == node.node_id)
        caps_res = await session.execute(caps_stmt)
        caps = caps_res.scalar_one_or_none()
        if not caps:
            caps = NodeCapabilitiesModel(node_id=node.node_id)
            session.add(caps)

        caps.cpu_model = report.cpu.model
        caps.cpu_vendor = report.cpu.vendor
        caps.cpu_cores = report.cpu.cores
        caps.cpu_threads = report.cpu.threads
        caps.ram_total_bytes = report.memory.total_bytes
        caps.storage_total_bytes = report.storage.total_bytes
        caps.storage_available_bytes = report.storage.available_bytes
        caps.estimated_bandwidth = report.network.estimated_bandwidth
        caps.runtime_capabilities = report.runtime_capabilities

        # Update GPUs
        await session.execute(
            delete(NodeGPUModel).where(NodeGPUModel.node_id == node.node_id)
        )

        for gpu in report.gpus:
            gpu_row = NodeGPUModel(
                node_id=node.node_id,
                vendor=gpu.vendor,
                name=gpu.name,
                vram_total_bytes=gpu.vram_total_bytes,
                vram_used_bytes=gpu.vram_used_bytes,
                vram_avail_bytes=gpu.vram_available_bytes,
                utilization_percent=gpu.utilization_percent,
                temperature_c=gpu.temperature_c,
                driver_version=gpu.driver_version,
                compute_capability=gpu.compute_capability,
            )
            session.add(gpu_row)

        await session.commit()

        await event_publisher.publish(
            ClusterEventType.NODE_REGISTERED,
            f"Node {node.node_id} ({node.hostname}) registered capabilities",
            node_id=node.node_id,
            metadata={
                "cpu_cores": report.cpu.cores,
                "ram_gb": round(report.memory.total_bytes / (1024**3), 2),
                "gpu_count": len(report.gpus),
            }
        )
        await event_publisher.publish(
            ClusterEventType.NODE_READY,
            f"Node {node.node_id} is READY",
            node_id=node.node_id
        )

        return {
            "registered": True,
            "cluster_node_id": f"cluster-{node.node_id[:8]}",
            "heartbeat_interval_seconds": 5,
            "telemetry_interval_seconds": 5,
            "message": "Node registration accepted"
        }

    async def get_node(self, session: AsyncSession, node_id: str) -> Optional[NodeModel]:
        stmt = select(NodeModel).where(NodeModel.node_id == node_id).options(
            selectinload(NodeModel.capabilities),
            selectinload(NodeModel.gpus),
            selectinload(NodeModel.installed_models)
        )
        res = await session.execute(stmt)
        return res.scalar_one_or_none()

    async def list_nodes(self, session: AsyncSession) -> List[NodeModel]:
        stmt = select(NodeModel).options(
            selectinload(NodeModel.capabilities),
            selectinload(NodeModel.gpus),
            selectinload(NodeModel.installed_models)
        )
        res = await session.execute(stmt)
        return list(res.scalars().all())

    async def set_node_status(self, session: AsyncSession, node_id: str, new_status: str) -> bool:
        node = await self.get_node(session, node_id)
        if not node:
            return False
        
        old_status = node.status
        node.status = new_status
        node.updated_at = datetime.datetime.utcnow()
        await session.commit()

        if old_status != new_status:
            event_type = ClusterEventType.NODE_READY if new_status == "READY" else (
                ClusterEventType.NODE_DRAINING if new_status == "DRAINING" else (
                    ClusterEventType.NODE_OFFLINE if new_status == "OFFLINE" else ClusterEventType.NODE_DEGRADED
                )
            )
            await event_publisher.publish(
                event_type,
                f"Node {node_id} status changed from {old_status} to {new_status}",
                node_id=node_id
            )
        return True

node_registry = NodeRegistry()
