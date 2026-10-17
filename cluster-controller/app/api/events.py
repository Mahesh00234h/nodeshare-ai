import asyncio
import json
from fastapi import APIRouter, WebSocket, WebSocketDisconnect
from app.events.publisher import event_publisher

router = APIRouter(prefix="/api/events", tags=["Live Events"])

@router.websocket("/ws")
async def websocket_events_endpoint(websocket: WebSocket):
    await websocket.accept()
    queue = event_publisher.subscribe()

    try:
        while True:
            event = await queue.get()
            await websocket.send_text(json.dumps(event))
            queue.task_done()
    except WebSocketDisconnect:
        event_publisher.unsubscribe(queue)
    except Exception:
        event_publisher.unsubscribe(queue)
