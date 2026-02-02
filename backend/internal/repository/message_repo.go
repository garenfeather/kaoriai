// Package repository 实现数据访问层
package repository

import (
	"database/sql"
	"fmt"

	"gpt-tools/backend/internal/model"
)

// MessageRepository 消息数据访问接口
type MessageRepository struct {
	db *sql.DB
}

// NewMessageRepository 创建消息仓库实例
func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// ListConversationMessages 获取对话的消息列表
func (r *MessageRepository) ListConversationMessages(conversationUUID string, page, pageSize int) ([]model.Message, int, error) {
	// 查询总数
	countQuery := `
		SELECT COUNT(*)
		FROM messages
		WHERE conversation_uuid = ? AND hidden_at IS NULL
	`
	var total int
	if err := r.db.QueryRow(countQuery, conversationUUID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询消息总数失败: %w", err)
	}

	// 查询消息列表
	query := `
		SELECT uuid, conversation_uuid, parent_uuid, round_index,
		       role, content_type, content, thinking, model, created_at, hidden_at
		FROM messages
		WHERE conversation_uuid = ? AND hidden_at IS NULL
		ORDER BY round_index, created_at
		LIMIT ? OFFSET ?
	`

	offset := (page - 1) * pageSize
	rows, err := r.db.Query(query, conversationUUID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询消息列表失败: %w", err)
	}
	defer rows.Close()

	messages := []model.Message{}
	for rows.Next() {
		var m model.Message
		err := rows.Scan(
			&m.UUID, &m.ConversationUUID, &m.ParentUUID, &m.RoundIndex,
			&m.Role, &m.ContentType, &m.Content, &m.Thinking, &m.Model, &m.CreatedAt, &m.HiddenAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("解析消息数据失败: %w", err)
		}
		messages = append(messages, m)
	}

	return messages, total, nil
}

// GetMessage 获取单条消息
func (r *MessageRepository) GetMessage(uuid string) (*model.Message, error) {
	query := `
		SELECT uuid, conversation_uuid, parent_uuid, round_index,
		       role, content_type, content, thinking, model, created_at, hidden_at
		FROM messages
		WHERE uuid = ? AND hidden_at IS NULL
	`

	var m model.Message
	err := r.db.QueryRow(query, uuid).Scan(
		&m.UUID, &m.ConversationUUID, &m.ParentUUID, &m.RoundIndex,
		&m.Role, &m.ContentType, &m.Content, &m.Thinking, &m.Model, &m.CreatedAt, &m.HiddenAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询消息失败: %w", err)
	}

	return &m, nil
}

// GetMessageContext 获取消息上下文(前后N条消息)
func (r *MessageRepository) GetMessageContext(uuid string, before, after int) ([]model.Message, error) {
	// 先获取当前消息的 round_index 和 conversation_uuid
	var roundIndex int
	var conversationUUID string
	err := r.db.QueryRow(
		"SELECT round_index, conversation_uuid FROM messages WHERE uuid = ?",
		uuid,
	).Scan(&roundIndex, &conversationUUID)

	if err != nil {
		return nil, fmt.Errorf("查询消息信息失败: %w", err)
	}

	// 查询上下文消息
	query := `
		SELECT uuid, conversation_uuid, parent_uuid, round_index,
		       role, content_type, content, thinking, model, created_at, hidden_at
		FROM messages
		WHERE conversation_uuid = ?
		  AND round_index BETWEEN ? AND ?
		  AND hidden_at IS NULL
		ORDER BY round_index, created_at
	`

	minRound := roundIndex - before
	if minRound < 1 {
		minRound = 1
	}
	maxRound := roundIndex + after

	rows, err := r.db.Query(query, conversationUUID, minRound, maxRound)
	if err != nil {
		return nil, fmt.Errorf("查询消息上下文失败: %w", err)
	}
	defer rows.Close()

	messages := []model.Message{}
	for rows.Next() {
		var m model.Message
		err := rows.Scan(
			&m.UUID, &m.ConversationUUID, &m.ParentUUID, &m.RoundIndex,
			&m.Role, &m.ContentType, &m.Content, &m.Thinking, &m.Model, &m.CreatedAt, &m.HiddenAt,
		)
		if err != nil {
			return nil, fmt.Errorf("解析消息数据失败: %w", err)
		}
		messages = append(messages, m)
	}

	return messages, nil
}
