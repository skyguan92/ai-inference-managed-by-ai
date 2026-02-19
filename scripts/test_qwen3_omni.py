#!/usr/bin/env python3
"""
Test Qwen3-Omni-30B model loading with Transformers
"""

import torch
from transformers import Qwen3OmniMoeForConditionalGeneration, AutoProcessor
from PIL import Image
import requests

model_path = "/mnt/data/models/.cache/models--Qwen--Qwen3-Omni-30B-A3B-Instruct"

print("Loading model...")
print(f"Path: {model_path}")
print(f"CUDA available: {torch.cuda.is_available()}")
print(f"CUDA device: {torch.cuda.get_device_name(0) if torch.cuda.is_available() else 'N/A'}")

try:
    # Load processor
    processor = AutoProcessor.from_pretrained(model_path, trust_remote_code=True)
    print("✓ Processor loaded")
    
    # Load model
    model = Qwen3OmniMoeForConditionalGeneration.from_pretrained(
        model_path,
        torch_dtype=torch.bfloat16,
        device_map="auto",
        trust_remote_code=True,
    )
    print("✓ Model loaded")
    print(f"Model device: {next(model.parameters()).device}")
    
    # Test text-only inference
    print("\nTesting text inference...")
    text = "Describe what you can do."
    messages = [
        {"role": "user", "content": text}
    ]
    
    text_input = processor.apply_chat_template(
        messages, tokenize=False, add_generation_prompt=True
    )
    inputs = processor(text=[text_input], return_tensors="pt").to(model.device)
    
    print("Generating response...")
    with torch.no_grad():
        outputs = model.generate(**inputs, max_new_tokens=100)
    
    response = processor.batch_decode(outputs, skip_special_tokens=True)[0]
    print(f"Response: {response}")
    print("\n✓ Text inference successful!")
    
except Exception as e:
    print(f"✗ Error: {e}")
    import traceback
    traceback.print_exc()
