# AI对话数据管理系统 - 架构设计文档

## 一、系统概述

### 1.1 系统定位

AI对话数据综合管理系统,用于集成和管理多个AI工具的对话历史,支持全文检索、多维度收藏、标签管理等功能。

**核心特性:**
- 多数据源支持: Claude、Claude Code、ChatGPT、Codex、Gemini、Gemini-CLI
- 多粒度操作: 支持对conversation、round、message、fragment级别操作
- 全文检索: 基于Bleve的中文/韩文优化搜索
- 移动优先: iOS客户端为主要使用场景,Web端用于调试

### 1.2 技术特点

- **单用户系统**: 无需复杂的权限管理
- **嵌入式存储**: SQLite + Bleve,无外部数据库依赖
- **轻量部署**: Go单一二进制 + 数据目录
- **数据规模**: 设计支持5GB以下文本数据

---

## 二、技术选型

### 2.1 核心技术栈

| 组件 | 技术 | 版本要求 | 选型理由 |
|------|------|---------|---------|
| 后端语言 | Go | 1.21+ | 性能优秀,部署简单,与现有parsers兼容 |
| Web框架 | Gin | 1.9+ | 轻量高性能,中文文档完善 |
| 数据库 | SQLite | 3.35+ | 嵌入式,无需独立服务,适合单用户场景 |
| 搜索引擎 | Bleve | 2.x | Go原生,CJK支持好,无外部依赖 |
| 存储后端 | BadgerDB | - | Bleve索引存储,比默认BoltDB更快 |
| 配置管理 | Viper | - | 支持多种配置格式 |
| 日志 | Zap | - | 结构化日志,性能好 |
| 前端(Web) | Vue 3 + Naive UI | - | 调试用,UI可简陋 |
| 前端(iOS) | Flutter | 3.x | 快速开发,参考Kelivo项目UI |

### 2.2 为什么选择Bleve?

**对比MySQL InnoDB全文索引:**
- ✅ **中文分词更优**: Bleve CJK analyzer效果好于MySQL ngram
- ✅ **零依赖**: 无需独立数据库服务
- ✅ **Go原生**: 编译到同一二进制,无需网络通信
- ✅ **部署简单**: 索引目录随应用一起部署
- ✅ **成本低**: 无需云数据库服务费用

**对比Meilisearch:**
- ✅ **架构简单**: 不需要独立Rust服务
- ✅ **生态一致**: Go项目内原生集成
- ✅ **性能足够**: 5GB数据规模下,Bleve延迟<50ms

### 2.3 为什么选择SQLite?

- ✅ **单用户场景**: 无需并发写入优化
- ✅ **数据规模小**: 5GB以内,SQLite性能足够
- ✅ **零运维**: 无需独立进程,文件即数据库
- ✅ **备份简单**: 文件复制即可
- ❌ **不适合**: 多用户高并发场景(非本项目目标)

---

## 三、系统架构

### 3.1 整体架构图

```
┌─────────────────────────────────────────────────────────┐
│                    iOS App (Flutter)                     │
│  - 对话列表                                               │
│  - 搜索界面                                               │
│  - 收藏管理                                               │
│  - 标签管理                                               │
└────────────────────┬────────────────────────────────────┘
                     │ HTTPS/JSON
                     ▼
┌─────────────────────────────────────────────────────────┐
│               Caddy (反向代理 + HTTPS)                   │
└────────────────────┬────────────────────────────────────┘
                     │ HTTP
                     ▼
┌─────────────────────────────────────────────────────────┐
│              Go API Server (Gin Framework)               │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │   Bleve      │  │   SQLite     │  │   Images     │ │
│  │  (搜索索引)  │  │  (结构化数据) │  │  (本地文件)  │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│                                                          │
│  Handler → Service → Repository                         │
└────────────────────┬────────────────────────────────────┘
                     ▲
                     │ HTTP/gRPC (内部接口)
                     │
┌────────────────────┴────────────────────────────────────┐
│           Data Sync Workers (独立进程)                   │
│  ┌──────────────────┐  ┌──────────────────┐            │
│  │ OpenAI Worker    │  │ Claude Worker    │  ...       │
│  │ (邮件监听)       │  │ (文件监听)       │            │
│  └──────────────────┘  └──────────────────┘            │
│                                                          │
│  - 数据采集                                              │
│  - 格式解析(复用scripts下的parsers)                      │
│  - 批量上传到API Server                                  │
└─────────────────────────────────────────────────────────┘
```

