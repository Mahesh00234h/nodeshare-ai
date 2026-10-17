import asyncio
import logging
import time
from typing import Dict
import grpc

from app.grpc.generated import node_controller_pb2, node_controller_pb2_grpc
from app.database.session import async_session_factory
from app.nodes.registry import node_registry
from app.nodes.health import heartbeat_service
from app.nodes.telemetry import telemetry_service
from app.nodes.capabilities import (
    NodeCapabilityReport, CPUInfo, MemoryInfo, GPUInfo, StorageInfo, NetworkInfo, NetworkInterfaceInfo
)
from app.jobs.manager import job_manager
from app.jobs.streaming import token_multiplexer
from app.models.registry import model_registry
from app.grpc.stream_manager import node_stream_manager

logger = logging.getLogger(__name__)


def safe_has_field(msg, field_name: str) -> bool:
    try:
        return msg.HasField(field_name)
    except Exception:
        return False

class NodeControllerServicer(node_controller_pb2_grpc.NodeControllerServiceServicer):
    async def EnrollNode(self, request: node_controller_pb2.EnrollmentRequest, context: grpc.aio.ServicerContext) -> node_controller_pb2.EnrollmentResponse:
        logger.info(f"[gRPC EnrollNode] Node={request.node_id}, Token={request.enrollment_token}, Host={request.hostname}")
        async with async_session_factory() as session:
            res = await node_registry.enroll_node(
                session=session,
                node_id=request.node_id,
                enrollment_token=request.enrollment_token,
                hostname=request.hostname,
                agent_version=request.agent_version
            )
            return node_controller_pb2.EnrollmentResponse(
                success=res["success"],
                auth_token=res.get("auth_token", ""),
                node_secret=res.get("node_secret", ""),
                message=res.get("message", "")
            )

    async def RegisterNode(self, request: node_controller_pb2.RegistrationRequest, context: grpc.aio.ServicerContext) -> node_controller_pb2.RegistrationResponse:
        try:
            logger.info(f"[gRPC RegisterNode] Node={request.node_id}, Host={request.hostname}, OS={request.os}")
            
            has_cpu = safe_has_field(request, "cpu")
            has_mem = safe_has_field(request, "memory")
            has_storage = safe_has_field(request, "storage")
            has_network = safe_has_field(request, "network")

            cpu_info = CPUInfo(
                model=request.cpu.model if has_cpu else "",
                vendor=request.cpu.vendor if has_cpu else "",
                cores=request.cpu.cores if has_cpu else 1,
                threads=request.cpu.threads if has_cpu else 1,
                utilization_percent=request.cpu.utilization_percent if has_cpu else 0.0
            )
            mem_info = MemoryInfo(
                total_bytes=request.memory.total_bytes if has_mem else 0,
                available_bytes=request.memory.available_bytes if has_mem else 0,
                used_bytes=request.memory.used_bytes if has_mem else 0,
                utilization_percent=request.memory.utilization_percent if has_mem else 0.0
            )
            gpu_list = [
                GPUInfo(
                    vendor=g.vendor,
                    name=g.name,
                    vram_total_bytes=g.vram_total_bytes,
                    vram_used_bytes=g.vram_used_bytes,
                    vram_available_bytes=g.vram_available_bytes,
                    utilization_percent=g.utilization_percent,
                    temperature_c=g.temperature_c,
                    driver_version=g.driver_version,
                    compute_capability=g.compute_capability
                ) for g in request.gpus
            ]
            storage_info = StorageInfo(
                total_bytes=request.storage.total_bytes if has_storage else 0,
                available_bytes=request.storage.available_bytes if has_storage else 0,
                used_bytes=request.storage.used_bytes if has_storage else 0,
                used_percent=request.storage.used_percent if has_storage else 0.0
            )
            net_ifaces = [
                NetworkInterfaceInfo(
                    name=i.name,
                    ips=list(i.ips),
                    mac=i.mac,
                    is_wired=i.is_wired,
                    estimated_bandwidth_bps=i.estimated_bandwidth_bps
                ) for i in (request.network.interfaces if has_network else [])
            ]
            net_info = NetworkInfo(
                interfaces=net_ifaces,
                estimated_bandwidth=request.network.estimated_bandwidth if has_network else 1000000000
            )

            report = NodeCapabilityReport(
                node_id=request.node_id,
                hostname=request.hostname,
                os=request.os,
                os_version=request.os_version,
                architecture=request.architecture,
                agent_version=request.agent_version,
                protocol_version=request.protocol_version,
                cpu=cpu_info,
                memory=mem_info,
                gpus=gpu_list,
                storage=storage_info,
                network=net_info,
                runtime_capabilities=list(request.runtime_capabilities)
            )

            async with async_session_factory() as session:
                res = await node_registry.register_node(session, report, request.auth_token)
                return node_controller_pb2.RegistrationResponse(
                    registered=res["registered"],
                    cluster_node_id=res.get("cluster_node_id", ""),
                    heartbeat_interval_seconds=res.get("heartbeat_interval_seconds", 5),
                    telemetry_interval_seconds=res.get("telemetry_interval_seconds", 5),
                    message=res["message"]
                )
        except Exception as e:
            logger.exception(f"[gRPC RegisterNode] Error handling registration for node {getattr(request, 'node_id', 'unknown')}: {e}")
            return node_controller_pb2.RegistrationResponse(
                registered=False,
                message=f"Internal registration error: {str(e)}"
            )

    async def SendHeartbeat(self, request: node_controller_pb2.HeartbeatRequest, context: grpc.aio.ServicerContext) -> node_controller_pb2.HeartbeatResponse:
        async with async_session_factory() as session:
            res = await heartbeat_service.process_heartbeat(
                session=session,
                node_id=request.node_id,
                auth_token=request.auth_token,
                node_state=request.state,
                active_job_ids=list(request.active_job_ids),
                cpu_util=request.cpu_utilization,
                ram_util=request.ram_utilization,
                gpu_util=request.gpu_utilization,
                vram_util=request.vram_utilization,
                gpu_temp=request.gpu_temperature,
                uptime=request.uptime_seconds
            )
            return node_controller_pb2.HeartbeatResponse(
                acknowledged=res["acknowledged"],
                desired_state=res.get("desired_state", "READY"),
                message=res["message"]
            )

    async def SendTelemetry(self, request: node_controller_pb2.TelemetryRequest, context: grpc.aio.ServicerContext) -> node_controller_pb2.TelemetryResponse:
        async with async_session_factory() as session:
            has_cpu = safe_has_field(request, "cpu")
            has_mem = safe_has_field(request, "memory")
            ok = await telemetry_service.process_telemetry(
                session=session,
                node_id=request.node_id,
                auth_token=request.auth_token,
                timestamp=request.timestamp,
                cpu_util=request.cpu.utilization_percent if has_cpu else 0.0,
                ram_util=request.memory.utilization_percent if has_mem else 0.0,
                ram_used_bytes=request.memory.used_bytes if has_mem else 0,
                gpu_util=request.gpus[0].utilization_percent if len(request.gpus) > 0 else 0.0,
                vram_util=(request.gpus[0].vram_used_bytes / request.gpus[0].vram_total_bytes * 100.0) if len(request.gpus) > 0 and request.gpus[0].vram_total_bytes > 0 else 0.0,
                gpu_temp=request.gpus[0].temperature_c if len(request.gpus) > 0 else 0,
                active_jobs=request.active_jobs,
                queued_jobs=request.queued_jobs,
                uptime=request.uptime_seconds
            )
            return node_controller_pb2.TelemetryResponse(received=ok)

    async def StreamControlChannel(self, request_iterator, context: grpc.aio.ServicerContext):
        node_id = None
        send_queue = None

        try:
            async for node_msg in request_iterator:
                if not node_id:
                    node_id = node_msg.node_id
                    send_queue = node_stream_manager.register_stream(node_id)
                    
                    # Start sender task for this stream
                    async def sender_loop():
                        try:
                            while True:
                                ctrl_msg = await send_queue.get()
                                await context.write(ctrl_msg)
                                send_queue.task_done()
                        except Exception as e:
                            logger.debug(f"[gRPC Stream Sender] Loop ended for {node_id}: {e}")

                    asyncio.create_task(sender_loop())

                if node_msg.HasField("job_progress"):
                    prog = node_msg.job_progress
                    async with async_session_factory() as session:
                        await job_manager.update_job_progress(
                            session=session,
                            job_id=prog.job_id,
                            status_str=prog.status,
                            chunk=prog.output_chunk,
                            tokens=prog.tokens_generated,
                            error_msg=prog.error_message
                        )

                    # Push token chunk to SSE / OpenAI HTTP streaming queue
                    token_multiplexer.push_token(
                        job_id=prog.job_id,
                        chunk=prog.output_chunk,
                        tokens=prog.tokens_generated,
                        status=prog.status,
                        error_msg=prog.error_message
                    )

                elif node_msg.drain_acknowledged:
                    async with async_session_factory() as session:
                        await node_registry.set_node_status(session, node_id, "DRAINED")

        except asyncio.CancelledError:
            logger.info(f"[gRPC Stream] Control channel cancelled for {node_id}")
        except Exception as e:
            logger.error(f"[gRPC Stream] Error in control channel for {node_id}: {e}")
        finally:
            if node_id:
                node_stream_manager.unregister_stream(node_id)
