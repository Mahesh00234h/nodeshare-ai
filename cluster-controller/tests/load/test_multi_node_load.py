import pytest
from app.scheduler.resources import NodeResources
from app.scheduler.scoring import score_node_for_job

def test_heterogeneous_multi_node_scheduling():
    node1 = NodeResources(
        node_id="node-1-rtx3050",
        gpu_count=1,
        vram_total_bytes=4 * 1024**3,
        vram_available_bytes=3 * 1024**3,
        ram_total_bytes=16 * 1024**3,
        ram_available_bytes=10 * 1024**3,
        installed_models=["llama2-7b"]
    )

    node2 = NodeResources(
        node_id="node-2-rtx3090",
        gpu_count=1,
        vram_total_bytes=24 * 1024**3,
        vram_available_bytes=20 * 1024**3,
        ram_total_bytes=64 * 1024**3,
        ram_available_bytes=48 * 1024**3,
        installed_models=["llama3-70b"]
    )

    node3 = NodeResources(
        node_id="node-3-cpu-only",
        gpu_count=0,
        vram_total_bytes=0,
        vram_available_bytes=0,
        ram_total_bytes=8 * 1024**3,
        ram_available_bytes=6 * 1024**3,
        installed_models=[]
    )

    # Workload A: Requires 16GB VRAM -> Must select Node 2 (Node 1 & 3 rejected)
    score1, _ = score_node_for_job(node1, model_id="llama3-70b", required_vram_bytes=16 * 1024**3, requires_gpu=True)
    score2, _ = score_node_for_job(node2, model_id="llama3-70b", required_vram_bytes=16 * 1024**3, requires_gpu=True)
    score3, _ = score_node_for_job(node3, model_id="llama3-70b", required_vram_bytes=16 * 1024**3, requires_gpu=True)

    assert score1 == 0.0  # Node 1 VRAM too small
    assert score3 == 0.0  # Node 3 no GPU
    assert score2 > 100.0 # Node 2 qualifies

    # Workload B: Small CPU job -> Node 3 or Node 1 or 2 qualifies
    score3_cpu, _ = score_node_for_job(node3, model_id="small-cpu-model", required_ram_bytes=2 * 1024**3, requires_gpu=False)
    assert score3_cpu > 0.0
