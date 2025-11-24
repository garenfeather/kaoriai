#!/usr/local/bin/python3
"""
对话树解析脚本

输入：一个 conversation 对象（包含 mapping 字段）
输出：树状结构，显示对话的层级关系和内容
"""

import json
import sys
from typing import Dict, List, Any, Optional


class ConversationNode:
    """对话节点类"""

    def __init__(self, node_id: str, data: Dict[str, Any]):
        self.id = node_id
        self.parent = data.get('parent')
        self.children = data.get('children', [])
        self.message = data.get('message')

    def get_content(self) -> str:
        """提取节点的文本内容"""
        if not self.message:
            return ""

        content = self.message.get('content', {})
        parts = content.get('parts', [])

        # 提取文本内容
        text_parts = []
        for part in parts:
            if isinstance(part, str):
                text_parts.append(part)
            elif isinstance(part, dict):
                # 跳过图片等非文本内容
                if part.get('content_type') not in ['image_asset_pointer', 'execution_output']:
                    continue

        return ''.join(text_parts).strip()

    def get_role(self) -> str:
        """获取节点的角色"""
        if not self.message:
            return "unknown"
        return self.message.get('author', {}).get('role', 'unknown')


def build_tree(conversation: Dict[str, Any]) -> Dict[str, ConversationNode]:
    """构建节点映射"""
    mapping = conversation.get('mapping', {})
    nodes = {}

    for node_id, node_data in mapping.items():
        nodes[node_id] = ConversationNode(node_id, node_data)

    return nodes


def find_root(nodes: Dict[str, ConversationNode]) -> Optional[str]:
    """找到根节点"""
    for node_id, node in nodes.items():
        if node.parent is None:
            return node_id
    return None


def print_tree(nodes: Dict[str, ConversationNode],
               node_id: str,
               show_ids: bool = False):
    """打印链条式树状结构

    格式：
    user: xxx
    - assistant: yyy
    - assistant: zzz
    user: aaa
    - assistant: bbb

    Args:
        nodes: 节点映射
        node_id: 当前节点ID
        show_ids: 是否显示节点ID
    """

    def print_chain(current_id: str, indent: str = ""):
        """打印一个消息链条"""
        if current_id not in nodes:
            return

        node = nodes[current_id]
        role = node.get_role()
        content = node.get_content()

        # 跳过system节点和空内容节点
        if role == "system" or (not content and role != "user"):
            for child_id in node.children:
                print_chain(child_id, indent)
            return

        # 显示当前节点
        content_preview = content[:80].replace('\n', ' ') if content else ""
        if len(content) > 80:
            content_preview += "..."

        if show_ids:
            print(f"{indent}{role} [{current_id[:8]}]: {content_preview}")
        else:
            print(f"{indent}{role}: {content_preview}")

        # 处理子节点
        children = node.children

        if len(children) == 0:
            return
        elif len(children) == 1:
            # 单分支，继续链条
            print_chain(children[0], "- ")
        else:
            # 多分支，每个作为独立的子链
            for child_id in children:
                print_chain(child_id, "- ")

    print_chain(node_id)


def print_full_content(nodes: Dict[str, ConversationNode],
                       node_id: str,
                       depth: int = 0):
    """打印完整内容（非树状，按深度缩进）

    Args:
        nodes: 节点映射
        node_id: 当前节点ID
        depth: 当前深度
    """
    if node_id not in nodes:
        return

    node = nodes[node_id]
    role = node.get_role()
    content = node.get_content()

    indent = "  " * depth

    if content:
        print(f"{indent}[{role.upper()}]")
        # 内容每行都缩进
        for line in content.split('\n'):
            print(f"{indent}  {line}")
        print()  # 空行分隔

    # 递归处理子节点
    for child_id in node.children:
        print_full_content(nodes, child_id, depth + 1)


def main():
    """主函数"""
    if len(sys.argv) < 2:
        print("用法:")
        print("  python parse_conversation_tree.py <conversation_file.json> [--full] [--ids]")
        print()
        print("参数:")
        print("  conversation_file.json  包含单个conversation的JSON文件")
        print("  --full                  显示完整内容（非树状结构）")
        print("  --ids                   显示节点ID")
        sys.exit(1)

    input_file = sys.argv[1]
    show_full = '--full' in sys.argv
    show_ids = '--ids' in sys.argv

    # 读取输入文件
    try:
        with open(input_file, 'r', encoding='utf-8') as f:
            conversation = json.load(f)
    except Exception as e:
        print(f"错误: 无法读取文件 {input_file}")
        print(f"详细信息: {e}")
        sys.exit(1)

    # 构建树
    nodes = build_tree(conversation)
    root_id = find_root(nodes)

    if not root_id:
        print("错误: 未找到根节点")
        sys.exit(1)

    # 打印标题
    title = conversation.get('title', '未命名对话')
    print(f"对话: {title}")
    print("=" * 60)
    print()

    # 根据参数选择输出格式
    if show_full:
        print_full_content(nodes, root_id)
    else:
        print_tree(nodes, root_id, show_ids=show_ids)


if __name__ == "__main__":
    main()
