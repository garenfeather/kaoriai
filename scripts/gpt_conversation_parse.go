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

// 数据结构定义
type Conversation struct {
	Title           string                 `json:"title"`
	CreateTime      float64                `json:"create_time"`
	UpdateTime      float64                `json:"update_time"`
	Mapping         map[string]MappingNode `json:"mapping"`
	CurrentNode     string                 `json:"current_node"`
	ConversationID  string                 `json:"conversation_id"`
	ID              string                 `json:"id"`
	GizmoID         *string                `json:"gizmo_id"`
}

type MappingNode struct {
	ID       string   `json:"id"`
	Message  *Message `json:"message"`
	Parent   *string  `json:"parent"`
	Children []string `json:"children"`
}

type Message struct {
	ID         string  `json:"id"`
	Author     Author  `json:"author"`
	CreateTime *float64 `json:"create_time"`
	UpdateTime *float64 `json:"update_time"`
	Content    Content `json:"content"`
	Status     string  `json:"status"`
}

type Author struct {
	Role string `json:"role"`
	Name *string `json:"name"`
}

type Content struct {
	ContentType string        `json:"content_type"`
	Parts       []interface{} `json:"parts"`
}

// 输出节点结构
type OutputNode struct {
	ID          string   `json:"id"`
	ParentID    string   `json:"parent_id"`
	ChildID     string   `json:"child_id"`
	Role        string   `json:"role"`
	ContentType string   `json:"content_type"`
	Content     string   `json:"content,omitempty"`
	Images      []string `json:"images,omitempty"`
	CreateTime  *float64 `json:"create_time"`
}

// 输出文件结构
type OutputFile struct {
	RoundCount int          `json:"round_count"` // 对话轮数（user/human消息数量）
	TotalCount int          `json:"total_count"` // 总消息数量
	ProjectID  string       `json:"project_id,omitempty"`
	Data       []OutputNode `json:"data"`
}

func main() {
	// 解析命令行参数
	inputFile := flag.String("input", "", "输入的JSON文件路径")
	outputDir := flag.String("output", "parsed/gpt/conversation", "输出目录")
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("错误: 必须指定输入文件")
		fmt.Println("用法: gpt_conversation_parse -input <file> [-output <dir>]")
		os.Exit(1)
	}

	// 读取输入文件
	data, err := ioutil.ReadFile(*inputFile)
	if err != nil {
		fmt.Printf("读取文件失败: %v\n", err)
		os.Exit(1)
	}

	// 解析JSON数组
	var conversations []Conversation
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
			fmt.Printf("处理第 %d 个对话失败 (ID: %s): %v\n", i+1, conv.ConversationID, err)
		} else {
			successCount++
		}
	}

	fmt.Printf("处理完成: 成功 %d/%d\n", successCount, len(conversations))
}

func processConversation(conv Conversation, outputDir string) error {
	// 如果没有current_node,跳过
	if conv.CurrentNode == "" {
		return fmt.Errorf("没有current_node")
	}

	// 从current_node开始向上追溯
	nodes := []OutputNode{}
	nodeIDList := []string{} // 保存节点ID顺序
	currentID := conv.CurrentNode

	for currentID != "" {
		node, exists := conv.Mapping[currentID]
		if !exists {
			break
		}

		// 提取消息信息
		if node.Message != nil {
			outputNode := extractNode(node)

			// 设置ID
			outputNode.ID = node.ID

			// 设置ParentID (来自原始数据)
			if node.Parent != nil && *node.Parent != "" {
				outputNode.ParentID = *node.Parent
			} else {
				outputNode.ParentID = ""
			}

			// 添加到列表头部(因为是从尾部向头部追溯)
			nodes = append([]OutputNode{outputNode}, nodes...)
			nodeIDList = append([]string{currentID}, nodeIDList...)
		}

		// 移动到父节点
		if node.Parent == nil || *node.Parent == "" {
			break
		}
		currentID = *node.Parent
	}

	// 如果没有有效节点,跳过
	if len(nodes) == 0 {
		return fmt.Errorf("没有有效的消息节点")
	}

	// 根据parent关系补全child_id
	for i := 0; i < len(nodes); i++ {
		nodes[i].ChildID = ""
		// 查找当前节点的子节点（即parent_id指向当前节点id的节点）
		for j := 0; j < len(nodes); j++ {
			if nodes[j].ParentID == nodes[i].ID {
				nodes[i].ChildID = nodes[j].ID
				break // 每个节点只有一个child
			}
		}
	}

	// 生成输出文件名
	conversationID := conv.ConversationID
	if conversationID == "" {
		conversationID = conv.ID
	}
	if conversationID == "" {
		conversationID = fmt.Sprintf("unknown_%d", conv.CreateTime)
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

	// 构建输出文件结构
	output := OutputFile{
		RoundCount: roundCount,
		TotalCount: totalCount,
		Data:       nodes,
	}
	// 如果gizmo_id不为null，则添加到输出
	if conv.GizmoID != nil && *conv.GizmoID != "" {
		output.ProjectID = *conv.GizmoID
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

func extractNode(node MappingNode) OutputNode {
	output := OutputNode{}

	if node.Message == nil {
		return output
	}

	// 提取角色
	output.Role = node.Message.Author.Role

	// 提取content_type
	output.ContentType = node.Message.Content.ContentType

	// 提取parts中的文本内容和图片
	var textParts []string
	var images []string
	for _, part := range node.Message.Content.Parts {
		switch v := part.(type) {
		case string:
			// 直接字符串
			if v != "" {
				textParts = append(textParts, v)
			}
		case map[string]interface{}:
			// 如果是对象,尝试提取文本字段
			if text, ok := v["text"].(string); ok && text != "" {
				textParts = append(textParts, text)
			}
			// 对于image_asset_pointer等类型,提取图片ID
			if contentType, ok := v["content_type"].(string); ok {
				if contentType == "image_asset_pointer" {
					// 提取asset_pointer字段
					if assetPointer, ok := v["asset_pointer"].(string); ok && assetPointer != "" {
						images = append(images, assetPointer)
					}
					textParts = append(textParts, "[图片]")
				}
			}
		}
	}

	output.Content = strings.Join(textParts, "\n")
	if len(images) > 0 {
		output.Images = images
	}

	// 提取create_time
	output.CreateTime = node.Message.CreateTime

	return output
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
