package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// 对话节点结构（来自gpt_conversation_parse.go输出）
type ConversationNode struct {
	ID          string   `json:"id"`
	ParentID    string   `json:"parent_id"`
	ChildID     string   `json:"child_id"`
	Role        string   `json:"role"`
	ContentType string   `json:"content_type"`
	Content     string   `json:"content"`
	Images      []string `json:"images,omitempty"`
	CreateTime  *float64 `json:"create_time"`
}

// 对话文件结构
type ConversationFile struct {
	ProjectID string             `json:"project_id,omitempty"`
	Data      []ConversationNode `json:"data"`
}

// 树节点结构
type TreeNode struct {
	Parent        string   `json:"parent"`
	Children      []string `json:"children"`
	Conversations []string `json:"conversations,omitempty"`
}

// 树结构
type Tree struct {
	Root          string              `json:"root"`
	Conversations []string            `json:"conversations"`
	Nodes         map[string]TreeNode `json:"nodes"`
}

func main() {
	// 解析命令行参数
	inputFiles := flag.String("input", "", "输入的JSON文件路径，多个文件用逗号分隔")
	outputDir := flag.String("output", "parsed/gpt/tree", "输出目录")
	flag.Parse()

	if *inputFiles == "" {
		fmt.Println("错误: 必须指定输入文件")
		fmt.Println("用法: gpt_branch_tree_merge -input <file1,file2,...> [-output <dir>]")
		os.Exit(1)
	}

	// 创建输出目录
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("创建输出目录失败: %v\n", err)
		os.Exit(1)
	}

	// 分割文件列表
	fileList := strings.Split(*inputFiles, ",")
	for i := range fileList {
		fileList[i] = strings.TrimSpace(fileList[i])
	}

	if len(fileList) < 2 {
		fmt.Println("错误: 至少需要两个输入文件进行合并")
		os.Exit(1)
	}

	// 读取第一个文件并判断类型
	firstFileData, err := ioutil.ReadFile(fileList[0])
	if err != nil {
		fmt.Printf("读取文件失败 %s: %v\n", fileList[0], err)
		os.Exit(1)
	}

	var tree *Tree
	var conversations [][]ConversationNode
	var conversationIDs []string

	// 尝试解析为树结构
	if isTreeStructure(firstFileData) {
		tree = &Tree{}
		if err := json.Unmarshal(firstFileData, tree); err != nil {
			fmt.Printf("解析树结构失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("检测到已有树结构，包含 %d 个对话\n", len(tree.Conversations))

		// 读取后续对话文件
		for i := 1; i < len(fileList); i++ {
			conv, convID, err := readConversationFile(fileList[i])
			if err != nil {
				fmt.Printf("读取对话文件失败 %s: %v\n", fileList[i], err)
				os.Exit(1)
			}
			conversations = append(conversations, conv)
			conversationIDs = append(conversationIDs, convID)
		}
	} else {
		// 所有文件都是对话文件
		fmt.Printf("检测到 %d 个对话文件，开始合并\n", len(fileList))
		for _, file := range fileList {
			conv, convID, err := readConversationFile(file)
			if err != nil {
				fmt.Printf("读取对话文件失败 %s: %v\n", file, err)
				os.Exit(1)
			}
			conversations = append(conversations, conv)
			conversationIDs = append(conversationIDs, convID)
		}
	}

	// 执行合并
	var result *Tree
	if tree != nil {
		// 合并到已有树
		result, err = mergeConversationsToTree(tree, conversations, conversationIDs)
	} else {
		// 创建新树
		result, err = createTreeFromConversations(conversations, conversationIDs)
	}

	if err != nil {
		fmt.Printf("合并失败: %v\n", err)
		os.Exit(1)
	}

	// 计算统计信息
	stats := calculateStatistics(result, conversations)
	fmt.Printf("\n合并成功！\n")
	fmt.Printf("共同节点数量: %d\n", stats.CommonNodeCount)
	fmt.Printf("最长树枝长度: %d\n", stats.MaxDepth)
	fmt.Printf("共同节点占比: %.2f%%\n", stats.Percentage)

	// 生成输出文件名（使用root id）
	outputFileName := result.Root + ".json"
	outputPath := *outputDir + "/" + outputFileName

	// 写入输出文件
	outputData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("序列化JSON失败: %v\n", err)
		os.Exit(1)
	}

	if err := ioutil.WriteFile(outputPath, outputData, 0644); err != nil {
		fmt.Printf("写入文件失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("结果已保存到: %s\n", outputPath)
}

// 判断是否为树结构
func isTreeStructure(data []byte) bool {
	var temp map[string]interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return false
	}
	_, hasRoot := temp["root"]
	_, hasNodes := temp["nodes"]
	_, hasConversations := temp["conversations"]
	return hasRoot && hasNodes && hasConversations
}

// 读取对话文件
func readConversationFile(filepath string) ([]ConversationNode, string, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, "", err
	}

	// 尝试解析新格式（带data字段的包装格式）
	var convFile ConversationFile
	if err := json.Unmarshal(data, &convFile); err != nil {
		return nil, "", err
	}

	// 从文件名提取conversation ID
	filename := filepath[strings.LastIndex(filepath, "/")+1:]
	conversationID := strings.TrimSuffix(filename, ".json")

	return convFile.Data, conversationID, nil
}

