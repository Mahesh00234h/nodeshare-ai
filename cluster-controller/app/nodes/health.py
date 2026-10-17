import datetime
import logging
from typing import Dict, Any
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from app.database.models import NodeModel
from app.events.publisher import event_publisher
from app.events.types import ClusterEventType

logger = logging.getLogger(__name__)

class HeartbeatService:
    async def process_heartbeat(
        self,
        session: AsyncSession,
        node_id: str,
        auth_token: str,
        node_state: str,
        active_job_ids: list,
        cpu_util: float,
        ram_util: float,
        gpu_util: float,
        vram_util: float,
        gpu_temp: int,
        uptime: int,
    ) -> Dict[str, Any]:
        stmt = select(NodeModel).where(NodeModel.node_id == node_id)
        res = await session.execute(stmt)
        node = res.scalar_one_or_none()

        if not node or node.auth_token != auth_token:
            return {"acknowledged": False, "desired_state": "RE_REGISTER", "message": "Node authentication failed"}

        now = datetime.datetime.utcnow()
        node.last_heartbeat_at = now
        node.last_seen_at = now
        
        desired_state = "READY"
        if node.status == "DRAINING":
            desired_state = "DRAIN"
        elif node_state == "DEGRADED":
            node.status = "DEGRADED"
        elif len(active_job_ids) > 0:
            node.status = "BUSY"
        elif node.status in ["OFFLINE", "CONNECTING", "DEGRADED", "UNKNOWN"]:
            node.status = "READY"
            await event_publisher.publish(ClusterEventType.NODE_ONLINE, f"Node {node_id} reconnected and online", node_id=node_id)
        elif node_state == "READY" and node.status != "BUSY":
            node.status = "READY"

        await session.commit()

        return {
            "acknowledged": True,
            "desired_state": desired_state,
            "message": "Heartbeat ACK"
        }

heartbeat_service = HeartbeatService()
