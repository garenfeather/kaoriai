package gpt

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// GPTConversation GPT对话原始结构
type GPTConversation struct {
	Title          string                    `json:"title"`
	CreateTime     float64                   `json:"create_time"`
	UpdateTime     float64                   `json:"update_time"`
	Mapping        map[string]GPTMappingNode `json:"mapping"`
	CurrentNode    string                    `json:"current_node"`
	ConversationID string                    `json:"conversation_id"`
	ID             string                    `json:"id"`
	GizmoID        *string                   `json:"gizmo_id"`
}

// GPTMappingNode GPT mapping节点
type GPTMappingNode struct {
	ID       string      `json:"id"`
	Message  *GPTMessage `json:"message"`
	Parent   *string     `json:"parent"`
	Children []string    `json:"children"`
}

// GPTMessage GPT消息
type GPTMessage struct {
	ID         string      `json:"id"`
	Author     GPTAuthor   `json:"author"`
	CreateTime *float64    `json:"create_time"`
	UpdateTime *float64    `json:"update_time"`
	Content    GPTContent  `json:"content"`
	Status     string      `json:"status"`
}

// GPTAuthor 消息作者
type GPTAuthor struct {
	Role string  `json:"role"`
	Name *string `json:"name"`
}

// GPTContent 消息内容
type GPTContent struct {
	ContentType string        `json:"content_type"`
	Parts       []interface{} `json:"parts"`
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
	UUID            string
	ParentUUID      string
	ChildUUID       string
	RoundIndex      int
	Role            string
	ContentType     string
	ContentText     string
	ContentImages   []string
	CreatedAt       time.Time
}

// ParseConversationsFile 解析conversations.json文件
func ParseConversationsFile(filePath string) ([]ParsedConversation, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var conversations []GPTConversation
	if err := json.Unmarshal(data, &conversations); err != nil {
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}

	var results []ParsedConversation
	for _, conv := range conversations {
		parsed, err := parseConversation(conv)
		if err != nil {
			// 记录错误但继续处理其他对话
			fmt.Printf("Error parsing conversation %s: %v\n", conv.ID, err)
			continue
		}
		results = append(results, parsed)
	}

	return results, nil
}

// parseConversation 解析单个对话
func parseConversation(conv GPTConversation) (ParsedConversation, error) {
	result := ParsedConversation{
		UUID:      conv.ID,
		Title:     conv.Title,
		CreatedAt: time.Unix(int64(conv.CreateTime), 0),
		UpdatedAt: time.Unix(int64(conv.UpdateTime), 0),
	}

	// 提取title
	if result.Title == "" {
		result.Title = extractTitleFromConversation(conv)
	}

	// 构建metadata
	metadata := map[string]interface{}{
		"conversation_id": conv.ConversationID,
	}
	if conv.GizmoID != nil {
		metadata["gizmo_id"] = *conv.GizmoID
	}
	metadataBytes, _ := json.Marshal(metadata)
	result.Metadata = string(metadataBytes)

	// 解析消息
	messages, err := parseMessages(conv)
	if err != nil {
		return result, err
	}
	result.Messages = messages

	return result, nil
}

// extractTitleFromConversation 从对话中提取标题
func extractTitleFromConversation(conv GPTConversation) string {
	// 尝试从mapping中找到第一条user消息
	if conv.CurrentNode == "" {
		return "Untitled Conversation"
	}

	// 从current_node开始回溯到根节点
	visited := make(map[string]bool)
	currentID := conv.CurrentNode

	for currentID != "" {
		if visited[currentID] {
			break
		}
		visited[currentID] = true

		node, exists := conv.Mapping[currentID]
		if !exists {
			break
		}

		// 检查是否是user消息
		if node.Message != nil && node.Message.Author.Role == "user" {
			if text := extractTextFromParts(node.Message.Content.Parts); text != "" {
				if len(text) > 50 {
					return text[:50] + "..."
				}
				return text
			}
		}

		// 移动到父节点
		if node.Parent == nil || *node.Parent == "" {
			break
		}
		currentID = *node.Parent
	}

	return "Untitled Conversation"
}

