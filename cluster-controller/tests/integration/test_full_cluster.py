import asyncio
import pytest
import httpx
from fastapi.testclient import TestClient

from app.main import app
from app.database.session import init_db, async_session_factory
from app.nodes.registry import node_registry
from app.nodes.capabilities import NodeCapabilityReport, CPUInfo, MemoryInfo, StorageInfo, NetworkInfo
from app.models.registry import model_registry

@pytest.mark.asyncio
async def test_full_controller_cluster_flow():
    # 1. Initialize DB
    await init_db()

    async with async_session_factory() as session:
        # 2. Register mock model
        await model_registry.register_model(
            session=session,
            model_id="mock-llama-3",
            name="Mock Llama 3 8B",
            parameter_count="8B"
        )

        # 3. Simulate Node Enrollment & Registration
        enroll_res = await node_registry.enroll_node(
            session=session,
            node_id="test-node-001",
            enrollment_token="dev-enrollment-token",
            hostname="test-host",
            agent_version="1.0.0"
        )
        assert enroll_res["success"] is True
        auth_token = enroll_res["auth_token"]

        report = NodeCapabilityReport(
            node_id="test-node-001",
            hostname="test-host",
            os="windows",
            agent_version="1.0.0",
            cpu=CPUInfo(model="Intel i5", cores=8, threads=8, utilization_percent=20.0),
            memory=MemoryInfo(total_bytes=16 * 1024**3, available_bytes=10 * 1024**3, used_bytes=6 * 1024**3),
            storage=StorageInfo(total_bytes=100 * 1024**3, available_bytes=50 * 1024**3),
            runtime_capabilities=["CPU_INFERENCE", "TEXT_GENERATION", "STREAMING"]
        )

        reg_res = await node_registry.register_node(session, report, auth_token)
        assert reg_res["registered"] is True

        # Register node installed model
        await model_registry.register_node_model(session, "test-node-001", "mock-llama-3")

    # 4. Query Cluster Status API
    async with httpx.AsyncClient(transport=httpx.ASGITransport(app=app), base_url="http://testserver") as client:
        headers = {"Authorization": "Bearer sk-cluster-controller-secret-key"}
        
        status_resp = await client.get("/api/cluster/status", headers=headers)
        assert status_resp.status_code == 200
        cluster_data = status_resp.json()
        
        assert cluster_data["status"] in ["OPERATIONAL", "DEGRADED", "NO_NODES_ONLINE"]
        assert cluster_data["nodes"]["total"] >= 1
        assert cluster_data["single_node_mode"] is True

        # 5. Query Nodes API
        nodes_resp = await client.get("/api/nodes", headers=headers)
        assert nodes_resp.status_code == 200
        nodes_list = nodes_resp.json()
        assert len(nodes_list) >= 1
        test_node = next((n for n in nodes_list if n["node_id"] == "test-node-001"), None)
        assert test_node is not None
        assert test_node["status"] == "READY"

        # 6. Query OpenAI /v1/models
        models_resp = await client.get("/v1/models", headers=headers)
        assert models_resp.status_code == 200
        models_data = models_resp.json()
        assert models_data["object"] == "list"
        assert len(models_data["data"]) >= 1

        # 7. Drain Node Test
        drain_resp = await client.post("/api/nodes/test-node-001/drain", headers=headers)
        assert drain_resp.status_code == 200
        assert drain_resp.json()["status"] == "DRAINING"

        # Resume Node Test
        resume_resp = await client.post("/api/nodes/test-node-001/resume", headers=headers)
        assert resume_resp.status_code == 200
        assert resume_resp.json()["status"] == "READY"
