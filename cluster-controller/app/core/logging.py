import json
import logging
import sys
from datetime import datetime, timezone
from app.core.config import settings

class JSONFormatter(logging.Formatter):
    def format(self, record):
        log_data = {
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "level": record.levelname,
            "component": "controller",
            "message": record.getMessage(),
            "logger": record.name,
        }
        if hasattr(record, "event"):
            log_data["event"] = record.event
        if hasattr(record, "node_id"):
            log_data["node_id"] = record.node_id
        if hasattr(record, "job_id"):
            log_data["job_id"] = record.job_id

        return json.dumps(log_data)

def setup_logging():
    log_level = getattr(logging, settings.LOG_LEVEL.upper(), logging.INFO)
    logger = logging.getLogger()
    logger.setLevel(log_level)

    handler = logging.StreamHandler(sys.stdout)
    handler.setFormatter(JSONFormatter())
    
    # Avoid duplicate handlers
    if not logger.handlers:
        logger.addHandler(handler)

    return logger

logger = setup_logging()
