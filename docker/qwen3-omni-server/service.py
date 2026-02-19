#!/usr/bin/env python3
"""
Qwen3-Omni FastAPI Inference Service
Provides OpenAI-compatible API with async model loading
"""

import os
import json
import base64
import io
import threading
import time
from typing import Optional, List, Union
from contextlib import asynccontextmanager

import torch
import numpy as np
from PIL import Image
from fastapi import FastAPI, HTTPException, File, UploadFile, Form
from fastapi.responses import JSONResponse, StreamingResponse
from pydantic import BaseModel, Field

# Model configuration
MODEL_PATH = os.environ.get("MODEL_PATH", "/models")
DEVICE = os.environ.get("DEVICE", "cuda" if torch.cuda.is_available() else "cpu")
MAX_MODEL_LEN = int(os.environ.get("MAX_MODEL_LEN", "4096"))
PORT = int(os.environ.get("PORT", "8000"))

# Global model and processor
model = None
processor = None
model_loading = False
model_loaded = False
loading_error = None


class ChatMessage(BaseModel):
    role: str
    content: Union[str, List[dict]]


class ChatRequest(BaseModel):
    model: str = "/models"
    messages: List[ChatMessage]
    max_tokens: Optional[int] = 512
    temperature: Optional[float] = 0.7
    stream: Optional[bool] = False


class TextRequest(BaseModel):
    text: str
    max_tokens: Optional[int] = 512
    temperature: Optional[float] = 0.7


class HealthResponse(BaseModel):
    status: str
    model: str
    device: str
    loaded: bool
    loading: bool


