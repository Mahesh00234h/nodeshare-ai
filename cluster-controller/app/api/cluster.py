from fastapi import APIRouter, Depends, Security
from sqlalchemy.ext.asyncio import AsyncSession
from app.database.session import get_db_session
from app.core.security import verify_client_api_key
from app.nodes.registry import node_registry
from app.jobs.manager import job_manager

router = APIRouter(prefix="/api/cluster", tags=["Cluster"])

@router.get("/status")
async def get_cluster_status(
    session: AsyncSession = Depends(get_db_session),
    _api_key: str = Depends(verify_client_api_key)
):
    nodes = await node_registry.list_nodes(session)
    
    total_nodes = len(nodes)
    online_nodes = sum(1 for n in nodes if n.status in ["READY", "BUSY"])
    offline_nodes = sum(1 for n in nodes if n.status == "OFFLINE")
    degraded_nodes = sum(1 for n in nodes if n.status == "DEGRADED")
    draining_nodes = sum(1 for n in nodes if n.status == "DRAINING")

    ram_total = sum(n.capabilities.ram_total_bytes for n in nodes if n.capabilities)
    ram_avail = sum(n.capabilities.storage_available_bytes for n in nodes if n.capabilities)
    vram_total = sum(sum(g.vram_total_bytes for g in n.gpus) for n in nodes if n.gpus)
    vram_avail = sum(sum(g.vram_avail_bytes for g in n.gpus) for n in nodes if n.gpus)

    jobs = await job_manager.list_jobs(session, limit=100)
    running_jobs = sum(1 for j in jobs if j.status in ["RUNNING", "STREAMING", "ASSIGNED"])
    queued_jobs = sum(1 for j in jobs if j.status == "QUEUED")

    cluster_status = "OPERATIONAL" if online_nodes > 0 else "NO_NODES_ONLINE"

    return {
        "status": cluster_status,
        "single_node_mode": True,
        "nodes": {
            "total": total_nodes,
            "online": online_nodes,
            "offline": offline_nodes,
            "degraded": degraded_nodes,
            "draining": draining_nodes,
        },
        "resources": {
            "ram_total_bytes": ram_total,
            "ram_available_bytes": ram_avail,
            "vram_total_bytes": vram_total,
            "vram_available_bytes": vram_avail,
        },
        "jobs": {
            "running": running_jobs,
            "queued": queued_jobs,
        }
    }
