import asyncio
import datetime
import logging
from sqlalchemy import select
from app.database.session import async_session_factory
from app.database.models import NodeModel, JobModel, JobAssignmentModel
from app.events.publisher import event_publisher
from app.events.types import ClusterEventType

logger = logging.getLogger(__name__)

async def start_health_monitor_worker(interval_seconds: int = 5):
    logger.info("[WORKER] Background health monitor worker started")

    while True:
        try:
            await asyncio.sleep(interval_seconds)
            async with async_session_factory() as session:
                now = datetime.datetime.utcnow()
                stmt = select(NodeModel).where(NodeModel.status.in_(["READY", "BUSY", "DEGRADED", "DRAINING"]))
                res = await session.execute(stmt)
                nodes = res.scalars().all()

                for node in nodes:
                    elapsed = (now - node.last_heartbeat_at).total_seconds()

                    # Threshold check:
                    # >30s (5+ missed heartbeats) -> OFFLINE
                    # 15s-30s (3-5 missed heartbeats) -> DEGRADED
                    if elapsed > 30.0:
                        old_status = node.status
                        node.status = "OFFLINE"
                        node.updated_at = now

                        # Fail active job assignments on this offline node
                        stmt_jobs = select(JobAssignmentModel).where(
                            JobAssignmentModel.node_id == node.node_id,
                            JobAssignmentModel.status.in_(["ASSIGNED", "RUNNING"])
                        )
                        assignments_res = await session.execute(stmt_jobs)
                        assignments = assignments_res.scalars().all()

                        for assign in assignments:
                            assign.status = "NODE_FAILURE"
                            assign.failure_reason = f"Node {node.node_id} went OFFLINE (heartbeat timeout)"
                            
                            # Mark main job as FAILED or RECOVERING
                            stmt_main_job = select(JobModel).where(JobModel.job_id == assign.job_id)
                            main_job_res = await session.execute(stmt_main_job)
                            main_job = main_job_res.scalar_one_or_none()
                            if main_job and main_job.status in ["RUNNING", "ASSIGNED"]:
                                main_job.status = "FAILED"
                                main_job.error_message = f"Execution node {node.node_id} failed / disconnected"
                                main_job.completed_at = now
                                await event_publisher.publish(
                                    ClusterEventType.JOB_FAILED,
                                    f"Job {main_job.job_id} failed due to node failure ({node.node_id})",
                                    job_id=main_job.job_id,
                                    node_id=node.node_id
                                )

                        await event_publisher.publish(
                            ClusterEventType.NODE_OFFLINE,
                            f"Node {node.node_id} ({node.hostname}) marked OFFLINE (no heartbeat for {int(elapsed)}s)",
                            node_id=node.node_id
                        )

                    elif elapsed > 15.0 and node.status in ["READY", "BUSY"]:
                        node.status = "DEGRADED"
                        node.updated_at = now
                        await event_publisher.publish(
                            ClusterEventType.NODE_DEGRADED,
                            f"Node {node.node_id} marked DEGRADED (missed heartbeats)",
                            node_id=node.node_id
                        )

                await session.commit()

        except asyncio.CancelledError:
            logger.info("[WORKER] Health monitor worker stopping")
            break
        except Exception as e:
            logger.error(f"[WORKER] Health monitor worker error: {e}", exc_info=True)
