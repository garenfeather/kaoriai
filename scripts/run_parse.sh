#!/bin/bash

# GPT Conversation Parser 编译和运行脚本

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$SCRIPT_DIR/../bin"
OUTPUT_DIR="$SCRIPT_DIR/../parsed/gpt/conversation"

# 创建bin目录
mkdir -p "$BIN_DIR"

# 删除旧的可执行文件
if [ -f "$BIN_DIR/gpt_conversation_parse" ]; then
    echo "删除旧的可执行文件..."
    rm -f "$BIN_DIR/gpt_conversation_parse"
fi

# 编译Go程序
echo "正在编译 gpt_conversation_parse..."
go build -o "$BIN_DIR/gpt_conversation_parse" "$SCRIPT_DIR/gpt_conversation_parse.go"

if [ $? -ne 0 ]; then
    echo "编译失败!"
    exit 1
fi

echo "编译成功!"

# 运行程序
if [ -z "$1" ]; then
    echo "用法: $0 <input_json_file> [output_dir]"
    echo "示例: $0 data/conversations_backup_account_modified.json"
    exit 1
fi

INPUT_FILE="$1"
if [ ! -z "$2" ]; then
    OUTPUT_DIR="$2"
fi

echo "输入文件: $INPUT_FILE"
echo "输出目录: $OUTPUT_DIR"
echo ""

"$BIN_DIR/gpt_conversation_parse" -input "$INPUT_FILE" -output "$OUTPUT_DIR"

# 保存退出码
EXIT_CODE=$?

# 删除可执行文件
if [ -f "$BIN_DIR/gpt_conversation_parse" ]; then
    echo ""
    echo "清理可执行文件..."
    rm -f "$BIN_DIR/gpt_conversation_parse"
fi

# 返回原始退出码
exit $EXIT_CODE
