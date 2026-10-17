from typing import List, Optional
from pydantic import BaseModel, Field

class NodeResources(BaseModel):
    node_id: str
    hostname: str = ""
    status: str = "READY"
    cpu_cores: int = 1
    cpu_utilization: float = 0.0
    ram_total_bytes: int = 0
    ram_available_bytes: int = 0
    gpu_count: int = 0
    vram_total_bytes: int = 0
    vram_available_bytes: int = 0
    gpu_utilization: float = 0.0
    gpu_temperature: int = 0
    active_jobs: int = 0
    installed_models: List[str] = Field(default_factory=list)
    runtime_capabilities: List[str] = Field(default_factory=list)
