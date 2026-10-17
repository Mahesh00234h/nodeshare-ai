from fastapi import APIRouter, Depends
from sqlalchemy import text
from app.core.config import settings
from app.database.session import get_db_session

router = APIRouter(tags=["Health"])

@router.get("/health")
async def health_check():
    return {"status": "HEALTHY", "service": settings.PROJECT_NAME}

@router.get("/ready")
async def readiness_check(session = Depends(get_db_session)):
    try:
        if hasattr(session, "command"):
            await session.command("ping")
            return {"status": "READY", "database": "CONNECTED", "backend": "MongoDB Atlas"}
        else:
            await session.execute(text("SELECT 1"))
            return {"status": "READY", "database": "CONNECTED", "backend": "SQL"}
    except Exception as e:
        return {"status": "NOT_READY", "database_error": str(e)}

@router.get("/version")
async def version_info():
    return {
        "controller_version": settings.VERSION,
        "protocol_version": settings.PROTOCOL_VERSION,
        "service": settings.PROJECT_NAME
    }
