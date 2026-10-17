from typing import Dict, Any, List
from pydantic import BaseModel, Field

class CPUInfo(BaseModel):
    model: str = ""
    vendor: str = ""
    cores: int = 1
    threads: int = 1
    utilization_percent: float = 0.0

class MemoryInfo(BaseModel):
    total_bytes: int = 0
    available_bytes: int = 0
    used_bytes: int = 0
    utilization_percent: float = 0.0

class GPUInfo(BaseModel):
    vendor: str = "NVIDIA"
    name: str = ""
    vram_total_bytes: int = 0
    vram_used_bytes: int = 0
    vram_available_bytes: int = 0
    utilization_percent: float = 0.0
    temperature_c: int = 0
    driver_version: str = ""
    compute_capability: str = ""

class StorageInfo(BaseModel):
    total_bytes: int = 0
    available_bytes: int = 0
    used_bytes: int = 0
    used_percent: float = 0.0

class NetworkInterfaceInfo(BaseModel):
    name: str = ""
    ips: List[str] = Field(default_factory=list)
    mac: str = ""
    is_wired: bool = True
    estimated_bandwidth_bps: int = 1000000000

class NetworkInfo(BaseModel):
    interfaces: List[NetworkInterfaceInfo] = Field(default_factory=list)
    estimated_bandwidth: int = 1000000000

class NodeCapabilityReport(BaseModel):
    node_id: str
    hostname: str = ""
    os: str = ""
    os_version: str = ""
    architecture: str = ""
    agent_version: str = "1.0.0"
    protocol_version: str = "v1"
    cpu: CPUInfo = Field(default_factory=CPUInfo)
    memory: MemoryInfo = Field(default_factory=MemoryInfo)
    gpus: List[GPUInfo] = Field(default_factory=list)
    storage: StorageInfo = Field(default_factory=StorageInfo)
    network: NetworkInfo = Field(default_factory=NetworkInfo)
    runtime_capabilities: List[str] = Field(default_factory=list)
