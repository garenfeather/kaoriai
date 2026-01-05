// Package server 实现所有 API 的请求处理函数
package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ========================================
// 对话管理相关 Handler
// ========================================

// ListConversations 获取对话列表
// GET /api/v1/conversations?page=1&page_size=20&source_type=gpt&date_from=xxx&date_to=xxx
func (h *Handler) ListConversations(c *gin.Context) {
	page, pageSize := parsePagination(c)
	sourceType := strings.TrimSpace(c.Query("source_type"))
	dateFrom := strings.TrimSpace(c.Query("date_from"))
	dateTo := strings.TrimSpace(c.Query("date_to"))

	conversations, total, err := h.conversationRepo.ListConversations(sourceType, dateFrom, dateTo, page, pageSize)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	writeOK(c, gin.H{
		"items":     conversations,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetConversation 获取单个对话的详细信息
// GET /api/v1/conversations/:uuid
func (h *Handler) GetConversation(c *gin.Context) {
	uuid := strings.TrimSpace(c.Param("uuid"))
	if uuid == "" {
		writeError(c, http.StatusBadRequest, 1, "conversation uuid required")
		return
	}

	conversation, err := h.conversationRepo.GetConversation(uuid)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	if conversation == nil {
		writeError(c, http.StatusNotFound, 404, "conversation not found")
		return
	}

	writeOK(c, conversation)
}

// ListConversationMessages 获取对话的消息列表
// GET /api/v1/conversations/:uuid/messages?page=1&page_size=20
func (h *Handler) ListConversationMessages(c *gin.Context) {
	uuid := strings.TrimSpace(c.Param("uuid"))
	if uuid == "" {
		writeError(c, http.StatusBadRequest, 1, "conversation uuid required")
		return
	}

	page, pageSize := parsePagination(c)

	messages, total, err := h.messageRepo.ListConversationMessages(uuid, page, pageSize)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	writeOK(c, gin.H{
		"items":     messages,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ========================================
// 消息管理相关 Handler
// ========================================

// GetMessage 获取单条消息的详细信息
// GET /api/v1/messages/:uuid
func (h *Handler) GetMessage(c *gin.Context) {
	uuid := strings.TrimSpace(c.Param("uuid"))
	if uuid == "" {
		writeError(c, http.StatusBadRequest, 1, "message uuid required")
		return
	}

	message, err := h.messageRepo.GetMessage(uuid)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	if message == nil {
		writeError(c, http.StatusNotFound, 404, "message not found")
		return
	}

	writeOK(c, message)
}

// GetMessageContext 获取消息的上下文
// GET /api/v1/messages/:uuid/context?before=2&after=2
func (h *Handler) GetMessageContext(c *gin.Context) {
	uuid := strings.TrimSpace(c.Param("uuid"))
	if uuid == "" {
		writeError(c, http.StatusBadRequest, 1, "message uuid required")
		return
	}

	before := parsePositiveInt(c.Query("before"), 2)
	after := parsePositiveInt(c.Query("after"), 2)

	messages, err := h.messageRepo.GetMessageContext(uuid, before, after)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	writeOK(c, gin.H{
		"items": messages,
	})
}

// ========================================
// 搜索相关 Handler
// ========================================

// Search 实现全文搜索功能 (TODO: 集成 Bleve)
// POST /api/v1/search
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

	// TODO: 实现 Bleve 搜索引擎集成
	writeOK(c, gin.H{
		"total":     0,
		"items":     []gin.H{},
		"page":      page,
		"page_size": pageSize,
	})
}

// ========================================
// 对话树管理相关 Handler
// ========================================

// ListTrees 获取对话树列表
// GET /api/v1/trees?page=1&page_size=20
func (h *Handler) ListTrees(c *gin.Context) {
	page, pageSize := parsePagination(c)

	trees, total, err := h.treeRepo.ListTrees(page, pageSize)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	// 只返回元信息,不包含树结构数据
	items := []gin.H{}
	for _, t := range trees {
		items = append(items, gin.H{
			"tree_id":    t.TreeID,
			"title":      t.Title,
			"created_at": t.CreatedAt.Format(time.RFC3339),
			"updated_at": t.UpdatedAt.Format(time.RFC3339),
		})
	}

	writeOK(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// UpdateTree 创建或更新对话树
// POST /api/v1/tree/update
func (h *Handler) UpdateTree(c *gin.Context) {
	var req struct {
		TreeID            string   `json:"tree_id"`
		Title             string   `json:"title"`
		Description       string   `json:"description"`
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

	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)

	// 构建简单的树结构数据(JSON格式)
	// 实际应用中可以根据业务需求构建更复杂的树结构
	treeDataMap := gin.H{
		"conversation_uuids": req.ConversationUUIDs,
		"node_count":         len(req.ConversationUUIDs),
	}
	treeDataJSON, err := jsonMarshal(treeDataMap)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, "failed to marshal tree data")
		return
	}

	// 保存到数据库
	err = h.treeRepo.UpsertTree(treeID, req.Title, req.Description, treeDataJSON)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	writeOK(c, gin.H{
		"tree_id":    treeID,
		"updated_at": nowRFC3339(),
	})
}

// GetTree 获取对话树的详细信息
// GET /api/v1/trees/:tree_id
func (h *Handler) GetTree(c *gin.Context) {
	treeID := strings.TrimSpace(c.Param("tree_id"))
	if treeID == "" {
		writeError(c, http.StatusBadRequest, 1, "tree_id required")
		return
	}

	tree, err := h.treeRepo.GetTree(treeID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	if tree == nil {
		writeError(c, http.StatusNotFound, 404, "tree not found")
		return
	}

	writeOK(c, tree)
}

// DeleteTree 删除对话树
// DELETE /api/v1/trees/:tree_id
func (h *Handler) DeleteTree(c *gin.Context) {
	treeID := strings.TrimSpace(c.Param("tree_id"))
	if treeID == "" {
		writeError(c, http.StatusBadRequest, 1, "tree_id required")
		return
	}

	err := h.treeRepo.DeleteTree(treeID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	writeOK(c, gin.H{"deleted": true, "tree_id": treeID})
}

// ========================================
// 收藏管理相关 Handler
// ========================================

// CreateFavorite 创建收藏
// POST /api/v1/favorites
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
	req.Category = strings.TrimSpace(req.Category)

	if req.TargetType == "" || req.TargetID == "" {
		writeError(c, http.StatusBadRequest, 1, "target_type and target_id required")
		return
	}

	allowed := map[string]bool{"conversation": true, "round": true, "message": true, "fragment": true}
	if !allowed[req.TargetType] {
		writeError(c, http.StatusBadRequest, 1, "invalid target_type")
		return
	}

	favorite, err := h.favoriteRepo.CreateFavorite(req.TargetType, req.TargetID, req.Category, req.Notes)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	writeOK(c, favorite)
}

// ListFavorites 获取收藏列表
// GET /api/v1/favorites?category=tech_solution&page=1&page_size=20
func (h *Handler) ListFavorites(c *gin.Context) {
	page, pageSize := parsePagination(c)
	category := strings.TrimSpace(c.Query("category"))

	favorites, total, err := h.favoriteRepo.ListFavorites(category, page, pageSize)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	writeOK(c, gin.H{
		"items":     favorites,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// DeleteFavorite 删除收藏
// DELETE /api/v1/favorites/:id
func (h *Handler) DeleteFavorite(c *gin.Context) {
	idStr := strings.TrimSpace(c.Param("id"))
	if idStr == "" {
		writeError(c, http.StatusBadRequest, 1, "favorite id required")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeError(c, http.StatusBadRequest, 1, "invalid id")
		return
	}

	err = h.favoriteRepo.DeleteFavorite(id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	writeOK(c, gin.H{"deleted": true})
}

// ========================================
// 标签管理相关 Handler
// ========================================

// ListTags 获取所有标签
// GET /api/v1/tags
func (h *Handler) ListTags(c *gin.Context) {
	tags, err := h.tagRepo.ListTags()
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	writeOK(c, gin.H{
		"items": tags,
	})
}

// CreateTag 创建新标签
// POST /api/v1/tags
func (h *Handler) CreateTag(c *gin.Context) {
	var req struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, 1, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Color = strings.TrimSpace(req.Color)

	if req.Name == "" {
		writeError(c, http.StatusBadRequest, 1, "name required")
		return
	}

	tag, err := h.tagRepo.CreateTag(req.Name, req.Color)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	writeOK(c, tag)
}

// AddConversationTag 为对话添加单个标签
// POST /api/v1/conversation-tags
func (h *Handler) AddConversationTag(c *gin.Context) {
	var req struct {
		TagID            int    `json:"tag_id"`
		ConversationUUID string `json:"conversation_uuid"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, 1, "invalid request body")
		return
	}
	req.ConversationUUID = strings.TrimSpace(req.ConversationUUID)

	if req.TagID <= 0 || req.ConversationUUID == "" {
		writeError(c, http.StatusBadRequest, 1, "tag_id and conversation_uuid required")
		return
	}

	err := h.tagRepo.AddConversationTag(req.TagID, req.ConversationUUID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	writeOK(c, gin.H{
		"tag_id":            req.TagID,
		"conversation_uuid": req.ConversationUUID,
		"created_at":        nowRFC3339(),
	})
}

// BatchAddConversationTags 批量为对话添加标签
// POST /api/v1/conversation-tags/batch-add
func (h *Handler) BatchAddConversationTags(c *gin.Context) {
	var req struct {
		ConversationUUID string `json:"conversation_uuid"`
		TagIDs           []int  `json:"tag_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, 1, "invalid request body")
		return
	}
	req.ConversationUUID = strings.TrimSpace(req.ConversationUUID)

	if req.ConversationUUID == "" || len(req.TagIDs) == 0 {
		writeError(c, http.StatusBadRequest, 1, "conversation_uuid and tag_ids required")
		return
	}

	err := h.tagRepo.BatchAddConversationTags(req.ConversationUUID, req.TagIDs)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	writeOK(c, gin.H{
		"conversation_uuid": req.ConversationUUID,
		"tag_ids":           req.TagIDs,
		"added_at":          nowRFC3339(),
	})
}

// BatchRemoveConversationTags 批量移除对话的标签
// POST /api/v1/conversation-tags/batch-remove
func (h *Handler) BatchRemoveConversationTags(c *gin.Context) {
	var req struct {
		ConversationUUID string `json:"conversation_uuid"`
		TagIDs           []int  `json:"tag_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, 1, "invalid request body")
		return
	}
	req.ConversationUUID = strings.TrimSpace(req.ConversationUUID)

	if req.ConversationUUID == "" || len(req.TagIDs) == 0 {
		writeError(c, http.StatusBadRequest, 1, "conversation_uuid and tag_ids required")
		return
	}

	err := h.tagRepo.BatchRemoveConversationTags(req.ConversationUUID, req.TagIDs)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	writeOK(c, gin.H{
		"conversation_uuid": req.ConversationUUID,
		"removed_tag_ids":   req.TagIDs,
	})
}

// DeleteConversationTag 删除对话和标签的关联
// DELETE /api/v1/conversation-tags/:id
func (h *Handler) DeleteConversationTag(c *gin.Context) {
	idStr := strings.TrimSpace(c.Param("id"))
	if idStr == "" {
		writeError(c, http.StatusBadRequest, 1, "id required")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeError(c, http.StatusBadRequest, 1, "invalid id")
		return
	}

	err = h.tagRepo.DeleteConversationTag(id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	writeOK(c, gin.H{"deleted": true})
}

// ListTagConversations 获取标签下的所有对话
// GET /api/v1/tags/:id/conversations?page=1&page_size=20
func (h *Handler) ListTagConversations(c *gin.Context) {
	idStr := strings.TrimSpace(c.Param("id"))
	if idStr == "" {
		writeError(c, http.StatusBadRequest, 1, "tag id required")
		return
	}

	tagID, err := strconv.Atoi(idStr)
	if err != nil || tagID <= 0 {
		writeError(c, http.StatusBadRequest, 1, "invalid tag id")
		return
	}

	page, pageSize := parsePagination(c)

	conversations, total, err := h.tagRepo.ListTagConversations(tagID, page, pageSize)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	writeOK(c, gin.H{
		"items":     conversations,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ========================================
// 统计相关 Handler
// ========================================

// StatsOverview 获取概览统计
// GET /api/v1/stats/overview
func (h *Handler) StatsOverview(c *gin.Context) {
	stats, err := h.statsRepo.GetOverview()
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	writeOK(c, stats)
}

// StatsByDate 获取按日期统计
// GET /api/v1/stats/by-date?date_from=2025-01-01&date_to=2025-11-30
func (h *Handler) StatsByDate(c *gin.Context) {
	dateFrom := strings.TrimSpace(c.Query("date_from"))
	dateTo := strings.TrimSpace(c.Query("date_to"))

	stats, err := h.statsRepo.GetStatsByDate(dateFrom, dateTo)
	if err != nil {
		writeError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	writeOK(c, stats)
}

// ========================================
// 内部 API Handler (仅供 Worker 调用)
// ========================================

// SyncBatch Worker 批量同步数据 (TODO: 实现同步逻辑)
// POST /internal/v1/sync/batch
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

	// TODO: 实现批量同步逻辑
	writeOK(c, gin.H{
		"success":                true,
		"inserted_conversations": len(req.Conversations),
		"inserted_messages":      0,
		"updated_conversations":  0,
		"updated_messages":       0,
	})
}

// ========================================
// 辅助函数
// ========================================

// parsePagination 解析分页参数,提供默认值
func parsePagination(c *gin.Context) (int, int) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	return page, pageSize
}

// parsePositiveInt 解析正整数,失败则返回默认值
func parsePositiveInt(val string, def int) int {
	if val == "" {
		return def
	}
	if n, err := strconv.Atoi(val); err == nil && n > 0 {
		return n
	}
	return def
}

// normalizePage 规范化分页参数
func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return page, pageSize
}

// timeNowUnix 返回当前 Unix 时间戳(秒)
func timeNowUnix() int64 {
	return time.Now().Unix()
}

// jsonMarshal 序列化对象为JSON字符串
func jsonMarshal(v interface{}) (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
