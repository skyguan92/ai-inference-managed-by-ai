#!/bin/bash
# Start Qwen3-Omni-30B inference service

set -e

echo "=== Starting Qwen3-Omni-30B Inference Service ==="

# Model path
MODEL_PATH="/mnt/data/models/.cache/models--Qwen--Qwen3-Omni-30B-A3B-Instruct"
PORT=8000

# Check if model exists
if [ ! -d "$MODEL_PATH" ]; then
    echo "Error: Model not found at $MODEL_PATH"
    exit 1
fi

echo "Model path: $MODEL_PATH"
echo "Port: $PORT"

# Check for GB10-compatible vLLM image
if docker images | grep -q "zhiwen-vllm"; then
    IMAGE="zhiwen-vllm:0128"
    echo "Using GB10-compatible image: $IMAGE"
else
    IMAGE="vllm/vllm-openai:v0.15.0"
    echo "Using official vLLM image: $IMAGE"
fi

# Stop existing container if exists
if docker ps -a | grep -q "aima-qwen3-omni"; then
    echo "Stopping existing container..."
    docker stop aima-qwen3-omni 2>/dev/null || true
    docker rm aima-qwen3-omni 2>/dev/null || true
fi

echo ""
echo "Starting container..."
docker run -d \
    --name aima-qwen3-omni \
    --gpus all \
    -p $PORT:$PORT \
    -v "$MODEL_PATH:/models" \
    --memory="80g" \
    --restart unless-stopped \
    $IMAGE \
    vllm serve /models \
    --port $PORT \
    --gpu-memory-utilization 0.85 \
    --max-model-len 8192 \
    --trust-remote-code \
    --enable-chunked-prefill \
    --tensor-parallel-size 1 \
    2>&1

if [ $? -eq 0 ]; then
    echo ""
    echo "✓ Container started successfully"
    echo "Container ID: $(docker ps -q -f name=aima-qwen3-omni)"
    echo "Endpoint: http://localhost:$PORT"
    echo ""
    echo "Waiting for service to be ready..."
    
    # Wait for health check
    for i in {1..60}; do
        if curl -s http://localhost:$PORT/health > /dev/null 2>&1; then
            echo "✓ Service is ready!"
            echo ""
            echo "Test command:"
            echo "  curl http://localhost:$PORT/v1/models"
            exit 0
        fi
        sleep 5
        echo "  Attempt $i/60..."
    done
    
    echo "⚠ Service start timeout. Check logs:"
    echo "  docker logs aima-qwen3-omni"
else
    echo "✗ Failed to start container"
    exit 1
fi
