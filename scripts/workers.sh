#!/bin/bash

# Workers 统一管理脚本
# 用法:
#   ./workers.sh                    # 编译并启动所有 workers
#   ./workers.sh gpt                # 仅启动 gpt worker
#   ./workers.sh gpt codex          # 启动 gpt 和 codex workers
#   ./workers.sh -b                 # 仅编译，不启动
#   ./workers.sh --stop             # 停止所有运行中的 workers

set -e

# 工作目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN_DIR="$PROJECT_ROOT/bin"
PID_DIR="$PROJECT_ROOT/data/pids"

# 创建必要目录
mkdir -p "$BIN_DIR"
mkdir -p "$PID_DIR"

# 配置参数
DB_PATH="${DB_PATH:-data/conversations.db}"
TEMP_DIR="${TEMP_DIR:-data/temp}"
CHECK_INTERVAL="${CHECK_INTERVAL:-30s}"

# 可用的 workers
AVAILABLE_WORKERS=("gpt" "claude" "codex" "claude_code" "gemini")

# 模式标志
BUILD_ONLY=false
STOP_MODE=false

# 解析参数
SELECTED_WORKERS=()
for arg in "$@"; do
    case "$arg" in
        -b|--build-only)
            BUILD_ONLY=true
            ;;
        --stop)
            STOP_MODE=true
            ;;
        -h|--help)
            echo "Workers 统一管理脚本"
            echo ""
            echo "用法:"
            echo "  $0                   # 编译并启动所有 workers"
            echo "  $0 <worker>...       # 编译并启动指定的 workers"
            echo "  $0 -b                # 仅编译所有 workers，不启动"
            echo "  $0 -b <worker>...    # 仅编译指定的 workers"
            echo "  $0 --stop            # 停止所有运行中的 workers"
            echo ""
            echo "可用的 workers:"
            echo "  gpt          - GPT 对话数据同步"
            echo "  claude       - Claude 对话数据同步"
            echo "  codex        - Codex 对话数据同步"
            echo "  claude_code  - Claude Code 对话数据同步"
            echo "  gemini       - Gemini 对话数据同步"
            echo ""
            echo "环境变量:"
            echo "  DB_PATH           数据库路径 (默认: data/conversations.db)"
            echo "  TEMP_DIR          临时目录 (默认: data/temp)"
            echo "  CHECK_INTERVAL    检查间隔 (默认: 30s)"
            echo "  IMAGES_DIR        图片目录 (默认: data/images, 仅 gpt worker)"
            echo ""
            echo "GPT Worker 环境变量:"
            echo "  GPT_WORKER_MODE                - 工作模式: email|file (默认: email)"
            echo ""
            echo "  email 模式额外需要:"
            echo "    GO_PROTON_API_TEST_USERNAME    - Proton Mail 用户名"
            echo "    GO_PROTON_API_TEST_PASSWORD    - Proton Mail 密码"
            echo "    SECURE_NEXT_AUTH_SESSION_TOKEN - Session Token"
            echo ""
            echo "  file 模式:"
            echo "    直接监测 data/original/gpt 目录中的 ZIP 文件"
            echo ""
            echo "示例:"
            echo "  $0                        # 启动所有 workers"
            echo "  $0 gpt                    # 只启动 gpt worker"
            echo "  $0 gpt codex              # 启动 gpt 和 codex workers"
            echo "  $0 -b                     # 只编译，不启动"
            echo "  $0 -b codex               # 只编译 codex worker"
            echo "  CHECK_INTERVAL=60s $0 gpt # 自定义检查间隔"
            exit 0
            ;;
        *)
            # 检查是否为有效的 worker 名称
            if [[ " ${AVAILABLE_WORKERS[@]} " =~ " ${arg} " ]]; then
                SELECTED_WORKERS+=("$arg")
            else
                echo "错误: 未知的 worker '$arg'"
                echo "可用的 workers: ${AVAILABLE_WORKERS[*]}"
                exit 1
            fi
            ;;
    esac
done

# 停止模式
if [ "$STOP_MODE" = true ]; then
    echo "停止所有运行中的 workers..."
    stopped_count=0
    for worker in "${AVAILABLE_WORKERS[@]}"; do
        pid_file="$PID_DIR/${worker}_worker.pid"
        if [ -f "$pid_file" ]; then
            pid=$(cat "$pid_file")
            if kill -0 "$pid" 2>/dev/null; then
                echo "停止 $worker worker (PID: $pid)..."
                kill "$pid"
                rm "$pid_file"
                stopped_count=$((stopped_count + 1))
            else
                echo "$worker worker (PID: $pid) 已经不在运行"
                rm "$pid_file"
            fi
        fi
    done
    if [ $stopped_count -eq 0 ]; then
        echo "没有运行中的 workers"
    else
        echo "✓ 已停止 $stopped_count 个 workers"
    fi
    exit 0
fi

