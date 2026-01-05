package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CodexMessage Codex JSONL 文件中的消息结构
type CodexMessage struct {
	Type      string        `json:"type"`
	Timestamp string        `json:"timestamp"`
	Payload   *CodexPayload `json:"payload"`
}

// CodexPayload 消息的 payload 部分
type CodexPayload struct {
	Type    string          `json:"type"`
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	ID      string          `json:"id"` // session_meta 中的 id
}

// ContentItem content 数组中的单个元素
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ParsedConversation 解析后的对话结构
type ParsedConversation struct {
	SessionID string
	Title     string
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

// ParseJSONLFile 解析 Codex JSONL 文件
func ParseJSONLFile(filePath string) (*ParsedConversation, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var sessionID string
	var rawMessages []CodexMessage

	scanner := bufio.NewScanner(file)
	// 增加缓冲区大小以处理长行
	const maxCapacity = 2 * 1024 * 1024 // 2MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	// 读取所有行
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg CodexMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// 忽略无法解析的行
			continue
		}

		// 提取 session ID
		if msg.Type == "session_meta" && msg.Payload != nil && msg.Payload.ID != "" {
			sessionID = msg.Payload.ID
		}

		// 只处理 response_item 类型且 payload 类型为 message 的记录
		if msg.Type == "response_item" && msg.Payload != nil && msg.Payload.Type == "message" {
			rawMessages = append(rawMessages, msg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan file: %w", err)
	}

	if sessionID == "" {
		return nil, fmt.Errorf("session_meta not found or missing id")
	}

	if len(rawMessages) == 0 {
		return nil, fmt.Errorf("no valid messages found")
	}

	// 转换为 ParsedMessage
	messages, err := convertMessages(sessionID, rawMessages)
	if err != nil {
		return nil, fmt.Errorf("convert messages: %w", err)
	}

	// 提取标题（从第一条 user 消息的前 50 个字符）
	title := extractTitle(messages)

	return &ParsedConversation{
		SessionID: sessionID,
		Title:     title,
		Messages:  messages,
	}, nil
}

// convertMessages 将原始消息转换为 ParsedMessage
func convertMessages(sessionID string, rawMessages []CodexMessage) ([]ParsedMessage, error) {
	var messages []ParsedMessage
	var prevUUID string
	roundIndex := 0

	for _, msg := range rawMessages {
		content := extractContent(msg.Payload)

		// 如果内容为空，跳过
		if content == "" {
			continue
		}

		// 生成 message UUID
		messageUUID, err := generateMessageUUID(sessionID, msg.Timestamp, msg.Payload)
		if err != nil {
			return nil, fmt.Errorf("generate UUID: %w", err)
		}

		// user 消息时递增 round_index
		if msg.Payload.Role == "user" || msg.Payload.Role == "human" {
			roundIndex++
		}

		// 解析时间戳
		createdAt, err := parseTimestamp(msg.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp: %w", err)
		}

		message := ParsedMessage{
			UUID:        messageUUID,
			ParentUUID:  prevUUID,
			RoundIndex:  roundIndex,
			Role:        msg.Payload.Role,
			ContentType: "text",
			Content:     content,
			CreatedAt:   createdAt,
		}

		messages = append(messages, message)
		prevUUID = messageUUID
	}

	return messages, nil
}

// generateMessageUUID 生成消息的 UUID v5
// 使用 sessionID 作为命名空间，timestamp + payload JSON 作为名字
func generateMessageUUID(sessionID, timestamp string, payload *CodexPayload) (string, error) {
	// 解析 sessionID 为 UUID 作为命名空间
	namespaceUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return "", fmt.Errorf("invalid session ID: %w", err)
	}

	// 序列化 payload 为 JSON（确保格式稳定）
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	// 拼接 timestamp + payload JSON 作为名字
	name := timestamp + string(payloadJSON)

	// 生成 UUID v5 (使用 SHA1)
	messageUUID := uuid.NewSHA1(namespaceUUID, []byte(name))
	return messageUUID.String(), nil
}

// extractContent 从 payload 中提取文本内容
func extractContent(payload *CodexPayload) string {
	if payload == nil || payload.Content == nil {
		return ""
	}

	// 尝试将 content 解析为数组
	var contentArray []ContentItem
	if err := json.Unmarshal(payload.Content, &contentArray); err == nil {
		var textParts []string
		for _, item := range contentArray {
			if item.Text != "" {
				textParts = append(textParts, item.Text)
			}
		}
		return strings.Join(textParts, "\n")
	}

	// 尝试将 content 解析为字符串
	var contentStr string
	if err := json.Unmarshal(payload.Content, &contentStr); err == nil {
		return contentStr
	}

	return ""
}

// extractTitle 从消息列表中提取标题（第一条 user 消息的前 50 个字符）
func extractTitle(messages []ParsedMessage) string {
	for _, msg := range messages {
		if msg.Role == "user" || msg.Role == "human" {
			content := msg.Content
			// 移除多余的空白
			content = strings.TrimSpace(content)
			// 取前 50 个字符
			if len(content) > 50 {
				return content[:50]
			}
			return content
		}
	}
	return "Untitled Conversation"
}

// parseTimestamp 解析 ISO8601 时间戳为数据库格式
func parseTimestamp(timestamp string) (time.Time, error) {
	// Codex 使用 ISO8601 格式：2025-10-03T12:39:28.166Z
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time: %w", err)
	}
	return t, nil
}
