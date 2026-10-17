import asyncio
import logging
from typing import Dict, Optional
from app.grpc.generated import node_controller_pb2
from app.nodes.registry import node_registry

logger = logging.getLogger(__name__)

class NodeStreamManager:
    def __init__(self):
        self._queues: Dict[str, asyncio.Queue] = {}

    def get_or_create_queue(self, node_id: str) -> asyncio.Queue:
        if node_id not in self._queues:
            self._queues[node_id] = asyncio.Queue()
            logger.info(f"[gRPC StreamManager] Created message queue for node {node_id}")
        return self._queues[node_id]

    def register_stream(self, node_id: str) -> asyncio.Queue:
        q = self.get_or_create_queue(node_id)
        logger.info(f"[gRPC StreamManager] Registered active control stream for node {node_id}")
        return q

    def unregister_stream(self, node_id: str):
        # Retain queue in case HTTP fallback polling is active
        logger.info(f"[gRPC StreamManager] Detached stream for node {node_id} (queue preserved)")

    async def get_pending_message(self, node_id: str, timeout_seconds: float = 0.1) -> Optional[node_controller_pb2.ControllerMessage]:
        q = self._queues.get(node_id)
        if not q:
            return None
        try:
            return await asyncio.wait_for(q.get(), timeout=timeout_seconds)
        except asyncio.TimeoutError:
            return None

    async def send_job_assignment(
        self,
        node_id: str,
        job_id: str,
        job_type: str,
        model_id: str,
        execution_id: str,
        payload: str,
        priority: int = 1,
        required_ram_bytes: int = 0,
        required_vram_bytes: int = 0
    ) -> bool:
        q = self.get_or_create_queue(node_id)

        assignment_msg = node_controller_pb2.ControllerMessage(
            message_id=f"msg-{job_id}",
            job_assignment=node_controller_pb2.JobAssignment(
                job_id=job_id,
                job_type=job_type,
                model_id=model_id,
                execution_id=execution_id,
                partition_id="partition-0",
                payload=payload,
                priority=priority,
                required_ram_bytes=required_ram_bytes,
                required_vram_bytes=required_vram_bytes
            )
        )
        await q.put(assignment_msg)
        logger.info(f"[gRPC StreamManager] Enqueued job assignment for node {node_id} (job_id={job_id}, queue_size={q.qsize()})")
        return True

    async def send_job_cancel(self, node_id: str, job_id: str, reason: str = ""):
        q = self._queues.get(node_id)
        if q:
            cancel_msg = node_controller_pb2.ControllerMessage(
                message_id=f"msg-cancel-{job_id}",
                job_cancel=node_controller_pb2.JobCancel(job_id=job_id, reason=reason)
            )
            await q.put(cancel_msg)

    async def send_drain_command(self, node_id: str, reason: str = ""):
        q = self._queues.get(node_id)
        if q:
            drain_msg = node_controller_pb2.ControllerMessage(
                message_id=f"msg-drain-{node_id}",
                drain_command=node_controller_pb2.DrainCommand(reason=reason, force=False)
            )
            await q.put(drain_msg)

node_stream_manager = NodeStreamManager()
