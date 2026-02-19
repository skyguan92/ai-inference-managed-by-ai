#!/bin/bash
# ASR Service Launcher - Runs on CPU
# Usage: launch_asr.sh <model_path> <port>

MODEL_PATH="${1:-/mnt/data/models/SenseVoiceSmall}"
PORT="${2:-8001}"

echo "Starting ASR service on CPU..."
echo "Model: $MODEL_PATH"
echo "Port: $PORT"

# Force CPU usage
export CUDA_VISIBLE_DEVICES=""

# Check if FunASR is installed
if ! python3 -c "import funasr" 2>/dev/null; then
    echo "Installing FunASR..."
    pip install funasr -q
fi

# Create a simple FastAPI server for ASR
cat > /tmp/asr_server.py << 'PYTHON_EOF'
import sys
import os
import uvicorn
from fastapi import FastAPI, File, UploadFile
from fastapi.responses import JSONResponse
import tempfile

model_path = sys.argv[1] if len(sys.argv) > 1 else "/mnt/data/models/SenseVoiceSmall"
port = int(sys.argv[2]) if len(sys.argv) > 2 else 8001

app = FastAPI(title="ASR Service")

# Lazy load model
model = None

def get_model():
    global model
    if model is None:
        from funasr import AutoModel
        print(f"Loading ASR model from {model_path}...")
        model = AutoModel(
            model=model_path,
            vad_model="fsmn-vad",
            punc_model="ct-punc",
            spk_model="cam++",
        )
        print("ASR model loaded!")
    return model

@app.post("/asr")
async def transcribe(audio: UploadFile = File(...)):
    try:
        # Save uploaded file
        with tempfile.NamedTemporaryFile(delete=False, suffix=".wav") as tmp:
            content = await audio.read()
            tmp.write(content)
            tmp_path = tmp.name
        
        # Run inference
        m = get_model()
        result = m.generate(
            input=tmp_path,
            batch_size_s=300,
            hotword='魔搭'
        )
        
        # Clean up
        os.unlink(tmp_path)
        
        return JSONResponse({
            "text": result[0]["text"] if result else "",
            "success": True
        })
    except Exception as e:
        return JSONResponse(
            {"error": str(e), "success": False},
            status_code=500
        )

@app.get("/health")
def health():
    return {"status": "healthy", "model_loaded": model is not None}

if __name__ == "__main__":
    print(f"Starting ASR server on port {port}")
    uvicorn.run(app, host="0.0.0.0", port=port, log_level="info")
PYTHON_EOF

python3 /tmp/asr_server.py "$MODEL_PATH" "$PORT"
