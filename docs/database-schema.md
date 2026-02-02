# 数据库设计文档

## 一、概述

### 1.1 数据库选型

**SQLite 3.35+**

选型理由:
- 嵌入式数据库,无需独立服务
- 单用户场景,无并发写入压力
- 数据规模<5GB,性能足够
- 备份简单(文件复制)
- WAL模式支持并发读

### 1.2 设计原则

1. **数据完整性**: 使用外键约束保证引用完整性
2. **数据隐藏**: 所有表使用hidden_at字段,便于数据恢复和隐藏
3. **冗余字段**: conversation_uuid在统计/视图中可冗余,便于快速查询
4. **JSON存储**: 复杂结构(如tool_use)使用JSON,灵活扩展
5. **时间戳**: 所有表包含created_at,重要表包含updated_at

---

## 二、核心数据表

### 2.1 conversations - 对话表

**用途:** 存储对话元数据

```sql
CREATE TABLE conversations (
    uuid TEXT PRIMARY KEY,                      -- 对话唯一标识(来源系统UUID)
    source_type TEXT NOT NULL,                  -- 数据来源: gpt|claude|claude_code|codex|gemini|gemini_cli
    title TEXT DEFAULT '',                      -- 对话标题
    metadata TEXT,                              -- JSON格式元数据

    -- 时间戳
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    hidden_at DATETIME                          -- 隐藏时间(隐藏后不在界面展示)
);

-- 索引
CREATE INDEX idx_conv_source ON conversations(source_type, created_at);
CREATE INDEX idx_conv_created ON conversations(created_at);
CREATE INDEX idx_conv_hidden ON conversations(hidden_at);
```

**metadata字段结构(JSON):**
```json
{
  "project_name": "learn-itv",
  "cwd": "/Users/xxx/projects/learn-itv",
  "git_branch": "main",
  "git_commit": "abc123",
  "session_id": "original-session-id",
  "duration_minutes": 45,
  "message_count": 128,
  "custom": {
    "key": "value"
  }
}
```

**说明:**
- `uuid`: 作为主键,来源系统的原始ID
- `source_type`: 枚举值,便于按来源过滤
- `hidden_at`: NULL表示未隐藏,NOT NULL表示已隐藏
- **分支树功能移至独立表conversation_trees（单表JSON存储）**

---

### 2.1.1 conversation_trees - 对话分支树表

**用途:** 管理对话分支树

```sql
CREATE TABLE conversation_trees (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tree_id TEXT NOT NULL UNIQUE,             -- 树唯一标识(UUID)
    title TEXT DEFAULT '',                     -- 树标题
    description TEXT,                          -- 树描述
    tree_data TEXT NOT NULL,                   -- JSON格式树结构
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tree_id ON conversation_trees(tree_id);
```

**tree_data字段结构(JSON):**
```json
{
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
```

**说明:**
- 单表设计，树结构直接存JSON
- tree_data包含完整的树层级结构
- 简单直接，无需复杂JOIN

---

### 2.2 messages - 消息表

**用途:** 存储对话中的每条消息

```sql
CREATE TABLE messages (
    uuid TEXT PRIMARY KEY,                      -- 消息唯一标识
    conversation_uuid TEXT NOT NULL,            -- 所属对话UUID
    parent_uuid TEXT DEFAULT '',                -- 父消息UUID(用于构建对话链)

    -- 消息定位
    round_index INTEGER NOT NULL,                -- round序号(从1开始)

    -- 消息属性
    role TEXT NOT NULL,                         -- user|assistant|system
    content_type TEXT NOT NULL,                 -- text|tool_use|tool_result|multipart
    content TEXT NOT NULL,                      -- JSON格式完整内容
    thinking TEXT,                              -- 思考过程(仅Gemini等支持,assistant消息)
    model TEXT,                                 -- 模型名称(如gemini-1.5-pro、gpt-4o等)

    -- 时间戳
    created_at DATETIME NOT NULL,
    hidden_at DATETIME,                         -- 屏蔽时间(屏蔽后不在界面展示)

    FOREIGN KEY (conversation_uuid) REFERENCES conversations(uuid) ON DELETE CASCADE
);

-- 索引
CREATE INDEX idx_msg_conv_round ON messages(conversation_uuid, round_index);
CREATE INDEX idx_msg_parent ON messages(parent_uuid);
CREATE INDEX idx_msg_role ON messages(role);
CREATE INDEX idx_msg_created ON messages(created_at);
CREATE INDEX idx_msg_hidden ON messages(hidden_at);
```

