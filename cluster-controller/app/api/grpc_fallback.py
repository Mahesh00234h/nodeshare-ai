import gzip
import json
import logging
import asyncio
import datetime
from fastapi import APIRouter, Request, Response
from google.protobuf.json_format import Parse, MessageToDict
from sqlalchemy import select
from app.grpc.generated import node_controller_pb2
from app.grpc.handlers import NodeControllerServicer
from app.grpc.stream_manager import node_stream_manager
from app.jobs.manager import job_manager
from app.jobs.streaming import token_multiplexer
from app.database.session import async_session_factory
from app.database.models import JobModel, JobAssignmentModel, NodeModel

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/node_controller.v1.NodeControllerService", tags=["gRPC HTTP Bridge"])
servicer = NodeControllerServicer()

def parse_proto_request(raw_body: bytes, req_obj, request: Request):
    if not raw_body:
        return

    content_type = request.headers.get("content-type", "").lower()
    grpc_encoding = request.headers.get("grpc-encoding", "").lower()

    def is_valid_parse(obj) -> bool:
        node_id = getattr(obj, "node_id", None)
        return bool(node_id) or bool(obj.ListFields())

    # Priority 1: JSON parse if content-type is JSON or payload starts with '{'
    if "json" in content_type or raw_body.strip().startswith(b"{"):
        try:
            Parse(raw_body.decode("utf-8", errors="ignore"), req_obj, ignore_unknown_fields=True)
            if is_valid_parse(req_obj):
                return
        except Exception:
            pass
        req_obj.Clear()

    # Strategy 1: Try gRPC 5-byte frame header stripping (with gzip support)
    if len(raw_body) >= 5:
        msg_len = int.from_bytes(raw_body[1:5], "big")
        payload = raw_body[5:5+msg_len] if (msg_len > 0 and len(raw_body) >= 5 + msg_len) else raw_body[5:]
        if raw_body[0] == 1 or "gzip" in grpc_encoding:
            try:
                payload = gzip.decompress(payload)
            except Exception:
                pass
        try:
            req_obj.ParseFromString(payload)
            if is_valid_parse(req_obj):
                return
        except Exception:
            pass
        req_obj.Clear()

    # Strategy 2: Try direct binary protobuf parse
    try:
        req_obj.ParseFromString(raw_body)
        if is_valid_parse(req_obj):
            return
    except Exception:
        pass
    req_obj.Clear()

    # Strategy 3: Try JSON parse fallback
    try:
        Parse(raw_body.decode("utf-8", errors="ignore"), req_obj, ignore_unknown_fields=True)
        if is_valid_parse(req_obj):
            return
    except Exception:
        pass

    logger.warning(f"[gRPC Bridge] Could not parse request payload for {request.url.path} (content-type: {content_type})")

def make_response(resp_obj, request: Request):
    content_type = request.headers.get("content-type", "").lower()
    serialized = resp_obj.SerializeToString()
    if "json" in content_type:
        return Response(content=json.dumps(MessageToDict(resp_obj, preserving_proto_field_name=True)), media_type="application/json")
    elif "grpc" in content_type:
        frame_header = b"\x00" + len(serialized).to_bytes(4, "big")
        return Response(
            content=frame_header + serialized,
            media_type="application/grpc",
            headers={"grpc-status": "0", "grpc-message": ""}
        )
    else:
        return Response(content=serialized, media_type="application/x-protobuf")

@router.post("/EnrollNode")
async def http_enroll_node(request: Request):
    raw_body = await request.body()
    req_obj = node_controller_pb2.EnrollmentRequest()
    parse_proto_request(raw_body, req_obj, request)
    try:
        resp_obj = await servicer.EnrollNode(req_obj, None)
        return make_response(resp_obj, request)
    except Exception as e:
        logger.exception(f"[gRPC Bridge Error] EnrollNode: {e}")
        return make_response(node_controller_pb2.EnrollmentResponse(success=False, message=str(e)), request)

