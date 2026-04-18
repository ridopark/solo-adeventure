import io
import os
import urllib.request

import numpy as np
import onnxruntime as ort
from fastapi import FastAPI, HTTPException
from fastapi.responses import Response
from huggingface_hub import hf_hub_download
from PIL import Image
from pydantic import BaseModel, Field

MODEL_REPO = os.environ.get("DEPTH_MODEL_REPO", "onnx-community/depth-anything-v2-small")
MODEL_FILE = os.environ.get("DEPTH_MODEL_FILE", "onnx/model.onnx")
THREADS = int(os.environ.get("DEPTH_THREADS", "4"))
INPUT_SIZE = 518
MEAN = np.array([0.485, 0.456, 0.406], dtype=np.float32)
STD = np.array([0.229, 0.224, 0.225], dtype=np.float32)

print(f"loading depth model: {MODEL_REPO}/{MODEL_FILE}")
_model_path = hf_hub_download(repo_id=MODEL_REPO, filename=MODEL_FILE)
_opts = ort.SessionOptions()
_opts.intra_op_num_threads = THREADS
_opts.graph_optimization_level = ort.GraphOptimizationLevel.ORT_ENABLE_ALL
session = ort.InferenceSession(_model_path, sess_options=_opts, providers=["CPUExecutionProvider"])
print(f"depth model ready: threads={THREADS}")

app = FastAPI(title="solo-adeventure depth")


class DepthRequest(BaseModel):
    image_url: str = Field(min_length=1)


@app.get("/health")
def health():
    return {"status": "ok", "model": MODEL_REPO, "threads": THREADS}


@app.post("/depth")
def depth(req: DepthRequest):
    try:
        fetch = urllib.request.Request(
            req.image_url,
            headers={
                "User-Agent": "Mozilla/5.0 (compatible; solo-adeventure-depth/1.0)",
                "Accept": "image/*",
            },
        )
        with urllib.request.urlopen(fetch, timeout=15) as r:
            raw = r.read()
        orig = Image.open(io.BytesIO(raw)).convert("RGB")
    except Exception as e:
        raise HTTPException(status_code=400, detail=f"fetch image: {e}")

    try:
        w, h = orig.size
        resized = orig.resize((INPUT_SIZE, INPUT_SIZE), Image.BILINEAR)
        x = np.asarray(resized, dtype=np.float32) / 255.0
        x = (x - MEAN) / STD
        x = np.transpose(x, (2, 0, 1))[None, ...]

        raw_depth = session.run(["predicted_depth"], {"pixel_values": x})[0][0]

        d_pil = Image.fromarray(raw_depth, mode="F").resize((w, h), Image.BILINEAR)
        d_arr = np.asarray(d_pil, dtype=np.float32)
        dmin, dmax = float(d_arr.min()), float(d_arr.max())
        if dmax - dmin < 1e-6:
            u16 = np.zeros_like(d_arr, dtype=np.uint16)
        else:
            u16 = ((d_arr - dmin) / (dmax - dmin) * 65535.0).astype(np.uint16)

        out = io.BytesIO()
        Image.fromarray(u16, mode="I;16").save(out, format="PNG", optimize=True)
        return Response(content=out.getvalue(), media_type="image/png")
    except Exception as e:
        raise HTTPException(status_code=502, detail=f"depth inference: {e}")
