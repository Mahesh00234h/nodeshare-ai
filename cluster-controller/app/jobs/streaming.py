import asyncio
import logging
from typing import Dict, Optional

logger = logging.getLogger(__name__)

class TokenStreamMultiplexer:
    def __init__(self):
        self._job_queues: Dict[str, asyncio.Queue] = {}

    def register_job_stream(self, job_id: str) -> asyncio.Queue:
        q = asyncio.Queue(maxsize=500)
        self._job_queues[job_id] = q
        return q

    def push_token(self, job_id: str, chunk: str, tokens: int, status: str, error_msg: str):
        q = self._job_queues.get(job_id)
        if q:
            try:
                q.put_nowait({
                    "chunk": chunk,
                    "tokens": tokens,
                    "status": status,
                    "error": error_msg,
                })
            except asyncio.QueueFull:
                logger.warning(f"Job stream queue full for {job_id}")

    def close_job_stream(self, job_id: str):
        q = self._job_queues.pop(job_id, None)
        if q:
            try:
                q.put_nowait({"status": "COMPLETED", "chunk": "", "tokens": 0, "error": ""})
            except asyncio.QueueFull:
                pass

token_multiplexer = TokenStreamMultiplexer()
