FROM zhiwen-vllm:0128

# Install system dependencies
RUN apt-get update && apt-get install -y \
    libsndfile1 \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
RUN pip install --no-cache-dir \
    fastapi \
    uvicorn \
    python-multipart \
    pillow \
    pydantic \
    accelerate

# Fix Qwen3-Omni bug in transformers
RUN sed -i 's/sliding_window: int | None = None,$/use_sliding_window: bool = False,\n        sliding_window: int | None = None,/g' /usr/local/lib/python3.12/dist-packages/transformers/models/qwen3_omni_moe/configuration_qwen3_omni_moe.py && \
    sed -i 's/self.sliding_window = sliding_window if self.use_sliding_window else None/self.use_sliding_window = use_sliding_window\n        self.sliding_window = sliding_window if use_sliding_window else None/g' /usr/local/lib/python3.12/dist-packages/transformers/models/qwen3_omni_moe/configuration_qwen3_omni_moe.py

# Set working directory
WORKDIR /app

# Copy service code
COPY service.py /app/

# Environment variables
ENV MODEL_PATH=/models
ENV PORT=8000
ENV PYTHONUNBUFFERED=1

# Expose port
EXPOSE 8000

# Start service
CMD ["python3", "-u", "/app/service.py"]
