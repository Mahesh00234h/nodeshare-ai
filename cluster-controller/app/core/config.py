import os
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    PROJECT_NAME: str = "NodeShare AI Cluster Controller"
    VERSION: str = "0.1.0"
    PROTOCOL_VERSION: str = "v1"

    # Security Keys
    CONTROLLER_API_KEY: str = os.getenv("CONTROLLER_API_KEY", "sk-cluster-controller-secret-key")
    NODE_ENROLLMENT_SECRET: str = os.getenv("NODE_ENROLLMENT_SECRET", "dev-enrollment-token")

    # Infrastructure Endpoints
    DATABASE_URL: str = os.getenv("DATABASE_URL", "mongodb://localhost:27017/nodeshare_db")
    MONGODB_URL: str = os.getenv("MONGODB_URL", os.getenv("DATABASE_URL", "mongodb://localhost:27017/nodeshare_db"))
    MONGODB_DB_NAME: str = os.getenv("MONGODB_DB_NAME", "nodeshare_db")
    REDIS_URL: str = os.getenv("REDIS_URL", "redis://localhost:6379/0")
    
    # Server Ports (Render defaults to PORT=10000)
    GRPC_PORT: int = int(os.getenv("GRPC_PORT", "50051"))
    HTTP_PORT: int = int(os.getenv("PORT", os.getenv("HTTP_PORT", "10000")))

    # Health & Heartbeat Thresholds
    HEARTBEAT_INTERVAL_SECONDS: int = int(os.getenv("HEARTBEAT_INTERVAL_SECONDS", "5"))
    HEARTBEAT_TIMEOUT_SECONDS: int = int(os.getenv("HEARTBEAT_TIMEOUT_SECONDS", "20"))
    MISSED_HEARTBEAT_DEGRADED_THRESHOLD: int = 3
    MISSED_HEARTBEAT_OFFLINE_THRESHOLD: int = 5

    # Logging
    LOG_LEVEL: str = os.getenv("LOG_LEVEL", "INFO")

    class Config:
        env_file = ".env"
        extra = "ignore"

settings = Settings()
