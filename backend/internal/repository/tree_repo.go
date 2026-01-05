// Package repository 实现数据访问层
package repository

import (
	"database/sql"
	"fmt"

	"gpt-tools/backend/internal/model"
)

// TreeRepository 对话树数据访问接口
type TreeRepository struct {
	db *sql.DB
}

// NewTreeRepository 创建对话树仓库实例
func NewTreeRepository(db *sql.DB) *TreeRepository {
	return &TreeRepository{db: db}
}

// ListTrees 获取对话树列表
func (r *TreeRepository) ListTrees(page, pageSize int) ([]model.ConversationTree, int, error) {
	// 查询总数
	countQuery := "SELECT COUNT(*) FROM conversation_trees"
	var total int
	if err := r.db.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询树总数失败: %w", err)
	}

	// 查询列表
	query := `
		SELECT id, tree_id, title, description, tree_data, created_at, updated_at
		FROM conversation_trees
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	offset := (page - 1) * pageSize
	rows, err := r.db.Query(query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询树列表失败: %w", err)
	}
	defer rows.Close()

	trees := []model.ConversationTree{}
	for rows.Next() {
		var t model.ConversationTree
		err := rows.Scan(
			&t.ID, &t.TreeID, &t.Title, &t.Description,
			&t.TreeData, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("解析树数据失败: %w", err)
		}
		trees = append(trees, t)
	}

	return trees, total, nil
}

// GetTree 获取对话树详情
func (r *TreeRepository) GetTree(treeID string) (*model.ConversationTree, error) {
	query := `
		SELECT id, tree_id, title, description, tree_data, created_at, updated_at
		FROM conversation_trees
		WHERE tree_id = ?
	`

	var t model.ConversationTree
	err := r.db.QueryRow(query, treeID).Scan(
		&t.ID, &t.TreeID, &t.Title, &t.Description,
		&t.TreeData, &t.CreatedAt, &t.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询树失败: %w", err)
	}

	return &t, nil
}

// UpsertTree 创建或更新对话树
func (r *TreeRepository) UpsertTree(treeID, title, description, treeData string) error {
	// 使用 INSERT OR REPLACE 实现 upsert
	query := `
		INSERT INTO conversation_trees (tree_id, title, description, tree_data, created_at, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(tree_id) DO UPDATE SET
			title = excluded.title,
			description = excluded.description,
			tree_data = excluded.tree_data,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.db.Exec(query, treeID, title, description, treeData)
	if err != nil {
		return fmt.Errorf("创建或更新对话树失败: %w", err)
	}

	return nil
}

// DeleteTree 删除对话树
func (r *TreeRepository) DeleteTree(treeID string) error {
	query := "DELETE FROM conversation_trees WHERE tree_id = ?"

	result, err := r.db.Exec(query, treeID)
	if err != nil {
		return fmt.Errorf("删除对话树失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取删除结果失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("对话树不存在")
	}

	return nil
}
