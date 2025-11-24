#!/bin/bash

# OpenAI 邮件监控编译和运行脚本

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$SCRIPT_DIR/../bin"

# 创建bin目录
mkdir -p "$BIN_DIR"

# 删除旧的可执行文件
if [ -f "$BIN_DIR/openai_email_monitor" ]; then
    echo "删除旧的可执行文件..."
    rm -f "$BIN_DIR/openai_email_monitor"
fi

# 整理依赖
echo "正在整理依赖..."
cd "$SCRIPT_DIR"
go mod tidy

if [ $? -ne 0 ]; then
    echo "依赖整理失败!"
    exit 1
fi

# 编译Go程序
echo "正在编译 openai_email_monitor..."
go build -o "$BIN_DIR/openai_email_monitor" "$SCRIPT_DIR/openai_email_monitor.go"

if [ $? -ne 0 ]; then
    echo "编译失败!"
    exit 1
fi

echo "编译成功!"

# 检查环境变量
if [ -z "$GO_PROTON_API_TEST_USERNAME" ] || [ -z "$GO_PROTON_API_TEST_PASSWORD" ]; then
    echo ""
    echo "错误: 必须设置环境变量:"
    echo "  export GO_PROTON_API_TEST_USERNAME=\"your_email\""
    echo "  export GO_PROTON_API_TEST_PASSWORD=\"your_password\""
    echo ""
    rm -f "$BIN_DIR/openai_email_monitor"
    exit 1
fi

echo ""
echo "启动 OpenAI 邮件监控..."
echo "邮箱: $GO_PROTON_API_TEST_USERNAME"
echo "检查间隔: 30秒"
echo "按 Ctrl+C 停止监控"
echo ""

# 运行程序
"$BIN_DIR/openai_email_monitor"

# 保存退出码
EXIT_CODE=$?

# 删除可执行文件
if [ -f "$BIN_DIR/openai_email_monitor" ]; then
    echo ""
    echo "清理可执行文件..."
    rm -f "$BIN_DIR/openai_email_monitor"
fi

# 返回原始退出码
exit $EXIT_CODE
