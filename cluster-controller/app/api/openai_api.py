import asyncio
import json
import time
from typing import List, Optional, Dict, Any, Union
from fastapi import APIRouter, Depends, HTTPException, status, Request
from fastapi.responses import StreamingResponse
from pydantic import BaseModel, Field
from sqlalchemy.ext.asyncio import AsyncSession

from app.database.session import get_db_session
from app.core.security import verify_client_api_key
from app.scheduler.scheduler import scheduler
from app.jobs.manager import job_manager
from app.jobs.streaming import token_multiplexer
from app.models.registry import model_registry

router = APIRouter(prefix="/v1", tags=["OpenAI-Compatible API"])

class ChatMessage(BaseModel):
    role: str
    content: str

class ChatCompletionRequest(BaseModel):
    model: str
    messages: List[ChatMessage]
    temperature: Optional[float] = 0.7
    max_tokens: Optional[int] = 512
    stream: Optional[bool] = False
    top_p: Optional[float] = 1.0
    stop: Optional[Union[str, List[str]]] = None

class EmbeddingRequest(BaseModel):
    model: str
    input: Union[str, List[str]]

@router.get("/models")
async def list_openai_models(
    session: AsyncSession = Depends(get_db_session),
    _api_key: str = Depends(verify_client_api_key)
):
    models = await model_registry.list_models(session)
    data = []
    
    # If database has registered models, use them
    for m in models:
        data.append({
            "id": m.model_id,
            "object": "model",
            "created": int(m.created_at.timestamp()),
            "owned_by": "distributed-ai-cluster",
        })
    
    # Default fallback models if empty
    if not data:
        data = [
            {"id": "mock-llama-3", "object": "model", "created": int(time.time()), "owned_by": "distributed-ai-cluster"},
            {"id": "llama3:8b", "object": "model", "created": int(time.time()), "owned_by": "distributed-ai-cluster"},
        ]

    return {"object": "list", "data": data}

