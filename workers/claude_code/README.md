# Claude Code Worker

Claude Code 数据同步 Worker，自动处理 Claude Code 导出的 ZIP 压缩包并同步到 SQLite 数据库。

## 功能

- 扫描 `data/original/claude_code/` 目录下的 ZIP 压缩包
- 自动解压并验证目录结构
- 解析 `projects/` 目录下的所有 JSONL 文件
- 提取对话和消息数据，写入 SQLite 数据库
- 处理完成后删除压缩包和临时数据
- 自动标记格式错误的压缩包为废弃状态
- 循环检测新的压缩包

## 数据格式

### 压缩包结构

```
claude_code.zip
└── projects/
    ├── project-name-1/
    │   ├── session-id-1.jsonl
    │   └── session-id-2.jsonl
    └── project-name-2/
        └── session-id-3.jsonl
```

或者：

```
claude_code.zip
└── claude_code/
    └── projects/
        └── ...
```

### JSONL 格式

每行一个 JSON 对象：

```json
{"type":"user","uuid":"xxx","parentUuid":"yyy","sessionId":"zzz","message":{"role":"user","content":"..."},"timestamp":"2025-01-01T00:00:00.000Z"}
```

## 使用方法

### 方式 1：使用启动脚本（推荐）

```bash
# 使用默认配置
./scripts/run_claude_code_worker.sh

# 自定义配置
DB_PATH=data/my.db CHECK_INTERVAL=60s ./scripts/run_claude_code_worker.sh
```

### 方式 2：统一编译后运行

```bash
# 编译所有 Workers
./scripts/build_workers.sh

# 运行
./bin/claude_code_worker \
  -db data/conversations.db \
  -data data/original/claude_code \
  -temp data/temp \
  -interval 30s
```

### 方式 3：手动编译运行

```bash
# 编译
cd workers/claude_code/cmd
go build -o ../../../bin/claude_code_worker .
cd ../../..

# 运行
./bin/claude_code_worker \
  -db data/conversations.db \
  -data data/original/claude_code \
  -temp data/temp \
  -interval 30s
```

## 参数说明

- `-db`: 数据库文件路径（默认：`data/conversations.db`）
- `-data`: 原始数据目录（默认：`data/original/claude_code`）
- `-temp`: 临时目录（默认：`data/temp`）
- `-interval`: 检查间隔（默认：`30s`）

## 数据处理规则

1. **标题生成**：使用第一条 user 消息的前 50 字符作为对话标题
2. **round_index**：根据 user 消息递增
3. **metadata**：暂时留空
4. **废弃处理**：
   - 解压失败 → 标记为 `xxx.abandoned_TIMESTAMP.zip`
   - 结构不符合 → 标记为 `xxx.abandoned_TIMESTAMP.zip`
   - 写入废弃原因到 `.reason` 文件

## 实现细节

- 使用 GPT Worker 方式直接写 SQLite
- 支持事务处理，确保数据一致性
- 自动跳过已处理的压缩包
- 容错处理：单个会话解析失败不影响其他会话

## 文件结构

```
workers/claude_code/
├── cmd/
│   └── main.go          # 入口文件
├── worker.go            # Worker 主逻辑
├── parser.go            # JSONL 解析
├── database.go          # 数据库操作
├── utils.go             # 工具函数
└── README.md            # 本文档
```
