#!/bin/bash

# GPT Conversation Parser 测试脚本

echo "======================================"
echo "GPT Conversation Parser - 自动测试"
echo "======================================"

# 检查输入文件
TEST_FILE="data/conversations_backup_account_modified.json"
if [ ! -f "$TEST_FILE" ]; then
    echo "❌ 测试文件不存在: $TEST_FILE"
    exit 1
fi
echo "✅ 找到测试文件: $TEST_FILE"

# 创建临时输出目录
TEST_OUTPUT="test_output_$$"
mkdir -p "$TEST_OUTPUT"

# 运行解析
echo ""
echo "正在运行解析测试..."
./bin/gpt_conversation_parse -input "$TEST_FILE" -output "$TEST_OUTPUT" > /dev/null 2>&1

if [ $? -ne 0 ]; then
    echo "❌ 解析失败"
    rm -rf "$TEST_OUTPUT"
    exit 1
fi

# 统计结果
FILE_COUNT=$(ls -1 "$TEST_OUTPUT"/*.json 2>/dev/null | wc -l | tr -d ' ')

if [ "$FILE_COUNT" -eq 0 ]; then
    echo "❌ 没有生成输出文件"
    rm -rf "$TEST_OUTPUT"
    exit 1
fi

echo "✅ 解析成功"
echo ""
echo "生成文件数: $FILE_COUNT"

# 验证JSON格式
FIRST_FILE=$(ls "$TEST_OUTPUT"/*.json | head -1)
if /usr/local/bin/python3 -m json.tool "$FIRST_FILE" > /dev/null 2>&1; then
    echo "✅ JSON格式正确"
else
    echo "❌ JSON格式错误"
    rm -rf "$TEST_OUTPUT"
    exit 1
fi

# 清理
rm -rf "$TEST_OUTPUT"

echo ""
echo "======================================"
echo "✅ 所有测试通过!"
echo "======================================"
