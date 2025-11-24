# 对话服务器

一个用于读取和提供对话 JSON 数据的 Go 后端服务器。

## 功能

- 支持多个对话来源：gpt、claude、claude_code、codex
- RESTful API 接口
- 自动查找 parsed 和 data 目录中的对话文件
- CORS 支持
- 健康检查端点
- 请求日志记录

## API 端点

### 1. 健康检查
```bash
GET /health
```

返回服务状态。

### 2. 列出所有对话
```bash
GET /list
```

返回所有来源的可用对话 ID 列表。

### 3. 获取对话数据
```bash
GET /{source}/{conversation_id}
```

参数：
- `source`: 对话来源（gpt、claude、claude_code、codex）
- `conversation_id`: 对话 ID

示例：
```bash
curl http://localhost:8080/gpt/d4d4ddf6-5452-4dbb-9c1c-8a59ebfdb8fa
```

## 启动服务

使用项目根目录的启动脚本：

```bash
cd /Users/rhinenoir/Downloads/cli-session-history
./start.sh
```

## 停止服务

```bash
./stop.sh
```

## 手动启动

如果需要手动启动：

```bash
cd backend
go build -o conversation-server main.go
./conversation-server
```

服务器将在端口 8080 上运行。

## 文件查找逻辑

服务器会按以下顺序查找对话文件：

1. `parsed/{source}/conversation/{conversation_id}.json`
2. `data/{source}/{conversation_id}.json`
3. `parsed/{source}/{conversation_id}.json`

## 日志

运行日志保存在：`backend/server.log`

## 进程管理

PID 文件：`backend/server.pid`