**content字段结构(JSON):**

**情况1: 纯文本消息(content_type=text)**
```json
{
  "type": "text",
  "text": "帮我设计一个监控方案"
}
```

**情况2: Tool Use消息(content_type=tool_use)**
```json
{
  "type": "tool_use",
  "tool_name": "Read",
  "tool_input": {
    "file_path": "/Users/xxx/main.go"
  },
  "tool_output": {
    "success": true,
    "content": "package main\n..."
  }
}
```

**情况3: 多部分消息(content_type=multipart)**
```json
{
  "type": "multipart",
  "parts": [
    {
      "type": "text",
      "text": "让我先查看项目结构"
    },
    {
      "type": "tool_use",
      "name": "Glob",
      "input": {"pattern": "**/*.go"}
    },
    {
      "type": "text",
      "text": "发现了以下文件..."
    }
  ]
}
```

**情况4: 包含图片(GPT, content_type=text/multipart)**
```json
{
  "text": "这是一张架构图，请帮我分析",
  "images": ["file-abc123xyz-sanitized", "file-def456"]
}
```

**情况5: 包含视频(Gemini, content_type=text/multipart/video)**
```json
{
  "text": "你觉得她唱歌时位置偏低还是偏高",
  "videos": ["c8d0a41aba76e19e-nadojoa22 2026-01-04T095204.mp4"]
}
```

**情况6: 同时包含图片和视频(Gemini)**
```json
{
  "text": "分析这些素材",
  "images": ["image1.jpg", "image2.png"],
  "videos": ["conversation-id-video1.mov", "conversation-id-video2.mp4"]
}
```

**说明:**
- `uuid`: 作为主键,消息唯一标识
- `conversation_uuid`: 外键关联conversations表
- `round_index` (round序号): 一问一答算一轮,便于查询上下文
- `parent_uuid`: 支持分支对话,构建对话树
- `content_type`: 实际存在的类型为 text | tool_use | tool_result | multipart | image | video
- `content`: 简化后的JSON格式，包含text、images、videos字段；全文索引用的纯文本在索引阶段从content实时提取（不落库）
- `thinking`: 模型思考过程的纯文本记录，仅部分来源（如Gemini）的assistant消息包含此字段，NULL表示无thinking数据
- `model`: 生成该消息的模型名称（如gemini-1.5-pro、gemini-2.0-flash-exp、gpt-4o等），主要用于assistant消息，便于按模型统计和筛选，NULL表示未记录模型信息
- `hidden_at`: 隐藏功能,NULL表示未隐藏,NOT NULL表示已隐藏

**多媒体文件处理:**
- **图片** (GPT/Gemini支持):
  - 文件存储在 `data/images/` 目录
  - content JSON中的 `images` 字段存储文件名数组
  - 文件名格式: `{image_id}.jpg` 或从URL提取的名称
- **视频** (仅Gemini支持):
  - 文件存储在 `data/videos/` 目录
  - content JSON中的 `videos` 字段存储文件名数组
  - 文件名格式: `{conversation_id}-{original_filename}`
- 所有来源的图片统一存放在 `data/images/`，视频统一存放在 `data/videos/`
- 不需要独立的images/videos表

---

## 三、功能数据表

### 3.1 favorites - 收藏表

**用途:** 收藏conversation、round、message或fragment

```sql
CREATE TABLE favorites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- 收藏目标
    target_type TEXT NOT NULL,                  -- conversation | round | message | fragment
    target_id TEXT NOT NULL,                    -- 目标对象ID

    -- 收藏属性
    category TEXT DEFAULT 'default',            -- 收藏分类
    notes TEXT,                                 -- 备注说明

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_fav_type_id ON favorites(target_type, target_id);
CREATE INDEX idx_fav_category ON favorites(category, created_at);
CREATE INDEX idx_fav_type ON favorites(target_type, created_at);
```