// 从多个对话创建新树
func createTreeFromConversations(conversations [][]ConversationNode, conversationIDs []string) (*Tree, error) {
	if len(conversations) == 0 {
		return nil, fmt.Errorf("没有对话数据")
	}

	tree := &Tree{
		Conversations: conversationIDs,
		Nodes:         make(map[string]TreeNode),
	}

	// 查找共同根节点
	rootID, err := findCommonRoot(conversations)
	if err != nil {
		return nil, err
	}
	tree.Root = rootID

	// 构建节点映射
	nodeConvMap := make(map[string][]string) // 记录每个节点属于哪些对话

	for i, conv := range conversations {
		convID := conversationIDs[i]
		for _, node := range conv {
			if _, exists := tree.Nodes[node.ID]; !exists {
				tree.Nodes[node.ID] = TreeNode{
					Parent:   node.ParentID,
					Children: []string{},
				}
			}
			nodeConvMap[node.ID] = append(nodeConvMap[node.ID], convID)
		}
	}

	// 构建children关系
	for nodeID, node := range tree.Nodes {
		for otherID, otherNode := range tree.Nodes {
			if otherNode.Parent == nodeID {
				// 检查是否已存在
				found := false
				for _, childID := range node.Children {
					if childID == otherID {
						found = true
						break
					}
				}
				if !found {
					node.Children = append(node.Children, otherID)
					tree.Nodes[nodeID] = node
				}
			}
		}
	}

	// 设置conversations字段（非所有对话共同节点）
	totalConvCount := len(conversationIDs)
	for nodeID, node := range tree.Nodes {
		convList := nodeConvMap[nodeID]
		if len(convList) < totalConvCount {
			node.Conversations = convList
			tree.Nodes[nodeID] = node
		}
	}

	return tree, nil
}

// 合并对话到已有树
func mergeConversationsToTree(tree *Tree, conversations [][]ConversationNode, conversationIDs []string) (*Tree, error) {
	// 检查新对话是否可以合并
	for i, conv := range conversations {
		if !canMerge(tree, conv) {
			return nil, fmt.Errorf("对话 %s 无法合并到现有树结构（没有共同节点）", conversationIDs[i])
		}
	}

	// 更新conversations列表
	for _, convID := range conversationIDs {
		if !contains(tree.Conversations, convID) {
			tree.Conversations = append(tree.Conversations, convID)
		}
	}

	// 构建节点所属对话映射
	nodeConvMap := make(map[string][]string)

	// 先统计现有树中的节点
	for nodeID, node := range tree.Nodes {
		if len(node.Conversations) > 0 {
			nodeConvMap[nodeID] = node.Conversations
		} else {
			// 如果conversations为空，说明是所有已有对话的共同节点
			nodeConvMap[nodeID] = tree.Conversations[:len(tree.Conversations)-len(conversationIDs)]
		}
	}

	// 合并新对话的节点
	for i, conv := range conversations {
		convID := conversationIDs[i]
		for _, node := range conv {
			// 添加或更新节点
			if _, exists := tree.Nodes[node.ID]; exists {
				// 节点已存在，更新所属对话
				nodeConvMap[node.ID] = append(nodeConvMap[node.ID], convID)
			} else {
				// 新节点
				tree.Nodes[node.ID] = TreeNode{
					Parent:   node.ParentID,
					Children: []string{},
				}
				nodeConvMap[node.ID] = []string{convID}
			}

			// 更新父节点的children
			if node.ParentID != "" {
				if parentNode, exists := tree.Nodes[node.ParentID]; exists {
					if !contains(parentNode.Children, node.ID) {
						parentNode.Children = append(parentNode.Children, node.ID)
						tree.Nodes[node.ParentID] = parentNode
					}
				}
			}
		}
	}

	// 更新所有节点的conversations字段
	totalConvCount := len(tree.Conversations)
	for nodeID, node := range tree.Nodes {
		convList := nodeConvMap[nodeID]
		// 去重
		convList = uniqueStrings(convList)

		if len(convList) < totalConvCount {
			node.Conversations = convList
		} else {
			node.Conversations = nil // 所有对话的共同节点
		}
		tree.Nodes[nodeID] = node
	}

	return tree, nil
}

