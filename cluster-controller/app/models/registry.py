import datetime
import logging
from typing import List, Optional
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.database.models import RegisteredModel, NodeInstalledModel

logger = logging.getLogger(__name__)

class ModelRegistry:
    async def register_model(
        self,
        session: AsyncSession,
        model_id: str,
        name: str,
        version: str = "1.0.0",
        parameter_count: str = "7B",
        quantization: str = "q4_0",
        size_bytes: int = 0,
        sha256_checksum: str = ""
    ) -> RegisteredModel:
        stmt = select(RegisteredModel).where(RegisteredModel.model_id == model_id)
        res = await session.execute(stmt)
        model = res.scalar_one_or_none()

        if not model:
            model = RegisteredModel(
                model_id=model_id,
                name=name,
                version=version,
                parameter_count=parameter_count,
                quantization=quantization,
                size_bytes=size_bytes,
                sha256_checksum=sha256_checksum,
                created_at=datetime.datetime.utcnow()
            )
            session.add(model)
        else:
            model.name = name
            model.version = version
            model.parameter_count = parameter_count
            model.quantization = quantization
            model.size_bytes = size_bytes
            model.sha256_checksum = sha256_checksum

        await session.commit()
        return model

    async def get_model(self, session: AsyncSession, model_id: str) -> Optional[RegisteredModel]:
        stmt = select(RegisteredModel).where(RegisteredModel.model_id == model_id)
        res = await session.execute(stmt)
        return res.scalar_one_or_none()

    async def list_models(self, session: AsyncSession) -> List[RegisteredModel]:
        stmt = select(RegisteredModel).order_by(RegisteredModel.model_id)
        res = await session.execute(stmt)
        return list(res.scalars().all())

    async def register_node_model(
        self,
        session: AsyncSession,
        node_id: str,
        model_id: str,
        status: str = "READY",
        size_bytes: int = 0,
        checksum: str = ""
    ):
        stmt = select(NodeInstalledModel).where(
            NodeInstalledModel.node_id == node_id,
            NodeInstalledModel.model_id == model_id
        )
        res = await session.execute(stmt)
        entry = res.scalar_one_or_none()

        if not entry:
            entry = NodeInstalledModel(
                node_id=node_id,
                model_id=model_id,
                status=status,
                size_bytes=size_bytes,
                checksum=checksum,
                last_verified_at=datetime.datetime.utcnow()
            )
            session.add(entry)
        else:
            entry.status = status
            entry.size_bytes = size_bytes
            entry.checksum = checksum
            entry.last_verified_at = datetime.datetime.utcnow()

        await session.commit()

model_registry = ModelRegistry()