// parseMessages 解析消息列表
func parseMessages(conv GPTConversation) ([]ParsedMessage, error) {
	if conv.CurrentNode == "" {
		return []ParsedMessage{}, nil
	}

	// 从current_node回溯到根节点，构建消息链
	var chain []string
	visited := make(map[string]bool)
	currentID := conv.CurrentNode

	for currentID != "" {
		if visited[currentID] {
			break
		}
		visited[currentID] = true

		node, exists := conv.Mapping[currentID]
		if !exists {
			break
		}

		// 跳过system消息
		if node.Message != nil && node.Message.Author.Role == "system" {
			if node.Parent == nil || *node.Parent == "" {
				break
			}
			currentID = *node.Parent
			continue
		}

		chain = append([]string{currentID}, chain...)

		if node.Parent == nil || *node.Parent == "" {
			break
		}
		currentID = *node.Parent
	}

	// 计算round_index
	roundIndex := 0
	var messages []ParsedMessage

	for _, nodeID := range chain {
		node := conv.Mapping[nodeID]

		if node.Message == nil {
			continue
		}

		// 跳过system消息
		if node.Message.Author.Role == "system" {
			continue
		}

		// user消息时，round_index递增
		if node.Message.Author.Role == "user" {
			roundIndex++
		}

		// 解析消息内容
		text, images := parseContent(node.Message.Content)

		// 确定content_type
		contentType := "text"
		if len(images) > 0 {
			if text != "" {
				contentType = "multipart"
			} else {
				contentType = "image"
			}
		}

		// 获取child_uuid
		childUUID := ""
		if len(node.Children) > 0 {
			childUUID = node.Children[0]
		}

		// 获取创建时间
		createdAt := time.Unix(int64(conv.CreateTime), 0)
		if node.Message.CreateTime != nil {
			createdAt = time.Unix(int64(*node.Message.CreateTime), 0)
		}

		msg := ParsedMessage{
			UUID:          node.ID,
			ParentUUID:    getParentUUID(node.Parent),
			ChildUUID:     childUUID,
			RoundIndex:    roundIndex,
			Role:          mapRole(node.Message.Author.Role),
			ContentType:   contentType,
			ContentText:   text,
			ContentImages: images,
			CreatedAt:     createdAt,
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// parseContent 解析消息内容
func parseContent(content GPTContent) (string, []string) {
	text := extractTextFromParts(content.Parts)
	images := extractImagesFromParts(content.Parts)
	return text, images
}

// extractTextFromParts 从parts中提取文本
func extractTextFromParts(parts []interface{}) string {
	var texts []string
	for _, part := range parts {
		switch v := part.(type) {
		case string:
			if v != "" {
				texts = append(texts, v)
			}
		case map[string]interface{}:
			// 可能是结构化数据，尝试提取text字段
			if textVal, ok := v["text"].(string); ok && textVal != "" {
				texts = append(texts, textVal)
			}
		}
	}
	return strings.Join(texts, "\n")
}

// extractImagesFromParts 从parts中提取图片引用
func extractImagesFromParts(parts []interface{}) []string {
	var images []string
	for _, part := range parts {
		if m, ok := part.(map[string]interface{}); ok {
			// 处理 sediment:// 格式的图片
			if assetPointer, ok := m["asset_pointer"].(string); ok {
				if strings.HasPrefix(assetPointer, "sediment://") {
					// 提取文件ID
					fileID := strings.TrimPrefix(assetPointer, "sediment://")
					// 转换为存储格式
					images = append(images, fileID+"-sanitized")
				} else if strings.HasPrefix(assetPointer, "file-service://") {
					// 提取文件ID
					fileID := strings.TrimPrefix(assetPointer, "file-service://")
					images = append(images, fileID)
				}
			}

			// 处理其他可能的图片字段
			if fileID, ok := m["file_id"].(string); ok && fileID != "" {
				images = append(images, fileID)
			}
		}
	}
	return images
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
	case "tool":
		return "tool"
	default:
		return role
	}
}

// getParentUUID 获取父节点UUID
func getParentUUID(parent *string) string {
	if parent == nil {
		return ""
	}
	return *parent
}
