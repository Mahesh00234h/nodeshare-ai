import asyncio
import logging
import grpc

from app.core.config import settings
from app.grpc.generated import node_controller_pb2_grpc
from app.grpc.handlers import NodeControllerServicer

logger = logging.getLogger(__name__)

class GRPCServerManager:
    def __init__(self):
        self.server: grpc.aio.Server = None

    async def start(self):
        self.server = grpc.aio.server()
        node_controller_pb2_grpc.add_NodeControllerServiceServicer_to_server(
            NodeControllerServicer(), self.server
        )
        
        listen_addr = f"0.0.0.0:{settings.GRPC_PORT}"
        self.server.add_insecure_port(listen_addr)
        
        logger.info(f"[gRPC Server] Starting gRPC Node Controller on {listen_addr}")
        await self.server.start()

    async def stop(self):
        if self.server:
            logger.info("[gRPC Server] Stopping gRPC server...")
            await self.server.stop(5)

grpc_server_manager = GRPCServerManager()