def load_model_async():
    """Load Qwen3-Omni model and processor in background thread"""
    global model, processor, model_loading, model_loaded, loading_error
    
    if model_loading or model_loaded:
        return
    
    model_loading = True
    
    print(f"[Background] Loading model from: {MODEL_PATH}")
    print(f"[Background] Device: {DEVICE}")
    print(f"[Background] CUDA available: {torch.cuda.is_available()}")
    
    if torch.cuda.is_available():
        print(f"[Background] CUDA device: {torch.cuda.get_device_name(0)}")
        print(f"[Background] CUDA memory: {torch.cuda.get_device_properties(0).total_memory / 1e9:.2f} GB")
    
    try:
        from transformers import Qwen3OmniMoeForConditionalGeneration, AutoProcessor
        
        # Load processor
        print("[Background] Loading processor...")
        processor = AutoProcessor.from_pretrained(MODEL_PATH, trust_remote_code=True)
        print("[Background] ✓ Processor loaded")
        
        # Load model with optimizations for Jetson
        print("[Background] Loading model (this may take 10-20 minutes)...")
        print("[Background] Loading 70GB model, please wait...")
        
        # Use device_map="auto" for efficient memory allocation
        model = Qwen3OmniMoeForConditionalGeneration.from_pretrained(
            MODEL_PATH,
            torch_dtype=torch.bfloat16,
            device_map="auto",
            trust_remote_code=True,
            low_cpu_mem_usage=True,
        )
        
        print(f"[Background] ✓ Model loaded on {model.device}")
        
        # Print model info
        total_params = sum(p.numel() for p in model.parameters())
        print(f"[Background] Total parameters: {total_params / 1e9:.2f}B")
        
        model_loaded = True
        loading_error = None
        
    except Exception as e:
        print(f"[Background] Error loading model: {e}")
        import traceback
        traceback.print_exc()
        loading_error = str(e)
    finally:
        model_loading = False


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Lifespan context manager for startup/shutdown"""
    global model_loading
    
    print("Starting Qwen3-Omni Service...")
    print("⚡ Async loading mode: Model will load in background")
    
    # Start model loading in background thread
    loading_thread = threading.Thread(target=load_model_async, daemon=True)
    loading_thread.start()
    
    yield
    
    # Shutdown
    print("Shutting down Qwen3-Omni Service...")


app = FastAPI(
    title="Qwen3-Omni Inference API",
    description="OpenAI-compatible API for Qwen3-Omni multimodal model",
    version="1.0.0",
    lifespan=lifespan
)


@app.get("/health")
async def health():
    """Health check endpoint - returns loading status"""
    return HealthResponse(
        status="loading" if model_loading else ("healthy" if model_loaded else "unhealthy"),
        model="Qwen3-Omni-30B-A3B",
        device=str(model.device) if model else DEVICE,
        loaded=model_loaded,
        loading=model_loading
    )


@app.get("/v1/models")
async def list_models():
    """List available models (OpenAI compatible)"""
    return {
        "object": "list",
        "data": [
            {
                "id": "/models",
                "object": "model",
                "created": 1700000000,
                "owned_by": "qwen"
            }
        ]
    }


@app.post("/v1/chat/completions")
async def chat_completion(request: ChatRequest):
    """Chat completion endpoint (OpenAI compatible)"""
    if not model_loaded:
        if model_loading:
            raise HTTPException(status_code=503, detail="Model is still loading, please wait...")
        else:
            raise HTTPException(status_code=503, detail="Model failed to load")
    
    try:
        # Build conversation
        conversation = []
        for msg in request.messages:
            if isinstance(msg.content, str):
                conversation.append({"role": msg.role, "content": msg.content})
            else:
                conversation.append({"role": msg.role, "content": msg.content})
        
        # Apply chat template
        text = processor.apply_chat_template(
            conversation,
            tokenize=False,
            add_generation_prompt=True
        )
        
        # Process inputs
        inputs = processor(
            text=[text],
            return_tensors="pt",
            padding=True
        ).to(model.device)
        
        # Generate
        with torch.no_grad():
            outputs = model.generate(
                **inputs,
                max_new_tokens=request.max_tokens,
                temperature=request.temperature,
                do_sample=request.temperature > 0,
            )
        
        # Decode response
        response_text = processor.batch_decode(
            outputs[:, inputs.input_ids.shape[1]:],
            skip_special_tokens=True
        )[0]
        
        return {
            "id": "chatcmpl-qwen3omni",
            "object": "chat.completion",
            "created": int(time.time()),
            "model": request.model,
            "choices": [
                {
                    "index": 0,
                    "message": {
                        "role": "assistant",
                        "content": response_text
                    },
                    "finish_reason": "stop"
                }
            ],
            "usage": {
                "prompt_tokens": inputs.input_ids.shape[1],
                "completion_tokens": len(outputs[0]) - inputs.input_ids.shape[1],
                "total_tokens": len(outputs[0])
            }
        }
        
    except Exception as e:
        print(f"Error in chat completion: {e}")
        import traceback
        traceback.print_exc()
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/inference/text")
async def text_inference(request: TextRequest):
    """Simple text inference endpoint"""
    if not model_loaded:
        if model_loading:
            raise HTTPException(status_code=503, detail="Model is still loading, please wait...")
        else:
            raise HTTPException(status_code=503, detail="Model failed to load")
    
    try:
        # Build conversation
        conversation = [
            {"role": "user", "content": request.text}
        ]
        
        # Apply chat template
        text = processor.apply_chat_template(
            conversation,
            tokenize=False,
            add_generation_prompt=True
        )
        
        # Process
        inputs = processor(text=[text], return_tensors="pt").to(model.device)
        
        # Generate
        with torch.no_grad():
            outputs = model.generate(
                **inputs,
                max_new_tokens=request.max_tokens,
                temperature=request.temperature,
                do_sample=request.temperature > 0,
            )
        
        # Decode
        response = processor.batch_decode(
            outputs[:, inputs.input_ids.shape[1]:],
            skip_special_tokens=True
        )[0]
        
        return {
            "text": response,
            "input_tokens": inputs.input_ids.shape[1],
            "output_tokens": len(outputs[0]) - inputs.input_ids.shape[1]
        }
        
    except Exception as e:
        print(f"Error in text inference: {e}")
        import traceback
        traceback.print_exc()
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/status")
async def status():
    """Get detailed loading status"""
    return {
        "model": "Qwen3-Omni-30B-A3B",
        "loaded": model_loaded,
        "loading": model_loading,
        "error": loading_error,
        "device": str(model.device) if model else DEVICE,
    }


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=PORT)
