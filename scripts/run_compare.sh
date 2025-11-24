#!/bin/bash

# JSON Compare 编译和运行脚本

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$SCRIPT_DIR/../bin"

# 创建bin目录
mkdir -p "$BIN_DIR"

# 删除旧的可执行文件
if [ -f "$BIN_DIR/compare_json" ]; then
    echo "删除旧的可执行文件..."
    rm -f "$BIN_DIR/compare_json"
fi

# 编译Go程序
echo "正在编译 compare_json..."
go build -o "$BIN_DIR/compare_json" "$SCRIPT_DIR/compare_json.go"

if [ $? -ne 0 ]; then
    echo "编译失败!"
    exit 1
fi

echo "编译成功!"
echo ""

# 运行程序
if [ $# -lt 2 ]; then
    echo "用法: $0 <file1.json> <file2.json> [limit]"
    echo "示例: $0 data/file1.json data/file2.json"
    echo "示例: $0 data/file1.json data/file2.json 100"
    exit 1
fi

FILE1="$1"
FILE2="$2"
LIMIT="${3:-50}"

echo "文件1: $FILE1"
echo "文件2: $FILE2"
echo "最多显示: $LIMIT 个差异"
echo ""

"$BIN_DIR/compare_json" "$FILE1" "$FILE2" -limit "$LIMIT"

# 保存退出码
EXIT_CODE=$?

# 删除可执行文件
if [ -f "$BIN_DIR/compare_json" ]; then
    echo ""
    echo "清理可执行文件..."
    rm -f "$BIN_DIR/compare_json"
fi

# 返回原始退出码
exit $EXIT_CODE
