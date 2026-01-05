package claude_code

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// ClaudeCodeMessage Claude Code消息原始结构
type ClaudeCodeMessage struct {
	Type        string          `json:"type"`
	UUID        string          `json:"uuid"`
	ParentUUID  *string         `json:"parentUuid"`
	SessionID   string          `json:"sessionId"`
	Message     *MessageContent `json:"message"`
	Timestamp   string          `json:"timestamp"`
	IsMeta      bool            `json:"isMeta,omitempty"`
	IsSidechain bool            `json:"isSidechain,omitempty"`
}

// MessageContent 消息内容
type MessageContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ParsedConversation 解析后的对话
type ParsedConversation struct {
	UUID      string
	Title     string
	Metadata  string
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
	ContentText string
	CreatedAt   time.Time
}

// ParseJSONLFile 解析单个 JSONL 文件
func ParseJSONLFile(filePath string) (*ParsedConversation, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var messages []ClaudeCodeMessage
	scanner := bufio.NewScanner(file)

	// 增加 buffer 大小以处理大行
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // 10MB max

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var msg ClaudeCodeMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// 跳过解析失败的行，继续处理
			continue
		}

		messages = append(messages, msg)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan file: %w", err)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no valid messages found")
	}

	return buildConversation(messages)
}

// buildConversation 从消息列表构建对话
func buildConversation(messages []ClaudeCodeMessage) (*ParsedConversation, error) {
	// 过滤出实际的对话消息（非 file-history-snapshot，有 message 字段）
	var validMessages []ClaudeCodeMessage
	var sessionID string
	var firstTimestamp, lastTimestamp time.Time

	for _, msg := range messages {
		// 记录 sessionID
		if msg.SessionID != "" {
			sessionID = msg.SessionID
		}

		// 解析时间戳
		if msg.Timestamp != "" {
			t, err := time.Parse(time.RFC3339, msg.Timestamp)
			if err == nil {
				if firstTimestamp.IsZero() || t.Before(firstTimestamp) {
					firstTimestamp = t
				}
				if lastTimestamp.IsZero() || t.After(lastTimestamp) {
					lastTimestamp = t
				}
			}
		}

		// 只保留有 message 内容的消息，跳过 meta 和 sidechain 消息
		if msg.Message != nil && msg.Message.Role != "" && !msg.IsMeta && !msg.IsSidechain {
			validMessages = append(validMessages, msg)
		}
	}

	if len(validMessages) == 0 {
		return nil, fmt.Errorf("no valid conversation messages")
	}

	// 使用 sessionID 作为对话 UUID
	if sessionID == "" {
		return nil, fmt.Errorf("no session ID found")
	}

	// 构建消息树并计算 round_index
	parsedMessages := buildMessageTree(validMessages)

	// 提取标题（第一条 user 消息的前 50 字符）
	title := extractTitle(parsedMessages)

	// 使用第一条和最后一条消息的时间戳
	if firstTimestamp.IsZero() {
		firstTimestamp = time.Now()
	}
	if lastTimestamp.IsZero() {
		lastTimestamp = firstTimestamp
	}

	return &ParsedConversation{
		UUID:      sessionID,
		Title:     title,
		Metadata:  "", // metadata 暂时留空
		CreatedAt: firstTimestamp,
		UpdatedAt: lastTimestamp,
		Messages:  parsedMessages,
	}, nil
}

// buildMessageTree 构建消息树并计算 round_index
func buildMessageTree(messages []ClaudeCodeMessage) []ParsedMessage {
	var result []ParsedMessage
	roundIndex := 0

	for _, msg := range messages {
		// 解析时间戳
		createdAt := time.Now()
		if msg.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339, msg.Timestamp); err == nil {
				createdAt = t
			}
		}

		// user 消息时 round_index 递增
		if msg.Message.Role == "user" {
			roundIndex++
		}

		// 确定 content_type
		contentType := "text"
		content := msg.Message.Content

		// 获取 parent_uuid
		parentUUID := ""
		if msg.ParentUUID != nil {
			parentUUID = *msg.ParentUUID
		}

		parsedMsg := ParsedMessage{
			UUID:        msg.UUID,
			ParentUUID:  parentUUID,
			RoundIndex:  roundIndex,
			Role:        mapRole(msg.Message.Role),
			ContentType: contentType,
			ContentText: content,
			CreatedAt:   createdAt,
		}

		result = append(result, parsedMsg)
	}

	return result
}

// extractTitle 提取标题（第一条 user 消息的前 50 字符）
func extractTitle(messages []ParsedMessage) string {
	for _, msg := range messages {
		if msg.Role == "user" && msg.ContentText != "" {
			content := strings.TrimSpace(msg.ContentText)
			// 移除可能的 XML 标签
			if strings.Contains(content, "<") {
				// 尝试提取纯文本
				content = extractPlainText(content)
			}
			if len(content) > 50 {
				return content[:50] + "..."
			}
			if content != "" {
				return content
			}
		}
	}
	return "Untitled Conversation"
}

// extractPlainText 从可能包含 XML 标签的内容中提取纯文本
func extractPlainText(content string) string {
	// 简单实现：移除 <> 标签
	var result strings.Builder
	inTag := false
	for _, ch := range content {
		if ch == '<' {
			inTag = true
		} else if ch == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(ch)
		}
	}
	return strings.TrimSpace(result.String())
}

// mapRole 映射角色名称
func mapRole(role string) string {
	switch role {
	case "user":
		return "user"
	case "assistant":
		return "assistant"
	case "system":
		return "system"
	default:
		return role
	}
}
