package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Claude Code 数据结构定义
type ClaudeCodeMessage struct {
	Type        string                 `json:"type"`
	UUID        string                 `json:"uuid"`
	ParentUUID  *string                `json:"parentUuid"`
	SessionID   string                 `json:"sessionId"`
	Timestamp   string                 `json:"timestamp"`
	Message     *MessageContent        `json:"message"`
	IsMeta      bool                   `json:"isMeta"`
	Children    []string               `json:"-"` // 用于构建关系
}

type MessageContent struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// 输出节点结构（与GPT格式保持一致）
type OutputNode struct {
	ID          string                 `json:"id"`
	ParentID    string                 `json:"parent_id"`
	ChildID     string                 `json:"child_id"`
	Role        string                 `json:"role"`
	ContentType string                 `json:"content_type"`
	Content     string                 `json:"content,omitempty"`
	Images      []string               `json:"images,omitempty"`
	ToolData    map[string]interface{} `json:"tool_data,omitempty"`
	CreateTime  *string                `json:"create_time"`
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
	inputFile := flag.String("input", "", "输入的JSONL文件路径")
	outputDir := flag.String("output", "parsed/claude_code/conversation", "输出目录")
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("错误: 必须指定输入文件")
		fmt.Println("用法: claude_code_conversation_parse -input <file> [-output <dir>]")
		os.Exit(1)
	}

	// 读取并解析JSONL文件
	messages, sessionID, err := readJSONL(*inputFile)
	if err != nil {
		fmt.Printf("读取文件失败: %v\n", err)
		os.Exit(1)
	}

	if len(messages) == 0 {
		fmt.Println("警告: 文件中没有有效的消息")
		os.Exit(0)
	}

	// 创建输出目录
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("创建输出目录失败: %v\n", err)
		os.Exit(1)
	}

	// 处理对话
	if err := processConversation(messages, sessionID, *outputDir); err != nil {
		fmt.Printf("处理对话失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("处理完成")
}

func readJSONL(filename string) ([]ClaudeCodeMessage, string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	var messages []ClaudeCodeMessage
	var sessionID string
	scanner := bufio.NewScanner(file)

	// 增加缓冲区大小以处理长行
	const maxCapacity = 1024 * 1024 // 1MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg ClaudeCodeMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// 忽略无法解析的行
			continue
		}

		// 只处理有消息内容的记录
		if msg.Message != nil && msg.UUID != "" {
			// 跳过元数据消息
			if msg.IsMeta {
				continue
			}
			messages = append(messages, msg)
			if sessionID == "" && msg.SessionID != "" {
				sessionID = msg.SessionID
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, "", err
	}

	return messages, sessionID, nil
}

func processConversation(messages []ClaudeCodeMessage, sessionID string, outputDir string) error {
	if len(messages) == 0 {
		return fmt.Errorf("没有有效的消息节点")
	}

	// 构建父子关系
	msgMap := make(map[string]*ClaudeCodeMessage)
	for i := range messages {
		msgMap[messages[i].UUID] = &messages[i]
	}

	// 构建子节点列表
	for i := range messages {
		if messages[i].ParentUUID != nil && *messages[i].ParentUUID != "" {
			if parent, exists := msgMap[*messages[i].ParentUUID]; exists {
				parent.Children = append(parent.Children, messages[i].UUID)
			}
		}
	}

	// 转换为输出格式
	nodes := []OutputNode{}
	for _, msg := range messages {
		// 跳过system角色的消息
		if msg.Message != nil && msg.Message.Role == "system" {
			continue
		}

		contentType, content, toolData := extractContent(msg.Message)

		// 如果内容为空且没有tool_data，跳过
		if content == "" && toolData == nil {
			continue
		}

		parentID := ""
		if msg.ParentUUID != nil {
			parentID = *msg.ParentUUID
		}

		childID := ""
		if len(msg.Children) > 0 {
			childID = msg.Children[0] // 取第一个子节点
		}

		node := OutputNode{
			ID:          msg.UUID,
			ParentID:    parentID,
			ChildID:     childID,
			Role:        msg.Message.Role,
			ContentType: contentType,
			Content:     content,
			ToolData:    toolData,
			CreateTime:  &msg.Timestamp,
		}

		nodes = append(nodes, node)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("没有有效的内容节点")
	}

	// 生成输出文件名
	if sessionID == "" {
		sessionID = messages[0].UUID
	}
	sessionID = sanitizeFilename(sessionID)
	outputFile := filepath.Join(outputDir, sessionID+".json")

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

func extractContent(msg *MessageContent) (string, string, map[string]interface{}) {
	if msg == nil {
		return "text", "", nil
	}

	// 尝试将content解析为字符串
	var contentStr string
	if err := json.Unmarshal(msg.Content, &contentStr); err == nil {
		return "text", contentStr, nil
	}

	// 尝试将content解析为数组
	var contentArray []interface{}
	if err := json.Unmarshal(msg.Content, &contentArray); err == nil {
		var textParts []string
		var contentType string = "text"
		var toolData map[string]interface{}

		for _, item := range contentArray {
			if itemMap, ok := item.(map[string]interface{}); ok {
				// 获取type字段
				if typeVal, ok := itemMap["type"].(string); ok && typeVal != "" {
					// 优先使用非text类型
					if contentType == "text" || typeVal != "text" {
						contentType = typeVal
					}
				}

				// 提取tool_use的数据
				if typeVal, ok := itemMap["type"].(string); ok && typeVal == "tool_use" {
					if name, ok := itemMap["name"].(string); ok && name != "" {
						toolData = make(map[string]interface{})
						toolData["name"] = name

						if input, ok := itemMap["input"].(map[string]interface{}); ok {
							// 特殊处理TodoWrite
							if name == "TodoWrite" {
								if todosData, ok := input["todos"].([]interface{}); ok {
									var todos []map[string]interface{}
									for _, todoItem := range todosData {
										if todoMap, ok := todoItem.(map[string]interface{}); ok {
											status, _ := todoMap["status"].(string)
											activeForm, _ := todoMap["activeForm"].(string)
											if status != "" && activeForm != "" {
												todos = append(todos, map[string]interface{}{
													"status":     status,
													"activeForm": activeForm,
												})
											}
										}
									}
									if len(todos) > 0 {
										toolData["todos"] = todos
									}
								}
							} else {
								// 其他tool直接保存input
								toolData["input"] = input
							}
						}
					}
				}

				// 提取text字段
				if text, ok := itemMap["text"].(string); ok && text != "" {
					textParts = append(textParts, text)
				}
				// 提取tool_result内容
				if content, ok := itemMap["content"].(string); ok && content != "" {
					textParts = append(textParts, content)
				}
			} else if str, ok := item.(string); ok {
				textParts = append(textParts, str)
			}
		}
		return contentType, strings.Join(textParts, "\n"), toolData
	}

	// 尝试将content解析为对象
	var contentObj map[string]interface{}
	if err := json.Unmarshal(msg.Content, &contentObj); err == nil {
		contentType := "text"
		if typeVal, ok := contentObj["type"].(string); ok {
			contentType = typeVal
		}

		if text, ok := contentObj["text"].(string); ok {
			return contentType, text, nil
		}
	}

	return "text", "", nil
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
