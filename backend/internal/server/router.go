// Package server 实现 API 服务器的路由和通用工具函数
package server

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gpt-tools/backend/internal/repository"
)

// APIResponse 定义统一的 API 响应结构
// 所有的 API 响应都使用这个结构体,确保前端可以统一处理
type APIResponse struct {
	Code    int         `json:"code"`              // 错误码: 0=成功, 非0=各种错误码
	Message string      `json:"message"`           // 响应消息: "ok" 或错误描述
	Data    interface{} `json:"data,omitempty"`    // 实际数据,成功时包含业务数据
}

// Handler 存放所有 HTTP 请求处理方法的结构体
// 注入各个 Repository 用于数据访问
type Handler struct{
	conversationRepo *repository.ConversationRepository
	messageRepo      *repository.MessageRepository
	treeRepo         *repository.TreeRepository
	favoriteRepo     *repository.FavoriteRepository
	tagRepo          *repository.TagRepository
	statsRepo        *repository.StatsRepository
}

// NewRouter 创建并配置 Gin 路由器
// 参数:
//   - db: 数据库连接
// 返回:
//   - *gin.Engine: 配置好的 Gin 引擎
func NewRouter(db *sql.DB) *gin.Engine {
	// 创建 Gin 引擎,不使用默认中间件
	r := gin.New()

	// 注册 Recovery 中间件,防止 panic 导致服务器崩溃
	r.Use(gin.Recovery())

	// 创建 Handler 实例,注入所有 Repository
	h := &Handler{
		conversationRepo: repository.NewConversationRepository(db),
		messageRepo:      repository.NewMessageRepository(db),
		treeRepo:         repository.NewTreeRepository(db),
		favoriteRepo:     repository.NewFavoriteRepository(db),
		tagRepo:          repository.NewTagRepository(db),
		statsRepo:        repository.NewStatsRepository(db),
	}

	// ===== 公共 API 路由组 (前端和客户端使用) =====
	api := r.Group("/api/v1")
	{
		// --- 对话管理 API ---
		api.GET("/conversations", h.ListConversations)                      // 获取对话列表(支持分页和过滤)
		api.GET("/conversations/:uuid", h.GetConversation)                  // 获取单个对话详情
		api.GET("/conversations/:uuid/messages", h.ListConversationMessages)// 获取对话的消息列表

		// --- 消息管理 API ---
		api.GET("/messages/:uuid", h.GetMessage)                           // 获取单条消息详情
		api.GET("/messages/:uuid/context", h.GetMessageContext)            // 获取消息的上下文(前后N条消息)

		// --- 搜索 API ---
		api.POST("/search", h.Search)                                      // 全文搜索(支持关键词、过滤、分页)

		// --- 对话树管理 API ---
		api.GET("/trees", h.ListTrees)                                     // 获取对话树列表
		api.POST("/tree/update", h.UpdateTree)                             // 创建或更新对话树
		api.GET("/trees/:tree_id", h.GetTree)                              // 获取对话树详情
		api.DELETE("/trees/:tree_id", h.DeleteTree)                        // 删除对话树

		// --- 收藏管理 API ---
		api.POST("/favorites", h.CreateFavorite)                           // 创建收藏
		api.GET("/favorites", h.ListFavorites)                             // 获取收藏列表
		api.DELETE("/favorites/:id", h.DeleteFavorite)                     // 删除收藏

		// --- 标签管理 API ---
		api.GET("/tags", h.ListTags)                                       // 获取所有标签
		api.POST("/tags", h.CreateTag)                                     // 创建新标签
		api.POST("/conversation-tags", h.AddConversationTag)               // 为对话添加单个标签
		api.POST("/conversation-tags/batch-add", h.BatchAddConversationTags)     // 批量添加标签
		api.POST("/conversation-tags/batch-remove", h.BatchRemoveConversationTags) // 批量移除标签
		api.DELETE("/conversation-tags/:id", h.DeleteConversationTag)      // 删除对话标签关联
		api.GET("/tags/:id/conversations", h.ListTagConversations)         // 获取标签下的所有对话

		// --- 统计 API ---
		api.GET("/stats/overview", h.StatsOverview)                        // 获取概览统计
		api.GET("/stats/by-date", h.StatsByDate)                          // 获取按日期统计
	}

	// ===== 内部 API 路由组 (仅供 Worker 调用) =====
	internal := r.Group("/internal/v1")
	{
		// Worker 批量同步数据接口
		// 用于数据采集 Worker 批量上传解析后的对话和消息数据
		internal.POST("/sync/batch", h.SyncBatch)
	}

	return r
}

// writeError 写入错误响应
// 参数:
//   - c: Gin 上下文
//   - httpStatus: HTTP 状态码(如 400, 404, 500)
//   - code: 业务错误码(自定义,如 1=参数错误, 2=未找到)
//   - msg: 错误消息
func writeError(c *gin.Context, httpStatus int, code int, msg string) {
	c.JSON(httpStatus, APIResponse{Code: code, Message: msg})
}

// writeOK 写入成功响应
// 参数:
//   - c: Gin 上下文
//   - data: 要返回的数据(会被包装在 APIResponse.Data 字段中)
func writeOK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{Code: 0, Message: "ok", Data: data})
}

// nonEmptyStrings 检查字符串数组中是否所有元素都非空
// 用于验证批量操作的参数
func nonEmptyStrings(values []string) bool {
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			return false
		}
	}
	return true
}

// nowRFC3339 返回当前 UTC 时间的 RFC3339 格式字符串
// 用于生成 API 响应中的时间戳字段
func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
