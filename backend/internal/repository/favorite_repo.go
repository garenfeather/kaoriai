// Package repository 实现数据访问层
package repository

import (
	"database/sql"
	"fmt"

	"gpt-tools/backend/internal/model"
)

// FavoriteRepository 收藏数据访问接口
type FavoriteRepository struct {
	db *sql.DB
}

// NewFavoriteRepository 创建收藏仓库实例
func NewFavoriteRepository(db *sql.DB) *FavoriteRepository {
	return &FavoriteRepository{db: db}
}

// ListFavorites 获取收藏列表
func (r *FavoriteRepository) ListFavorites(category string, page, pageSize int) ([]model.Favorite, int, error) {
	// 构建查询条件
	query := `
		SELECT id, target_type, target_id, category, notes, created_at
		FROM favorites
		WHERE 1=1
	`
	countQuery := "SELECT COUNT(*) FROM favorites WHERE 1=1"
	args := []interface{}{}
	countArgs := []interface{}{}

	// 添加分类过滤
	if category != "" {
		query += " AND category = ?"
		countQuery += " AND category = ?"
		args = append(args, category)
		countArgs = append(countArgs, category)
	}

	// 查询总数
	var total int
	if err := r.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询收藏总数失败: %w", err)
	}

	// 添加排序和分页
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)

	// 执行查询
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询收藏列表失败: %w", err)
	}
	defer rows.Close()

	favorites := []model.Favorite{}
	for rows.Next() {
		var f model.Favorite
		err := rows.Scan(
			&f.ID, &f.TargetType, &f.TargetID,
			&f.Category, &f.Notes, &f.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("解析收藏数据失败: %w", err)
		}
		favorites = append(favorites, f)
	}

	return favorites, total, nil
}

// CreateFavorite 创建收藏
func (r *FavoriteRepository) CreateFavorite(targetType, targetID, category, notes string) (*model.Favorite, error) {
	query := `
		INSERT INTO favorites (target_type, target_id, category, notes, created_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	result, err := r.db.Exec(query, targetType, targetID, category, notes)
	if err != nil {
		return nil, fmt.Errorf("创建收藏失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("获取收藏ID失败: %w", err)
	}

	// 查询并返回创建的收藏
	return r.GetFavorite(int(id))
}

// GetFavorite 根据ID获取收藏
func (r *FavoriteRepository) GetFavorite(id int) (*model.Favorite, error) {
	query := `
		SELECT id, target_type, target_id, category, notes, created_at
		FROM favorites
		WHERE id = ?
	`

	var f model.Favorite
	err := r.db.QueryRow(query, id).Scan(
		&f.ID, &f.TargetType, &f.TargetID,
		&f.Category, &f.Notes, &f.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询收藏失败: %w", err)
	}

	return &f, nil
}

// DeleteFavorite 删除收藏
func (r *FavoriteRepository) DeleteFavorite(id int) error {
	query := "DELETE FROM favorites WHERE id = ?"

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除收藏失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取删除结果失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("收藏不存在")
	}

	return nil
}
