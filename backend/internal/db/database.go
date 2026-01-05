// Package db 提供数据库连接和初始化功能
package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// InitDB 初始化数据库连接并执行初始化脚本
// 参数:
//   - dbPath: 数据库文件路径
// 返回:
//   - *sql.DB: 数据库连接
//   - error: 错误信息
func InitDB(dbPath string) (*sql.DB, error) {
	// 打开 SQLite 数据库
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(1) // SQLite 建议单连接
	db.SetMaxIdleConns(1)

	// 执行数据库初始化
	if err := initSchema(db); err != nil {
		return nil, fmt.Errorf("初始化数据库 schema 失败: %w", err)
	}

	log.Printf("数据库初始化成功: %s", dbPath)
	return db, nil
}

// initSchema 初始化数据库表结构
func initSchema(db *sql.DB) error {
	// 启用外键约束
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return err
	}

	// 性能优化配置
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA cache_size = -64000;",
		"PRAGMA temp_store = MEMORY;",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("执行 PRAGMA 失败 [%s]: %w", pragma, err)
		}
	}

	// 执行建表脚本
	schema := `
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

CREATE INDEX IF NOT EXISTS idx_conv_source ON conversations(source_type, created_at);
CREATE INDEX IF NOT EXISTS idx_conv_created ON conversations(created_at);
CREATE INDEX IF NOT EXISTS idx_conv_hidden ON conversations(hidden_at);

-- 2. 对话树表
CREATE TABLE IF NOT EXISTS conversation_trees (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tree_id TEXT NOT NULL UNIQUE,
    title TEXT DEFAULT '',
    description TEXT,
    tree_data TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tree_id ON conversation_trees(tree_id);

-- 3. 消息表
CREATE TABLE IF NOT EXISTS messages (
    uuid TEXT PRIMARY KEY,
    conversation_uuid TEXT NOT NULL,
    parent_uuid TEXT DEFAULT '',
    round_index INTEGER NOT NULL,
    role TEXT NOT NULL,
    content_type TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    hidden_at DATETIME,
    FOREIGN KEY (conversation_uuid) REFERENCES conversations(uuid) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_msg_conv_round ON messages(conversation_uuid, round_index);
CREATE INDEX IF NOT EXISTS idx_msg_parent ON messages(parent_uuid);
CREATE INDEX IF NOT EXISTS idx_msg_role ON messages(role);
CREATE INDEX IF NOT EXISTS idx_msg_created ON messages(created_at);
CREATE INDEX IF NOT EXISTS idx_msg_hidden ON messages(hidden_at);

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

CREATE INDEX IF NOT EXISTS idx_fav_type_id ON favorites(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_fav_category ON favorites(category, created_at);
CREATE INDEX IF NOT EXISTS idx_fav_type ON favorites(target_type, created_at);

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

CREATE INDEX IF NOT EXISTS idx_frag_conv ON fragments(conversation_uuid);
CREATE INDEX IF NOT EXISTS idx_frag_msg ON fragments(message_uuid);
CREATE INDEX IF NOT EXISTS idx_frag_type ON fragments(fragment_type);
CREATE INDEX IF NOT EXISTS idx_frag_hidden ON fragments(hidden_at);

-- 6. 标签表
CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    color TEXT DEFAULT '#3B82F6',
    usage_count INTEGER DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tag_usage ON tags(usage_count DESC);

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

CREATE INDEX IF NOT EXISTS idx_conv_tag_tag ON conversation_tags(tag_id, created_at);
CREATE INDEX IF NOT EXISTS idx_conv_tag_conv ON conversation_tags(conversation_uuid);

-- ============================================
-- 触发器
-- ============================================

-- 标签使用次数自动维护
DROP TRIGGER IF EXISTS trg_tag_usage_inc;
CREATE TRIGGER trg_tag_usage_inc AFTER INSERT ON conversation_tags
BEGIN
    UPDATE tags SET usage_count = usage_count + 1
    WHERE id = NEW.tag_id;
END;

DROP TRIGGER IF EXISTS trg_tag_usage_dec;
CREATE TRIGGER trg_tag_usage_dec AFTER DELETE ON conversation_tags
BEGIN
    UPDATE tags SET usage_count = usage_count - 1
    WHERE id = OLD.tag_id;
END;

-- 对话更新时间自动维护
DROP TRIGGER IF EXISTS trg_conv_updated_at;
CREATE TRIGGER trg_conv_updated_at AFTER UPDATE ON conversations
WHEN OLD.updated_at = NEW.updated_at
BEGIN
    UPDATE conversations SET updated_at = CURRENT_TIMESTAMP
    WHERE uuid = NEW.uuid;
END;

-- ============================================
-- 视图
-- ============================================

DROP VIEW IF EXISTS conversation_stats_view;
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

DROP VIEW IF EXISTS message_with_context_view;
CREATE VIEW message_with_context_view AS
SELECT
    m.*,
    c.title as conversation_title,
    c.source_type,
    c.metadata as conversation_metadata
FROM messages m
JOIN conversations c ON m.conversation_uuid = c.uuid
WHERE m.hidden_at IS NULL AND c.hidden_at IS NULL;
`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("执行建表脚本失败: %w", err)
	}

	// 插入预设标签
	insertTags := `
INSERT OR IGNORE INTO tags (name, color) VALUES
('监控', '#3B82F6'),
('数据库', '#10B981'),
('前端', '#F59E0B'),
('后端', '#EF4444'),
('架构', '#8B5CF6'),
('性能优化', '#EC4899'),
('Bug修复', '#DC2626'),
('需求讨论', '#6366F1');
`
	if _, err := db.Exec(insertTags); err != nil {
		return fmt.Errorf("插入预设标签失败: %w", err)
	}

	return nil
}
