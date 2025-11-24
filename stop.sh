#!/bin/bash

# 对话服务器停止脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/backend"
BINARY_NAME="conversation-server"
PID_FILE="$BACKEND_DIR/server.pid"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== 停止对话服务器 ===${NC}"
echo ""

STOPPED=false

# 通过 PID 文件停止
if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if ps -p "$PID" > /dev/null 2>&1; then
        echo "正在停止服务 (PID: $PID)..."
        kill "$PID" 2>/dev/null || true
        sleep 1

        # 检查是否还在运行
        if ps -p "$PID" > /dev/null 2>&1; then
            echo "强制停止服务..."
            kill -9 "$PID" 2>/dev/null || true
            sleep 1
        fi

        echo -e "${GREEN}✓ 服务已停止${NC}"
        STOPPED=true
    else
        echo -e "${YELLOW}PID 文件存在但进程未运行${NC}"
    fi
    rm -f "$PID_FILE"
fi

# 通过进程名停止（备用方案）
PIDS=$(pgrep -f "$BINARY_NAME" 2>/dev/null || true)
if [ ! -z "$PIDS" ]; then
    echo "发现其他运行中的服务进程，正在清理..."
    for pid in $PIDS; do
        kill "$pid" 2>/dev/null || true
    done
    sleep 1
    echo -e "${GREEN}✓ 已清理所有服务进程${NC}"
    STOPPED=true
fi

if [ "$STOPPED" = false ]; then
    echo -e "${YELLOW}没有运行中的服务${NC}"
fi

echo ""
echo -e "${GREEN}=== 完成 ===${NC}"