### 3.2 目录结构

```
/opt/conversation-manager/
├── bin/
│   ├── api-server              # API服务主程序
│   ├── openai-sync-worker      # GPT数据同步Worker
│   ├── claude-sync-worker      # Claude数据同步Worker
│   └── codex-sync-worker       # Codex数据同步Worker
│
├── data/
│   ├── conversation.db         # SQLite数据库
│   ├── bleve_index/            # Bleve索引目录
│   │   ├── index_meta.json
│   │   └── store/              # BadgerDB存储
│   └── images/                 # 图片本地存储
│       ├── gpt/
│       ├── claude/
│       ├── claude_code/
│       └── codex/
│
├── config/
│   └── config.yaml             # 配置文件
│
└── logs/
    ├── api-server.log
    └── sync-worker.log
```

### 3.3 部署架构

```
┌─────────────────────────────────────────┐
│         VPS服务器 (2核4G)                │
│                                          │
│  ┌────────────────────────────────────┐ │
│  │  Caddy (端口443)                   │ │
│  │  - HTTPS自动证书                   │ │
│  │  - 反向代理到API Server            │ │
│  └──────────────┬─────────────────────┘ │
│                 │                        │
│  ┌──────────────▼─────────────────────┐ │
│  │  API Server (端口8080)             │ │
│  │  - Gin Web服务                     │ │
│  │  - SQLite + Bleve                  │ │
│  └────────────────────────────────────┘ │
│                                          │
│  ┌────────────────────────────────────┐ │
│  │  Sync Workers (后台进程)           │ │
│  │  - 定时检查数据源                   │ │
│  │  - 解析并上传到API Server          │ │
│  └────────────────────────────────────┘ │
│                                          │
│  /opt/conversation-manager/data/        │
│  - 数据库文件                            │
│  - 索引文件                              │
│  - 图片文件                              │
└─────────────────────────────────────────┘
```

---

## 四、数据流设计

### 4.1 数据写入流程

```
┌─────────────────┐
│  数据源         │
│  - 邮件         │
│  - 本地文件     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Sync Worker     │
│ 1. 监听/定时    │
│ 2. 计算MD5      │
│ 3. 检查变更     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Parser          │
│ (复用scripts)   │
│ - 解析JSON      │
│ - 提取字段      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ API Server      │
│ POST /internal/ │
│     sync/batch  │
└────────┬────────┘
         │
    ┌────┴────┐
    ▼         ▼
┌────────┐ ┌──────┐
│ SQLite │ │Bleve │
│ 写入   │ │索引  │
└────────┘ └──────┘
```

**关键点:**
1. Worker独立于API Server,解耦数据采集和服务
2. 通过MD5检查避免重复处理
3. SQLite和Bleve同步写入,保持一致性
4. 仅索引assistant的文字回复到Bleve

### 4.2 搜索查询流程

```
┌─────────────┐
│ iOS App     │
│ 输入关键词  │
└──────┬──────┘
       │ POST /api/v1/search
       │ {keyword, filters, page}
       ▼
┌─────────────┐
│ API Server  │
│ SearchService│
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Bleve       │
│ 1. CJK分词  │
│ 2. 全文搜索 │
│ 3. 过滤     │
│ 4. 排序     │
└──────┬──────┘
      │ 返回message_uuids
       ▼
┌─────────────┐
│ SQLite      │
│ 根据IDs查询 │
│ 完整数据    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 组装响应    │
│ - 消息内容  │
│ - 高亮片段  │
│ - 元数据    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ iOS App     │
│ 渲染结果    │
└─────────────┘
```

**关键点:**
1. Bleve只用于搜索,返回message_uuid列表
2. SQLite存储完整数据,根据UUIDs查询详情
3. 高亮信息由Bleve自动生成
4. 支持多维度过滤(时间、来源、标签等)

### 4.3 数据同步策略

**职责分工:**

