from typing import Tuple, Optional
from app.scheduler.resources import NodeResources

def score_node_for_job(
    node: NodeResources,
    model_id: str,
    required_ram_bytes: int = 0,
    required_vram_bytes: int = 0,
    requires_gpu: bool = False,
) -> Tuple[float, Optional[str]]:
    # 1. Hard Eligibility Checks
    if node.status not in ["READY", "BUSY", "DEGRADED"]:
        return 0.0, f"Node is in state {node.status}"

    if requires_gpu and node.gpu_count == 0:
        return 0.0, "Node does not have GPU support"

    if required_vram_bytes > 0:
        if node.vram_available_bytes < required_vram_bytes:
            return 0.0, f"Insufficient VRAM: available {node.vram_available_bytes} bytes < required {required_vram_bytes} bytes"

    if required_ram_bytes > 0:
        if node.ram_available_bytes < required_ram_bytes:
            return 0.0, f"Insufficient RAM: available {node.ram_available_bytes} bytes < required {required_ram_bytes} bytes"

    # 2. Dynamic Fitness Scoring
    score = 100.0

    # Model Locality Bonus
    if model_id in node.installed_models:
        score += 50.0

    # VRAM Headroom Score
    if node.vram_total_bytes > 0:
        vram_ratio = node.vram_available_bytes / float(node.vram_total_bytes)
        score += vram_ratio * 30.0

    # RAM Headroom Score
    if node.ram_total_bytes > 0:
        ram_ratio = node.ram_available_bytes / float(node.ram_total_bytes)
        score += ram_ratio * 20.0

    # Load Penalties
    if node.status == "DEGRADED":
        score -= 40.0
    score -= (node.gpu_utilization / 100.0) * 25.0
    score -= (node.cpu_utilization / 100.0) * 15.0
    score -= node.active_jobs * 10.0

    return max(score, 1.0), None