# 如果没有指定 workers，使用所有可用的
if [ ${#SELECTED_WORKERS[@]} -eq 0 ]; then
    SELECTED_WORKERS=("${AVAILABLE_WORKERS[@]}")
fi

echo "========================================"
echo "Workers 管理"
echo "========================================"
echo "选中的 workers: ${SELECTED_WORKERS[*]}"
echo "数据库: $DB_PATH"
echo "临时目录: $TEMP_DIR"
echo "检查间隔: $CHECK_INTERVAL"
echo ""

# 编译 workers
echo "开始编译 workers..."
echo ""

for worker in "${SELECTED_WORKERS[@]}"; do
    echo "[$worker] 正在编译..."

    worker_dir="$PROJECT_ROOT/workers/$worker"
    if [ ! -d "$worker_dir/cmd" ]; then
        echo "[$worker] 错误: 找不到 cmd 目录"
        continue
    fi

    cd "$worker_dir/cmd"
    if go build -o "$BIN_DIR/${worker}_worker" . ; then
        echo "[$worker] ✓ 编译成功"
    else
        echo "[$worker] ✗ 编译失败"
        exit 1
    fi
    echo ""
done

cd "$PROJECT_ROOT"

# 仅编译模式
if [ "$BUILD_ONLY" = true ]; then
    echo "========================================"
    echo "编译完成 (仅编译模式)"
    echo "========================================"
    echo ""
    echo "可执行文件列表:"
    for worker in "${SELECTED_WORKERS[@]}"; do
        if [ -f "$BIN_DIR/${worker}_worker" ]; then
            size=$(ls -lh "$BIN_DIR/${worker}_worker" | awk '{print $5}')
            echo "  ${worker}_worker ($size)"
        fi
    done
    exit 0
fi

# 启动 workers
echo "========================================"
echo "启动 workers..."
echo "========================================"
echo ""

# 获取每个 worker 的数据目录
get_data_dir() {
    local worker=$1
    case "$worker" in
        gpt)
            echo "data/original/gpt"
            ;;
        claude)
            echo "data/original/claude"
            ;;
        codex)
            echo "data/original/codex"
            ;;
        claude_code)
            echo "data/original/claude_code"
            ;;
        gemini)
            echo "data/original/gemini"
            ;;
        *)
            echo "data/original/$worker"
            ;;
    esac
}

# 启动单个 worker
start_worker() {
    local worker=$1
    local data_dir=$(get_data_dir "$worker")

    echo "[$worker] 启动中..."
    echo "[$worker] 数据目录: $data_dir"

    # 根据不同的 worker 使用不同的参数
    case "$worker" in
        gpt)
            # GPT worker 需要特殊的环境变量和参数
            # 检查工作模式
            gpt_mode="${GPT_WORKER_MODE:-email}"

            if [ "$gpt_mode" = "email" ]; then
                # email 模式需要检查环境变量
                if [ -z "$GO_PROTON_API_TEST_USERNAME" ] || [ -z "$GO_PROTON_API_TEST_PASSWORD" ] || [ -z "$SECURE_NEXT_AUTH_SESSION_TOKEN" ]; then
                    echo "[$worker] ✗ 错误: GPT worker (email 模式) 需要以下环境变量:"
                    echo "         GO_PROTON_API_TEST_USERNAME"
                    echo "         GO_PROTON_API_TEST_PASSWORD"
                    echo "         SECURE_NEXT_AUTH_SESSION_TOKEN"
                    echo ""
                    echo "         或者设置 GPT_WORKER_MODE=file 使用文件模式"
                    echo ""
                    return 1
                fi
            else
                echo "[$worker] 使用文件模式，监测目录: $data_dir"
            fi

            "$BIN_DIR/${worker}_worker" \
                -db "$DB_PATH" \
                -original-dir "$data_dir" \
                -images-dir "${IMAGES_DIR:-data/images}" \
                -temp-dir "$TEMP_DIR" \
                -interval "$CHECK_INTERVAL" \
                > "$PROJECT_ROOT/data/logs/${worker}_worker.log" 2>&1 &
            ;;
        gemini)
            # Gemini worker 需要额外的视频目录参数
            "$BIN_DIR/${worker}_worker" \
                -db "$DB_PATH" \
                -data "$data_dir" \
                -images-dir "${IMAGES_DIR:-data/images}" \
                -videos-dir "${VIDEOS_DIR:-data/videos}" \
                -interval "$CHECK_INTERVAL" \
                > "$PROJECT_ROOT/data/logs/${worker}_worker.log" 2>&1 &
            ;;
        *)
            # 其他 workers 使用标准参数
            "$BIN_DIR/${worker}_worker" \
                -db "$DB_PATH" \
                -data "$data_dir" \
                -temp "$TEMP_DIR" \
                -interval "$CHECK_INTERVAL" \
                > "$PROJECT_ROOT/data/logs/${worker}_worker.log" 2>&1 &
            ;;
    esac

    local worker_pid=$!
    echo "$worker_pid" > "$PID_DIR/${worker}_worker.pid"

    # 等待一下确保启动成功
    sleep 1
    if kill -0 "$worker_pid" 2>/dev/null; then
        echo "[$worker] ✓ 启动成功 (PID: $worker_pid)"
    else
        echo "[$worker] ✗ 启动失败，查看日志: data/logs/${worker}_worker.log"
        rm "$PID_DIR/${worker}_worker.pid"
    fi
    echo ""
}

for worker in "${SELECTED_WORKERS[@]}"; do
    start_worker "$worker"
done

echo "========================================"
echo "所有 workers 已启动"
echo "========================================"
echo ""
echo "日志目录: data/logs/"
echo "PID 目录: data/pids/"
echo ""
echo "查看日志:"
for worker in "${SELECTED_WORKERS[@]}"; do
    echo "  tail -f data/logs/${worker}_worker.log"
done
echo ""
echo "停止所有 workers:"
echo "  $0 --stop"
