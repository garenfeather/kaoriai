// Package repository 实现数据访问层
package repository

import (
	"database/sql"
	"fmt"

	"gpt-tools/backend/internal/model"
)

// ConversationRepository 对话数据访问接口
type ConversationRepository struct {
	db *sql.DB
}

// NewConversationRepository 创建对话仓库实例
func NewConversationRepository(db *sql.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

// ListConversations 获取对话列表(带分页和过滤)
func (r *ConversationRepository) ListConversations(sourceType string, dateFrom, dateTo string, page, pageSize int) ([]model.ConversationStats, int, error) {
	// 构建查询条件
	query := `
		SELECT
			uuid, title, source_type, created_at,
			message_count, user_message_count, assistant_message_count,
			max_round_index, first_message_at, last_message_at
		FROM conversation_stats_view
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM conversations WHERE hidden_at IS NULL`
	args := []interface{}{}
	countArgs := []interface{}{}

	// 添加来源过滤
	if sourceType != "" {
		query += " AND source_type = ?"
		countQuery += " AND source_type = ?"
		args = append(args, sourceType)
		countArgs = append(countArgs, sourceType)
	}

	// 添加日期过滤
	if dateFrom != "" {
		query += " AND created_at >= ?"
		countQuery += " AND created_at >= ?"
		args = append(args, dateFrom)
		countArgs = append(countArgs, dateFrom)
	}
	if dateTo != "" {
		query += " AND created_at <= ?"
		countQuery += " AND created_at <= ?"
		args = append(args, dateTo)
		countArgs = append(countArgs, dateTo)
	}

	// 查询总数
	var total int
	if err := r.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询总数失败: %w", err)
	}

	// 添加排序和分页
	query += " ORDER BY last_message_at DESC LIMIT ? OFFSET ?"
	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)

	// 执行查询
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询对话列表失败: %w", err)
	}
	defer rows.Close()

	// 解析结果
	conversations := []model.ConversationStats{}
	for rows.Next() {
		var c model.ConversationStats
		err := rows.Scan(
			&c.UUID, &c.Title, &c.SourceType, &c.CreatedAt,
			&c.MessageCount, &c.UserMessageCount, &c.AssistantMessageCount,
			&c.MaxRoundIndex, &c.FirstMessageAt, &c.LastMessageAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("解析对话数据失败: %w", err)
		}
		conversations = append(conversations, c)
	}

	return conversations, total, nil
}

// GetConversation 获取单个对话详情
func (r *ConversationRepository) GetConversation(uuid string) (*model.Conversation, error) {
	query := `
		SELECT uuid, source_type, title, metadata, created_at, updated_at, hidden_at
		FROM conversations
		WHERE uuid = ? AND hidden_at IS NULL
	`

	var c model.Conversation
	err := r.db.QueryRow(query, uuid).Scan(
		&c.UUID, &c.SourceType, &c.Title, &c.Metadata,
		&c.CreatedAt, &c.UpdatedAt, &c.HiddenAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询对话失败: %w", err)
	}

	return &c, nil
}

// GetConversationStats 获取对话统计信息
func (r *ConversationRepository) GetConversationStats(uuid string) (*model.ConversationStats, error) {
	query := `
		SELECT
			uuid, title, source_type, created_at,
			message_count, user_message_count, assistant_message_count,
			max_round_index, first_message_at, last_message_at
		FROM conversation_stats_view
		WHERE uuid = ?
	`

	var c model.ConversationStats
	err := r.db.QueryRow(query, uuid).Scan(
		&c.UUID, &c.Title, &c.SourceType, &c.CreatedAt,
		&c.MessageCount, &c.UserMessageCount, &c.AssistantMessageCount,
		&c.MaxRoundIndex, &c.FirstMessageAt, &c.LastMessageAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询对话统计失败: %w", err)
	}

	return &c, nil
}
