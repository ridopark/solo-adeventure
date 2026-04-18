# Depth sidecar

Runs Depth-Anything-V2-Small (ONNX) on CPU to produce a 16-bit PNG depth map
per illustration. The frontend uses it to drive a three.js parallax plane that
makes the still image feel 3D.

Model: `onnx-community/depth-anything-v2-small` (fp32, 99 MB). Cached by
huggingface_hub on first boot, reused thereafter.

## Run locally

    python3 -m venv venv
    ./venv/bin/pip install -r requirements.txt
    ./venv/bin/uvicorn main:app --host 127.0.0.1 --port 8086

## Endpoint

    POST /depth  { "image_url": "https://..." }
      -> 200 image/png (16-bit grayscale, same dims as input; brighter = closer)
      -> 400 fetch failure
      -> 502 inference failure

Warm inference on 4 vCPU ARM: ~250-500 ms.
Cold boot (first request after process start): +2-4 s for model load.
