from fastapi import APIRouter, Depends, HTTPException, status, Security
from sqlalchemy.ext.asyncio import AsyncSession
from app.database.session import get_db_session
from app.core.security import verify_client_api_key
from app.nodes.registry import node_registry
from app.nodes.telemetry import telemetry_service
from app.grpc.stream_manager import node_stream_manager

router = APIRouter(prefix="/api/nodes", tags=["Nodes Admin"])

@router.get("")
async def list_cluster_nodes(
    session: AsyncSession = Depends(get_db_session),
    _api_key: str = Depends(verify_client_api_key)
):
    nodes = await node_registry.list_nodes(session)
    result = []
    for n in nodes:
        result.append({
            "node_id": n.node_id,
            "hostname": n.hostname,
            "status": n.status,
            "os": n.os,
            "os_version": n.os_version,
            "architecture": n.architecture,
            "agent_version": n.agent_version,
            "protocol_version": n.protocol_version,
            "last_heartbeat_at": n.last_heartbeat_at.isoformat() if n.last_heartbeat_at else None,
            "cpu": {
                "model": n.capabilities.cpu_model if n.capabilities else "",
                "cores": n.capabilities.cpu_cores if n.capabilities else 1,
                "threads": n.capabilities.cpu_threads if n.capabilities else 1,
            } if n.capabilities else None,
            "ram_total_bytes": n.capabilities.ram_total_bytes if n.capabilities else 0,
            "gpu_count": len(n.gpus),
            "gpus": [
                {
                    "name": g.name,
                    "vram_total_bytes": g.vram_total_bytes,
                    "vram_avail_bytes": g.vram_avail_bytes,
                    "temperature_c": g.temperature_c,
                } for g in n.gpus
            ],
            "installed_models": [m.model_id for m in n.installed_models if m.status == "READY"],
        })
    return result

@router.get("/{node_id}")
async def get_node_details(
    node_id: str,
    session: AsyncSession = Depends(get_db_session),
    _api_key: str = Depends(verify_client_api_key)
):
    node = await node_registry.get_node(session, node_id)
    if not node:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=f"Node {node_id} not found")
    
    return {
        "node_id": node.node_id,
        "hostname": node.hostname,
        "status": node.status,
        "os": node.os,
        "os_version": node.os_version,
        "architecture": node.architecture,
        "agent_version": node.agent_version,
        "first_seen_at": node.first_seen_at.isoformat(),
        "last_heartbeat_at": node.last_heartbeat_at.isoformat(),
        "capabilities": {
            "cpu_model": node.capabilities.cpu_model if node.capabilities else "",
            "cpu_cores": node.capabilities.cpu_cores if node.capabilities else 1,
            "ram_total_bytes": node.capabilities.ram_total_bytes if node.capabilities else 0,
            "runtime_capabilities": node.capabilities.runtime_capabilities if node.capabilities else [],
        } if node.capabilities else None,
        "gpus": [{"name": g.name, "vram_total_bytes": g.vram_total_bytes} for g in node.gpus],
    }

@router.get("/{node_id}/telemetry")
async def get_node_telemetry(
    node_id: str,
    limit: int = 50,
    session: AsyncSession = Depends(get_db_session),
    _api_key: str = Depends(verify_client_api_key)
):
    history = await telemetry_service.get_node_telemetry_history(session, node_id, limit=limit)
    return [
        {
            "timestamp": t.timestamp.isoformat(),
            "cpu_utilization": t.cpu_utilization,
            "ram_utilization": t.ram_utilization,
            "ram_used_bytes": t.ram_used_bytes,
            "gpu_utilization": t.gpu_utilization,
            "vram_utilization": t.vram_utilization,
            "gpu_temperature": t.gpu_temperature,
            "active_jobs": t.active_jobs,
            "uptime_seconds": t.uptime_seconds,
        } for t in history
    ]

@router.post("/{node_id}/drain")
async def drain_node(
    node_id: str,
    reason: str = "Admin requested drain",
    session: AsyncSession = Depends(get_db_session),
    _api_key: str = Depends(verify_client_api_key)
):
    ok = await node_registry.set_node_status(session, node_id, "DRAINING")
    if not ok:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=f"Node {node_id} not found")
    
    await node_stream_manager.send_drain_command(node_id, reason=reason)
    return {"status": "DRAINING", "node_id": node_id, "message": "Drain command dispatched to node agent"}

@router.post("/{node_id}/resume")
async def resume_node(
    node_id: str,
    session: AsyncSession = Depends(get_db_session),
    _api_key: str = Depends(verify_client_api_key)
):
    ok = await node_registry.set_node_status(session, node_id, "READY")
    if not ok:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=f"Node {node_id} not found")
    return {"status": "READY", "node_id": node_id, "message": "Node resumed to READY state"}
