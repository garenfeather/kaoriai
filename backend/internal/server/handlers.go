package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ListConversations 返回对话列表
func (h *Handler) ListConversations(c *gin.Context) {
	page, pageSize := parsePagination(c)
	writeOK(c, gin.H{
		"items": []gin.H{
			{"uuid": "conv-demo-1", "title": "示例对话", "source_type": "gpt"},
		},
		"total":     1,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetConversation 返回对话详情
func (h *Handler) GetConversation(c *gin.Context) {
	uuid := strings.TrimSpace(c.Param("uuid"))
	if uuid == "" {
		writeError(c, http.StatusBadRequest, 1, "conversation uuid required")
		return
	}
	writeOK(c, gin.H{
		"uuid":        uuid,
		"title":       "示例对话",
		"source_type": "gpt",
	})
}

// ListConversationMessages 返回指定对话的消息列表
func (h *Handler) ListConversationMessages(c *gin.Context) {
	if strings.TrimSpace(c.Param("uuid")) == "" {
		writeError(c, http.StatusBadRequest, 1, "conversation uuid required")
		return
	}
	page, pageSize := parsePagination(c)
	writeOK(c, gin.H{
		"items": []gin.H{
			{"uuid": "msg-demo-1", "role": "user", "content_type": "text"},
		},
		"total":     1,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetMessage 返回消息详情
func (h *Handler) GetMessage(c *gin.Context) {
	uuid := strings.TrimSpace(c.Param("uuid"))
	if uuid == "" {
		writeError(c, http.StatusBadRequest, 1, "message uuid required")
		return
	}
	writeOK(c, gin.H{
		"uuid":         uuid,
		"role":         "assistant",
		"content_type": "text",
		"content": gin.H{
			"type": "text",
			"text": "示例内容",
		},
	})
}

// GetMessageContext 返回消息上下文
func (h *Handler) GetMessageContext(c *gin.Context) {
	uuid := strings.TrimSpace(c.Param("uuid"))
	if uuid == "" {
		writeError(c, http.StatusBadRequest, 1, "message uuid required")
		return
	}
	writeOK(c, gin.H{
		"items": []gin.H{
			{"uuid": uuid, "role": "assistant", "content_type": "text"},
		},
	})
}

// Search 实现搜索请求校验与示例响应
func (h *Handler) Search(c *gin.Context) {
	var req struct {
		Keyword  string   `json:"keyword"`
		Sources  []string `json:"sources"`
		DateFrom string   `json:"date_from"`
		DateTo   string   `json:"date_to"`
		Tags     []int    `json:"tags"`
		Page     int      `json:"page"`
		PageSize int      `json:"page_size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, 1, "invalid request body")
		return
	}
	req.Keyword = strings.TrimSpace(req.Keyword)
	if req.Keyword == "" {
		writeError(c, http.StatusBadRequest, 1, "keyword required")
		return
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	writeOK(c, gin.H{
		"total": 1,
		"items": []gin.H{
			{
				"message_uuid":       "msg-search-1",
				"conversation_uuid":  "conv-search-1",
				"conversation_title": "搜索示例",
				"role":               "assistant",
				"content_type":       "text",
				"content_preview":    "...示例高亮...",
				"highlight":          []string{"...示例高亮..."},
				"source_type":        "gpt",
				"created_at":         nowRFC3339(),
				"score":              1.0,
			},
		},
		"page":      page,
		"page_size": pageSize,
	})
}

// ListTrees 返回对话树列表
func (h *Handler) ListTrees(c *gin.Context) {
	page, pageSize := parsePagination(c)
	writeOK(c, gin.H{
		"items": []gin.H{
			{
				"tree_id":    "tree-demo",
				"created_at": nowRFC3339(),
				"updated_at": nowRFC3339(),
			},
		},
		"total":     1,
		"page":      page,
		"page_size": pageSize,
	})
}

// UpdateTree 创建或更新对话树（根据tree_id是否为空区分）
func (h *Handler) UpdateTree(c *gin.Context) {
	var req struct {
		TreeID            string   `json:"tree_id"`
		ConversationUUIDs []string `json:"conversation_uuids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, 1, "invalid request body")
		return
	}
	if len(req.ConversationUUIDs) == 0 || !nonEmptyStrings(req.ConversationUUIDs) {
		writeError(c, http.StatusBadRequest, 1, "conversation_uuids required and must be non-empty")
		return
	}
	treeID := strings.TrimSpace(req.TreeID)
	if treeID == "" {
		treeID = "tree-" + strconv.FormatInt(timeNowUnix(), 10)
	}
	writeOK(c, gin.H{
		"tree_id":    treeID,
		"updated_at": nowRFC3339(),
	})
}

// GetTree 返回树详情
func (h *Handler) GetTree(c *gin.Context) {
	treeID := strings.TrimSpace(c.Param("tree_id"))
	if treeID == "" {
		writeError(c, http.StatusBadRequest, 1, "tree_id required")
		return
	}
	writeOK(c, gin.H{
		"tree_id": treeID,
		"tree_data": gin.H{
			"nodes": []gin.H{},
		},
		"created_at": nowRFC3339(),
		"updated_at": nowRFC3339(),
	})
}

// DeleteTree 删除树记录
func (h *Handler) DeleteTree(c *gin.Context) {
	treeID := strings.TrimSpace(c.Param("tree_id"))
	if treeID == "" {
		writeError(c, http.StatusBadRequest, 1, "tree_id required")
		return
	}
	writeOK(c, gin.H{"tree_id": treeID})
}

// CreateFavorite 创建收藏
func (h *Handler) CreateFavorite(c *gin.Context) {
	var req struct {
		TargetType string `json:"target_type"`
		TargetID   string `json:"target_id"`
		Category   string `json:"category"`
		Notes      string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, 1, "invalid request body")
		return
	}
	req.TargetType = strings.TrimSpace(req.TargetType)
	req.TargetID = strings.TrimSpace(req.TargetID)
	if req.TargetType == "" || req.TargetID == "" {
		writeError(c, http.StatusBadRequest, 1, "target_type and target_id required")
		return
	}
	allowed := map[string]bool{"conversation": true, "round": true, "message": true, "fragment": true}
	if !allowed[req.TargetType] {
		writeError(c, http.StatusBadRequest, 1, "invalid target_type")
		return
	}
	writeOK(c, gin.H{
		"id":          "fav-demo-1",
		"target_type": req.TargetType,
		"target_id":   req.TargetID,
		"category":    req.Category,
		"notes":       req.Notes,
		"created_at":  nowRFC3339(),
	})
}

// ListFavorites 收藏列表
func (h *Handler) ListFavorites(c *gin.Context) {
	page, pageSize := parsePagination(c)
	writeOK(c, gin.H{
		"items": []gin.H{
			{"id": "fav-demo-1", "target_type": "message", "target_id": "msg-demo-1", "category": "default"},
		},
		"total":     1,
		"page":      page,
		"page_size": pageSize,
	})
}

// DeleteFavorite 删除收藏
func (h *Handler) DeleteFavorite(c *gin.Context) {
	if strings.TrimSpace(c.Param("id")) == "" {
		writeError(c, http.StatusBadRequest, 1, "favorite id required")
		return
	}
	writeOK(c, gin.H{"deleted": true})
}

// ListTags 标签列表
func (h *Handler) ListTags(c *gin.Context) {
	writeOK(c, gin.H{
		"items": []gin.H{
			{"id": 1, "name": "监控", "color": "#3B82F6", "usage_count": 0},
		},
	})
}

// CreateTag 创建标签
func (h *Handler) CreateTag(c *gin.Context) {
	var req struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, 1, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(c, http.StatusBadRequest, 1, "name required")
		return
	}
	writeOK(c, gin.H{"id": 1, "name": req.Name, "color": req.Color})
}

// AddConversationTag 单个添加
func (h *Handler) AddConversationTag(c *gin.Context) {
	var req struct {
		TagID            int    `json:"tag_id"`
		ConversationUUID string `json:"conversation_uuid"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, 1, "invalid request body")
		return
	}
	if req.TagID <= 0 || strings.TrimSpace(req.ConversationUUID) == "" {
		writeError(c, http.StatusBadRequest, 1, "tag_id and conversation_uuid required")
		return
	}
	writeOK(c, gin.H{"tag_id": req.TagID, "conversation_uuid": req.ConversationUUID})
}

// BatchAddConversationTags 批量添加标签
func (h *Handler) BatchAddConversationTags(c *gin.Context) {
	var req struct {
		ConversationUUID string `json:"conversation_uuid"`
		TagIDs           []int  `json:"tag_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, 1, "invalid request body")
		return
	}
	if strings.TrimSpace(req.ConversationUUID) == "" || len(req.TagIDs) == 0 {
		writeError(c, http.StatusBadRequest, 1, "conversation_uuid and tag_ids required")
		return
	}
	writeOK(c, gin.H{
		"conversation_uuid": req.ConversationUUID,
		"tag_ids":           req.TagIDs,
	})
}

// BatchRemoveConversationTags 批量删除标签
func (h *Handler) BatchRemoveConversationTags(c *gin.Context) {
	var req struct {
		ConversationUUID string `json:"conversation_uuid"`
		TagIDs           []int  `json:"tag_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, 1, "invalid request body")
		return
	}
	if strings.TrimSpace(req.ConversationUUID) == "" || len(req.TagIDs) == 0 {
		writeError(c, http.StatusBadRequest, 1, "conversation_uuid and tag_ids required")
		return
	}
	writeOK(c, gin.H{
		"conversation_uuid": req.ConversationUUID,
		"removed_tag_ids":   req.TagIDs,
	})
}

// DeleteConversationTag 删除单个标签关联
func (h *Handler) DeleteConversationTag(c *gin.Context) {
	if strings.TrimSpace(c.Param("id")) == "" {
		writeError(c, http.StatusBadRequest, 1, "id required")
		return
	}
	writeOK(c, gin.H{"deleted": true})
}

// ListTagConversations 标签下的对话列表
func (h *Handler) ListTagConversations(c *gin.Context) {
	if strings.TrimSpace(c.Param("id")) == "" {
		writeError(c, http.StatusBadRequest, 1, "tag id required")
		return
	}
	page, pageSize := parsePagination(c)
	writeOK(c, gin.H{
		"items": []gin.H{
			{"uuid": "conv-demo-1", "title": "示例对话"},
		},
		"total":     1,
		"page":      page,
		"page_size": pageSize,
	})
}

// StatsOverview 总览统计
func (h *Handler) StatsOverview(c *gin.Context) {
	writeOK(c, gin.H{
		"total_conversations": 1,
		"total_messages":      1,
		"sources": gin.H{
			"gpt":    1,
			"claude": 0,
		},
	})
}

// StatsByDate 按日期统计
func (h *Handler) StatsByDate(c *gin.Context) {
	writeOK(c, []gin.H{
		{"date": "2025-11-20", "count": 1},
	})
}

// SyncBatch Worker批量同步
func (h *Handler) SyncBatch(c *gin.Context) {
	var req struct {
		SourceType    string                   `json:"source_type"`
		Conversations []map[string]interface{} `json:"conversations"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, 1, "invalid request body")
		return
	}
	if strings.TrimSpace(req.SourceType) == "" {
		writeError(c, http.StatusBadRequest, 1, "source_type required")
		return
	}
	if len(req.Conversations) == 0 {
		writeError(c, http.StatusBadRequest, 1, "conversations required")
		return
	}
	writeOK(c, gin.H{
		"success":                true,
		"inserted_conversations": len(req.Conversations),
		"inserted_messages":      0,
		"updated_conversations":  0,
		"updated_messages":       0,
	})
}

// parsePagination 解析分页参数，提供默认值
func parsePagination(c *gin.Context) (int, int) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	return page, pageSize
}

func parsePositiveInt(val string, def int) int {
	if val == "" {
		return def
	}
	if n, err := strconv.Atoi(val); err == nil && n > 0 {
		return n
	}
	return def
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return page, pageSize
}

func timeNowUnix() int64 {
	return time.Now().Unix()
}
