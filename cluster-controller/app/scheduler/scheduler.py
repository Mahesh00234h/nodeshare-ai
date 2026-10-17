import logging
from typing import List, Optional, Tuple
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.database.models import NodeModel
from app.scheduler.resources import NodeResources
from app.scheduler.scoring import score_node_for_job

logger = logging.getLogger(__name__)

class Scheduler:
    async def get_node_resources(self, session: AsyncSession, node_id: str) -> Optional[NodeResources]:
        stmt = select(NodeModel).where(NodeModel.node_id == node_id).options(
            selectinload(NodeModel.capabilities),
            selectinload(NodeModel.gpus),
            selectinload(NodeModel.installed_models)
        )
        res = await session.execute(stmt)
        node = res.scalar_one_or_none()
        if not node:
            return None
        return self._to_node_resources(node)

    async def list_available_node_resources(self, session: AsyncSession) -> List[NodeResources]:
        stmt = select(NodeModel).where(NodeModel.status.in_(["READY", "BUSY", "DEGRADED"])).options(
            selectinload(NodeModel.capabilities),
            selectinload(NodeModel.gpus),
            selectinload(NodeModel.installed_models)
        )
        res = await session.execute(stmt)
        nodes = res.scalars().all()
        return [self._to_node_resources(n) for n in nodes]

    async def select_node(
        self,
        session: AsyncSession,
        model_id: str,
        required_ram_bytes: int = 0,
        required_vram_bytes: int = 0,
        requires_gpu: bool = False,
    ) -> Tuple[Optional[NodeResources], Optional[str]]:
        nodes = await self.list_available_node_resources(session)
        if not nodes:
            return None, "No online compute nodes available in cluster"

        best_node: Optional[NodeResources] = None
        best_score = -1.0
        rejection_reasons = []

        for node in nodes:
            score, reason = score_node_for_job(
                node,
                model_id=model_id,
                required_ram_bytes=required_ram_bytes,
                required_vram_bytes=required_vram_bytes,
                requires_gpu=requires_gpu
            )
            if reason:
                rejection_reasons.append(f"Node {node.node_id}: {reason}")
            elif score > best_score:
                best_score = score
                best_node = node

        if not best_node:
            err_detail = "; ".join(rejection_reasons) if rejection_reasons else "Insufficient cluster capacity"
            return None, err_detail

        logger.info(f"[SCHEDULER] Selected node {best_node.node_id} (score={best_score:.2f}) for model {model_id}")
        return best_node, None

    def _to_node_resources(self, node: NodeModel) -> NodeResources:
        cpu_cores = node.capabilities.cpu_cores if node.capabilities else 1
        cpu_model = node.capabilities.cpu_model if node.capabilities else ""
        ram_total = node.capabilities.ram_total_bytes if node.capabilities else 0
        ram_avail = ram_total if ram_total > 0 else (16 * 1024 * 1024 * 1024)

        gpu_count = len(node.gpus)
        vram_total = sum(g.vram_total_bytes for g in node.gpus) if node.gpus else 0
        vram_avail = sum(g.vram_avail_bytes for g in node.gpus) if node.gpus else 0
        gpu_util = node.gpus[0].utilization_percent if node.gpus else 0.0

        installed = [m.model_id for m in node.installed_models if m.status == "READY"]
        runtimes = node.capabilities.runtime_capabilities if node.capabilities else []

        return NodeResources(
            node_id=node.node_id,
            hostname=node.hostname,
            status=node.status,
            cpu_cores=cpu_cores,
            ram_total_bytes=ram_total,
            ram_available_bytes=ram_avail,
            gpu_count=gpu_count,
            vram_total_bytes=vram_total,
            vram_available_bytes=vram_avail,
            gpu_utilization=gpu_util,
            installed_models=installed,
            runtime_capabilities=runtimes,
        )

scheduler = Scheduler()
