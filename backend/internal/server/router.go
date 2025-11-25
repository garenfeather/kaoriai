package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// APIResponse 定义统一响应结构
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Handler 存放所有路由处理方法
type Handler struct{}

// NewRouter 创建路由并注册所有API（仅校验并返回测试数据）
func NewRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	h := &Handler{}

	api := r.Group("/api/v1")
	{
		api.GET("/conversations", h.ListConversations)
		api.GET("/conversations/:uuid", h.GetConversation)
		api.GET("/conversations/:uuid/messages", h.ListConversationMessages)

		api.GET("/messages/:uuid", h.GetMessage)
		api.GET("/messages/:uuid/context", h.GetMessageContext)

		api.POST("/search", h.Search)

		api.GET("/trees", h.ListTrees)
		api.POST("/tree/update", h.UpdateTree)
		api.GET("/trees/:tree_id", h.GetTree)
		api.DELETE("/trees/:tree_id", h.DeleteTree)

		api.POST("/favorites", h.CreateFavorite)
		api.GET("/favorites", h.ListFavorites)
		api.DELETE("/favorites/:id", h.DeleteFavorite)

		api.GET("/tags", h.ListTags)
		api.POST("/tags", h.CreateTag)
		api.POST("/conversation-tags", h.AddConversationTag)
		api.POST("/conversation-tags/batch-add", h.BatchAddConversationTags)
		api.POST("/conversation-tags/batch-remove", h.BatchRemoveConversationTags)
		api.DELETE("/conversation-tags/:id", h.DeleteConversationTag)
		api.GET("/tags/:id/conversations", h.ListTagConversations)

		api.GET("/stats/overview", h.StatsOverview)
		api.GET("/stats/by-date", h.StatsByDate)
	}

	internal := r.Group("/internal/v1")
	{
		internal.POST("/sync/batch", h.SyncBatch)
	}

	return r
}

func writeError(c *gin.Context, httpStatus int, code int, msg string) {
	c.JSON(httpStatus, APIResponse{Code: code, Message: msg})
}

func writeOK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{Code: 0, Message: "ok", Data: data})
}

func nonEmptyStrings(values []string) bool {
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			return false
		}
	}
	return true
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
