import datetime
import logging
from typing import Dict, Any
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from app.database.models import NodeModel, NodeTelemetryModel

logger = logging.getLogger(__name__)

class TelemetryService:
    async def process_telemetry(
        self,
        session: AsyncSession,
        node_id: str,
        auth_token: str,
        timestamp: int,
        cpu_util: float,
        ram_util: float,
        ram_used_bytes: int,
        gpu_util: float,
        vram_util: float,
        gpu_temp: int,
        active_jobs: int,
        queued_jobs: int,
        uptime: int,
    ) -> bool:
        stmt = select(NodeModel).where(NodeModel.node_id == node_id)
        res = await session.execute(stmt)
        node = res.scalar_one_or_none()

        if not node or node.auth_token != auth_token:
            return False

        telemetry_row = NodeTelemetryModel(
            node_id=node_id,
            timestamp=datetime.datetime.utcnow(),
            cpu_utilization=cpu_util,
            ram_utilization=ram_util,
            ram_used_bytes=ram_used_bytes,
            gpu_utilization=gpu_util,
            vram_utilization=vram_util,
            gpu_temperature=gpu_temp,
            active_jobs=active_jobs,
            queued_jobs=queued_jobs,
            uptime_seconds=uptime,
        )
        session.add(telemetry_row)
        await session.commit()
        return True

    async def get_node_telemetry_history(self, session: AsyncSession, node_id: str, limit: int = 50) -> list:
        stmt = (
            select(NodeTelemetryModel)
            .where(NodeTelemetryModel.node_id == node_id)
            .order_by(NodeTelemetryModel.timestamp.desc())
            .limit(limit)
        )
        res = await session.execute(stmt)
        return list(res.scalars().all())

telemetry_service = TelemetryService()
