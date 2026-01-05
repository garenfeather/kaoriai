package claude

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Database 数据库操作封装
type Database struct {
	db *sql.DB
}

// NewDatabase 创建数据库连接
func NewDatabase(dbPath string) (*Database, error) {
	// 确保数据目录存在
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	// 检查数据库文件是否存在
	dbExists := false
	if _, err := os.Stat(dbPath); err == nil {
		dbExists = true
	}

	// 打开数据库连接
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// 如果数据库不存在，需要初始化
	if !dbExists {
		if err := initDatabase(db); err != nil {
			db.Close()
			return nil, fmt.Errorf("init database: %w", err)
		}
		log.Println("Database initialized successfully")
	}

	// 配置数据库
	_, _ = db.Exec("PRAGMA foreign_keys = ON")
	_, _ = db.Exec("PRAGMA journal_mode = WAL")
	_, _ = db.Exec("PRAGMA synchronous = NORMAL")

	return &Database{db: db}, nil
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	return d.db.Close()
}

// BeginTx 开始事务
func (d *Database) BeginTx() (*sql.Tx, error) {
	return d.db.Begin()
}

// InsertConversationTx 在事务中插入对话
func (d *Database) InsertConversationTx(tx *sql.Tx, uuid, sourceType, title, metadata string, createdAt, updatedAt time.Time) error {
	query := `
		INSERT INTO conversations (uuid, source_type, title, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(uuid) DO UPDATE SET
			title = excluded.title,
			metadata = excluded.metadata,
			updated_at = excluded.updated_at
	`
	_, err := tx.Exec(query, uuid, sourceType, title, metadata, createdAt, updatedAt)
	return err
}

// InsertMessageTx 在事务中插入消息
func (d *Database) InsertMessageTx(tx *sql.Tx, uuid, conversationUUID, parentUUID string, roundIndex int,
	role, contentType, content string, createdAt time.Time) error {
	query := `
		INSERT INTO messages (uuid, conversation_uuid, parent_uuid, round_index, role, content_type, content, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(uuid) DO UPDATE SET
			parent_uuid = excluded.parent_uuid,
			round_index = excluded.round_index,
			content = excluded.content
	`
	_, err := tx.Exec(query, uuid, conversationUUID, parentUUID, roundIndex, role, contentType, content, createdAt)
	return err
}

// initDatabase 初始化数据库表结构
func initDatabase(db *sql.DB) error {
	// 尝试多个可能的路径
	possiblePaths := []string{
		"scripts/init_database.sql",                  // 从项目根目录运行
		"../../scripts/init_database.sql",            // 从 workers/claude 目录运行
		"../../../scripts/init_database.sql",         // 从 workers/claude/cmd 目录运行
	}

	var sqlBytes []byte
	var err error
	var foundPath string

	for _, path := range possiblePaths {
		sqlBytes, err = os.ReadFile(path)
		if err == nil {
			foundPath = path
			break
		}
	}

	if foundPath == "" {
		return fmt.Errorf("init sql file not found, tried paths: %v, last error: %w", possiblePaths, err)
	}

	log.Printf("Using init SQL from: %s", foundPath)

	// 执行SQL
	if _, err := db.Exec(string(sqlBytes)); err != nil {
		return fmt.Errorf("exec init sql: %w", err)
	}

	return nil
}

// ContentJSON 消息内容JSON结构
type ContentJSON struct {
	Text string `json:"text,omitempty"`
}

// BuildContentJSON 构建content JSON字符串
func BuildContentJSON(text string) (string, error) {
	content := ContentJSON{
		Text: text,
	}

	data, err := json.Marshal(content)
	if err != nil {
		return "", fmt.Errorf("marshal content: %w", err)
	}

	return string(data), nil
}