| 组件 | 职责 | 存储内容 |
|------|------|---------|
| **SQLite** | 结构化数据存储 | conversations, messages(完整JSON), favorites, tags, fragments |
| **Bleve** | 全文搜索索引 | message_uuid + content_text + 过滤字段(source_type, role, created_at) |
| **文件系统** | 二进制文件 | 图片文件(JPEG/PNG等) |

**同步规则:**
1. 写入message时,同时写SQLite和Bleve
2. 当`content_text`非空时,索引到Bleve(包括user和assistant)
3. 图片不索引,只在SQLite记录metadata,文件存本地
4. 删除操作软删除(hidden_at)

---

## 五、API设计

### 5.1 RESTful API规范

**基础路径:** `/api/v1`

**通用响应格式:**
```json
{
  "code": 0,          // 0=成功, 非0=错误码
  "message": "ok",
  "data": {...}
}
```

**分页参数:**
```json
{
  "page": 1,          // 页码,从1开始
  "page_size": 20,    // 每页条数
  "total": 100        // 总数
}
```

### 5.2 核心API端点

#### 5.2.1 对话管理

```
GET    /api/v1/conversations
       查询参数: source_type, date_from, date_to, page, page_size
       响应: {items: [...], total, page, page_size}

GET    /api/v1/conversations/:uuid
       响应: conversation详情 + metadata

GET    /api/v1/conversations/:uuid/messages
       查询参数: page, page_size
       响应: message列表(时间线顺序)
```

#### 5.2.1.1 对话树管理

依赖数据库表 `conversation_trees`，`tree_data` JSON 结构与 `docs/database-schema.md` 中的定义一致（根节点及其children递归构成一棵树）。

```
GET    /api/v1/trees
       查询参数: page, page_size
       响应:
       {
         "items": [
           {
             "tree_id": "tree-xxx",
             "title": "监控方案演进",
             "description": "从Prometheus到多种方案对比",
             "created_at": "2025-11-20T10:00:00Z",
             "updated_at": "2025-11-20T10:00:00Z"
           }
         ],
         "total": 1,
         "page": 1,
         "page_size": 20
       }

POST   /api/v1/trees
       请求:
       {
         "title": "监控方案演进",
         "description": "从Prometheus到多种方案对比",
         "tree_data": {
           "nodes": [
             {
               "conversation_uuid": "conv-abc123",
               "parent_uuid": null,
               "order": 0,
               "notes": "初始方案",
               "children": [
                 {
                   "conversation_uuid": "conv-def456",
                   "parent_uuid": "conv-abc123",
                   "order": 0,
                   "notes": "改进方案",
                   "children": []
                  },
                  {
                    "conversation_uuid": "conv-ghi789",
                    "parent_uuid": "conv-abc123",
                    "order": 1,
                    "notes": "并行方案",
                    "children": []
                  }
                ]
              }
            ]
          }
       }
       响应:
       {
         "tree_id": "tree-xxx",
         "created_at": "2025-11-20T10:00:00Z",
         "updated_at": "2025-11-20T10:00:00Z"
       }
       功能: 创建新的对话树，后端生成`tree_id`并将完整结构写入conversation_trees.tree_data

GET    /api/v1/trees/:tree_id
       响应:
       {
         "tree_id": "tree-xxx",
         "title": "监控方案演进",
         "description": "从Prometheus到多种方案对比",
         "tree_data": {
           "nodes": [
             {
               "conversation_uuid": "conv-abc123",
               "parent_uuid": null,
               "order": 0,
               "notes": "初始方案",
                "children": [
                 {
                   "conversation_uuid": "conv-def456",
                   "parent_uuid": "conv-abc123",
                   "order": 0,
                   "notes": "改进方案",
                   "children": []
                 }
               ]
             }
           ]
         },
         "created_at": "2025-11-20T10:00:00Z",
         "updated_at": "2025-11-20T10:00:00Z"
       }

PUT    /api/v1/trees/:tree_id
       请求:
       {
         "title": "监控方案演进",
         "description": "从Prometheus到多种方案对比",
         "tree_data": {
           "nodes": [
             {
               "conversation_uuid": "conv-abc123",
               "parent_uuid": null,
               "order": 0,
               "notes": "初始方案",
               "children": [
                 {
                   "conversation_uuid": "conv-def456",
                   "parent_uuid": "conv-abc123",
                   "order": 0,
                   "notes": "改进方案",
                   "children": []
                 },
                 {
                   "conversation_uuid": "conv-jkl012",
                   "parent_uuid": "conv-abc123",
                   "order": 1,
                   "notes": "新方案",
                   "children": []
                 }
               ]
             }
           ]
        }
       }
       功能: 覆盖更新元数据与tree_data（整棵树）

PATCH  /api/v1/trees/:tree_id
       请求: {title, description}
       功能: 仅更新树的元数据（不改tree_data）

DELETE /api/v1/trees/:tree_id
       功能: 删除conversation_trees中的整棵树记录
```

