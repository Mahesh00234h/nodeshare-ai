import logging
from typing import AsyncGenerator, Optional, Any
from app.core.config import settings

logger = logging.getLogger(__name__)

is_mongo = settings.DATABASE_URL.startswith("mongodb://") or settings.DATABASE_URL.startswith("mongodb+srv://")

mongo_client: Optional[Any] = None
mongo_db: Optional[Any] = None

if is_mongo:
    from motor.motor_asyncio import AsyncIOMotorClient
    mongo_client = AsyncIOMotorClient(settings.DATABASE_URL)
    mongo_db = mongo_client.get_database(settings.MONGODB_DB_NAME)

from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession, async_sessionmaker
sql_url = settings.DATABASE_URL if not is_mongo else "sqlite+aiosqlite:///./cluster_controller.db"
engine = create_async_engine(sql_url, echo=False, future=True)
async_session_factory = async_sessionmaker(
    bind=engine,
    class_=AsyncSession,
    expire_on_commit=False,
    autoflush=False,
)

async def init_db():
    if is_mongo and mongo_db is not None:
        logger.info(f"Initialized MongoDB connection to database '{settings.MONGODB_DB_NAME}'")
        try:
            await mongo_db.nodes.create_index("node_id", unique=True)
            await mongo_db.models.create_index("model_id", unique=True)
            await mongo_db.jobs.create_index("job_id", unique=True)
        except Exception as e:
            logger.warning(f"MongoDB index initialization notice: {e}")
    from app.database.models import Base
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)

async def get_db_session() -> AsyncGenerator[Any, None]:
    async with async_session_factory() as session:
        try:
            yield session
            await session.commit()
        except Exception:
            await session.rollback()
            raise
        finally:
            await session.close()