**说明:**
- 统一使用target_id字段存储收藏对象的ID
- target_type标明收藏类型，便于索引和回源
- 支持conversation、round、message、fragment四种类型的收藏
- 简化设计，便于扩展

**category建议值:**
- `default`: 默认收藏
- `tech_solution`: 技术方案
- `inspiration`: 灵感
- `reference`: 参考
- 用户可自定义

---

### 3.2 fragments - 片段表

**用途:** 存储从消息中提取的代码片段、关键信息等

```sql
CREATE TABLE fragments (
    uuid TEXT PRIMARY KEY,                      -- 片段唯一标识
    conversation_uuid TEXT NOT NULL,            -- 所属对话UUID
    message_uuid TEXT NOT NULL,                 -- 来源消息UUID

    -- 片段属性
    fragment_type TEXT NOT NULL,                -- 片段类型: code|text|table|image
    content TEXT NOT NULL,                      -- 片段内容
    language TEXT,                              -- 编程语言(仅code类型)
    start_line INTEGER,                         -- 起始行号(仅code类型)
    end_line INTEGER,                           -- 结束行号(仅code类型)

    -- 时间戳
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    hidden_at DATETIME,                         -- 隐藏时间

    FOREIGN KEY (conversation_uuid) REFERENCES conversations(uuid) ON DELETE CASCADE,
    FOREIGN KEY (message_uuid) REFERENCES messages(uuid) ON DELETE CASCADE
);

-- 索引
CREATE INDEX idx_frag_conv ON fragments(conversation_uuid);
CREATE INDEX idx_frag_msg ON fragments(message_uuid);
CREATE INDEX idx_frag_type ON fragments(fragment_type);
CREATE INDEX idx_frag_hidden ON fragments(hidden_at);
```

**说明:**
- 用于存储从消息中提取的细粒度内容
- 支持代码片段、文本片段、表格、图片等类型
- 便于用户收藏和管理特定内容
- 与收藏表配合使用，实现多粒度收藏功能

---

### 3.3 tags - 标签表

**用途:** 标签基础数据

```sql
CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,                  -- 标签名称
    color TEXT DEFAULT '#3B82F6',               -- 标签颜色(HEX格式)
    usage_count INTEGER DEFAULT 0,              -- 使用次数(冗余字段,便于排序)
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_tag_usage ON tags(usage_count DESC);
```

**预设标签建议:**
```sql
INSERT INTO tags (name, color) VALUES
('监控', '#3B82F6'),
('数据库', '#10B981'),
('前端', '#F59E0B'),
('后端', '#EF4444'),
('架构', '#8B5CF6'),
('性能优化', '#EC4899'),
('Bug修复', '#DC2626'),
('需求讨论', '#6366F1');
```

---

### 3.4 conversation_tags - 对话标签关联表

**用途:** conversation与tags的多对多关联

```sql
CREATE TABLE conversation_tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tag_id INTEGER NOT NULL,
    conversation_uuid TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(tag_id, conversation_uuid),
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_uuid) REFERENCES conversations(uuid) ON DELETE CASCADE
);

-- 索引
CREATE INDEX idx_conv_tag_tag ON conversation_tags(tag_id, created_at);
CREATE INDEX idx_conv_tag_conv ON conversation_tags(conversation_uuid);
```

**触发器(自动维护usage_count):**
```sql
-- 打标签时增加计数
CREATE TRIGGER trg_tag_usage_inc AFTER INSERT ON conversation_tags
BEGIN
    UPDATE tags SET usage_count = usage_count + 1
    WHERE id = NEW.tag_id;
END;

-- 删除标签时减少计数
CREATE TRIGGER trg_tag_usage_dec AFTER DELETE ON conversation_tags
BEGIN
    UPDATE tags SET usage_count = usage_count - 1
    WHERE id = OLD.tag_id;
END;
```

**说明:**
- 简化为只支持对conversation打标签
- 表名改为conversation_tags更清晰
- message/fragment级别的标签暂不支持

---

## 四、视图定义

### 4.1 conversation_stats_view - 对话统计视图

**用途:** 快速查询对话的统计信息

