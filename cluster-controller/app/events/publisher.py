import asyncio
import datetime
import logging
from typing import Dict, Any, Set
from app.events.types import ClusterEventType

logger = logging.getLogger(__name__)

class EventPublisher:
    def __init__(self):
        self._subscribers: Set[asyncio.Queue] = set()

    def subscribe(self) -> asyncio.Queue:
        q = asyncio.Queue(maxsize=100)
        self._subscribers.add(q)
        return q

    def unsubscribe(self, q: asyncio.Queue):
        self._subscribers.discard(q)

    async def publish(self, event_type: ClusterEventType, message: str, node_id: str = None, job_id: str = None, metadata: Dict[str, Any] = None):
        event_payload = {
            "event_type": event_type.value,
            "timestamp": datetime.datetime.utcnow().isoformat(),
            "node_id": node_id,
            "job_id": job_id,
            "message": message,
            "metadata": metadata or {}
        }
        
        logger.info(f"[EVENT] {event_type.value}: {message}", extra={"event": event_type.value, "node_id": node_id, "job_id": job_id})

        dead_queues = set()
        for q in self._subscribers:
            try:
                q.put_nowait(event_payload)
            except asyncio.QueueFull:
                dead_queues.add(q)

        for dq in dead_queues:
            self._subscribers.discard(dq)

event_publisher = EventPublisher()
