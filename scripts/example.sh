#!/bin/bash

# 对话树解析工具使用示例

echo "=== 示例1: 解析第0个conversation（链条树格式） ==="
python3 extract_and_parse.py ../conversations_backup_account_modified.json 0

echo ""
echo "=== 示例2: 解析第1个conversation（完整内容） ==="
python3 extract_and_parse.py ../conversations_backup_account_modified.json 1 --full | head -50

echo ""
echo "=== 示例3: 显示节点ID ==="
python3 extract_and_parse.py ../conversations_backup_account_modified.json 0 --ids | head -20