```sql
CREATE VIEW conversation_stats_view AS
SELECT
    c.uuid,
    c.title,
    c.source_type,
    c.created_at,
    COUNT(DISTINCT m.uuid) as message_count,
    COUNT(DISTINCT CASE WHEN m.role = 'user' THEN m.uuid END) as user_message_count,
    COUNT(DISTINCT CASE WHEN m.role = 'assistant' THEN m.uuid END) as assistant_message_count,
    MAX(m.round_index) as max_round_index,
    MIN(m.created_at) as first_message_at,
    MAX(m.created_at) as last_message_at
FROM conversations c
LEFT JOIN messages m ON c.uuid = m.conversation_uuid AND m.hidden_at IS NULL
WHERE c.hidden_at IS NULL
GROUP BY c.uuid;
```

**使用场景:**
```sql
-- 查询对话列表(带统计信息)
SELECT * FROM conversation_stats_view
ORDER BY last_message_at DESC
LIMIT 20;
```

---

### 4.2 message_with_context_view - 消息上下文视图

**用途:** 查询消息时附带对话信息

```sql
CREATE VIEW message_with_context_view AS
SELECT
    m.*,
    c.title as conversation_title,
    c.source_type,
    c.metadata as conversation_metadata
FROM messages m
JOIN conversations c ON m.conversation_uuid = c.uuid
WHERE m.hidden_at IS NULL AND c.hidden_at IS NULL;
```

**使用场景:**
```sql
-- 搜索结果查询(带对话上下文)
SELECT * FROM message_with_context_view
WHERE uuid IN (?, ?, ...);
```

---

## 五、初始化脚本

### 5.1 完整建表脚本

