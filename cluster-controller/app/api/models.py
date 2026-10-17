from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel
from sqlalchemy.ext.asyncio import AsyncSession
from app.database.session import get_db_session
from app.core.security import verify_client_api_key
from app.models.registry import model_registry

router = APIRouter(prefix="/api/models", tags=["Models Admin"])

class RegisterModelRequest(BaseModel):
    model_id: str
    name: str
    version: str = "1.0.0"
    parameter_count: str = "7B"
    quantization: str = "q4_0"
    size_bytes: int = 0
    sha256_checksum: str = ""

@router.get("")
async def list_registered_models(
    session: AsyncSession = Depends(get_db_session),
    _api_key: str = Depends(verify_client_api_key)
):
    models = await model_registry.list_models(session)
    return [
        {
            "model_id": m.model_id,
            "name": m.name,
            "version": m.version,
            "parameter_count": m.parameter_count,
            "quantization": m.quantization,
            "size_bytes": m.size_bytes,
            "sha256_checksum": m.sha256_checksum,
            "created_at": m.created_at.isoformat(),
        } for m in models
    ]

@router.post("/register")
async def register_new_model(
    req: RegisterModelRequest,
    session: AsyncSession = Depends(get_db_session),
    _api_key: str = Depends(verify_client_api_key)
):
    m = await model_registry.register_model(
        session=session,
        model_id=req.model_id,
        name=req.name,
        version=req.version,
        parameter_count=req.parameter_count,
        quantization=req.quantization,
        size_bytes=req.size_bytes,
        sha256_checksum=req.sha256_checksum
    )
    return {
        "status": "REGISTERED",
        "model_id": m.model_id,
        "name": m.name,
    }
