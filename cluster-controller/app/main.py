import sys
import os
import asyncio
import logging
from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

# Ensure generated gRPC stubs can resolve relative imports across environments
generated_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "grpc", "generated"))
if generated_path not in sys.path:
    sys.path.insert(0, generated_path)

from app.core.config import settings
from app.core.logging import logger
from app.database.session import init_db
from app.grpc.server import grpc_server_manager
from app.workers.health_worker import start_health_monitor_worker

from app.api import health, cluster, nodes, jobs, models, events, openai_api, grpc_fallback

@asynccontextmanager
async def lifespan(app: FastAPI):
    logger.info(f"=== Starting {settings.PROJECT_NAME} v{settings.VERSION} ===")
    
    # 1. Initialize Database Tables
    await init_db()
    logger.info("[Lifespan] Database initialized successfully")

    # 2. Start gRPC Server
    await grpc_server_manager.start()

    # 3. Start Health Monitoring Worker Task
    health_task = asyncio.create_task(start_health_monitor_worker(interval_seconds=5))

    yield

    # Shutdown
    logger.info("[Lifespan] Shutting down Cluster Controller...")
    health_task.cancel()
    await grpc_server_manager.stop()
    logger.info("[Lifespan] Cluster Controller shutdown complete")

app = FastAPI(
    title=settings.PROJECT_NAME,
    version=settings.VERSION,
    description="Central control plane for private distributed LLM compute cluster",
    lifespan=lifespan
)

# CORS Middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Mount API Routers
app.include_router(health.router)
app.include_router(cluster.router)
app.include_router(nodes.router)
app.include_router(jobs.router)
app.include_router(models.router)
app.include_router(events.router)
app.include_router(openai_api.router)
app.include_router(grpc_fallback.router)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run("app.main:app", host="0.0.0.0", port=settings.HTTP_PORT, reload=True)
