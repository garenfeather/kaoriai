package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Claude数据结构定义
type ClaudeConversation struct {
	UUID         string              `json:"uuid"`
	Name         string              `json:"name"`
	Summary      string              `json:"summary"`
	CreatedAt    string              `json:"created_at"`
	UpdatedAt    string              `json:"updated_at"`
	ChatMessages []ClaudeChatMessage `json:"chat_messages"`
}

type ClaudeChatMessage struct {
	UUID      string          `json:"uuid"`
	Text      string          `json:"text"`
	Content   []ClaudeContent `json:"content"`
	Sender    string          `json:"sender"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
	Files     []interface{}   `json:"files"`
}

type ClaudeContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// 输出节点结构（与GPT格式保持一致）
type OutputNode struct {
	ID          string   `json:"id"`
	ParentID    string   `json:"parent_id"`
	ChildID     string   `json:"child_id"`
	Role        string   `json:"role"`
	ContentType string   `json:"content_type"`
	Content     string   `json:"content,omitempty"`
	Images      []string `json:"images,omitempty"`
	CreateTime  *string  `json:"create_time"`
}

// 输出文件结构
type OutputFile struct {
	Title      string       `json:"title"`       // 对话标题
	RoundCount int          `json:"round_count"` // 对话轮数（user/human消息数量）
	TotalCount int          `json:"total_count"` // 总消息数量
	ProjectID  string       `json:"project_id,omitempty"`
	Data       []OutputNode `json:"data"`
}

func main() {
	// 解析命令行参数
	inputFile := flag.String("input", "", "输入的JSON文件路径")
	outputDir := flag.String("output", "parsed/claude/conversation", "输出目录")
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("错误: 必须指定输入文件")
		fmt.Println("用法: claude_conversation_parse -input <file> [-output <dir>]")
		os.Exit(1)
	}

	// 读取输入文件
	data, err := ioutil.ReadFile(*inputFile)
	if err != nil {
		fmt.Printf("读取文件失败: %v\n", err)
		os.Exit(1)
	}

	// 解析JSON数组
	var conversations []ClaudeConversation
	if err := json.Unmarshal(data, &conversations); err != nil {
		fmt.Printf("解析JSON失败: %v\n", err)
		os.Exit(1)
	}

	// 创建输出目录
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("创建输出目录失败: %v\n", err)
		os.Exit(1)
	}

	// 处理每个对话
	successCount := 0
	for i, conv := range conversations {
		if err := processConversation(conv, *outputDir); err != nil {
			fmt.Printf("处理第 %d 个对话失败 (ID: %s): %v\n", i+1, conv.UUID, err)
		} else {
			successCount++
		}
	}

	fmt.Printf("处理完成: 成功 %d/%d\n", successCount, len(conversations))
}

func processConversation(conv ClaudeConversation, outputDir string) error {
	// 如果没有消息,跳过
	if len(conv.ChatMessages) == 0 {
		return fmt.Errorf("没有聊天消息")
	}

	// 转换消息为输出格式
	nodes := []OutputNode{}
	var prevID string

	for _, msg := range conv.ChatMessages {
		// 跳过system角色的消息
		role := mapSenderToRole(msg.Sender)
		if role == "system" {
			continue
		}

		node := OutputNode{
			ID:          msg.UUID,
			ParentID:    prevID,
			ChildID:     "",
			Role:        role,
			ContentType: "text",
			Content:     extractTextContent(msg.Content),
			CreateTime:  &msg.CreatedAt,
		}

		// 提取图片（如果content中有图片类型）
		images := extractImages(msg.Content)
		if len(images) > 0 {
			node.Images = images
		}

		nodes = append(nodes, node)

		// 更新上一个节点的child_id
		if len(nodes) > 1 {
			nodes[len(nodes)-2].ChildID = msg.UUID
		}

		prevID = msg.UUID
	}

	// 生成输出文件名
	conversationID := conv.UUID
	if conversationID == "" {
		conversationID = fmt.Sprintf("unknown_%s", conv.CreatedAt)
	}

	// 清理文件名中的非法字符
	conversationID = sanitizeFilename(conversationID)
	outputFile := filepath.Join(outputDir, conversationID+".json")

	// 计算统计信息
	totalCount := len(nodes)
	roundCount := 0
	for _, node := range nodes {
		if node.Role == "user" || node.Role == "human" {
			roundCount++
		}
	}

	// 提取标题
	title := conv.Name
	if title == "" {
		title = conv.Summary
	}
	if title == "" {
		title = extractTitleFromMessages(nodes)
	}

	// 构建输出文件结构
	output := OutputFile{
		Title:      title,
		RoundCount: roundCount,
		TotalCount: totalCount,
		Data:       nodes,
	}

	// 序列化为JSON
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化JSON失败: %v", err)
	}

	// 写入文件
	if err := ioutil.WriteFile(outputFile, jsonData, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	fmt.Printf("已生成: %s (共 %d 条消息)\n", outputFile, len(nodes))
	return nil
}

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

func extractTextContent(contents []ClaudeContent) string {
	var textParts []string
	for _, content := range contents {
		if content.Type == "text" && content.Text != "" {
			textParts = append(textParts, content.Text)
		}
	}
	return strings.Join(textParts, "\n")
}

func extractImages(contents []ClaudeContent) []string {
	var images []string
	// Claude 数据格式中图片可能在content中，这里预留扩展
	// 根据实际数据格式调整
	return images
}

func extractTitleFromMessages(nodes []OutputNode) string {
	// 从第一个user消息提取前50字符作为标题
	for _, node := range nodes {
		if (node.Role == "user" || node.Role == "human") && node.Content != "" {
			content := strings.TrimSpace(node.Content)
			if len(content) > 50 {
				return content[:50] + "..."
			}
			return content
		}
	}
	return "Untitled Conversation"
}

func sanitizeFilename(name string) string {
	// 替换文件名中的非法字符
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "*", "_")
	name = strings.ReplaceAll(name, "?", "_")
	name = strings.ReplaceAll(name, "\"", "_")
	name = strings.ReplaceAll(name, "<", "_")
	name = strings.ReplaceAll(name, ">", "_")
	name = strings.ReplaceAll(name, "|", "_")
	return name
}
