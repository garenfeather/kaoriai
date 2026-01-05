#!/bin/bash

# API 服务器重启脚本
# 功能: 检查端口占用 -> 杀掉旧进程 -> 重新编译 -> 启动服务器

set -e  # 遇到错误立即退出

# ==================== 配置 ====================
PORT=${1:-8080}  # 默认端口8080，可通过第一个参数指定
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY_PATH="$PROJECT_ROOT/bin/api-server"
LOG_DIR="$PROJECT_ROOT/logs"
LOG_FILE="$LOG_DIR/api-server.log"
DB_PATH="$PROJECT_ROOT/data/conversation.db"

# ==================== 颜色定义 ====================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# ==================== 1. 检查并杀掉占用端口的进程 ====================
log_info "检查端口 $PORT 是否被占用..."

# 查找占用端口的进程
PID=$(lsof -ti:$PORT || echo "")

if [ -n "$PID" ]; then
    log_warn "发现端口 $PORT 被进程 $PID 占用"

    # 获取进程名称
    PROCESS_NAME=$(ps -p $PID -o comm= || echo "未知进程")
    log_info "进程名称: $PROCESS_NAME"

    # 杀掉进程
    log_info "正在终止进程 $PID..."
    kill -9 $PID 2>/dev/null || true

    # 等待进程完全终止
    sleep 1

    # 再次检查
    if lsof -ti:$PORT >/dev/null 2>&1; then
        log_error "无法终止端口 $PORT 上的进程"
        exit 1
    fi

    log_success "进程 $PID 已终止"
else
    log_info "端口 $PORT 未被占用"
fi

# ==================== 2. 清理旧的进程 ====================
log_info "清理残留的 api-server 进程..."

# 查找所有 api-server 进程（排除当前脚本和 grep 自身）
OLD_PIDS=$(pgrep -f "api-server" || echo "")

if [ -n "$OLD_PIDS" ]; then
    for OLD_PID in $OLD_PIDS; do
        # 确认不是当前脚本的进程
        if [ "$OLD_PID" != "$$" ]; then
            log_warn "发现残留进程: $OLD_PID"
            kill -9 $OLD_PID 2>/dev/null || true
        fi
    done
    log_success "残留进程已清理"
else
    log_info "没有发现残留的 api-server 进程"
fi

# ==================== 3. 编译项目 ====================
log_info "开始编译 API 服务器..."

cd "$PROJECT_ROOT"

# 检查是否存在 go.mod
if [ ! -f "go.mod" ]; then
    log_error "未找到 go.mod 文件，请确保在正确的项目目录"
    exit 1
fi

# 编译
if go build -o "$BINARY_PATH" ./backend/cmd/api; then
    log_success "编译成功: $BINARY_PATH"
else
    log_error "编译失败"
    exit 1
fi

# 验证二进制文件
if [ ! -f "$BINARY_PATH" ]; then
    log_error "编译后的二进制文件不存在"
    exit 1
fi

# 添加执行权限
chmod +x "$BINARY_PATH"

# ==================== 4. 准备日志目录 ====================
log_info "准备日志目录..."

mkdir -p "$LOG_DIR"

# 备份旧日志（如果存在）
if [ -f "$LOG_FILE" ]; then
    BACKUP_LOG="$LOG_DIR/api-server.$(date +%Y%m%d_%H%M%S).log"
    mv "$LOG_FILE" "$BACKUP_LOG"
    log_info "旧日志已备份: $BACKUP_LOG"
fi

# ==================== 5. 启动服务器 ====================
log_info "启动 API 服务器..."

cd "$PROJECT_ROOT"

# 后台启动服务器，重定向日志
nohup "$BINARY_PATH" -port "$PORT" -db "$DB_PATH" > "$LOG_FILE" 2>&1 &
SERVER_PID=$!

# 等待服务器启动
sleep 2

# 验证服务器是否正在运行
if ps -p $SERVER_PID > /dev/null 2>&1; then
    log_success "API 服务器已启动"
    log_info "进程 ID: $SERVER_PID"
    log_info "监听端口: $PORT"
    log_info "数据库: $DB_PATH"
    log_info "日志文件: $LOG_FILE"
    echo ""
    log_info "使用以下命令查看日志:"
    echo -e "  ${GREEN}tail -f $LOG_FILE${NC}"
    echo ""
    log_info "使用以下命令测试 API:"
    echo -e "  ${GREEN}curl http://localhost:$PORT/api/v1/tags${NC}"
    echo ""
    log_info "使用以下命令停止服务器:"
    echo -e "  ${GREEN}kill $SERVER_PID${NC}"
    echo -e "  或者"
    echo -e "  ${GREEN}$PROJECT_ROOT/scripts/stop-api-server.sh${NC}"
    echo ""

    # 输出最近的日志
    log_info "最近的日志输出:"
    echo "----------------------------------------"
    tail -10 "$LOG_FILE" 2>/dev/null || echo "暂无日志"
    echo "----------------------------------------"
else
    log_error "API 服务器启动失败"
    log_info "查看日志获取详细信息:"
    echo -e "  ${GREEN}cat $LOG_FILE${NC}"
    exit 1
fi

exit 0
