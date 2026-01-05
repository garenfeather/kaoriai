#!/bin/bash

# JSON Value Decoder 编译和运行脚本

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN_DIR="$PROJECT_ROOT/bin"

# 创建bin目录
mkdir -p "$BIN_DIR"

# 删除旧的可执行文件
if [ -f "$BIN_DIR/decode_json_value" ]; then
    echo "删除旧的可执行文件..."
    rm -f "$BIN_DIR/decode_json_value"
fi

# 编译Go程序
echo "正在编译 decode_json_value..."
go build -o "$BIN_DIR/decode_json_value" "$SCRIPT_DIR/go/decode_json_value.go"

if [ $? -ne 0 ]; then
    echo "编译失败!"
    exit 1
fi

echo "编译成功!"

# 运行程序
if [ -z "$1" ]; then
    echo "用法: $0 <input_json_file>"
    echo "示例: $0 data/conversations.json"
    exit 1
fi

INPUT_FILE="$1"

echo "输入文件: $INPUT_FILE"
echo ""

"$BIN_DIR/decode_json_value" "$INPUT_FILE"

# 保存退出码
EXIT_CODE=$?

# 删除可执行文件
if [ -f "$BIN_DIR/decode_json_value" ]; then
    echo ""
    echo "清理可执行文件..."
    rm -f "$BIN_DIR/decode_json_value"
fi

# 返回原始退出码
exit $EXIT_CODE
