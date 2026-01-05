package gemini

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// GeminiConversation Gemini对话原始结构
type GeminiConversation struct {
	Title      string         `json:"title"`
	RoundCount int            `json:"round_count"`
	TotalCount int            `json:"total_count"`
	Data       []GeminiMessage `json:"data"`
}

// GeminiMessage Gemini消息
type GeminiMessage struct {
	Role        string        `json:"role"`
	ContentType string        `json:"content_type"`
	Content     string        `json:"content"`
	Files       []GeminiFile  `json:"files,omitempty"`
}

// GeminiFile Gemini文件（图片或视频）
type GeminiFile struct {
	Type     string `json:"type"`     // "image" or "video"
	URL      string `json:"url"`
	Filename string `json:"filename"`
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
	UUID          string
	ParentUUID    string
	RoundIndex    int
	Role          string
	ContentType   string
	ContentText   string
	ContentImages []string
	ContentVideos []string
	CreatedAt     time.Time
}

// ParseJSONFile 解析Gemini JSON文件
func ParseJSONFile(filePath string, conversationID string) (*ParsedConversation, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var geminiConv GeminiConversation
	if err := json.Unmarshal(data, &geminiConv); err != nil {
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}

	return parseConversation(geminiConv, conversationID)
}

// parseConversation 解析Gemini对话
func parseConversation(geminiConv GeminiConversation, conversationID string) (*ParsedConversation, error) {
	// 当前时间作为对话时间（Gemini数据中没有时间戳）
	now := time.Now()

	conv := &ParsedConversation{
		UUID:      conversationID,
		Title:     geminiConv.Title,
		Metadata:  buildMetadata(geminiConv),
		CreatedAt: now,
		UpdatedAt: now,
		Messages:  []ParsedMessage{},
	}

	// 解析消息
	roundIndex := 1
	for i, msg := range geminiConv.Data {
		// 生成消息UUID
		msgUUID := fmt.Sprintf("%s-msg-%d", conversationID, i)

		// 解析内容和文件
		text := msg.Content
		images := []string{}
		videos := []string{}

		if msg.ContentType == "mixed" && len(msg.Files) > 0 {
			for _, file := range msg.Files {
				if file.Type == "image" {
					images = append(images, file.URL)
				} else if file.Type == "video" {
					// 视频文件名格式：{conversation_id}-{original_filename}
					videos = append(videos, fmt.Sprintf("%s-%s", conversationID, file.Filename))
				}
			}
		}

		// 确定content_type
		contentType := "text"
		if len(images) > 0 || len(videos) > 0 {
			if text != "" {
				contentType = "multipart"
			} else if len(images) > 0 {
				contentType = "image"
			} else {
				contentType = "video"
			}
		}

		parsedMsg := ParsedMessage{
			UUID:          msgUUID,
			ParentUUID:    "", // Gemini数据中没有parent信息，留空
			RoundIndex:    roundIndex,
			Role:          msg.Role,
			ContentType:   contentType,
			ContentText:   text,
			ContentImages: images,
			ContentVideos: videos,
			CreatedAt:     now.Add(time.Duration(i) * time.Second), // 模拟时间递增
		}

		conv.Messages = append(conv.Messages, parsedMsg)

		// 如果是assistant消息，round_index递增
		if msg.Role == "assistant" {
			roundIndex++
		}
	}

	return conv, nil
}

// buildMetadata 构建metadata JSON
func buildMetadata(geminiConv GeminiConversation) string {
	metadata := map[string]interface{}{
		"round_count": geminiConv.RoundCount,
		"total_count": geminiConv.TotalCount,
	}

	data, _ := json.Marshal(metadata)
	return string(data)
}
