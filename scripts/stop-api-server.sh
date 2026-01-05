#!/bin/bash

# API 服务器停止脚本
# 功能: 优雅地停止 API 服务器

set -e

# ==================== 配置 ====================
PORT=${1:-8080}  # 默认端口8080
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# ==================== 颜色定义 ====================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# ==================== 工具函数 ====================
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# ==================== 停止服务器 ====================
log_info "正在停止 API 服务器..."

# 查找所有 api-server 进程
PIDS=$(pgrep -f "api-server" || echo "")

if [ -z "$PIDS" ]; then
    log_warn "未发现运行中的 API 服务器进程"
    exit 0
fi

# 停止每个进程
STOPPED_COUNT=0
for PID in $PIDS; do
    # 获取进程信息
    PROCESS_INFO=$(ps -p $PID -o pid,command | tail -1)
    log_info "发现进程: $PROCESS_INFO"

    # 先尝试优雅停止 (SIGTERM)
    log_info "发送 SIGTERM 信号到进程 $PID..."
    kill $PID 2>/dev/null || true

    # 等待进程退出
    for i in {1..5}; do
        if ! ps -p $PID > /dev/null 2>&1; then
            log_success "进程 $PID 已停止"
            STOPPED_COUNT=$((STOPPED_COUNT + 1))
            break
        fi
        sleep 1
    done

    # 如果还在运行,强制杀掉
    if ps -p $PID > /dev/null 2>&1; then
        log_warn "进程 $PID 未响应 SIGTERM，发送 SIGKILL..."
        kill -9 $PID 2>/dev/null || true
        sleep 1

        if ! ps -p $PID > /dev/null 2>&1; then
            log_success "进程 $PID 已强制停止"
            STOPPED_COUNT=$((STOPPED_COUNT + 1))
        else
            log_error "无法停止进程 $PID"
        fi
    fi
done

# 检查端口是否还被占用
if [ -n "$PORT" ]; then
    PID_ON_PORT=$(lsof -ti:$PORT 2>/dev/null || echo "")
    if [ -n "$PID_ON_PORT" ]; then
        log_warn "端口 $PORT 仍被进程 $PID_ON_PORT 占用"
        log_info "强制终止该进程..."
        kill -9 $PID_ON_PORT 2>/dev/null || true
    fi
fi

log_success "共停止 $STOPPED_COUNT 个进程"
echo ""

exit 0
