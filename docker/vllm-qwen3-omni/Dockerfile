FROM zhiwen-vllm:0128

# Install latest transformers for Qwen3-Omni support
RUN pip install --no-cache-dir --upgrade \
    git+https://github.com/huggingface/transformers.git@main \
    2>/dev/null || pip install --no-cache-dir transformers>=5.0.0

# Install accelerate for model loading
RUN pip install --no-cache-dir accelerate

ENV VLLM_ATTENTION_BACKEND=FLASH_ATTN
ENV VLLM_WORKER_MULTIPROC_METHOD=spawn

EXPOSE 8000
ENTRYPOINT ["/opt/nvidia/nvidia_entrypoint.sh"]
