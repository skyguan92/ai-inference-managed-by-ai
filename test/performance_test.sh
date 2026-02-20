#!/bin/bash
# AIMA 推理性能测试脚本
# 测试 GLM-4.7-Flash (GPU), SenseVoice (CPU), Qwen3-TTS (CPU) 同时运行时的性能

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 测试配置
LLM_ITERATIONS=${LLM_ITERATIONS:-5}
TTS_ITERATIONS=${TTS_ITERATIONS:-5}
CONCURRENT_REQUESTS=${CONCURRENT_REQUESTS:-3}

# 结果存储
RESULTS_DIR="/tmp/aima-perf-results"
mkdir -p "$RESULTS_DIR"

echo "========================================"
echo "AIMA 推理性能测试"
echo "========================================"
echo ""
echo "测试配置:"
echo "  - LLM 迭代次数: $LLM_ITERATIONS"
echo "  - TTS 迭代次数: $TTS_ITERATIONS"
echo "  - 并发请求数: $CONCURRENT_REQUESTS"
echo ""

# 检查服务可用性
check_services() {
    echo -e "${BLUE}检查服务可用性...${NC}"
    
    local all_ok=true
    
    # LLM
    if curl -s http://localhost:8000/v1/models > /dev/null 2>&1; then
        echo -e "  LLM (GLM-4.7-Flash): ${GREEN}✓${NC}"
    else
        echo -e "  LLM (GLM-4.7-Flash): ${RED}✗${NC}"
        all_ok=false
    fi
    
    # ASR
    if curl -s http://localhost:8001/health | grep -q "healthy"; then
        echo -e "  ASR (SenseVoice): ${GREEN}✓${NC}"
    else
        echo -e "  ASR (SenseVoice): ${RED}✗${NC}"
        all_ok=false
    fi
    
    # TTS
    if curl -s http://localhost:8002/health | grep -q "healthy"; then
        echo -e "  TTS (Qwen3-TTS): ${GREEN}✓${NC}"
    else
        echo -e "  TTS (Qwen3-TTS): ${RED}✗${NC}"
        all_ok=false
    fi
    
    if [ "$all_ok" = false ]; then
        echo -e "${RED}错误: 部分服务不可用${NC}"
        exit 1
    fi
    echo ""
}

# 记录资源使用
record_resources() {
    local label=$1
    docker stats --no-stream --format "{{.Name}},{{.CPUPerc}},{{.MemUsage}}" > "$RESULTS_DIR/resources_${label}.csv"
}

