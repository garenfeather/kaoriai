#!/usr/local/bin/python3
"""
从 conversations 数组中提取并解析指定的对话

用法：
  python extract_and_parse.py <json_file> <index> [--full] [--ids]

示例：
  python extract_and_parse.py ../conversations_backup_account_modified.json 0
  python extract_and_parse.py ../conversations_backup_account_modified.json 5 --full
"""

import json
import sys
import subprocess
import tempfile
import os


def main():
    if len(sys.argv) < 3:
        print("用法:")
        print("  python extract_and_parse.py <json_file> <index> [--full] [--ids]")
        print()
        print("参数:")
        print("  json_file  包含conversations数组的JSON文件")
        print("  index      要解析的conversation索引（从0开始）")
        print("  --full     显示完整内容")
        print("  --ids      显示节点ID")
        print()
        print("示例:")
        print("  python extract_and_parse.py ../conversations_backup_account_modified.json 0")
        print("  python extract_and_parse.py ../conversations_backup_account_modified.json 5 --full")
        sys.exit(1)

    json_file = sys.argv[1]

    try:
        index = int(sys.argv[2])
    except ValueError:
        print(f"错误: 索引必须是整数，当前值: {sys.argv[2]}")
        sys.exit(1)

    # 其他参数
    extra_args = [arg for arg in sys.argv[3:] if arg.startswith('--')]

    # 读取文件
    try:
        with open(json_file, 'r', encoding='utf-8') as f:
            data = json.load(f)
    except Exception as e:
        print(f"错误: 无法读取文件 {json_file}")
        print(f"详细信息: {e}")
        sys.exit(1)

    # 检查是否是数组
    if not isinstance(data, list):
        print(f"错误: 文件内容不是数组")
        sys.exit(1)

    # 检查索引范围
    if index < 0 or index >= len(data):
        print(f"错误: 索引 {index} 超出范围 (0-{len(data)-1})")
        sys.exit(1)

    # 提取指定的conversation
    conversation = data[index]

    # 创建临时文件
    with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False, encoding='utf-8') as f:
        temp_file = f.name
        json.dump(conversation, f, ensure_ascii=False, indent=2)

    try:
        # 调用 parse_conversation_tree.py
        script_dir = os.path.dirname(os.path.abspath(__file__))
        parser_script = os.path.join(script_dir, 'parse_conversation_tree.py')

        cmd = ['/usr/local/bin/python3', parser_script, temp_file] + extra_args
        subprocess.run(cmd)
    finally:
        # 清理临时文件
        os.unlink(temp_file)


if __name__ == "__main__":
    main()
