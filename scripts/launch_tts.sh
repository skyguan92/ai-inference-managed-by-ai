#!/bin/bash
# TTS Service Launcher - Runs on CPU
# Usage: launch_tts.sh <model_path> <port>

MODEL_PATH="${1:-/mnt/data/models/Qwen3-TTS-0.6B}"
PORT="${2:-8002}"

echo "Starting TTS service on CPU..."
echo "Model: $MODEL_PATH"
echo "Port: $PORT"

# Force CPU usage
export CUDA_VISIBLE_DEVICES=""

# Check if required packages are installed
if ! python3 -c "import transformers" 2>/dev/null; then
    echo "Installing transformers..."
    pip install transformers torch -q
fi

# Create a simple FastAPI server for TTS
cat > /tmp/tts_server.py << 'PYTHON_EOF'
import sys
import os
import uvicorn
from fastapi import FastAPI
from fastapi.responses import JSONResponse, Response
from pydantic import BaseModel
import base64
import tempfile

model_path = sys.argv[1] if len(sys.argv) > 1 else "/mnt/data/models/Qwen3-TTS-0.6B"
port = int(sys.argv[2]) if len(sys.argv) > 2 else 8002

app = FastAPI(title="TTS Service")

# Lazy load model
tts_pipeline = None

def get_pipeline():
    global tts_pipeline
    if tts_pipeline is None:
        from transformers import pipeline
        import torch
        print(f"Loading TTS model from {model_path}...")
        device = "cpu"
        tts_pipeline = pipeline(
            "text-to-speech",
            model=model_path,
            device=device,
            torch_dtype=torch.float32
        )
        print("TTS model loaded!")
    return tts_pipeline

class TTSRequest(BaseModel):
    text: str
    voice: str = "default"
    speed: float = 1.0

@app.post("/tts")
def synthesize(request: TTSRequest):
    try:
        pipeline = get_pipeline()
        
        # Generate speech
        result = pipeline(request.text)
        
        # Convert to base64
        audio_bytes = result["audio"]
        audio_b64 = base64.b64encode(audio_bytes).decode('utf-8')
        
        return JSONResponse({
            "audio": audio_b64,
            "sampling_rate": result.get("sampling_rate", 24000),
            "success": True
        })
    except Exception as e:
        import traceback
        traceback.print_exc()
        return JSONResponse(
            {"error": str(e), "success": False},
            status_code=500
        )

@app.get("/health")
def health():
    return {"status": "healthy", "model_loaded": tts_pipeline is not None}

if __name__ == "__main__":
    print(f"Starting TTS server on port {port}")
    uvicorn.run(app, host="0.0.0.0", port=port, log_level="info")
PYTHON_EOF

python3 /tmp/tts_server.py "$MODEL_PATH" "$PORT"
