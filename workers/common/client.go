package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient API客户端
type APIClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewAPIClient 创建API客户端
func NewAPIClient(baseURL, token string, timeout int) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		token:   token,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// SyncBatchRequest 批量同步请求
type SyncBatchRequest struct {
	SourceType    string         `json:"source_type"`
	Conversations []Conversation `json:"conversations"`
}

// Conversation 对话数据结构
type Conversation struct {
	UUID      string            `json:"uuid"`
	Title     string            `json:"title"`
	Metadata  map[string]string `json:"metadata"`
	Messages  []Message         `json:"messages"`
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
}

// Message 消息数据结构
type Message struct {
	UUID        string                 `json:"uuid"`
	ParentUUID  string                 `json:"parent_uuid"`
	RoundIndex  int                    `json:"round_index"`
	Role        string                 `json:"role"`
	ContentType string                 `json:"content_type"`
	Content     map[string]interface{} `json:"content"`
	CreatedAt   string                 `json:"created_at"`
}

// SyncBatchResponse 批量同步响应
type SyncBatchResponse struct {
	Success                int `json:"success"`
	InsertedConversations  int `json:"inserted_conversations"`
	InsertedMessages       int `json:"inserted_messages"`
	UpdatedConversations   int `json:"updated_conversations"`
	UpdatedMessages        int `json:"updated_messages"`
}

// APIResponse 通用API响应
type APIResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// SyncBatch 批量同步数据到API服务器
func (c *APIClient) SyncBatch(req *SyncBatchRequest) (*SyncBatchResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/internal/v1/sync/batch", c.baseURL)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if apiResp.Code != 0 {
		return nil, fmt.Errorf("API error: code=%d, message=%s", apiResp.Code, apiResp.Message)
	}

	var syncResp SyncBatchResponse
	if err := json.Unmarshal(apiResp.Data, &syncResp); err != nil {
		return nil, fmt.Errorf("unmarshal data: %w", err)
	}

	return &syncResp, nil
}
