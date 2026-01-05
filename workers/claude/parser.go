package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// ClaudeConversation Claude 对话结构
type ClaudeConversation struct {
	UUID         string              `json:"uuid"`
	Name         string              `json:"name"`
	Summary      string              `json:"summary"`
	CreatedAt    string              `json:"created_at"`
	UpdatedAt    string              `json:"updated_at"`
	ChatMessages []ClaudeChatMessage `json:"chat_messages"`
}

// ClaudeChatMessage Claude 消息结构
type ClaudeChatMessage struct {
	UUID      string          `json:"uuid"`
	Text      string          `json:"text"`
	Content   []ClaudeContent `json:"content"`
	Sender    string          `json:"sender"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

// ClaudeContent 内容结构
type ClaudeContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ParsedConversation 解析后的对话结构
type ParsedConversation struct {
	UUID      string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
	Messages  []ParsedMessage
}

// ParsedMessage 解析后的消息
type ParsedMessage struct {
	UUID        string
	ParentUUID  string
	RoundIndex  int
	Role        string
	ContentType string
	Content     string
	CreatedAt   time.Time
}

// ParseConversationsJSON 解析 conversations.json 文件
func ParseConversationsJSON(filePath string) ([]*ParsedConversation, error) {
	// 读取文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// 解析 JSON 数组
	var conversations []ClaudeConversation
	if err := json.Unmarshal(data, &conversations); err != nil {
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}

	// 转换为 ParsedConversation
	var results []*ParsedConversation
	for _, conv := range conversations {
		parsed, err := convertConversation(conv)
		if err != nil {
			// 跳过无效的对话
			continue
		}
		results = append(results, parsed)
	}

	return results, nil
}

// convertConversation 转换单个对话
func convertConversation(conv ClaudeConversation) (*ParsedConversation, error) {
	// 如果没有消息，跳过
	if len(conv.ChatMessages) == 0 {
		return nil, fmt.Errorf("no chat messages")
	}

	// 解析时间戳
	createdAt, err := parseTimestamp(conv.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}

	updatedAt, err := parseTimestamp(conv.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}

	// 转换消息
	messages, err := convertMessages(conv.ChatMessages)
	if err != nil {
		return nil, fmt.Errorf("convert messages: %w", err)
	}

	// 如果没有有效消息，跳过
	if len(messages) == 0 {
		return nil, fmt.Errorf("no valid messages")
	}

	// 提取标题（优先级：name → summary → 第一条 user 消息前 50 字符）
	title := extractTitle(conv.Name, conv.Summary, messages)

	return &ParsedConversation{
		UUID:      conv.UUID,
		Title:     title,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Messages:  messages,
	}, nil
}

// convertMessages 转换消息列表
func convertMessages(messages []ClaudeChatMessage) ([]ParsedMessage, error) {
	var results []ParsedMessage
	var prevUUID string
	roundIndex := 0

	for _, msg := range messages {
		// 映射 sender 到 role
		role := mapSenderToRole(msg.Sender)

		// 跳过 system 角色的消息
		if role == "system" {
			continue
		}

		// 提取文本内容（从 content 数组）
		content := extractTextContent(msg.Content)
		if content == "" {
			continue
		}

		// user 消息时递增 round_index
		if role == "user" || role == "human" {
			roundIndex++
		}

		// 解析时间戳
		createdAt, err := parseTimestamp(msg.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("parse message timestamp: %w", err)
		}

		message := ParsedMessage{
			UUID:        msg.UUID,
			ParentUUID:  prevUUID,
			RoundIndex:  roundIndex,
			Role:        role,
			ContentType: "text",
			Content:     content,
			CreatedAt:   createdAt,
		}

		results = append(results, message)
		prevUUID = msg.UUID
	}

	return results, nil
}

// mapSenderToRole 将 sender 映射到 role
func mapSenderToRole(sender string) string {
	switch sender {
	case "human":
		return "user"
	case "assistant":
		return "assistant"
	default:
		return sender
	}
}

// extractTextContent 从 content 数组提取文本内容
func extractTextContent(contents []ClaudeContent) string {
	var textParts []string
	for _, content := range contents {
		if content.Type == "text" && content.Text != "" {
			textParts = append(textParts, content.Text)
		}
	}
	return strings.Join(textParts, "\n")
}

// extractTitle 提取标题
// 优先级：name → summary → 第一条 user 消息前 50 字符
func extractTitle(name, summary string, messages []ParsedMessage) string {
	// 优先使用 name
	if name != "" {
		return name
	}

	// 其次使用 summary
	if summary != "" {
		return summary
	}

	// 最后从第一条 user 消息提取
	for _, msg := range messages {
		if msg.Role == "user" || msg.Role == "human" {
			content := strings.TrimSpace(msg.Content)
			if len(content) > 50 {
				return content[:50]
			}
			return content
		}
	}

	return "Untitled Conversation"
}

// parseTimestamp 解析 Claude 的时间戳
// 格式：2025-11-02T22:33:11.888588Z
func parseTimestamp(timestamp string) (time.Time, error) {
	// 尝试多种格式
	formats := []string{
		time.RFC3339Nano,  // 2025-11-02T22:33:11.888588Z
		time.RFC3339,      // 2025-11-02T22:33:11Z
	}

	for _, format := range formats {
		t, err := time.Parse(format, timestamp)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported timestamp format: %s", timestamp)
}
