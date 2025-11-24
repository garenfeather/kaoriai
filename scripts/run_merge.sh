#!/bin/bash

# GPT Branch Tree Merge 编译和运行脚本

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$SCRIPT_DIR/../bin"
OUTPUT_DIR="$SCRIPT_DIR/../parsed/gpt/tree"

# 创建bin目录
mkdir -p "$BIN_DIR"

# 删除旧的可执行文件
if [ -f "$BIN_DIR/gpt_branch_tree_merge" ]; then
    echo "删除旧的可执行文件..."
    rm -f "$BIN_DIR/gpt_branch_tree_merge"
fi

# 编译Go程序
echo "正在编译 gpt_branch_tree_merge..."
go build -o "$BIN_DIR/gpt_branch_tree_merge" "$SCRIPT_DIR/gpt_branch_tree_merge.go"

if [ $? -ne 0 ]; then
    echo "编译失败!"
    exit 1
fi

echo "编译成功!"

# 运行程序
if [ -z "$1" ]; then
    echo "用法: $0 <input_files> [output_dir]"
    echo "示例: $0 conv_a.json,conv_b.json"
    echo "示例: $0 tree.json,conv_c.json gpt_tree"
    echo ""
    echo "说明:"
    echo "  - input_files: 用逗号分隔的多个JSON文件路径"
    echo "  - 如果第一个文件是树结构，会将后续对话合并到该树中"
    echo "  - 如果都是对话文件，会合并成新的树结构"
    exit 1
fi

INPUT_FILES="$1"
if [ ! -z "$2" ]; then
    OUTPUT_DIR="$2"
fi

echo "输入文件: $INPUT_FILES"
echo "输出目录: $OUTPUT_DIR"
echo ""

"$BIN_DIR/gpt_branch_tree_merge" -input "$INPUT_FILES" -output "$OUTPUT_DIR"

# 保存退出码
EXIT_CODE=$?

# 删除可执行文件
if [ -f "$BIN_DIR/gpt_branch_tree_merge" ]; then
    echo ""
    echo "清理可执行文件..."
    rm -f "$BIN_DIR/gpt_branch_tree_merge"
fi

# 返回原始退出码
exit $EXIT_CODE
