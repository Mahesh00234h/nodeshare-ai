from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.ext.asyncio import AsyncSession
from app.database.session import get_db_session
from app.core.security import verify_client_api_key
from app.jobs.manager import job_manager

router = APIRouter(prefix="/api/jobs", tags=["Jobs Admin"])

@router.get("")
async def list_jobs(
    limit: int = 50,
    session: AsyncSession = Depends(get_db_session),
    _api_key: str = Depends(verify_client_api_key)
):
    jobs = await job_manager.list_jobs(session, limit=limit)
    return [
        {
            "job_id": j.job_id,
            "request_id": j.request_id,
            "model_id": j.model_id,
            "status": j.status,
            "priority": j.priority,
            "execution_id": j.execution_id,
            "error_message": j.error_message,
            "created_at": j.created_at.isoformat(),
            "started_at": j.started_at.isoformat() if j.started_at else None,
            "completed_at": j.completed_at.isoformat() if j.completed_at else None,
            "assignments": [
                {
                    "node_id": a.node_id,
                    "role": a.role,
                    "status": a.status,
                    "assigned_at": a.assigned_at.isoformat()
                } for a in j.assignments
            ] if j.assignments else []
        } for j in jobs
    ]

@router.get("/{job_id}")
async def get_job_details(
    job_id: str,
    session: AsyncSession = Depends(get_db_session),
    _api_key: str = Depends(verify_client_api_key)
):
    job = await job_manager.get_job(session, job_id)
    if not job:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=f"Job {job_id} not found")
    
    return {
        "job_id": job.job_id,
        "model_id": job.model_id,
        "status": job.status,
        "payload": job.payload,
        "execution_id": job.execution_id,
        "error_message": job.error_message,
        "created_at": job.created_at.isoformat(),
        "started_at": job.started_at.isoformat() if job.started_at else None,
        "completed_at": job.completed_at.isoformat() if job.completed_at else None,
        "assignments": [
            {
                "node_id": a.node_id,
                "role": a.role,
                "status": a.status,
                "failure_reason": a.failure_reason,
            } for a in job.assignments
        ] if job.assignments else []
    }

@router.post("/{job_id}/cancel")
async def cancel_job(
    job_id: str,
    reason: str = "Admin requested cancellation",
    session: AsyncSession = Depends(get_db_session),
    _api_key: str = Depends(verify_client_api_key)
):
    ok = await job_manager.cancel_job(session, job_id, reason=reason)
    if not ok:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=f"Job {job_id} could not be cancelled (already finished or not found)")
    return {"job_id": job_id, "status": "CANCELLED", "reason": reason}
