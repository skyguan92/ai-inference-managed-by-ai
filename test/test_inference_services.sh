#!/bin/bash
# AIMA 推理服务测试脚本

set -e

echo "========================================"
echo "AIMA 推理服务测试"
echo "========================================"
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查服务健康状态
check_health() {
    local name=$1
    local url=$2
    
    echo -n "检查 $name 服务... "
    if curl -s "$url" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 健康${NC}"
        return 0
    else
        echo -e "${RED}✗ 不可用${NC}"
        return 1
    fi
}

# 测试 LLM 服务
test_llm() {
    echo ""
    echo "========================================"
    echo "1. LLM 服务测试 (Qwen3-Coder)"
    echo "========================================"
    
    # 测试 1: 简单问候
    echo ""
    echo "测试 1.1: 简单问候"
    echo "Prompt: 你好，请简短介绍一下自己"
    curl -s http://localhost:8000/v1/chat/completions \
        -H "Content-Type: application/json" \
        -d '{
            "model": "qwen3-coder-next-fp8",
            "messages": [{"role": "user", "content": "你好，请简短介绍一下自己"}],
            "max_tokens": 100,
            "temperature": 0.7
        }' | jq -r '.choices[0].message.content' 2>/dev/null || echo "请求失败"
    
    # 测试 2: 代码生成
    echo ""
    echo "测试 1.2: Python 代码生成"
    echo "Prompt: 写一个 Python 函数计算斐波那契数列第 n 项"
    curl -s http://localhost:8000/v1/chat/completions \
        -H "Content-Type: application/json" \
        -d '{
            "model": "qwen3-coder-next-fp8",
            "messages": [{"role": "user", "content": "写一个 Python 函数计算斐波那契数列第 n 项"}],
            "max_tokens": 200,
            "temperature": 0.3
        }' | jq -r '.choices[0].message.content' 2>/dev/null || echo "请求失败"
    
    # 测试 3: 数学问题
    echo ""
    echo "测试 1.3: 数学问题"
    echo "Prompt: 解方程：2x + 5 = 15"
    curl -s http://localhost:8000/v1/chat/completions \
        -H "Content-Type: application/json" \
        -d '{
            "model": "qwen3-coder-next-fp8",
            "messages": [{"role": "user", "content": "解方程：2x + 5 = 15"}],
            "max_tokens": 100,
            "temperature": 0.3
        }' | jq -r '.choices[0].message.content' 2>/dev/null || echo "请求失败"
}

# 测试 ASR 服务
test_asr() {
    echo ""
    echo "========================================"
    echo "2. ASR 服务测试 (SenseVoice)"
    echo "========================================"
    
    # 检查参考音频文件
    if [ -f "/mnt/data/models/Qwen3-TTS-0.6B/reference.wav" ]; then
        echo ""
        echo "测试 2.1: 语音识别"
        echo "使用参考音频文件进行测试"
        
        # 尝试使用参考音频测试
        # 注意：实际 API 端点需要根据 ASR 服务的具体实现调整
        echo "尝试调用 ASR API..."
        curl -s -X POST http://localhost:8001/asr \
            -F "audio=@/mnt/data/models/Qwen3-TTS-0.6B/reference.wav" 2>/dev/null | jq . 2>/dev/null || echo "ASR API 调用失败，请检查 API 端点"
    else
        echo "参考音频文件不存在，跳过 ASR 测试"
    fi
}

# 测试 TTS 服务
test_tts() {
    echo ""
    echo "========================================"
    echo "3. TTS 服务测试 (Qwen3-TTS)"
    echo "========================================"
    
    # 测试 1: 简单文本
    echo ""
    echo "测试 3.1: 简单文本合成"
    echo "Text: 你好，这是一个测试"
    
    # 合成并保存音频
    curl -s -X POST http://localhost:8002/tts \
        -H "Content-Type: application/json" \
        -d '{"text": "你好，这是一个测试", "voice": "default"}' \
        --output /tmp/tts_test1.wav 2>/dev/null || echo "TTS 请求失败"
    
    if [ -f "/tmp/tts_test1.wav" ] && [ -s "/tmp/tts_test1.wav" ]; then
        echo -e "${GREEN}✓ 音频合成成功${NC}"
        ls -lh /tmp/tts_test1.wav
    else
        echo -e "${RED}✗ 音频合成失败${NC}"
    fi
    
    # 测试 2: 长文本
    echo ""
    echo "测试 3.2: 长文本合成"
    echo "Text: 人工智能正在改变我们的生活方式"
    
    curl -s -X POST http://localhost:8002/tts \
        -H "Content-Type: application/json" \
        -d '{"text": "人工智能正在改变我们的生活方式", "voice": "default"}' \
        --output /tmp/tts_test2.wav 2>/dev/null || echo "TTS 请求失败"
    
    if [ -f "/tmp/tts_test2.wav" ] && [ -s "/tmp/tts_test2.wav" ]; then
        echo -e "${GREEN}✓ 音频合成成功${NC}"
        ls -lh /tmp/tts_test2.wav
    else
        echo -e "${RED}✗ 音频合成失败${NC}"
    fi
}

# 主测试流程
main() {
    echo "开始测试前准备..."
    
    # 检查依赖
    if ! command -v curl > /dev/null 2>&1; then
        echo "错误：需要安装 curl"
        exit 1
    fi
    
    if ! command -v jq > /dev/null 2>&1; then
        echo "警告：建议安装 jq 以获得更好的输出格式"
    fi
    
    # 检查服务健康状态
    echo ""
    echo "检查服务健康状态..."
    
    LLM_HEALTH=$(check_health "LLM" "http://localhost:8000/health")
    ASR_HEALTH=$(check_health "ASR" "http://localhost:8001/")
    TTS_HEALTH=$(check_health "TTS" "http://localhost:8002/health")
    
    # 等待 vLLM 加载完成
    echo ""
    echo "等待 LLM 服务就绪..."
    MAX_WAIT=300
    WAITED=0
    while ! curl -s http://localhost:8000/v1/models > /dev/null 2>&1; do
        if [ $WAITED -ge $MAX_WAIT ]; then
            echo -e "${RED}等待超时，LLM 服务未就绪${NC}"
            break
        fi
        echo -n "."
        sleep 5
        WAITED=$((WAITED + 5))
    done
    echo ""
    
    # 执行测试
    if curl -s http://localhost:8000/v1/models > /dev/null 2>&1; then
        test_llm
    else
        echo "LLM 服务未就绪，跳过 LLM 测试"
    fi
    
    if curl -s http://localhost:8001/ > /dev/null 2>&1; then
        test_asr
    else
        echo "ASR 服务未就绪，跳过 ASR 测试"
    fi
    
    if curl -s http://localhost:8002/health > /dev/null 2>&1; then
        test_tts
    else
        echo "TTS 服务未就绪，跳过 TTS 测试"
    fi
    
    echo ""
    echo "========================================"
    echo "测试完成"
    echo "========================================"
}

# 运行测试
main
