# Scripts 工具集

## Go 工具

### 1. GPT Conversation Parser (对话解析器)
- `gpt_conversation_parse.go` - 将 GPT 对话转为线性消息链
- `run_parse.sh` - 编译运行脚本
- 详细文档: `README_parse.md`

### 2. JSON Value Decoder (JSON 解码器)
- `decode_json_value.go` - 解码 Unicode 转义和控制字符
- `run_decode.sh` - 编译运行脚本
- Python 原版: `json_openai_transfer.py`

## Python 工具

### 对话树解析工具
- `parse_conversation_tree.py` - 核心解析脚本，将单个conversation对象解析为树状结构
- `extract_and_parse.py` - 便捷脚本，从conversations数组中提取指定索引的对话并解析
- `test_conversation.json` - 测试用的单个conversation样例

## 使用方法

### 方法1：直接解析单个conversation文件

```bash
python parse_conversation_tree.py <conversation_file.json> [选项]
```

**参数：**
- `conversation_file.json` - 包含单个conversation对象的JSON文件
- `--full` - 显示完整内容（按层级缩进，非树状图）
- `--ids` - 显示节点ID

**示例：**
```bash
# 树状结构预览
python parse_conversation_tree.py test_conversation.json

# 显示完整内容
python parse_conversation_tree.py test_conversation.json --full

# 显示节点ID（便于调试）
python parse_conversation_tree.py test_conversation.json --ids
```

### 方法2：从数组中提取并解析

```bash
python extract_and_parse.py <json_file> <index> [选项]
```

**参数：**
- `json_file` - 包含conversations数组的JSON文件
- `index` - 要解析的conversation索引（从0开始）
- `--full` - 显示完整内容
- `--ids` - 显示节点ID

**示例：**
```bash
# 解析第一个conversation（索引0）
python extract_and_parse.py ../conversations_backup_account_modified.json 0

# 解析第6个conversation，显示完整内容
python extract_and_parse.py ../conversations_backup_account_modified.json 5 --full
```

## 输出格式

### 链条树模式（默认）

```
对话: 身材对比分析
============================================================

user: 来对比一下这俩身材（👀语言简洁点
- assistant: 第一张：肩膀宽、胸肌饱满、腹部线条明显...
- assistant: 简洁对比：图1（上图）胸肌厚、饱满...
- user: 让你来选选哪个👀
- assistant: 如果是以"舞台角色冲击力"来看——我选图1...
```

- 显示对话的链条结构，每条消息作为一个节点
- 使用 `-` 连接后续回复
- 多个分支会并列显示为多个 `-` 项
- 自动跳过 system 节点和空内容节点
- 内容预览限制为前80字符

### 完整内容模式（--full）

```
[USER]
  来对比一下这俩身材（👀语言简洁点

  [ASSISTANT]
    第一张：肩膀宽、胸肌饱满、腹部线条明显，整体偏结实健美型。
    第二张：身材更纤长...
```

- 按层级缩进显示完整对话内容
- 适合阅读完整对话流程

## JSON 数据结构说明

输入的conversation对象包含：
- `title` - 对话标题
- `mapping` - 节点映射字典，key为节点ID
  - 每个节点包含：
    - `id` - 节点唯一标识
    - `parent` - 父节点ID
    - `children` - 子节点ID数组
    - `message` - 消息内容
      - `author.role` - 角色（user/assistant/system）
      - `content.parts` - 文本内容数组
