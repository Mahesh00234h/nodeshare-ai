import pytest
from app.scheduler.resources import NodeResources
from app.scheduler.scoring import score_node_for_job

def test_node_scoring_hard_eligibility():
    node_no_gpu = NodeResources(
        node_id="node-1",
        status="READY",
        gpu_count=0,
        ram_total_bytes=16 * 1024**3,
        ram_available_bytes=8 * 1024**3
    )

    # Job requires GPU -> Should reject node without GPU
    score, reason = score_node_for_job(node_no_gpu, model_id="llama3", requires_gpu=True)
    assert score == 0.0
    assert "does not have GPU support" in reason

    # Job requires 10GB RAM, node has 8GB -> Should reject
    score, reason = score_node_for_job(node_no_gpu, model_id="llama3", required_ram_bytes=10 * 1024**3)
    assert score == 0.0
    assert "Insufficient RAM" in reason

def test_node_scoring_locality_and_headroom():
    node_gpu = NodeResources(
        node_id="node-gpu",
        status="READY",
        gpu_count=1,
        vram_total_bytes=12 * 1024**3,
        vram_available_bytes=10 * 1024**3,
        ram_total_bytes=32 * 1024**3,
        ram_available_bytes=24 * 1024**3,
        installed_models=["mock-llama-3"]
    )

    score, reason = score_node_for_job(
        node_gpu,
        model_id="mock-llama-3",
        required_vram_bytes=4 * 1024**3,
        requires_gpu=True
    )
    assert reason is None
    assert score > 150.0  # Includes model locality + headroom bonuses
