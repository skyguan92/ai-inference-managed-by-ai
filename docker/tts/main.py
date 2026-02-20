#!/usr/bin/env python3
"""
Qwen3-TTS-0.6B CPU Inference Service
FastAPI wrapper for Qwen3-TTS real model inference using qwen-tts package
"""

import os
import io
import base64
import tempfile
from typing import Optional, Literal
from contextlib import asynccontextmanager
from pydantic import BaseModel, Field

import torch
import numpy as np
from scipy.io import wavfile
from fastapi import FastAPI, HTTPException
from fastapi.responses import FileResponse, JSONResponse

# Import Qwen3-TTS
from qwen_tts import Qwen3TTSModel

# Configuration
MODEL_PATH = os.getenv("MODEL_PATH", "/model")
DEVICE = os.getenv("DEVICE", "cpu")
CACHE_DIR = os.getenv("CACHE_DIR", "/cache")
REFERENCE_AUDIO = os.getenv("REFERENCE_AUDIO", "/model/reference.wav")
REFERENCE_TEXT = os.getenv("REFERENCE_TEXT", "your power is sufficient i said")

# Global model
model = None
model_loaded = False


class TTSRequest(BaseModel):
    """TTS request body."""
    text: str = Field(..., description="Text to synthesize", min_length=1, max_length=5000)
    voice: str = Field("default", description="Voice to use")
    speed: float = Field(1.0, description="Speech speed multiplier", ge=0.5, le=2.0)
    response_format: Literal["wav"] = Field("wav", description="Output audio format")


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Load model on startup and cleanup on shutdown."""
    global model_loaded, model

    print(f"Loading TTS service...")
    print(f"Model path: {MODEL_PATH}")
    print(f"Using device: {DEVICE}")
    print(f"Reference audio: {REFERENCE_AUDIO}")

    # Check if model files exist
    model_files_exist = os.path.exists(MODEL_PATH) and any(
        f.endswith(('.bin', '.safetensors', '.pt', '.pth'))
        for f in os.listdir(MODEL_PATH)
    ) if os.path.exists(MODEL_PATH) else False

    if model_files_exist:
        try:
            print("Loading real Qwen3-TTS model...")
            model = Qwen3TTSModel.from_pretrained(MODEL_PATH)
            print(f"✓ Model loaded successfully!")
            print(f"✓ Model type: {type(model).__name__}")
            model_loaded = True
        except Exception as e:
            print(f"✗ Error loading model: {e}")
            import traceback
            traceback.print_exc()
            print("Service will run in fallback mode")
            model_loaded = False
    else:
        print(f"✗ Model files not found at {MODEL_PATH}")
        print("Service will run in fallback mode")
        model_loaded = False

    yield

    # Cleanup
    print("Shutting down TTS service...")
    model_loaded = False
    model = None


app = FastAPI(
    title="Qwen3-TTS-0.6B Service",
    description="CPU-based TTS inference service using Qwen3-TTS real model",
    version="1.0.0",
    lifespan=lifespan
)


def generate_fallback_audio(text: str, speed: float = 1.0, sample_rate: int = 24000):
    """Generate fallback audio (sine wave) when model is not available."""
    has_chinese = any('\u4e00' <= c <= '\u9fff' for c in text)
    chars_per_sec = 3.0 if has_chinese else 5.0
    duration = max(1.0, len(text) / chars_per_sec / speed)

    t = np.linspace(0, duration, int(sample_rate * duration), False)
    freq = 440
    audio = (0.3 * np.sin(2 * np.pi * freq * t) +
             0.15 * np.sin(2 * np.pi * freq * 2 * t) +
             0.1 * np.sin(2 * np.pi * freq * 3 * t))

    envelope = np.ones_like(audio)
    fade_samples = min(int(0.01 * sample_rate), len(audio) // 4)
    envelope[:fade_samples] = np.linspace(0, 1, fade_samples)
    envelope[-fade_samples:] = np.linspace(1, 0, fade_samples)
    audio = audio * envelope

    return (audio * 32767).astype(np.int16), sample_rate, duration


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    return {
        "status": "healthy",
        "model": "Qwen3-TTS-0.6B",
        "device": DEVICE,
        "loaded": model_loaded,
        "real_model": model is not None,
        "fallback_mode": model is None
    }


@app.get("/")
async def root():
    """Root endpoint with service info."""
    return {
        "service": "Qwen3-TTS-0.6B",
        "version": "1.0.0",
        "model_loaded": model_loaded,
        "device": DEVICE,
        "endpoints": {
            "health": "/health",
            "tts": "/v1/tts (POST)",
            "openai_compat": "/v1/audio/speech (POST)"
        }
    }


@app.post("/v1/tts")
async def text_to_speech(request: TTSRequest):
    """
    Convert text to speech using Qwen3-TTS model.
    """
    try:
        if model is not None and model_loaded:
            # Use real Qwen3-TTS model
            print(f"Generating speech for: {request.text[:50]}...")
            
            # Check if reference audio exists
            ref_audio = REFERENCE_AUDIO if os.path.exists(REFERENCE_AUDIO) else None
            ref_text = REFERENCE_TEXT if ref_audio else None
            
            # Generate speech with voice cloning
            audios, sample_rate = model.generate_voice_clone(
                text=request.text,
                ref_audio=ref_audio,
                ref_text=ref_text,
                language='auto'
            )
            
            # Convert to int16
            if isinstance(audios[0], np.floating):
                audio_int16 = (audios[0] * 32767).astype(np.int16)
            else:
                audio_int16 = audios[0]
            
            # Save to buffer
            buffer = io.BytesIO()
            wavfile.write(buffer, sample_rate, audio_int16)
            buffer.seek(0)
            
            audio_base64 = base64.b64encode(buffer.read()).decode("utf-8")
            duration = len(audio_int16) / sample_rate
            
            print(f"✓ Generated {duration:.2f}s of audio")
            
            return {
                "text": request.text,
                "audio_base64": audio_base64,
                "sample_rate": sample_rate,
                "format": request.response_format,
                "duration_seconds": round(duration, 2),
                "real_model": True
            }
        else:
            # Fallback mode
            audio_array, sample_rate, duration = generate_fallback_audio(
                request.text, request.speed
            )
            
            buffer = io.BytesIO()
            wavfile.write(buffer, sample_rate, audio_array)
            buffer.seek(0)
            
            audio_base64 = base64.b64encode(buffer.read()).decode("utf-8")
            
            return {
                "text": request.text,
                "audio_base64": audio_base64,
                "sample_rate": sample_rate,
                "format": request.response_format,
                "duration_seconds": round(duration, 2),
                "real_model": False,
                "note": "Fallback mode - sine wave audio"
            }

    except Exception as e:
        import traceback
        traceback.print_exc()
        raise HTTPException(status_code=500, detail=f"TTS error: {str(e)}")


# OpenAI-compatible endpoint
class OpenAITTSRequest(BaseModel):
    """OpenAI-compatible TTS request."""
    model: str = Field("qwen3-tts", description="Model to use")
    input: str = Field(..., description="Text to synthesize", min_length=1, max_length=5000)
    voice: str = Field("alloy", description="Voice to use")
    response_format: Literal["wav"] = Field("wav", description="Output format")
    speed: float = Field(1.0, description="Speech speed", ge=0.25, le=4.0)


@app.post("/v1/audio/speech")
async def openai_compatible_tts(request: OpenAITTSRequest):
    """OpenAI-compatible text-to-speech endpoint."""
    if model is None or not model_loaded:
        raise HTTPException(status_code=503, detail="Model not loaded")

    try:
        # Use real Qwen3-TTS model
        ref_audio = REFERENCE_AUDIO if os.path.exists(REFERENCE_AUDIO) else None
        ref_text = REFERENCE_TEXT if ref_audio else None
        
        audios, sample_rate = model.generate_voice_clone(
            text=request.input,
            ref_audio=ref_audio,
            ref_text=ref_text,
            language='auto'
        )
        
        # Convert to int16
        if isinstance(audios[0], np.floating):
            audio_int16 = (audios[0] * 32767).astype(np.int16)
        else:
            audio_int16 = audios[0]
        
        # Save to temporary file
        with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
            tmp_path = tmp.name
            wavfile.write(tmp_path, sample_rate, audio_int16)
        
        return FileResponse(
            tmp_path,
            media_type="audio/wav",
            filename="speech.wav"
        )

    except Exception as e:
        import traceback
        traceback.print_exc()
        raise HTTPException(status_code=500, detail=f"TTS error: {str(e)}")


if __name__ == "__main__":
    import uvicorn
    import argparse
    
    parser = argparse.ArgumentParser(description="Qwen3-TTS Service")
    parser.add_argument("--model", type=str, default=MODEL_PATH, help="Model path")
    parser.add_argument("--port", type=int, default=8002, help="Server port")
    parser.add_argument("--device", type=str, default=DEVICE, help="Device (cpu/cuda)")
    args = parser.parse_args()
    
    MODEL_PATH = args.model
    DEVICE = args.device
    
    print(f"Starting Qwen3-TTS server on port {args.port}")
    uvicorn.run(app, host="0.0.0.0", port=args.port, log_level="info")
