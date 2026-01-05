// Package repository 实现数据访问层
package repository

import (
	"database/sql"
	"fmt"

	"gpt-tools/backend/internal/model"
)

// TagRepository 标签数据访问接口
type TagRepository struct {
	db *sql.DB
}

// NewTagRepository 创建标签仓库实例
func NewTagRepository(db *sql.DB) *TagRepository {
	return &TagRepository{db: db}
}

// ListTags 获取所有标签(按使用次数排序)
func (r *TagRepository) ListTags() ([]model.Tag, error) {
	query := `
		SELECT id, name, color, usage_count, created_at
		FROM tags
		ORDER BY usage_count DESC, name
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询标签列表失败: %w", err)
	}
	defer rows.Close()

	tags := []model.Tag{}
	for rows.Next() {
		var t model.Tag
		err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.UsageCount, &t.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("解析标签数据失败: %w", err)
		}
		tags = append(tags, t)
	}

	return tags, nil
}

// ListTagConversations 获取标签下的所有对话
func (r *TagRepository) ListTagConversations(tagID int, page, pageSize int) ([]model.ConversationStats, int, error) {
	// 查询总数
	countQuery := `
		SELECT COUNT(*)
		FROM conversation_tags ct
		JOIN conversations c ON ct.conversation_uuid = c.uuid
		WHERE ct.tag_id = ? AND c.hidden_at IS NULL
	`
	var total int
	if err := r.db.QueryRow(countQuery, tagID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询标签对话总数失败: %w", err)
	}

	// 查询对话列表
	query := `
		SELECT
			csv.uuid, csv.title, csv.source_type, csv.created_at,
			csv.message_count, csv.user_message_count, csv.assistant_message_count,
			csv.max_round_index, csv.first_message_at, csv.last_message_at
		FROM conversation_tags ct
		JOIN conversation_stats_view csv ON ct.conversation_uuid = csv.uuid
		WHERE ct.tag_id = ?
		ORDER BY ct.created_at DESC
		LIMIT ? OFFSET ?
	`

	offset := (page - 1) * pageSize
	rows, err := r.db.Query(query, tagID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询标签对话列表失败: %w", err)
	}
	defer rows.Close()

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

// CreateTag 创建新标签
func (r *TagRepository) CreateTag(name, color string) (*model.Tag, error) {
	query := `
		INSERT INTO tags (name, color, usage_count, created_at)
		VALUES (?, ?, 0, CURRENT_TIMESTAMP)
	`

	result, err := r.db.Exec(query, name, color)
	if err != nil {
		return nil, fmt.Errorf("创建标签失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("获取标签ID失败: %w", err)
	}

	// 查询并返回创建的标签
	return r.GetTag(int(id))
}

// GetTag 根据ID获取标签
func (r *TagRepository) GetTag(id int) (*model.Tag, error) {
	query := `
		SELECT id, name, color, usage_count, created_at
		FROM tags
		WHERE id = ?
	`

	var t model.Tag
	err := r.db.QueryRow(query, id).Scan(
		&t.ID, &t.Name, &t.Color, &t.UsageCount, &t.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}

	return &t, nil
}

// AddConversationTag 为对话添加单个标签
func (r *TagRepository) AddConversationTag(tagID int, conversationUUID string) error {
	// 使用 INSERT OR IGNORE 避免重复添加
	query := `
		INSERT OR IGNORE INTO conversation_tags (tag_id, conversation_uuid, created_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`

	result, err := r.db.Exec(query, tagID, conversationUUID)
	if err != nil {
		return fmt.Errorf("添加对话标签失败: %w", err)
	}

	// 如果成功插入，更新标签使用次数
	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected > 0 {
		r.incrementTagUsageCount(tagID)
	}

	return nil
}

// BatchAddConversationTags 批量为对话添加标签
func (r *TagRepository) BatchAddConversationTags(conversationUUID string, tagIDs []int) error {
	// 开始事务
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 批量插入
	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO conversation_tags (tag_id, conversation_uuid, created_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		return fmt.Errorf("准备插入语句失败: %w", err)
	}
	defer stmt.Close()

	addedTagIDs := []int{}
	for _, tagID := range tagIDs {
		result, err := stmt.Exec(tagID, conversationUUID)
		if err != nil {
			return fmt.Errorf("插入标签关联失败: %w", err)
		}

		// 记录成功插入的标签ID
		if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
			addedTagIDs = append(addedTagIDs, tagID)
		}
	}

	// 更新标签使用次数
	if len(addedTagIDs) > 0 {
		for _, tagID := range addedTagIDs {
			_, err := tx.Exec("UPDATE tags SET usage_count = usage_count + 1 WHERE id = ?", tagID)
			if err != nil {
				return fmt.Errorf("更新标签使用次数失败: %w", err)
			}
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// BatchRemoveConversationTags 批量移除对话的标签
func (r *TagRepository) BatchRemoveConversationTags(conversationUUID string, tagIDs []int) error {
	// 开始事务
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 批量删除
	stmt, err := tx.Prepare(`
		DELETE FROM conversation_tags
		WHERE conversation_uuid = ? AND tag_id = ?
	`)
	if err != nil {
		return fmt.Errorf("准备删除语句失败: %w", err)
	}
	defer stmt.Close()

	removedTagIDs := []int{}
	for _, tagID := range tagIDs {
		result, err := stmt.Exec(conversationUUID, tagID)
		if err != nil {
			return fmt.Errorf("删除标签关联失败: %w", err)
		}

		// 记录成功删除的标签ID
		if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
			removedTagIDs = append(removedTagIDs, tagID)
		}
	}

	// 更新标签使用次数
	if len(removedTagIDs) > 0 {
		for _, tagID := range removedTagIDs {
			_, err := tx.Exec("UPDATE tags SET usage_count = usage_count - 1 WHERE id = ?", tagID)
			if err != nil {
				return fmt.Errorf("更新标签使用次数失败: %w", err)
			}
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// DeleteConversationTag 删除对话标签关联（通过关联ID）
func (r *TagRepository) DeleteConversationTag(id int) error {
	// 先查询tag_id用于更新使用次数
	var tagID int
	err := r.db.QueryRow("SELECT tag_id FROM conversation_tags WHERE id = ?", id).Scan(&tagID)
	if err == sql.ErrNoRows {
		return nil // 记录不存在，视为删除成功
	}
	if err != nil {
		return fmt.Errorf("查询标签关联失败: %w", err)
	}

	// 删除关联
	result, err := r.db.Exec("DELETE FROM conversation_tags WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除标签关联失败: %w", err)
	}

	// 如果删除成功，减少标签使用次数
	if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
		r.decrementTagUsageCount(tagID)
	}

	return nil
}

// incrementTagUsageCount 增加标签使用次数（内部方法）
func (r *TagRepository) incrementTagUsageCount(tagID int) {
	r.db.Exec("UPDATE tags SET usage_count = usage_count + 1 WHERE id = ?", tagID)
}

// decrementTagUsageCount 减少标签使用次数（内部方法）
func (r *TagRepository) decrementTagUsageCount(tagID int) {
	r.db.Exec("UPDATE tags SET usage_count = CASE WHEN usage_count > 0 THEN usage_count - 1 ELSE 0 END WHERE id = ?", tagID)
}