```sql
-- ============================================
-- AI对话数据管理系统 - 数据库初始化脚本
-- SQLite 3.35+
-- ============================================

-- 启用外键约束
PRAGMA foreign_keys = ON;

-- 性能优化配置
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -64000;
PRAGMA temp_store = MEMORY;

-- ============================================
-- 核心表
-- ============================================

-- 1. 对话表
CREATE TABLE IF NOT EXISTS conversations (
    uuid TEXT PRIMARY KEY,
    source_type TEXT NOT NULL,
    title TEXT DEFAULT '',
    metadata TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    hidden_at DATETIME
);

CREATE INDEX idx_conv_source ON conversations(source_type, created_at);
CREATE INDEX idx_conv_created ON conversations(created_at);
CREATE INDEX idx_conv_hidden ON conversations(hidden_at);

CREATE TABLE IF NOT EXISTS conversation_trees (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tree_id TEXT NOT NULL UNIQUE,             -- 树唯一标识(UUID)
    title TEXT DEFAULT '',                    -- 树标题
    description TEXT,                         -- 树描述
    tree_data TEXT NOT NULL,                  -- JSON格式树结构,结构与上文示例一致
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tree_id ON conversation_trees(tree_id);

-- 3. 消息表
CREATE TABLE IF NOT EXISTS messages (
    uuid TEXT PRIMARY KEY,
    conversation_uuid TEXT NOT NULL,
    parent_uuid TEXT DEFAULT '',
    round_index INTEGER NOT NULL,                -- round序号(从1开始)
    role TEXT NOT NULL,
    content_type TEXT NOT NULL,
    content TEXT NOT NULL,
    thinking TEXT,
    model TEXT,
    created_at DATETIME NOT NULL,
    hidden_at DATETIME,
    FOREIGN KEY (conversation_uuid) REFERENCES conversations(uuid) ON DELETE CASCADE
);

CREATE INDEX idx_msg_conv_round ON messages(conversation_uuid, round_index);
CREATE INDEX idx_msg_parent ON messages(parent_uuid);
CREATE INDEX idx_msg_role ON messages(role);
CREATE INDEX idx_msg_created ON messages(created_at);
CREATE INDEX idx_msg_hidden ON messages(hidden_at);

-- ============================================
-- 功能表
-- ============================================

-- 4. 收藏表
CREATE TABLE IF NOT EXISTS favorites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    category TEXT DEFAULT 'default',
    notes TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_fav_type_id ON favorites(target_type, target_id);
CREATE INDEX idx_fav_category ON favorites(category, created_at);
CREATE INDEX idx_fav_type ON favorites(target_type, created_at);

-- 5. 片段表
CREATE TABLE IF NOT EXISTS fragments (
    uuid TEXT PRIMARY KEY,
    conversation_uuid TEXT NOT NULL,
    message_uuid TEXT NOT NULL,
    fragment_type TEXT NOT NULL,
    content TEXT NOT NULL,
    language TEXT,
    start_line INTEGER,
    end_line INTEGER,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    hidden_at DATETIME,
    FOREIGN KEY (conversation_uuid) REFERENCES conversations(uuid) ON DELETE CASCADE,
    FOREIGN KEY (message_uuid) REFERENCES messages(uuid) ON DELETE CASCADE
);

CREATE INDEX idx_frag_conv ON fragments(conversation_uuid);
CREATE INDEX idx_frag_msg ON fragments(message_uuid);
CREATE INDEX idx_frag_type ON fragments(fragment_type);
CREATE INDEX idx_frag_hidden ON fragments(hidden_at);

-- 6. 标签表
CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    color TEXT DEFAULT '#3B82F6',
    usage_count INTEGER DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tag_usage ON tags(usage_count DESC);

-- 7. 对话标签关联表
CREATE TABLE IF NOT EXISTS conversation_tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tag_id INTEGER NOT NULL,
    conversation_uuid TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tag_id, conversation_uuid),
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_uuid) REFERENCES conversations(uuid) ON DELETE CASCADE
);

CREATE INDEX idx_conv_tag_tag ON conversation_tags(tag_id, created_at);
CREATE INDEX idx_conv_tag_conv ON conversation_tags(conversation_uuid);

-- ============================================
-- 触发器
-- ============================================

-- 标签使用次数自动维护
CREATE TRIGGER IF NOT EXISTS trg_tag_usage_inc AFTER INSERT ON conversation_tags
BEGIN
    UPDATE tags SET usage_count = usage_count + 1
    WHERE id = NEW.tag_id;
END;

CREATE TRIGGER IF NOT EXISTS trg_tag_usage_dec AFTER DELETE ON conversation_tags
BEGIN
    UPDATE tags SET usage_count = usage_count - 1
    WHERE id = OLD.tag_id;
END;

-- 对话更新时间自动维护
CREATE TRIGGER IF NOT EXISTS trg_conv_updated_at AFTER UPDATE ON conversations
WHEN OLD.updated_at = NEW.updated_at
BEGIN
    UPDATE conversations SET updated_at = CURRENT_TIMESTAMP
    WHERE uuid = NEW.uuid;
END;

-- ============================================
-- 视图
-- ============================================

CREATE VIEW IF NOT EXISTS conversation_stats_view AS
SELECT
    c.uuid,
    c.title,
    c.source_type,
    c.created_at,
    COUNT(DISTINCT m.uuid) as message_count,
    COUNT(DISTINCT CASE WHEN m.role = 'user' THEN m.uuid END) as user_message_count,
    COUNT(DISTINCT CASE WHEN m.role = 'assistant' THEN m.uuid END) as assistant_message_count,
    MAX(m.round_index) as max_round_index,
    MIN(m.created_at) as first_message_at,
    MAX(m.created_at) as last_message_at
FROM conversations c
LEFT JOIN messages m ON c.uuid = m.conversation_uuid AND m.hidden_at IS NULL
WHERE c.hidden_at IS NULL
GROUP BY c.uuid;

CREATE VIEW IF NOT EXISTS message_with_context_view AS
SELECT
    m.*,
    c.title as conversation_title,
    c.source_type,
    c.metadata as conversation_metadata
FROM messages m
JOIN conversations c ON m.conversation_uuid = c.uuid
WHERE m.hidden_at IS NULL AND c.hidden_at IS NULL;

-- ============================================
-- 初始数据
-- ============================================

-- 预设标签
INSERT OR IGNORE INTO tags (name, color) VALUES
('监控', '#3B82F6'),
('数据库', '#10B981'),
('前端', '#F59E0B'),
('后端', '#EF4444'),
('架构', '#8B5CF6'),
('性能优化', '#EC4899'),
('Bug修复', '#DC2626'),
('需求讨论', '#6366F1');
```

---

## 七、常用查询示例

### 7.1 对话查询

