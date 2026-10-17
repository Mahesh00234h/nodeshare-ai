import pytest
import asyncio

from app.database.session import async_session_factory, init_db
from app.nodes.registry import node_registry
from app.nodes.capabilities import NodeCapabilityReport, CPUInfo, MemoryInfo, GPUInfo, StorageInfo, NetworkInfo
from app.jobs.manager import job_manager
from app.scheduler.scheduler import scheduler
from app.grpc.stream_manager import node_stream_manager

@pytest.mark.asyncio
async def test_direct_node_dispatch():
    await init_db()
    async with async_session_factory() as session:
        # 1. Register test node
        node_id = "test-node-d4970fa6"
        report = NodeCapabilityReport(
            node_id=node_id,
            hostname="TEST-PC",
            os="Windows 11",
            os_version="10.0",
            architecture="x64",
            agent_version="1.0.0",
            protocol_version="v1",
            cpu=CPUInfo(model="Intel i7", vendor="Intel", cores=8, threads=16, utilization_percent=10.0),
            memory=MemoryInfo(total_bytes=16*1024*1024*1024, available_bytes=8*1024*1024*1024, used_bytes=8*1024*1024*1024, utilization_percent=50.0),
            gpus=[GPUInfo(vendor="NVIDIA", name="RTX 3060", vram_total_bytes=12*1024*1024*1024, vram_used_bytes=2*1024*1024*1024, vram_available_bytes=10*1024*1024*1024, utilization_percent=15.0, temperature_c=50, driver_version="535.11", compute_capability="8.6")],
            storage=StorageInfo(total_bytes=500*1024*1024*1024, available_bytes=200*1024*1024*1024, used_bytes=300*1024*1024*1024, used_percent=60.0),
            network=NetworkInfo(interfaces=[], estimated_bandwidth=1000000000),
            runtime_capabilities=["llama.cpp", "cuda"]
        )

        reg_res = await node_registry.register_node(session, report, auth_token="test-auth-token")
        assert reg_res["registered"] is True

        # Set node to READY
        await node_registry.set_node_status(session, node_id, "READY")
        node = await node_registry.get_node(session, node_id)
        assert node.status == "READY"

        # 2. Select Node via Scheduler
        selected_node, err_msg = await scheduler.select_node(
            session=session,
            model_id="qwen2.5-0.5b-instruct-q4_k_m.gguf",
            required_ram_bytes=512*1024*1024,
            requires_gpu=False
        )
        assert selected_node is not None
        assert selected_node.node_id == node_id

        # 3. Create & Dispatch Job
        job = await job_manager.create_job(
            session=session,
            model_id="qwen2.5-0.5b-instruct-q4_k_m.gguf",
            payload="Test prompt payload",
            priority=1
        )
        assert job.job_id is not None

        dispatched = await job_manager.assign_and_dispatch_job(
            session=session,
            job_id=job.job_id,
            node_id=selected_node.node_id,
            required_ram_bytes=512*1024*1024
        )
        assert dispatched is True

        # 4. Verify message in pending queue
        pending_msg = await node_stream_manager.get_pending_message(node_id, timeout_seconds=0.5)
        assert pending_msg is not None
        assert pending_msg.HasField("job_assignment")
        assert pending_msg.job_assignment.job_id == job.job_id
        assert pending_msg.job_assignment.model_id == "qwen2.5-0.5b-instruct-q4_k_m.gguf"