@router.post("/RegisterNode")
async def http_register_node(request: Request):
    raw_body = await request.body()
    req_obj = node_controller_pb2.RegistrationRequest()
    parse_proto_request(raw_body, req_obj, request)
    if not req_obj.node_id:
        return make_response(node_controller_pb2.RegistrationResponse(registered=False, message="node_id is required"), request)
    try:
        resp_obj = await servicer.RegisterNode(req_obj, None)
        return make_response(resp_obj, request)
    except Exception as e:
        logger.exception(f"[gRPC Bridge Error] RegisterNode: {e}")
        return make_response(node_controller_pb2.RegistrationResponse(registered=False, message=str(e)), request)

@router.post("/SendHeartbeat")
async def http_send_heartbeat(request: Request):
    raw_body = await request.body()
    req_obj = node_controller_pb2.HeartbeatRequest()
    parse_proto_request(raw_body, req_obj, request)
    if not req_obj.node_id:
        return make_response(node_controller_pb2.HeartbeatResponse(acknowledged=False, message="node_id is required"), request)
    try:
        resp_obj = await servicer.SendHeartbeat(req_obj, None)
        return make_response(resp_obj, request)
    except Exception as e:
        logger.exception(f"[gRPC Bridge Error] SendHeartbeat: {e}")
        return make_response(node_controller_pb2.HeartbeatResponse(acknowledged=False, message=str(e)), request)

@router.post("/SendTelemetry")
async def http_send_telemetry(request: Request):
    raw_body = await request.body()
    req_obj = node_controller_pb2.TelemetryRequest()
    parse_proto_request(raw_body, req_obj, request)
    if not req_obj.node_id:
        return make_response(node_controller_pb2.TelemetryResponse(received=False), request)
    try:
        resp_obj = await servicer.SendTelemetry(req_obj, None)
        return make_response(resp_obj, request)
    except Exception as e:
        logger.exception(f"[gRPC Bridge Error] SendTelemetry: {e}")
        return make_response(node_controller_pb2.TelemetryResponse(received=False), request)