```sql
-- 查询对话列表(带统计)
SELECT * FROM conversation_stats_view
WHERE source_type = 'gpt'
ORDER BY last_message_at DESC
LIMIT 20 OFFSET 0;

-- 查询对话详情
SELECT * FROM conversations
WHERE uuid = ?;

-- 查询对话的所有消息(时间线)
SELECT * FROM messages
WHERE conversation_uuid = ?
  AND hidden_at IS NULL
ORDER BY round_index, created_at;

```

### 7.1.1 分支树查询

```sql
-- 查询所有分支树列表
SELECT tree_id, title, description, created_at
FROM conversation_trees
ORDER BY created_at DESC;

-- 查询特定树的完整数据（包含JSON树结构）
SELECT * FROM conversation_trees
WHERE tree_id = ?;

-- 前端直接解析tree_data的JSON获取树结构
```

### 7.2 消息查询

```sql
-- 查询消息详情(带对话信息)
SELECT * FROM message_with_context_view
WHERE uuid = ?;

-- 查询消息上下文(前后N条)
SELECT * FROM messages
WHERE conversation_uuid = (
    SELECT conversation_uuid FROM messages WHERE uuid = ?
)
AND round_index BETWEEN
    (SELECT round_index - 2 FROM messages WHERE uuid = ?) AND
    (SELECT round_index + 2 FROM messages WHERE uuid = ?)
AND hidden_at IS NULL
ORDER BY round_index, created_at;

-- 根据UUIDs批量查询消息
SELECT * FROM message_with_context_view
WHERE uuid IN (?, ?, ?, ...);
```

### 7.3 收藏查询

```sql
-- 查询所有收藏
SELECT
    f.*,
    CASE f.target_type
        WHEN 'conversation' THEN (
            SELECT title FROM conversations WHERE uuid = f.target_id AND hidden_at IS NULL
        )
        WHEN 'round' THEN (
            SELECT 'Round ' || round_index FROM messages
            WHERE conversation_uuid || '-' || round_index = f.target_id
            AND hidden_at IS NULL LIMIT 1
        )
        WHEN 'message' THEN (
            SELECT ''  -- 预览需在应用层解析content生成摘要
        )
        WHEN 'fragment' THEN (
            SELECT SUBSTR(content, 1, 100) || '...' FROM fragments WHERE uuid = f.target_id AND hidden_at IS NULL
        )
        ELSE NULL
    END as target_preview
FROM favorites f
ORDER BY f.created_at DESC;

-- 按分类查询收藏
SELECT * FROM favorites
WHERE category = 'tech_solution'
ORDER BY created_at DESC;
```

### 7.4 标签查询

```sql
-- 查询热门标签
SELECT * FROM tags
ORDER BY usage_count DESC
LIMIT 20;

-- 查询某标签下的所有对话
SELECT
    c.*,
    ct.created_at as tagged_at
FROM conversations c
JOIN conversation_tags ct ON c.uuid = ct.conversation_uuid
WHERE ct.tag_id = ?
ORDER BY ct.created_at DESC;

-- 查询某对话的所有标签
SELECT t.* FROM tags t
JOIN conversation_tags ct ON t.id = ct.tag_id
WHERE ct.conversation_uuid = ?;
```

### 7.5 统计查询

```sql
-- 按来源统计对话数
SELECT source_type, COUNT(*) as count
FROM conversations
WHERE hidden_at IS NULL
GROUP BY source_type;

-- 按日期统计消息数
SELECT DATE(created_at) as date, COUNT(*) as count
FROM messages
WHERE hidden_at IS NULL
GROUP BY DATE(created_at)
ORDER BY date DESC
LIMIT 30;

-- 最活跃的对话(消息数最多)
SELECT * FROM conversation_stats_view
ORDER BY message_count DESC
LIMIT 10;
```

---

## 八、注意事项

### 8.1 并发控制

SQLite的WAL模式支持:
- ✅ 多个读取并发
- ✅ 单个写入 + 多个读取并发
- ❌ 多个写入并发

### 8.2 JSON字段查询

SQLite 3.38+支持JSON函数:
```sql
-- 查询metadata中的project_name
SELECT * FROM conversations
WHERE json_extract(metadata, '$.project_name') = 'learn-itv';

-- 查询content中的tool_name
SELECT * FROM messages
WHERE json_extract(content, '$.tool_name') = 'Read';
```