# 测试 LLM 性能
test_llm() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}1. LLM 性能测试 (GLM-4.7-Flash on GPU)${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    local total_time=0
    local total_tokens=0
    local times=()
    
    # 测试不同长度的请求
    local prompts=(
        "你好"  # 短
        "请简单介绍一下人工智能的发展历史"  # 中
        "请详细解释一下深度学习中的注意力机制是如何工作的，包括自注意力和多头注意力的原理"  # 长
    )
    
    for prompt in "${prompts[@]}"; do
        echo ""
        echo "测试 Prompt: ${prompt:0:30}..."
        
        for i in $(seq 1 $LLM_ITERATIONS); do
            local start=$(date +%s.%N)
            
            local response=$(curl -s http://localhost:8000/v1/chat/completions \
                -H "Content-Type: application/json" \
                -d "{
                    \"model\": \"/models\",
                    \"messages\": [{\"role\": \"user\", \"content\": \"$prompt\"}],
                    \"max_tokens\": 100,
                    \"temperature\": 0.7
                }")
            
            local end=$(date +%s.%N)
            local duration=$(echo "$end - $start" | bc)
            
            # 提取生成的 token 数
            local tokens=$(echo "$response" | jq -r '.usage.completion_tokens // 0')
            if [ "$tokens" = "null" ] || [ "$tokens" = "0" ]; then
                # 估算 token 数（约 1.5 字符/token）
                local content=$(echo "$response" | jq -r '.choices[0].message.content // ""')
                tokens=$((${#content} / 2))
            fi
            
            times+=("$duration")
            total_time=$(echo "$total_time + $duration" | bc)
            total_tokens=$((total_tokens + tokens))
            
            printf "  迭代 %d: %.2fs, ~%d tokens\n" $i $duration $tokens
        done
    done
    
    # 计算统计数据
    local avg_time=$(echo "scale=2; $total_time / ($LLM_ITERATIONS * ${#prompts[@]})" | bc)
    local avg_tps=$(echo "scale=1; $total_tokens / $total_time" | bc)
    
    # 计算标准差
    local sum_sq=0
    for t in "${times[@]}"; do
        local diff=$(echo "$t - $avg_time" | bc)
        sum_sq=$(echo "$sum_sq + ($diff * $diff)" | bc)
    done
    local std_dev=$(echo "scale=2; sqrt($sum_sq / ${#times[@]})" | bc)
    
    echo ""
    echo -e "${GREEN}LLM 性能摘要:${NC}"
    echo "  平均延迟: ${avg_time}s"
    echo "  标准差: ${std_dev}s"
    echo "  平均吞吐量: ${avg_tps} tokens/s"
    echo "  总请求数: $((${#prompts[@]} * LLM_ITERATIONS))"
    echo "  总 Token 数: $total_tokens"
    
    # 保存结果
    echo "llm_avg_latency=$avg_time" >> "$RESULTS_DIR/results.txt"
    echo "llm_throughput=$avg_tps" >> "$RESULTS_DIR/results.txt"
    
    record_resources "llm"
}

# 测试 TTS 性能
test_tts() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}2. TTS 性能测试 (Qwen3-TTS on CPU)${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    local total_time=0
    local total_chars=0
    local times=()
    
    # 测试不同长度的文本
    local texts=(
        "你好"  # 短
        "这是一个语音合成测试，用于评估TTS服务的性能表现"  # 中
        "人工智能技术正在快速发展，语音合成作为其中的重要组成部分，已经在许多领域得到了广泛应用，包括智能助手、有声读物、教育培训等场景。"  # 长
    )
    
    for text in "${texts[@]}"; do
        echo ""
        echo "测试文本: ${text:0:30}... (${#text} 字符)"
        
        for i in $(seq 1 $TTS_ITERATIONS); do
            local start=$(date +%s.%N)
            
            local response=$(curl -s -X POST http://localhost:8002/v1/tts \
                -H "Content-Type: application/json" \
                -d "{\"text\": \"$text\", \"voice\": \"default\"}")
            
            local end=$(date +%s.%N)
            local duration=$(echo "$end - $start" | bc)
            
            # 获取音频大小
            local audio_size=$(echo "$response" | jq -r '.audio_base64' | wc -c)
            
            times+=("$duration")
            total_time=$(echo "$total_time + $duration" | bc)
            total_chars=$((total_chars + ${#text}))
            
            printf "  迭代 %d: %.2fs, 音频大小: %d bytes\n" $i $duration $audio_size
        done
    done
    
    # 计算统计数据
    local avg_time=$(echo "scale=2; $total_time / ($TTS_ITERATIONS * ${#texts[@]})" | bc)
    local avg_cps=$(echo "scale=1; $total_chars / $total_time" | bc)
    
    echo ""
    echo -e "${GREEN}TTS 性能摘要:${NC}"
    echo "  平均延迟: ${avg_time}s"
    echo "  处理速度: ${avg_cps} 字符/s"
    echo "  总请求数: $((${#texts[@]} * TTS_ITERATIONS))"
    echo "  总字符数: $total_chars"
    
    # 保存结果
    echo "tts_avg_latency=$avg_time" >> "$RESULTS_DIR/results.txt"
    echo "tts_chars_per_sec=$avg_cps" >> "$RESULTS_DIR/results.txt"
    
    record_resources "tts"
}

# 测试 ASR 性能（使用预录音频）
test_asr() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}3. ASR 性能测试 (SenseVoice on CPU)${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    # 检查是否有测试音频
    local test_audio="/mnt/data/models/Qwen3-TTS-0.6B/reference.wav"
    
    if [ ! -f "$test_audio" ]; then
        echo -e "${YELLOW}警告: 未找到测试音频文件，跳过 ASR 性能测试${NC}"
        echo "asr_status=skipped" >> "$RESULTS_DIR/results.txt"
        return
    fi
    
    echo "使用测试音频: $test_audio"
    echo ""
    
    local total_time=0
    local times=()
    
    for i in $(seq 1 3); do
        local start=$(date +%s.%N)
        
        local response=$(curl -s -X POST http://localhost:8001/asr \
            -F "audio=@$test_audio")
        
        local end=$(date +%s.%N)
        local duration=$(echo "$end - $start" | bc)
        
        times+=("$duration")
        total_time=$(echo "$total_time + $duration" | bc)
        
        local text=$(echo "$response" | jq -r '.text // .result // "N/A"')
        printf "  迭代 %d: %.2fs, 结果: %s\n" $i $duration "${text:0:50}..."
    done
    
    local avg_time=$(echo "scale=2; $total_time / 3" | bc)
    
    echo ""
    echo -e "${GREEN}ASR 性能摘要:${NC}"
    echo "  平均延迟: ${avg_time}s"
    echo "  总请求数: 3"
    
    echo "asr_avg_latency=$avg_time" >> "$RESULTS_DIR/results.txt"
    
    record_resources "asr"
}

# 并发测试
test_concurrent() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}4. 并发性能测试${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo "同时发送 $CONCURRENT_REQUESTS 个请求到各服务..."
    
    local start=$(date +%s.%N)
    
    # 并发发送请求
    (
        curl -s http://localhost:8000/v1/chat/completions \
            -H "Content-Type: application/json" \
            -d '{"model": "/models", "messages": [{"role": "user", "content": "你好"}], "max_tokens": 50}' \
            > "$RESULTS_DIR/concurrent_llm.json" &
        
        curl -s -X POST http://localhost:8002/v1/tts \
            -H "Content-Type: application/json" \
            -d '{"text": "并发测试", "voice": "default"}' \
            > "$RESULTS_DIR/concurrent_tts.json" &
        
        curl -s http://localhost:8001/health \
            > "$RESULTS_DIR/concurrent_asr.json" &
        
        wait
    )
    
    local end=$(date +%s.%N)
    local total_duration=$(echo "$end - $start" | bc)
    
    echo ""
    echo -e "${GREEN}并发测试结果:${NC}"
    echo "  总耗时: ${total_duration}s"
    echo "  LLM 响应: $(cat "$RESULTS_DIR/concurrent_llm.json" | jq -r '.choices[0].message.content' 2>/dev/null | head -c 50)..."
    echo "  TTS 响应: $(cat "$RESULTS_DIR/concurrent_tts.json" | jq -r '.audio_base64' 2>/dev/null | wc -c) bytes"
    echo "  ASR 健康检查: $(cat "$RESULTS_DIR/concurrent_asr.json" | jq -r '.status' 2>/dev/null)"
    
    echo "concurrent_total_time=$total_duration" >> "$RESULTS_DIR/results.txt"
    
    record_resources "concurrent"
}

# 持续负载测试
test_sustained_load() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}5. 持续负载测试 (30秒)${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    
    local duration=30
    local llm_count=0
    local tts_count=0
    local start_time=$(date +%s)
    local end_time=$((start_time + duration))
    
    echo "运行持续负载测试..."
    
    while [ $(date +%s) -lt $end_time ]; do
        # LLM 请求
        curl -s http://localhost:8000/v1/chat/completions \
            -H "Content-Type: application/json" \
            -d '{"model": "/models", "messages": [{"role": "user", "content": "测试"}], "max_tokens": 20}' \
            > /dev/null 2>&1 &
        llm_count=$((llm_count + 1))
        
        # TTS 请求
        curl -s -X POST http://localhost:8002/v1/tts \
            -H "Content-Type: application/json" \
            -d '{"text": "测试", "voice": "default"}' \
            > /dev/null 2>&1 &
        tts_count=$((tts_count + 1))
        
        sleep 1
    done
    
    wait
    
    echo ""
    echo -e "${GREEN}持续负载测试结果:${NC}"
    echo "  测试时长: ${duration}s"
    echo "  LLM 请求数: $llm_count"
    echo "  TTS 请求数: $tts_count"
    echo "  LLM QPS: $(echo "scale=2; $llm_count / $duration" | bc)"
    echo "  TTS QPS: $(echo "scale=2; $tts_count / $duration" | bc)"
    
    echo "sustained_llm_qps=$(echo "scale=2; $llm_count / $duration" | bc)" >> "$RESULTS_DIR/results.txt"
    echo "sustained_tts_qps=$(echo "scale=2; $tts_count / $duration" | bc)" >> "$RESULTS_DIR/results.txt"
    
    record_resources "sustained"
}

# 生成报告
generate_report() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}性能测试报告${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    
    # 读取结果
    if [ -f "$RESULTS_DIR/results.txt" ]; then
        source "$RESULTS_DIR/results.txt"
    fi
    
    echo "| 服务 | 模型 | 设备 | 平均延迟 | 吞吐量 |"
    echo "|------|------|------|----------|--------|"
    echo "| LLM | GLM-4.7-Flash | GPU | ${llm_avg_latency:-N/A}s | ${llm_throughput:-N/A} tokens/s |"
    echo "| TTS | Qwen3-TTS-0.6B | CPU | ${tts_avg_latency:-N/A}s | ${tts_chars_per_sec:-N/A} chars/s |"
    echo "| ASR | SenseVoiceSmall | CPU | ${asr_avg_latency:-N/A}s | - |"
    echo ""
    
    echo "**并发性能:**"
    echo "- 并发请求总耗时: ${concurrent_total_time:-N/A}s"
    echo "- 持续负载 LLM QPS: ${sustained_llm_qps:-N/A}"
    echo "- 持续负载 TTS QPS: ${sustained_tts_qps:-N/A}"
    echo ""
    
    # 最终资源使用
    echo "**最终资源使用:**"
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.GPUPerc}}"
    
    echo ""
    echo -e "${GREEN}测试完成！结果保存在: $RESULTS_DIR${NC}"
}

# 主函数
main() {
    # 清理旧结果
    rm -f "$RESULTS_DIR"/*.txt "$RESULTS_DIR"/*.csv "$RESULTS_DIR"/*.json 2>/dev/null || true
    
    check_services
    
    # 运行测试
    test_llm
    test_tts
    test_asr
    test_concurrent
    test_sustained_load
    
    # 生成报告
    generate_report
}

main "$@"
