import io
import os

import edge_tts
from fastapi import FastAPI, HTTPException
from fastapi.responses import Response
from pydantic import BaseModel, Field

app = FastAPI(title="solo-adeventure tts")

DEFAULT_VOICE = os.environ.get("TTS_DEFAULT_VOICE", "en-US-AndrewNeural")
DEFAULT_RATE = os.environ.get("TTS_DEFAULT_RATE", "+0%")
MAX_CHARS = int(os.environ.get("TTS_MAX_CHARS", "6000"))


class TTSRequest(BaseModel):
    text: str = Field(min_length=1, max_length=MAX_CHARS)
    voice: str | None = None
    rate: str | None = None


@app.get("/health")
def health():
    return {"status": "ok", "voice": DEFAULT_VOICE}


@app.post("/tts")
async def tts(req: TTSRequest):
    voice = req.voice or DEFAULT_VOICE
    rate = req.rate or DEFAULT_RATE
    try:
        communicate = edge_tts.Communicate(req.text, voice, rate=rate)
        buf = io.BytesIO()
        async for chunk in communicate.stream():
            if chunk.get("type") == "audio":
                buf.write(chunk.get("data", b""))
        data = buf.getvalue()
    except Exception as e:
        raise HTTPException(status_code=502, detail=f"edge-tts: {e}") from e
    if not data:
        raise HTTPException(status_code=502, detail="edge-tts returned no audio")
    return Response(content=data, media_type="audio/mpeg")
