// Package model 定义数据库模型和数据结构
package model

import (
	"database/sql"
	"time"
)

// Conversation 对话模型
type Conversation struct {
	UUID       string         `json:"uuid"`        // 对话唯一标识
	SourceType string         `json:"source_type"` // 数据来源
	Title      string         `json:"title"`       // 对话标题
	Metadata   sql.NullString `json:"metadata"`    // JSON格式元数据
	CreatedAt  time.Time      `json:"created_at"`  // 创建时间
	UpdatedAt  time.Time      `json:"updated_at"`  // 更新时间
	HiddenAt   sql.NullTime   `json:"hidden_at"`   // 隐藏时间
}

// Message 消息模型
type Message struct {
	UUID             string         `json:"uuid"`              // 消息唯一标识
	ConversationUUID string         `json:"conversation_uuid"` // 所属对话UUID
	ParentUUID       string         `json:"parent_uuid"`       // 父消息UUID
	RoundIndex       int            `json:"round_index"`       // round序号
	Role             string         `json:"role"`              // user|assistant|system
	ContentType      string         `json:"content_type"`      // text|tool_use|tool_result|multipart
	Content          string         `json:"content"`           // JSON格式完整内容
	Thinking         sql.NullString `json:"thinking"`          // 思考过程(仅部分来源如Gemini)
	Model            sql.NullString `json:"model"`             // 模型名称(如gemini-1.5-pro)
	CreatedAt        time.Time      `json:"created_at"`        // 创建时间
	HiddenAt         sql.NullTime   `json:"hidden_at"`         // 隐藏时间
}

// ConversationTree 对话树模型
type ConversationTree struct {
	ID          int       `json:"id"`           // 自增ID
	TreeID      string    `json:"tree_id"`      // 树唯一标识
	Title       string    `json:"title"`        // 树标题
	Description string    `json:"description"`  // 树描述
	TreeData    string    `json:"tree_data"`    // JSON格式树结构
	CreatedAt   time.Time `json:"created_at"`   // 创建时间
	UpdatedAt   time.Time `json:"updated_at"`   // 更新时间
}

// Favorite 收藏模型
type Favorite struct {
	ID         int       `json:"id"`          // 自增ID
	TargetType string    `json:"target_type"` // conversation|round|message|fragment
	TargetID   string    `json:"target_id"`   // 目标对象ID
	Category   string    `json:"category"`    // 收藏分类
	Notes      string    `json:"notes"`       // 备注说明
	CreatedAt  time.Time `json:"created_at"`  // 创建时间
}

// Tag 标签模型
type Tag struct {
	ID         int       `json:"id"`          // 自增ID
	Name       string    `json:"name"`        // 标签名称
	Color      string    `json:"color"`       // 标签颜色
	UsageCount int       `json:"usage_count"` // 使用次数
	CreatedAt  time.Time `json:"created_at"`  // 创建时间
}

// ConversationTag 对话标签关联模型
type ConversationTag struct {
	ID               int       `json:"id"`                // 自增ID
	TagID            int       `json:"tag_id"`            // 标签ID
	ConversationUUID string    `json:"conversation_uuid"` // 对话UUID
	CreatedAt        time.Time `json:"created_at"`        // 创建时间
}

// Fragment 片段模型
type Fragment struct {
	UUID             string         `json:"uuid"`              // 片段唯一标识
	ConversationUUID string         `json:"conversation_uuid"` // 所属对话UUID
	MessageUUID      string         `json:"message_uuid"`      // 来源消息UUID
	FragmentType     string         `json:"fragment_type"`     // code|text|table|image
	Content          string         `json:"content"`           // 片段内容
	Language         sql.NullString `json:"language"`          // 编程语言(仅code类型)
	StartLine        sql.NullInt32  `json:"start_line"`        // 起始行号
	EndLine          sql.NullInt32  `json:"end_line"`          // 结束行号
	CreatedAt        time.Time      `json:"created_at"`        // 创建时间
	HiddenAt         sql.NullTime   `json:"hidden_at"`         // 隐藏时间
}

// ConversationStats 对话统计信息(用于列表展示)
type ConversationStats struct {
	UUID                 string    `json:"uuid"`
	Title                string    `json:"title"`
	SourceType           string    `json:"source_type"`
	CreatedAt            time.Time `json:"created_at"`
	MessageCount         int       `json:"message_count"`
	UserMessageCount     int       `json:"user_message_count"`
	AssistantMessageCount int      `json:"assistant_message_count"`
	MaxRoundIndex        int       `json:"max_round_index"`
	FirstMessageAt       sql.NullTime `json:"first_message_at"`
	LastMessageAt        sql.NullTime `json:"last_message_at"`
}

// MessageWithContext 带上下文的消息(用于搜索结果)
type MessageWithContext struct {
	Message
	ConversationTitle    string `json:"conversation_title"`
	ConversationMetadata string `json:"conversation_metadata"`
}

// StatsOverview 概览统计
type StatsOverview struct {
	TotalConversations int            `json:"total_conversations"`
	TotalMessages      int            `json:"total_messages"`
	Sources            map[string]int `json:"sources"`
}

// StatsByDate 按日期统计
type StatsByDate struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}
