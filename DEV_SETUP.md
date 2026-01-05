# 开发环境配置指南

## 一、环境要求

### 1.1 必需软件

| 软件 | 版本要求 | 当前版本 | 安装状态 |
|------|---------|---------|---------|
| Go | 1.21+ | 1.25.4 | ✅ |
| Python3 | 3.8+ | 3.12.8 | ✅ |
| SQLite3 | 3.35+ | 3.43.2 | ✅ |
| Git | 2.0+ | - | ✅ |

### 1.2 可选软件

| 软件 | 用途 | 说明 |
|------|------|------|
| Docker | 容器化部署 | 生产环境可选 |
| Caddy | HTTPS反向代理 | 生产环境必需 |

---

## 二、快速开始

### 2.1 克隆项目

```bash
cd ~/Downloads
# 项目已存在于: /Users/rhinenoir/Downloads/cli-session-history
cd cli-session-history
```

### 2.2 初始化数据库

```bash
# 执行数据库初始化脚本
./scripts/init_db.sh
```

这将创建：
- `data/conversation.db` - SQLite数据库
- `data/images/` - 图片存储目录
- 预设的标签数据

### 2.3 配置文件

```bash
# 复制示例配置文件
cp config.yaml.example config.yaml

# 编辑配置文件
vim config.yaml
```

**重要配置项：**
- `security.auth_token` - 修改为随机字符串
- `security.cors.allow_origins` - 生产环境设置为具体域名

### 2.4 安装Go依赖

```bash
# 项目根目录
go mod download
go mod tidy

# scripts目录
cd scripts
go mod download
go mod tidy
cd ..
```

---

## 三、开发工具配置

### 3.1 Go开发环境

**推荐IDE：**
- VS Code + Go extension
- GoLand

**VS Code配置 (`.vscode/settings.json`)：**
```json
{
  "go.useLanguageServer": true,
  "go.toolsManagement.autoUpdate": true,
  "editor.formatOnSave": true,
  "[go]": {
    "editor.codeActionsOnSave": {
      "source.organizeImports": true
    }
  }
}
```

### 3.2 Python开发环境

**推荐使用 /usr/local/bin/python3**

```bash
# 检查Python版本
/usr/local/bin/python3 --version

# 安装依赖（如需要）
/usr/local/bin/python3 -m pip install -r requirements.txt
```

---

## 四、运行项目

### 4.1 启动Backend服务器

**方法1：使用启动脚本**
```bash
./start.sh
```

**方法2：手动启动**
```bash
cd backend
go build -o conversation-server main.go
./conversation-server
```

服务器将在 `http://localhost:8080` 运行

### 4.2 测试服务器

```bash
# 健康检查
curl http://localhost:8080/health

# 列出所有对话
curl http://localhost:8080/list
```

### 4.3 停止服务器

```bash
./stop.sh
```

---

## 五、数据导入

### 5.1 解析GPT对话

```bash
cd scripts
./run_parse.sh <conversation_file.json>
```

### 5.2 解析对话树

```bash
cd scripts
python3 parse_conversation_tree.py <conversation_file.json>
```

---

## 六、项目结构

```
cli-session-history/
├── backend/                  # Go后端服务
│   ├── main.go
│   └── conversation-server   # 编译产物
├── scripts/                  # 工具脚本
│   ├── init_db.sh           # 数据库初始化
│   ├── init_database.sql    # SQL初始化脚本
│   ├── gpt_conversation_parse.go
│   ├── parse_conversation_tree.py
│   └── go.mod
├── data/                     # 数据目录
│   ├── conversation.db      # SQLite数据库
│   ├── bleve_index/         # 搜索索引
│   ├── images/              # 图片存储
│   ├── gpt/                 # GPT数据
│   ├── claude/              # Claude数据
│   ├── claude_code/         # Claude Code数据
│   └── codex/               # Codex数据
├── docs/                     # 技术文档
│   ├── architecture.md      # 架构设计
│   ├── database-schema.md   # 数据库设计
│   └── search-index.md      # 搜索索引设计
├── config.yaml              # 主配置文件
├── config.yaml.example      # 配置示例
├── go.mod                   # Go依赖
├── start.sh                 # 启动脚本
├── stop.sh                  # 停止脚本
└── DEV_SETUP.md            # 本文档
```

---

## 七、常见问题

### 7.1 数据库初始化失败

**问题：** `sqlite3: command not found`

**解决：**
```bash
# macOS
brew install sqlite3

# Ubuntu/Debian
sudo apt-get install sqlite3
```

### 7.2 Go依赖下载失败

**问题：** 网络问题导致依赖下载失败

**解决：**
```bash
# 使用Go代理
export GOPROXY=https://goproxy.cn,direct
go mod download
```

### 7.3 端口被占用

**问题：** `bind: address already in use`

**解决：**
```bash
# 查找占用8080端口的进程
lsof -i :8080

# 杀死进程
kill -9 <PID>

# 或修改config.yaml中的端口
```

### 7.4 Python版本问题

**问题：** 脚本提示Python版本不兼容

**解决：**
```bash
# 使用指定的Python版本
/usr/local/bin/python3 script.py
```

---

## 八、开发工作流

### 8.1 日常开发

1. 启动服务器：`./start.sh`
2. 修改代码
3. 重启服务器：`./stop.sh && ./start.sh`
4. 测试API
5. 提交代码

### 8.2 数据库变更

1. 修改 `scripts/init_database.sql`
2. 备份现有数据：`cp data/conversation.db data/conversation.db.backup`
3. 重新初始化：`./scripts/init_db.sh`
4. 测试迁移

### 8.3 Git工作流

```bash
# 查看状态
git status

# 添加文件
git add .

# 提交（注意：不要提交config.yaml和数据文件）
git commit -m "描述信息"

# 推送
git push
```

---

## 九、下一步

### 9.1 核心功能开发

根据 `docs/architecture.md`，需要实现：

1. **API Server (Go + Gin)**
   - RESTful API端点
   - SQLite数据访问
   - Bleve搜索集成
   - 认证中间件

2. **数据同步Worker**
   - 监听数据源变更
   - 解析JSON数据
   - 批量写入数据库
   - 索引更新

3. **前端开发**
   - Web端 (Vue 3 + Naive UI) - 调试用
   - iOS端 (Flutter) - 主要使用

### 9.2 技术栈学习

- [Gin框架文档](https://gin-gonic.com/)
- [Bleve搜索引擎](https://blevesearch.com/)
- [SQLite文档](https://www.sqlite.org/docs.html)
- [Go标准库](https://pkg.go.dev/std)

---

## 十、联系方式

遇到问题请查阅：
- 技术文档：`docs/`目录
- Issue跟踪：项目Git仓库
