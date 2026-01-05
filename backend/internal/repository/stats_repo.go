// Package repository 实现数据访问层
package repository

import (
	"database/sql"
	"fmt"

	"gpt-tools/backend/internal/model"
)

// StatsRepository 统计数据访问接口
type StatsRepository struct {
	db *sql.DB
}

// NewStatsRepository 创建统计仓库实例
func NewStatsRepository(db *sql.DB) *StatsRepository {
	return &StatsRepository{db: db}
}

// GetOverview 获取概览统计
func (r *StatsRepository) GetOverview() (*model.StatsOverview, error) {
	stats := &model.StatsOverview{
		Sources: make(map[string]int),
	}

	// 查询总对话数
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM conversations WHERE hidden_at IS NULL
	`).Scan(&stats.TotalConversations)
	if err != nil {
		return nil, fmt.Errorf("查询总对话数失败: %w", err)
	}

	// 查询总消息数
	err = r.db.QueryRow(`
		SELECT COUNT(*) FROM messages WHERE hidden_at IS NULL
	`).Scan(&stats.TotalMessages)
	if err != nil {
		return nil, fmt.Errorf("查询总消息数失败: %w", err)
	}

	// 查询各来源的对话数量
	rows, err := r.db.Query(`
		SELECT source_type, COUNT(*)
		FROM conversations
		WHERE hidden_at IS NULL
		GROUP BY source_type
	`)
	if err != nil {
		return nil, fmt.Errorf("查询来源统计失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sourceType string
		var count int
		if err := rows.Scan(&sourceType, &count); err != nil {
			return nil, fmt.Errorf("解析来源统计失败: %w", err)
		}
		stats.Sources[sourceType] = count
	}

	return stats, nil
}

// GetStatsByDate 获取按日期统计
func (r *StatsRepository) GetStatsByDate(dateFrom, dateTo string) ([]model.StatsByDate, error) {
	query := `
		SELECT DATE(created_at) as date, COUNT(*) as count
		FROM messages
		WHERE hidden_at IS NULL
	`
	args := []interface{}{}

	// 添加日期过滤
	if dateFrom != "" {
		query += " AND created_at >= ?"
		args = append(args, dateFrom)
	}
	if dateTo != "" {
		query += " AND created_at <= ?"
		args = append(args, dateTo)
	}

	query += " GROUP BY DATE(created_at) ORDER BY date DESC LIMIT 30"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询日期统计失败: %w", err)
	}
	defer rows.Close()

	stats := []model.StatsByDate{}
	for rows.Next() {
		var s model.StatsByDate
		if err := rows.Scan(&s.Date, &s.Count); err != nil {
			return nil, fmt.Errorf("解析日期统计失败: %w", err)
		}
		stats = append(stats, s)
	}

	return stats, nil
}