@router.post("/StreamControlChannel")
async def http_stream_control_channel(request: Request):
    raw_body = await request.body()
    node_msg = node_controller_pb2.NodeMessage()
    parse_proto_request(raw_body, node_msg, request)

    node_id = node_msg.node_id
    if not node_id:
        try:
            body_dict = json.loads(raw_body.decode("utf-8"))
            node_id = body_dict.get("node_id", "")
        except Exception:
            pass

    if not node_id:
        return make_response(node_controller_pb2.ControllerMessage(message_id="err-no-node-id"), request)

    # Process job progress update if present
    prog_data = None
    if node_msg.HasField("job_progress"):
        prog = node_msg.job_progress
        prog_data = {
            "job_id": prog.job_id,
            "status": prog.status,
            "output_chunk": prog.output_chunk,
            "tokens_generated": prog.tokens_generated,
            "error_message": prog.error_message
        }
    else:
        try:
            body_dict = json.loads(raw_body.decode("utf-8"))
            if "job_progress" in body_dict and isinstance(body_dict["job_progress"], dict):
                jp = body_dict["job_progress"]
                prog_data = {
                    "job_id": jp.get("job_id", ""),
                    "status": jp.get("status", ""),
                    "output_chunk": jp.get("output_chunk", ""),
                    "tokens_generated": jp.get("tokens_generated", 0),
                    "error_message": jp.get("error_message", "")
                }
        except Exception:
            pass

    if prog_data and prog_data["job_id"]:
        token_multiplexer.push_token(
            job_id=prog_data["job_id"],
            chunk=prog_data["output_chunk"],
            tokens=prog_data["tokens_generated"],
            status=prog_data["status"],
            error_msg=prog_data["error_message"]
        )
        async with async_session_factory() as session:
            await job_manager.update_job_progress(
                session=session,
                job_id=prog_data["job_id"],
                status_str=prog_data["status"],
                chunk=prog_data["output_chunk"],
                tokens=prog_data["tokens_generated"],
                error_msg=prog_data["error_message"]
            )

    # Update node last_seen so it stays READY
    async with async_session_factory() as session:
        try:
            from sqlalchemy import update as sql_update
            await session.execute(
                sql_update(NodeModel)
                .where(NodeModel.node_id == node_id)
                .values(last_seen_at=datetime.datetime.utcnow(), last_heartbeat_at=datetime.datetime.utcnow())
            )
            await session.commit()
        except Exception:
            pass

    # Poll pending message for node (in-memory queue)
    pending = await node_stream_manager.get_pending_message(node_id, timeout_seconds=0.5)
    if pending:
        return make_response(pending, request)

    # DB-backed job recovery: re-dispatch any ASSIGNED jobs that didn't reach this node
    # (handles Render restarts where in-memory queue is wiped)
    async with async_session_factory() as session:
        try:
            recovery_cutoff = datetime.datetime.utcnow() - datetime.timedelta(seconds=60)
            stmt = (
                select(JobModel, JobAssignmentModel)
                .join(JobAssignmentModel, JobModel.job_id == JobAssignmentModel.job_id)
                .where(
                    JobAssignmentModel.node_id == node_id,
                    JobModel.status.in_(["ASSIGNED", "RUNNING"]),
                    JobAssignmentModel.status == "ASSIGNED",
                    JobAssignmentModel.assigned_at <= recovery_cutoff
                )
                .limit(1)
            )
            res = await session.execute(stmt)
            row = res.first()
            if row:
                job, assignment = row
                logger.info(f"[gRPC Bridge] Re-dispatching stuck job {job.job_id} to node {node_id} via DB recovery")
                recovery_msg = node_controller_pb2.ControllerMessage(
                    message_id=f"recovery-{job.job_id}",
                    job_assignment=node_controller_pb2.JobAssignment(
                        job_id=job.job_id,
                        job_type="INFERENCE",
                        model_id=job.model_id,
                        execution_id=job.execution_id,
                        partition_id="partition-0",
                        payload=job.payload,
                        priority=job.priority,
                    )
                )
                return make_response(recovery_msg, request)
        except Exception as e:
            logger.warning(f"[gRPC Bridge] DB recovery query failed: {e}")

    return make_response(node_controller_pb2.ControllerMessage(message_id="idle"), request)

@router.post("/SendNodeMessage")
async def http_send_node_message(request: Request):
    raw_body = await request.body()
    node_msg = node_controller_pb2.NodeMessage()
    parse_proto_request(raw_body, node_msg, request)

    prog_data = None
    if node_msg.HasField("job_progress"):
        prog = node_msg.job_progress
        prog_data = {
            "job_id": prog.job_id,
            "status": prog.status,
            "output_chunk": prog.output_chunk,
            "tokens_generated": prog.tokens_generated,
            "error_message": prog.error_message
        }
    else:
        try:
            body_dict = json.loads(raw_body.decode("utf-8"))
            if "job_progress" in body_dict and isinstance(body_dict["job_progress"], dict):
                jp = body_dict["job_progress"]
                prog_data = {
                    "job_id": jp.get("job_id", ""),
                    "status": jp.get("status", ""),
                    "output_chunk": jp.get("output_chunk", ""),
                    "tokens_generated": jp.get("tokens_generated", 0),
                    "error_message": jp.get("error_message", "")
                }
        except Exception:
            pass

    if prog_data and prog_data["job_id"]:
        token_multiplexer.push_token(
            job_id=prog_data["job_id"],
            chunk=prog_data["output_chunk"],
            tokens=prog_data["tokens_generated"],
            status=prog_data["status"],
            error_msg=prog_data["error_message"]
        )
        async with async_session_factory() as session:
            await job_manager.update_job_progress(
                session=session,
                job_id=prog_data["job_id"],
                status_str=prog_data["status"],
                chunk=prog_data["output_chunk"],
                tokens=prog_data["tokens_generated"],
                error_msg=prog_data["error_message"]
            )

    return make_response(node_controller_pb2.ControllerMessage(message_id="ack"), request)
