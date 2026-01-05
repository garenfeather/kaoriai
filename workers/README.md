# Data Sync Workers

AI对话数据管理系统的数据同步Workers。

## 目录结构

```
workers/
├── common/              # 公共模块
│   ├── config.go       # 配置管理
│   ├── client.go       # API客户端
│   └── types.go        # 公共类型定义
├── openai/             # OpenAI/GPT Worker
│   ├── worker.go
│   └── main.go
├── claude/             # Claude Worker
│   ├── worker.go
│   └── main.go
├── claude_code/        # Claude Code Worker
│   ├── worker.go
│   └── main.go
├── codex/              # Codex Worker
│   ├── worker.go
│   └── main.go
├── gemini/             # Gemini Worker (TODO)
│   └── worker.go
├── gemini_cli/         # Gemini CLI Worker (TODO)
│   └── worker.go
├── config.yaml.example # 配置文件示例
└── README.md           # 本文件
```

## Worker功能

每个Worker负责特定数据源的数据同步：

| Worker | 数据源 | 同步方式 | 状态 |
|--------|--------|---------|------|
| openai | ChatGPT | 邮件监听/文件监听 | 框架完成 |
| claude | Claude | 文件监听 | 框架完成 |
| claude_code | Claude Code | 文件监听 | 框架完成 |
| codex | Codex | 文件监听 | 框架完成 |
| gemini | Gemini | 文件监听 | TODO |
| gemini_cli | Gemini CLI | 文件监听 | TODO |

## 核心功能

### 1. 数据监听与检测
- 定时检查数据源（文件/邮件）
- MD5校验避免重复处理
- 支持增量同步

### 2. 数据解析
- 复用 `scripts/go/` 下的parser
- 统一转换为标准格式
- 支持批量处理

### 3. 数据上传
- 批量提交到API服务器
- 自动重试机制
- 状态跟踪与报告

### 4. Worker管理
- 优雅启动/停止
- 状态监控
- 错误日志记录

## 配置说明

复制配置模板并修改：

```bash
cp config.yaml.example config.yaml
```

主要配置项：

```yaml
server:
  base_url: "http://localhost:8080"  # API服务器地址
  timeout: 30

worker:
  check_interval: 60   # 检查间隔(秒)
  batch_size: 100      # 批量上传大小

security:
  worker_token: "xxx"  # Worker访问Token
```

## 使用方法

### 编译Worker

```bash
# 编译所有Worker
cd workers
go build -o ../bin/openai-sync-worker ./openai/main.go
go build -o ../bin/claude-sync-worker ./claude/main.go
go build -o ../bin/claude-code-sync-worker ./claude_code/main.go
go build -o ../bin/codex-sync-worker ./codex/main.go
```

### 运行Worker

#### OpenAI Worker
```bash
./bin/openai-sync-worker -config config.yaml
```

环境变量：
- `OPENAI_EMAIL_USERNAME`: 邮箱用户名
- `OPENAI_EMAIL_PASSWORD`: 邮箱密码

#### Claude Worker
```bash
./bin/claude-sync-worker -config config.yaml -data-dir /path/to/claude/data
```

#### Claude Code Worker
```bash
./bin/claude-code-sync-worker -config config.yaml -projects-dir /path/to/claude-code/projects
```

#### Codex Worker
```bash
./bin/codex-sync-worker -config config.yaml -sessions-dir /path/to/codex/sessions
```

## API端点

Worker通过以下API与服务器交互：

### 批量同步
```
POST /internal/v1/sync/batch
Authorization: Bearer <worker_token>

请求体:
{
  "source_type": "gpt",
  "conversations": [...]
}

响应:
{
  "success": true,
  "inserted_conversations": 10,
  "inserted_messages": 50
}
```

## 开发指南

### 添加新的数据源Worker

1. 在 `workers/` 下创建新目录
2. 实现 `common.Worker` 接口：
   ```go
   type Worker interface {
       Name() string
       SourceType() string
       Start() error
       Stop() error
       Check() error
   }
   ```
3. 实现数据获取和解析逻辑
4. 创建 `main.go` 作为入口

### Worker实现要点

1. **线程安全**: 使用 `sync.RWMutex` 保护状态
2. **优雅退出**: 监听停止信号，清理资源
3. **错误处理**: 记录详细日志，支持重试
4. **状态跟踪**: 更新状态信息供监控使用

## TODO

- [ ] 实现parser调用逻辑
- [ ] 添加重试机制
- [ ] 完善错误处理
- [ ] 实现Gemini Worker
- [ ] 实现Gemini CLI Worker
- [ ] 添加Worker监控API
- [ ] 支持配置热加载

## 参考

- 架构文档: `../docs/architecture.md`
- 数据库设计: `../docs/database-schema.md`
- Parser脚本: `../scripts/go/`
