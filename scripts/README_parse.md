# GPT Conversation Parser

GPT 对话历史解析工具,将 ChatGPT 导出的 JSON 格式对话转换为线性结构的消息链。

## 功能说明

- 读取 ChatGPT 导出的对话 JSON 文件
- 从每个对话的 `current_node` 开始,通过 `parent` 字段向上追溯完整对话链
- 提取关键信息:角色、内容类型、消息内容、创建时间
- 按从头到尾的顺序输出为独立的 JSON 文件
- 自动处理图片标记、多模态内容

## 使用方法

### 方式1: 使用运行脚本(推荐)

```bash
# 基本用法
./scripts/run_parse.sh <输入文件> [输出目录]

# 示例
./scripts/run_parse.sh data/conversations_backup_account_modified.json

# 指定输出目录
./scripts/run_parse.sh data/conversations.json parsed_output
```

### 方式2: 手动编译运行

```bash
# 编译
cd scripts
go build -o ../bin/gpt_conversation_parse gpt_conversation_parse.go

# 运行
../bin/gpt_conversation_parse -input <文件路径> -output <输出目录>

# 示例
../bin/gpt_conversation_parse \
  -input data/conversations_backup_account_modified.json \
  -output parsed_conversations
```

## 参数说明

- `-input`: 必需,输入的 JSON 文件路径
- `-output`: 可选,输出目录(默认: `output`)

## 输出格式

每个对话生成一个独立的 JSON 文件,文件名为 `conversation_id.json`,包含消息数组:

```json
[
  {
    "role": "user",
    "content_type": "text",
    "content": "你好",
    "create_time": 1752791273.511
  },
  {
    "role": "assistant",
    "content_type": "text",
    "content": "你好!有什么我可以帮助你的吗?",
    "create_time": 1752791275.472
  }
]
```

## 测试结果

已使用 `conversations_backup_account_modified.json` 测试:

- ✅ 成功解析 184 个对话
- ✅ 消息总数: 1800+ 条
- ✅ 最长对话: 63 条消息
- ✅ 正确处理图片、多模态内容
- ✅ 完整保留对话链顺序

## 技术细节

### 数据结构处理

1. **链追溯**: 从 `current_node` 开始,通过 `parent` 字段递归向上
2. **顺序保证**: 追溯时倒序插入,确保最终顺序为从头到尾
3. **内容提取**:
   - 文本直接提取
   - 图片转换为 `[图片]` 标记
   - 多模态内容合并处理

### 文件名处理

自动清理非法字符: `/ \ : * ? " < > |` → `_`

## 注意事项

1. 输入文件必须是 ChatGPT 导出的标准 JSON 格式
2. 需要 Go 1.16+ 环境
3. 空消息节点会被保留(如 system 消息)
4. 输出目录会自动创建