// 查找共同根节点
func findCommonRoot(conversations [][]ConversationNode) (string, error) {
	if len(conversations) == 0 {
		return "", fmt.Errorf("没有对话数据")
	}

	// 收集第一个对话的所有节点ID
	firstConvNodes := make(map[string]bool)
	for _, node := range conversations[0] {
		firstConvNodes[node.ID] = true
	}

	// 找到所有对话都包含的节点
	commonNodes := make(map[string]bool)
	for nodeID := range firstConvNodes {
		isCommon := true
		for i := 1; i < len(conversations); i++ {
			found := false
			for _, node := range conversations[i] {
				if node.ID == nodeID {
					found = true
					break
				}
			}
			if !found {
				isCommon = false
				break
			}
		}
		if isCommon {
			commonNodes[nodeID] = true
		}
	}

	if len(commonNodes) == 0 {
		return "", fmt.Errorf("对话之间没有共同节点，无法合并")
	}

	// 找到最顶层的共同节点（parent_id为空或不在共同节点中）
	for _, node := range conversations[0] {
		if commonNodes[node.ID] {
			if node.ParentID == "" || !commonNodes[node.ParentID] {
				return node.ID, nil
			}
		}
	}

	// 如果没找到，返回第一个共同节点
	for nodeID := range commonNodes {
		return nodeID, nil
	}

	return "", fmt.Errorf("无法确定根节点")
}

// 检查对话是否可以合并到树
func canMerge(tree *Tree, conversation []ConversationNode) bool {
	for _, node := range conversation {
		if _, exists := tree.Nodes[node.ID]; exists {
			return true
		}
	}
	return false
}

// 统计信息
type Statistics struct {
	CommonNodeCount int
	MaxDepth        int
	Percentage      float64
}

// 计算统计信息
func calculateStatistics(tree *Tree, conversations [][]ConversationNode) Statistics {
	stats := Statistics{}

	// 找到最长对话的节点数
	maxLen := 0
	for _, conv := range conversations {
		if len(conv) > maxLen {
			maxLen = len(conv)
		}
	}
	stats.MaxDepth = maxLen

	// 计算共同节点数量（从root开始到第一个分叉点）
	commonCount := 0
	currentID := tree.Root

	for currentID != "" {
		node := tree.Nodes[currentID]

		// 检查是否所有对话都包含此节点
		isCommon := true
		if len(node.Conversations) > 0 {
			isCommon = false
		}

		if isCommon {
			commonCount++
			// 如果有多个children，说明开始分叉了
			if len(node.Children) > 1 {
				break
			}
			// 继续沿着单一路径
			if len(node.Children) == 1 {
				currentID = node.Children[0]
			} else {
				break
			}
		} else {
			break
		}
	}

	stats.CommonNodeCount = commonCount
	if stats.MaxDepth > 0 {
		stats.Percentage = float64(commonCount) / float64(stats.MaxDepth) * 100
	}

	return stats
}

// 辅助函数：检查字符串是否在切片中
func contains(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

// 辅助函数：字符串切片去重
func uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
