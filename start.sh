#!/bin/bash

# 对话服务器一键启动脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/backend"
BINARY_NAME="conversation-server"
BINARY_PATH="$BACKEND_DIR/$BINARY_NAME"
PID_FILE="$BACKEND_DIR/server.pid"
LOG_FILE="$BACKEND_DIR/server.log"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== 对话服务器启动脚本 ===${NC}"
echo ""

# 1. 检查并杀死已存在的进程
echo -e "${YELLOW}[1/4] 检查已存在的服务进程...${NC}"

# 通过 PID 文件检查
if [ -f "$PID_FILE" ]; then
    OLD_PID=$(cat "$PID_FILE")
    if ps -p "$OLD_PID" > /dev/null 2>&1; then
        echo "发现运行中的服务 (PID: $OLD_PID)，正在停止..."
        kill "$OLD_PID" 2>/dev/null || true
        sleep 1

        # 如果进程还在运行，强制杀死
        if ps -p "$OLD_PID" > /dev/null 2>&1; then
            echo "强制停止服务..."
            kill -9 "$OLD_PID" 2>/dev/null || true
            sleep 1
        fi
        echo -e "${GREEN}✓ 已停止旧服务${NC}"
    fi
    rm -f "$PID_FILE"
fi

# 通过进程名检查（备用方案）
PIDS=$(pgrep -f "$BINARY_NAME" 2>/dev/null || true)
if [ ! -z "$PIDS" ]; then
    echo "发现其他运行中的服务进程，正在清理..."
    for pid in $PIDS; do
        kill "$pid" 2>/dev/null || true
    done
    sleep 1
    echo -e "${GREEN}✓ 已清理所有服务进程${NC}"
else
    echo -e "${GREEN}✓ 没有运行中的服务${NC}"
fi

echo ""

# 2. 编译 Go 服务器
echo -e "${YELLOW}[2/4] 编译 Go 服务器...${NC}"

cd "$BACKEND_DIR"

# 删除旧的二进制文件
if [ -f "$BINARY_NAME" ]; then
    rm -f "$BINARY_NAME"
    echo "已删除旧的二进制文件"
fi

# 编译
go build -o "$BINARY_NAME" main.go

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ 编译失败!${NC}"
    exit 1
fi

echo -e "${GREEN}✓ 编译成功${NC}"
echo ""

# 3. 启动服务器
echo -e "${YELLOW}[3/4] 启动服务器...${NC}"

# 创建或清空日志文件
> "$LOG_FILE"

# 后台运行服务器
nohup "$BINARY_PATH" > "$LOG_FILE" 2>&1 &
SERVER_PID=$!

# 保存 PID
echo "$SERVER_PID" > "$PID_FILE"

# 等待服务启动
echo "等待服务启动..."
sleep 2

# 4. 验证服务状态
echo ""
echo -e "${YELLOW}[4/4] 验证服务状态...${NC}"

if ps -p "$SERVER_PID" > /dev/null 2>&1; then
    # 尝试访问健康检查端点
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 服务启动成功!${NC}"
        echo ""
        echo -e "${GREEN}=== 服务信息 ===${NC}"
        echo "PID: $SERVER_PID"
        echo "端口: 8080"
        echo "日志文件: $LOG_FILE"
        echo ""
        echo -e "${GREEN}=== API 端点 ===${NC}"
        echo "健康检查: http://localhost:8080/health"
        echo "列出对话: http://localhost:8080/list"
        echo "获取对话: http://localhost:8080/{source}/{conversation_id}"
        echo ""
        echo "示例:"
        echo "  curl http://localhost:8080/health"
        echo "  curl http://localhost:8080/list"
        echo "  curl http://localhost:8080/gpt/d4d4ddf6-5452-4dbb-9c1c-8a59ebfdb8fa"
        echo ""
        echo -e "${YELLOW}查看日志: tail -f $LOG_FILE${NC}"
        echo -e "${YELLOW}停止服务: ./stop.sh${NC}"
    else
        echo -e "${YELLOW}⚠ 服务已启动，但健康检查失败${NC}"
        echo "请检查日志: tail -f $LOG_FILE"
    fi
else
    echo -e "${RED}✗ 服务启动失败!${NC}"
    echo "请查看日志获取详细信息:"
    echo "  cat $LOG_FILE"
    rm -f "$PID_FILE"
    exit 1
fi

echo ""
echo -e "${GREEN}=== 启动完成 ===${NC}"
