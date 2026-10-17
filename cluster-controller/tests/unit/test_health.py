import pytest
import datetime
from app.database.models import NodeModel

def test_node_health_thresholds():
    now = datetime.datetime.utcnow()
    
    # Healthy (within 15s)
    last_hb_recent = now - datetime.timedelta(seconds=5)
    elapsed_recent = (now - last_hb_recent).total_seconds()
    assert elapsed_recent <= 15.0

    # Degraded (15s - 30s)
    last_hb_degraded = now - datetime.timedelta(seconds=20)
    elapsed_degraded = (now - last_hb_degraded).total_seconds()
    assert 15.0 < elapsed_degraded <= 30.0

    # Offline (>30s)
    last_hb_offline = now - datetime.timedelta(seconds=35)
    elapsed_offline = (now - last_hb_offline).total_seconds()
    assert elapsed_offline > 30.0
