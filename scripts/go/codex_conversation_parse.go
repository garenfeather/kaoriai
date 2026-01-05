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

	"github.com/google/uuid"
)

// Codex 数据结构定义
type CodexMessage struct {
	Type      string         `json:"type"`
	Timestamp string         `json:"timestamp"`
	Payload   *CodexPayload  `json:"payload"`
}

type CodexPayload struct {
	Type    string          `json:"type"`
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	Summary json.RawMessage `json:"summary"`
	ID      string          `json:"id"` // session_meta 中的 id
}

type ContentItem struct {
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
	RoundCount int          `json:"round_count"` // 对话轮数（user/human消息数量）
	TotalCount int          `json:"total_count"` // 总消息数量
	ProjectID  string       `json:"project_id,omitempty"`
	Data       []OutputNode `json:"data"`
}

func main() {
	// 解析命令行参数
	inputFile := flag.String("input", "", "输入的JSONL文件路径")
	outputDir := flag.String("output", "parsed/codex/conversation", "输出目录")
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("错误: 必须指定输入文件")
		fmt.Println("用法: codex_conversation_parse -input <file> [-output <dir>]")
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
	if err := processConversation(messages, sessionID, *inputFile, *outputDir); err != nil {
		fmt.Printf("处理对话失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("处理完成")
}

func readJSONL(filename string) ([]CodexMessage, string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	var messages []CodexMessage
	var sessionID string
	scanner := bufio.NewScanner(file)

	// 增加缓冲区大小以处理长行
	const maxCapacity = 2 * 1024 * 1024 // 2MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

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
			messages = append(messages, msg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, "", err
	}

	return messages, sessionID, nil
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

	// 生成 UUID v5
	messageUUID := uuid.NewSHA1(namespaceUUID, []byte(name))
	return messageUUID.String(), nil
}

func processConversation(messages []CodexMessage, sessionID string, inputFile string, outputDir string) error {
	if len(messages) == 0 {
		return fmt.Errorf("没有有效的消息节点")
	}

	// 转换为输出格式
	nodes := []OutputNode{}
	var prevID string

	for _, msg := range messages {
		content := extractContent(msg.Payload)

		// 如果内容为空，跳过
		if content == "" {
			continue
		}

		// 使用新的 UUID v5 方案生成节点ID
		nodeID, err := generateMessageUUID(sessionID, msg.Timestamp, msg.Payload)
		if err != nil {
			return fmt.Errorf("generate message UUID: %w", err)
		}

		parentID := prevID
		childID := ""

		node := OutputNode{
			ID:          nodeID,
			ParentID:    parentID,
			ChildID:     childID,
			Role:        msg.Payload.Role,
			ContentType: "text",
			Content:     content,
			CreateTime:  &msg.Timestamp,
		}

		nodes = append(nodes, node)

		// 更新上一个节点的child_id
		if len(nodes) >= 2 {
			nodes[len(nodes)-2].ChildID = nodeID
		}

		prevID = nodeID
	}

	if len(nodes) == 0 {
		return fmt.Errorf("没有有效的内容节点")
	}

	// 生成输出文件名
	// 从输入文件名提取session ID（只保留最后的UUID部分）
	inputFileName := filepath.Base(inputFile)
	if inputFileName != "" {
		// 去掉 .jsonl 后缀
		baseName := strings.TrimSuffix(inputFileName, ".jsonl")
		// 文件名格式：rollout-2025-10-14T01-04-12-0199de87-9743-7533-afcd-751a16622fca
		// 提取最后的UUID部分（最后5段，用-连接）
		parts := strings.Split(baseName, "-")
		if len(parts) >= 5 {
			// 取最后5段作为 UUID
			sessionID = strings.Join(parts[len(parts)-5:], "-")
		} else {
			sessionID = baseName
		}
	}

	if sessionID == "" {
		sessionID = messages[0].Timestamp
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

func extractContent(payload *CodexPayload) string {
	if payload == nil || payload.Content == nil {
		return ""
	}

	// 尝试将content解析为数组
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

	// 尝试将content解析为字符串
	var contentStr string
	if err := json.Unmarshal(payload.Content, &contentStr); err == nil {
		return contentStr
	}

	return ""
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
