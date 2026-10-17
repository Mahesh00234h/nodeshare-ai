import logging
from typing import Dict, Any

logger = logging.getLogger(__name__)

class JobRecoveryPlanner:
    def evaluate_recovery(self, job_type: str, status: str, has_checkpoint: bool = False) -> Dict[str, Any]:
        # For V1 Node Agent model inference, request-level inference is non-recoverable once mid-stream unless retried from start
        if job_type == "INFERENCE":
            return {
                "recoverable_type": "NON_RECOVERABLE",
                "strategy": "RETRY_FROM_START",
                "can_retry": True
            }
        elif job_type == "DOWNLOAD_MODEL":
            return {
                "recoverable_type": "RECOVERABLE",
                "strategy": "RESUME_DOWNLOAD",
                "can_retry": True
            }
        return {
            "recoverable_type": "UNKNOWN",
            "strategy": "NONE",
            "can_retry": False
        }

job_recovery_planner = JobRecoveryPlanner()
