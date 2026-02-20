# Qwen3-TTS-0.6B Real TTS Service
FROM python:3.11-slim

# Set environment variables
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
ENV MODEL_PATH=/model
ENV DEVICE=cpu

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    libsndfile1 \
    ffmpeg \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements first for better caching
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY main.py .

# Create model directory
RUN mkdir -p /model

# Expose port
EXPOSE 8002

# Health check
HEALTHCHECK --interval=10s --timeout=5s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8002/health || exit 1

# Run the service
CMD ["python", "main.py", "--model", "/model", "--port", "8002", "--device", "cpu"]
