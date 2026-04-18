# TTS sidecar

Wraps the `edge-tts` Python package (Microsoft Edge's free Azure Neural voices)
as a single-endpoint HTTP service. The Go backend posts narrative text here and
receives MP3 bytes back.

Isolated as a sidecar so that when MS rotates the undocumented endpoint, the fix
is `pip install -U edge-tts` + systemd restart without touching the Go binary.

## Run locally

    python3 -m venv venv
    ./venv/bin/pip install -r requirements.txt
    ./venv/bin/uvicorn main:app --host 127.0.0.1 --port 8085

## Endpoint

    POST /tts  { "text": "...", "voice": "en-US-AndrewNeural", "rate": "+0%" }
      -> 200 audio/mpeg
      -> 502 on edge-tts failure (rotate voice/rate or wait for upstream fix)
