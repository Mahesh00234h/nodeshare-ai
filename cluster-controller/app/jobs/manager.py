import datetime
import uuid
import logging
from typing import Optional, Dict, Any, List
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.database.models import JobModel, JobAssignmentModel
from app.events.publisher import event_publisher
from app.events.types import ClusterEventType
from app.grpc.stream_manager import node_stream_manager

logger = logging.getLogger(__name__)

class JobManager:
    async def create_job(
        self,
        session: AsyncSession,
        model_id: str,
        payload: str,
        priority: int = 1,
        request_id: str = None
    ) -> JobModel:
        job_id = f"job-{uuid.uuid4().hex[:12]}"
        exec_id = f"exec-{uuid.uuid4().hex[:8]}"
        
        job = JobModel(
            job_id=job_id,
            request_id=request_id or "",
            model_id=model_id,
            status="QUEUED",
            priority=priority,
            payload=payload,
            execution_id=exec_id,
            created_at=datetime.datetime.utcnow(),
        )
        session.add(job)
        await session.commit()

        await event_publisher.publish(
            ClusterEventType.JOB_CREATED,
            f"Job {job_id} created for model {model_id}",
            job_id=job_id
        )
        return job

    async def assign_and_dispatch_job(
        self,
        session: AsyncSession,
        job_id: str,
        node_id: str,
        required_ram_bytes: int = 0,
        required_vram_bytes: int = 0
    ) -> bool:
        stmt = select(JobModel).where(JobModel.job_id == job_id)
        res = await session.execute(stmt)
        job = res.scalar_one_or_none()

        if not job:
            return False

        now = datetime.datetime.utcnow()
        job.status = "ASSIGNED"
        job.started_at = now

        assignment = JobAssignmentModel(
            job_id=job_id,
            node_id=node_id,
            role="worker",
            partition_id="partition-0",
            status="ASSIGNED",
            assigned_at=now
        )
        session.add(assignment)
        await session.commit()

        # Send gRPC JobAssignment to connected node
        dispatched = await node_stream_manager.send_job_assignment(
            node_id=node_id,
            job_id=job.job_id,
            job_type="INFERENCE",
            model_id=job.model_id,
            execution_id=job.execution_id,
            payload=job.payload,
            priority=job.priority,
            required_ram_bytes=required_ram_bytes,
            required_vram_bytes=required_vram_bytes
        )

        if not dispatched:
            job.status = "FAILED"
            job.error_message = f"Node {node_id} control stream unavailable"
            job.completed_at = now
            assignment.status = "FAILED"
            assignment.failure_reason = "Control stream unavailable"
            await session.commit()
            return False

        job.status = "RUNNING"
        await session.commit()

        await event_publisher.publish(
            ClusterEventType.JOB_ASSIGNED,
            f"Job {job_id} assigned to node {node_id}",
            job_id=job_id,
            node_id=node_id
        )
        return True

    async def update_job_progress(
        self,
        session: AsyncSession,
        job_id: str,
        status_str: str,
        chunk: str,
        tokens: int,
        error_msg: str
    ):
        stmt = select(JobModel).where(JobModel.job_id == job_id).options(selectinload(JobModel.assignments))
        res = await session.execute(stmt)
        job = res.scalar_one_or_none()

        if not job:
            return

        now = datetime.datetime.utcnow()
        if status_str in ["RUNNING", "STREAMING"]:
            # Mark assignment as RUNNING so DB recovery doesn't re-dispatch
            for assign in job.assignments:
                if assign.status == "ASSIGNED":
                    assign.status = "RUNNING"
                    assign.started_at = now
            if job.status not in ["STREAMING", "RUNNING"]:
                job.status = "STREAMING"
            await session.commit()
            return
        elif status_str == "COMPLETED":
            job.status = "COMPLETED"
            job.completed_at = now
            for assign in job.assignments:
                if assign.status in ["ASSIGNED", "RUNNING"]:
                    assign.status = "COMPLETED"
                    assign.completed_at = now
            await event_publisher.publish(ClusterEventType.JOB_COMPLETED, f"Job {job_id} completed", job_id=job_id)
        elif status_str in ["FAILED", "REJECTED"]:
            job.status = "FAILED"
            job.error_message = error_msg
            job.completed_at = now
            for assign in job.assignments:
                if assign.status in ["ASSIGNED", "RUNNING"]:
                    assign.status = "FAILED"
                    assign.failure_reason = error_msg
                    assign.completed_at = now
            await event_publisher.publish(ClusterEventType.JOB_FAILED, f"Job {job_id} failed: {error_msg}", job_id=job_id)
        elif status_str == "CANCELLED":
            job.status = "CANCELLED"
            job.completed_at = now
            for assign in job.assignments:
                if assign.status in ["ASSIGNED", "RUNNING"]:
                    assign.status = "CANCELLED"
                    assign.completed_at = now
            await event_publisher.publish(ClusterEventType.JOB_CANCELLED, f"Job {job_id} cancelled", job_id=job_id)

        await session.commit()

    async def cancel_job(self, session: AsyncSession, job_id: str, reason: str = "User cancelled") -> bool:
        stmt = select(JobModel).where(JobModel.job_id == job_id).options(selectinload(JobModel.assignments))
        res = await session.execute(stmt)
        job = res.scalar_one_or_none()

        if not job or job.status in ["COMPLETED", "FAILED", "CANCELLED"]:
            return False

        job.status = "CANCELLED"
        job.completed_at = datetime.datetime.utcnow()

        for assign in job.assignments:
            assign.status = "CANCELLED"
            assign.failure_reason = reason
            # Send gRPC cancellation to node
            await node_stream_manager.send_job_cancel(assign.node_id, job_id, reason)

        await session.commit()
        await event_publisher.publish(ClusterEventType.JOB_CANCELLED, f"Job {job_id} cancelled ({reason})", job_id=job_id)
        return True

    async def get_job(self, session: AsyncSession, job_id: str) -> Optional[JobModel]:
        stmt = select(JobModel).where(JobModel.job_id == job_id).options(selectinload(JobModel.assignments))
        res = await session.execute(stmt)
        return res.scalar_one_or_none()

    async def list_jobs(self, session: AsyncSession, limit: int = 50) -> List[JobModel]:
        stmt = select(JobModel).options(selectinload(JobModel.assignments)).order_by(JobModel.created_at.desc()).limit(limit)
        res = await session.execute(stmt)
        return list(res.scalars().all())

job_manager = JobManager()