#### 5.2.2 搜索

```
POST   /api/v1/search
       请求:
       {
         "keyword": "prometheus",
         "sources": ["gpt", "claude"],     // 可选
         "date_from": "2025-01-01",        // 可选
         "date_to": "2025-11-30",          // 可选
         "tags": [1, 2],                   // 可选
         "page": 1,
         "page_size": 20
       }

       响应:
       {
         "total": 42,
         "items": [
           {
             "message_uuid": "msg-abc123",
             "conversation_uuid": "conv-def456",
             "conversation_title": "监控方案讨论",
             "role": "assistant",
             "content_type": "text",
             "content_preview": "...使用<mark>Prometheus</mark>...",
             "highlight": ["...高亮片段1...", "...高亮片段2..."],
             "source_type": "gpt",
             "created_at": "2025-11-20T10:30:00Z",
             "score": 4.52
           }
         ]
       }
```

#### 5.2.3 消息

```
GET    /api/v1/messages/:uuid
       响应: 完整message数据(包含content JSON)

GET    /api/v1/messages/:uuid/context
       查询参数: before=2, after=2
       响应: 上下文消息列表(前N条+当前+后N条)
```

#### 5.2.4 收藏

```
POST   /api/v1/favorites
       请求:
       {
         "target_type": "round",              // conversation | round | message | fragment
         "target_id": "conv-abc123-1",        // conversation: conv-abc123
                                             // round: conv-abc123-1 (conversation_uuid-轮次序号)
                                             // message: msg-def456
                                             // fragment: frag-xyz
         "category": "tech_solution",
         "notes": "Prometheus监控架构设计"
       }

GET    /api/v1/favorites
       查询参数: category, page, page_size
       响应: 收藏列表(字段: id, target_type, target_id, category, notes, created_at)

DELETE /api/v1/favorites/:id

PATCH  /api/v1/favorites/:id
       请求: {notes: "新备注", category: "inspiration"}
```

#### 5.2.5 标签

```
GET    /api/v1/tags
       响应: 所有标签列表(按使用次数排序)

POST   /api/v1/tags
       请求: {name: "监控", color: "#3B82F6"}

POST   /api/v1/conversation-tags
       请求:
       {
         "tag_id": 1,
         "conversation_uuid": "conv-abc123"
       }

DELETE /api/v1/conversation-tags/:id

GET    /api/v1/tags/:id/conversations
       查询参数: page, page_size
       响应: 该标签下的所有对话
```

#### 5.2.6 图片

```
GET    /api/v1/images/:image_id
       响应: 图片文件(Content-Type: image/jpeg等)
```

#### 5.2.8 统计

```
GET    /api/v1/stats/overview
       响应:
       {
         "total_conversations": 1234,
         "total_messages": 56789,
         "sources": {
           "gpt": 500,
           "claude": 300,
           ...
         }
       }

GET    /api/v1/stats/by-date
       查询参数: date_from, date_to
       响应: 按日期统计的消息数量
```

### 5.3 内部API(仅供Worker调用)

```
POST   /internal/v1/sync/batch
       请求头: Authorization: Bearer <worker_token>
       请求:
       {
         "source_type": "gpt",
         "conversations": [
           {
             "uuid": "xxx-xxx",
             "title": "...",
             "metadata": {...},
             "messages": [
               {
                 "uuid": "msg-1",
                 "parent_uuid": "",
                 "round_index": 1,          // round序号
                 "role": "user",
                 "content_type": "text",
                 "content": {...},
                 "content_text": "...",
                 "created_at": "2025-11-20T10:00:00Z"
               }
             ]
           }
         ]
       }

       响应:
       {
         "success": true,
         "inserted_conversations": 1,
         "inserted_messages": 2,
         "updated_conversations": 0,
         "updated_messages": 0
       }
```