@router.post("/chat/completions")
async def chat_completions(
    req: ChatCompletionRequest,
    raw_req: Request,
    session: AsyncSession = Depends(get_db_session),
    _api_key: str = Depends(verify_client_api_key)
):
    # 1. Select compute node via Scheduler
    selected_node, err_msg = await scheduler.select_node(
        session=session,
        model_id=req.model,
        required_ram_bytes=1024 * 1024 * 512, # 512MB default requirement
        requires_gpu=False
    )

    if not selected_node:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail=f"No compute node available to execute job for model {req.model}: {err_msg}"
        )

    # 2. Extract last user message prompt
    prompt_text = ""
    for msg in req.messages:
        prompt_text += f"{msg.role.upper()}: {msg.content}\n"

    # 3. Create Job
    job = await job_manager.create_job(
        session=session,
        model_id=req.model,
        payload=prompt_text,
        priority=1
    )

    # 4. Register job stream queue
    queue = token_multiplexer.register_job_stream(job.job_id)

    # 5. Dispatch job to selected node
    dispatched = await job_manager.assign_and_dispatch_job(
        session=session,
        job_id=job.job_id,
        node_id=selected_node.node_id,
        required_ram_bytes=1024 * 1024 * 512
    )

    if not dispatched:
        token_multiplexer.close_job_stream(job.job_id)
        raise HTTPException(
            status_code=status.HTTP_502_BAD_GATEWAY,
            detail=f"Failed to dispatch job to compute node {selected_node.node_id}"
        )

    # 6. Stream SSE if requested
    if req.stream:
        async def event_generator():
            try:
                while True:
                    try:
                        data = await asyncio.wait_for(queue.get(), timeout=1.0)
                    except asyncio.TimeoutError:
                        if await raw_req.is_disconnected():
                            await job_manager.cancel_job(session, job.job_id, reason="Client HTTP stream disconnected")
                            break
                        continue

                    status_val = data.get("status")
                    chunk_text = data.get("chunk", "")
                    err_val = data.get("error", "")

                    if err_val:
                        err_chunk = {
                            "id": f"chatcmpl-{job.job_id}",
                            "object": "chat.completion.chunk",
                            "created": int(time.time()),
                            "model": req.model,
                            "choices": [{"index": 0, "delta": {"content": f"\n[ERROR: {err_val}]"}, "finish_reason": "error"}]
                        }
                        yield f"data: {json.dumps(err_chunk)}\n\n"
                        break

                    if chunk_text:
                        sse_chunk = {
                            "id": f"chatcmpl-{job.job_id}",
                            "object": "chat.completion.chunk",
                            "created": int(time.time()),
                            "model": req.model,
                            "choices": [{"index": 0, "delta": {"content": chunk_text}, "finish_reason": None}]
                        }
                        yield f"data: {json.dumps(sse_chunk)}\n\n"

                    if status_val in ["COMPLETED", "FAILED", "CANCELLED"]:
                        done_chunk = {
                            "id": f"chatcmpl-{job.job_id}",
                            "object": "chat.completion.chunk",
                            "created": int(time.time()),
                            "model": req.model,
                            "choices": [{"index": 0, "delta": {}, "finish_reason": "stop"}]
                        }
                        yield f"data: {json.dumps(done_chunk)}\n\n"
                        yield "data: [DONE]\n\n"
                        break
                    
                    queue.task_done()

            finally:
                token_multiplexer.close_job_stream(job.job_id)

        return StreamingResponse(event_generator(), media_type="text/event-stream")

    # Non-streaming execution
    full_output = ""
    execution_error = ""
    final_status = ""
    try:
        while True:
            try:
                data = await asyncio.wait_for(queue.get(), timeout=110.0)
            except asyncio.TimeoutError:
                break
            status_val = data.get("status")
            final_status = status_val or final_status
            full_output += data.get("chunk", "")
            execution_error = data.get("error", "") or execution_error
            if status_val in ["COMPLETED", "FAILED", "CANCELLED"]:
                break
            queue.task_done()
    finally:
        token_multiplexer.close_job_stream(job.job_id)

    if final_status in ["FAILED", "CANCELLED"] or execution_error:
        detail = execution_error or f"Node execution ended with status {final_status}"
        raise HTTPException(status_code=status.HTTP_502_BAD_GATEWAY, detail=detail)
    if not full_output.strip():
        raise HTTPException(
            status_code=status.HTTP_504_GATEWAY_TIMEOUT,
            detail=f"Node produced no inference output before the response deadline (job {job.job_id})"
        )

    return {
        "id": f"chatcmpl-{job.job_id}",
        "object": "chat.completion",
        "created": int(time.time()),
        "model": req.model,
        "choices": [
            {
                "index": 0,
                "message": {
                    "role": "assistant",
                    "content": full_output,
                },
                "finish_reason": "stop"
            }
        ],
        "usage": {
            "prompt_tokens": len(prompt_text.split()),
            "completion_tokens": len(full_output.split()),
            "total_tokens": len(prompt_text.split()) + len(full_output.split()),
        }
    }

@router.post("/embeddings")
async def create_embeddings(
    req: EmbeddingRequest,
    _api_key: str = Depends(verify_client_api_key)
):
    inputs = [req.input] if isinstance(req.input, str) else req.input
    data_list = []
    for idx, text_str in enumerate(inputs):
        # Deterministic dummy vector for V1 foundation
        dummy_vector = [0.01 * (i % 10) for i in range(128)]
        data_list.append({
            "object": "embedding",
            "embedding": dummy_vector,
            "index": idx
        })

    return {
        "object": "list",
        "data": data_list,
        "model": req.model,
        "usage": {
            "prompt_tokens": len(inputs[0].split()) if inputs else 0,
            "total_tokens": len(inputs[0].split()) if inputs else 0
        }
    }
