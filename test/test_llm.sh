#!/bin/bash
# 简化的 LLM 测试脚本

echo "========================================"
echo "LLM 服务测试 (vLLM)"
echo "========================================"

# 等待 LLM 就绪
echo "等待 LLM 服务就绪..."
MAX_WAIT=600
WAITED=0
while ! curl -s http://localhost:8000/v1/models > /dev/null 2>&1; do
    if [ $WAITED -ge $MAX_WAIT ]; then
        echo "等待超时，LLM 服务未就绪"
        exit 1
    fi
    echo -n "."
    sleep 10
    WAITED=$((WAITED + 10))
done
echo ""
echo "✓ LLM 服务已就绪"

# 测试 1: 检查模型列表
echo ""
echo "测试 1: 查看可用模型"
curl -s http://localhost:8000/v1/models | jq -r '.data[].id' 2>/dev/null || echo "无法获取模型列表"

# 测试 2: 简单对话
echo ""
echo "测试 2: 简单对话"
echo "Prompt: 你好，请简短介绍一下自己"
curl -s http://localhost:8000/v1/chat/completions \
    -H "Content-Type: application/json" \
    -d '{
        "model": "qwen3-coder-next-fp8",
        "messages": [{"role": "user", "content": "你好，请简短介绍一下自己"}],
        "max_tokens": 100,
        "temperature": 0.7
    }' | jq -r '.choices[0].message.content' 2>/dev/null || echo "请求失败"

# 测试 3: 代码生成
echo ""
echo "测试 3: Python 代码生成"
echo "Prompt: 写一个 Python 函数计算斐波那契数列前 n 项"
curl -s http://localhost:8000/v1/chat/completions \
    -H "Content-Type: application/json" \
    -d '{
        "model": "qwen3-coder-next-fp8",
        "messages": [{"role": "user", "content": "写一个 Python 函数计算斐波那契数列前 n 项"}],
        "max_tokens": 200,
        "temperature": 0.3
    }' | jq -r '.choices[0].message.content' 2>/dev/null || echo "请求失败"

# 测试 4: 数学问题
echo ""
echo "测试 4: 数学问题"
echo "Prompt: 解方程：2x + 5 = 15"
curl -s http://localhost:8000/v1/chat/completions \
    -H "Content-Type: application/json" \
    -d '{
        "model": "qwen3-coder-next-fp8",
        "messages": [{"role": "user", "content": "解方程：2x + 5 = 15"}],
        "max_tokens": 100,
        "temperature": 0.3
    }' | jq -r '.choices[0].message.content' 2>/dev/null || echo "请求失败"

echo ""
echo "========================================"
echo "LLM 测试完成"
echo "========================================"