---

## 六、前端架构

### 6.1 Web端(Vue 3)

**定位:** 开发调试工具,UI可简陋,重功能轻交互

**核心页面:**
1. 对话列表页 - 简单table展示
2. 对话详情页 - 时间线展示消息
3. 搜索页 - 搜索框+结果列表
4. 收藏管理 - CRUD操作
5. 标签管理 - CRUD操作

**技术栈:**
- Vue 3 + Composition API
- Naive UI (组件库)
- Pinia (状态管理)
- Vue Router
- Axios

### 6.2 iOS端(Flutter)

**定位:** 主要使用场景,快速开发

**参考项目:** [Kelivo](https://github.com/Chevey339/kelivo)
- Flutter LLM聊天客户端(748 stars)
- 可复用对话列表、消息气泡、Tool Use展示等UI组件

**核心功能模块:**
1. 对话列表(按来源分组、搜索过滤)
2. 搜索页面(高级过滤、结果高亮)
3. 对话详情(时间线、Tool Use折叠、代码高亮、图片预览)
4. 收藏管理(分类展示、备注编辑)
5. 标签管理(CRUD、快捷打标签)

**技术栈:**
- Flutter 3.x + Dart
- Dio (HTTP客户端)
- Provider (状态管理)
- flutter_markdown (Markdown渲染)
- flutter_highlight (代码高亮)
- cached_network_image (图片缓存)

---

## 七、安全性设计

### 7.1 单用户鉴权

**方案:** 简单Token鉴权

```yaml
# config.yaml
security:
  auth_token: "your-secret-token-here"  # 随机生成的长字符串
```

**实现:**
```go
// 中间件
func AuthMiddleware(token string) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader != "Bearer "+token {
            c.JSON(401, gin.H{"error": "Unauthorized"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

**iOS端存储:**
- Token存储在Keychain
- 每次请求携带Token

### 7.2 HTTPS

**Caddy自动证书:**
```
Caddyfile:
yourdomain.com {
    reverse_proxy localhost:8080
}
```

Caddy自动申请和续期Let's Encrypt证书

---

## 八、性能优化

### 8.1 预期性能指标

| 指标 | 目标 | 说明 |
|------|------|------|
| 搜索延迟 | <100ms | 50万条消息规模 |
| 列表加载 | <50ms | 分页查询 |
| 详情加载 | <30ms | 单条conversation |
| 索引速度 | >1000 docs/s | 全量重建 |
| 并发支持 | 10 req/s | 单用户足够 |

### 8.2 优化策略

**SQLite优化:**
```sql
PRAGMA journal_mode = WAL;        -- 并发读写
PRAGMA synchronous = NORMAL;      -- 平衡性能和安全
PRAGMA cache_size = -64000;       -- 64MB缓存
PRAGMA temp_store = MEMORY;       -- 内存临时表
```

**Bleve优化:**
```go
// 使用BadgerDB作为存储后端
indexConfig := map[string]interface{}{
    "create_if_missing": true,
    "error_if_exists":   false,
}

index, err := bleve.NewUsing(indexPath, mapping,
    scorch.Name, badger.Name, indexConfig)
```

**批量操作优化:**
```go
// 批量索引
batch := index.NewBatch()
for _, msg := range messages {
    batch.Index(msg.UUID, msg)
    if batch.Size() >= 1000 {
        index.Batch(batch)
        batch.Reset()
    }
}
```

---

## 九、开发规范

### 9.1 代码结构约定

```
api-server/
├── cmd/server/main.go          # 入口
├── internal/
│   ├── handler/                # HTTP处理器
│   ├── service/                # 业务逻辑
│   ├── repository/             # 数据访问
│   ├── model/                  # 数据模型
│   ├── middleware/             # 中间件
│   └── config/                 # 配置
├── pkg/                        # 可复用的公共包
│   ├── parser/                 # 复用scripts下的parsers
│   └── utils/
└── go.mod
```

---

## 附录

### A. 参考资料

- [Bleve官方文档](https://blevesearch.com/)
- [Gin框架文档](https://gin-gonic.com/)
- [SQLite文档](https://www.sqlite.org/docs.html)
- [SwiftUI教程](https://developer.apple.com/tutorials/swiftui)
