// Package main 实现简单的对话文件服务器
// 这是一个原始的 HTTP 服务器,用于提供对话 JSON 文件的访问
// 注意: 这是早期实现,与 backend/cmd/api 是两个独立的服务器
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// 常量定义
const (
	Port       = "8080"     // HTTP 服务器监听端口
	ParsedDir  = "parsed"   // 解析后的对话数据目录
	ProjectDir = ".."       // 项目根目录的相对路径
)

// projectRoot 全局变量,存储项目根目录的绝对路径
var projectRoot string

// init 在 main 函数之前自动执行,用于初始化项目根目录路径
func init() {
	// 获取当前可执行文件的路径
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal("无法获取执行路径:", err)
	}
	execDir := filepath.Dir(execPath)

	// 假设 backend 目录在项目根目录下,向上查找一级
	projectRoot = filepath.Join(execDir, ProjectDir)

	// 如果是开发模式(使用 go run 命令),路径会包含 "go-build"
	// 此时使用当前工作目录的上级作为项目根目录
	if strings.Contains(execPath, "go-build") {
		wd, err := os.Getwd()
		if err == nil {
			projectRoot = filepath.Dir(wd)
		}
	}

	log.Printf("项目根目录: %s", projectRoot)
}

// ConversationHandler 处理对话请求的 HTTP 处理器
// 路径格式: /{source}/{conversation_id}
// 例如: /gpt/d4d4ddf6-5452-4dbb-9c1c-8a59ebfdb8fa
func ConversationHandler(w http.ResponseWriter, r *http.Request) {
	// === 设置 CORS 头,允许跨域访问 ===
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理 OPTIONS 预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 仅支持 GET 请求
	if r.Method != "GET" {
		http.Error(w, "仅支持 GET 请求", http.StatusMethodNotAllowed)
		return
	}

	// === 解析 URL 路径 ===
	// 路径格式: /{source}/{conversation_id}
	// source: 数据来源(gpt, claude, claude_code, codex)
	// conversation_id: 对话的唯一标识符
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	if len(pathParts) < 2 {
		http.Error(w, "无效的路径格式。应为: /{source}/{conversation_id}", http.StatusBadRequest)
		log.Printf("错误: 无效的路径格式: %s", r.URL.Path)
		return
	}

	source := pathParts[0]
	conversationID := pathParts[1]

	// === 验证数据来源 ===
	validSources := map[string]bool{
		"gpt":         true,
		"claude":      true,
		"claude_code": true,
		"codex":       true,
	}

	if !validSources[source] {
		http.Error(w, fmt.Sprintf("无效的来源: %s。有效值为: gpt, claude, claude_code, codex", source), http.StatusBadRequest)
		log.Printf("错误: 无效的来源: %s", source)
		return
	}

	// === 构建文件路径并查找文件 ===
	var filePath string

	// 优先级1: parsed/{source}/conversation/{conversation_id}.json
	// 这是标准的解析后的对话文件路径
	parsedPath := filepath.Join(projectRoot, ParsedDir, source, "conversation", conversationID+".json")

	// 优先级2: data/{source}/{conversation_id}.json
	// 备用路径,用于存储原始数据
	dataPath := filepath.Join(projectRoot, "data", source, conversationID+".json")

	// 优先级3: parsed/{source}/{conversation_id}.json
	// 替代路径(某些早期数据可能存储在这里)
	altPath := filepath.Join(projectRoot, ParsedDir, source, conversationID+".json")

	// 按优先级检查文件是否存在
	if _, err := os.Stat(parsedPath); err == nil {
		filePath = parsedPath
	} else if _, err := os.Stat(dataPath); err == nil {
		filePath = dataPath
	} else if _, err := os.Stat(altPath); err == nil {
		filePath = altPath
	} else {
		// 所有路径都不存在,返回 404
		http.Error(w, fmt.Sprintf("未找到对话文件: %s", conversationID), http.StatusNotFound)
		log.Printf("错误: 未找到对话文件。尝试的路径: %s, %s, %s", parsedPath, dataPath, altPath)
		return
	}

	log.Printf("请求: source=%s, id=%s, 文件路径=%s", source, conversationID, filePath)

	// === 读取文件内容 ===
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("读取文件失败: %v", err), http.StatusInternalServerError)
		log.Printf("错误: 读取文件失败 %s: %v", filePath, err)
		return
	}

	// === 验证 JSON 格式 ===
	var jsonData interface{}
	if err := json.Unmarshal(fileData, &jsonData); err != nil {
		http.Error(w, fmt.Sprintf("JSON 格式错误: %v", err), http.StatusInternalServerError)
		log.Printf("错误: JSON 格式错误 %s: %v", filePath, err)
		return
	}

	// === 返回 JSON 数据 ===
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(fileData)

	log.Printf("成功: 返回对话数据 %s/%s", source, conversationID)
}

// HealthHandler 健康检查端点
// 用于监控服务是否正常运行
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, `{"status":"ok","service":"conversation-api"}`)
}

// ListSourcesHandler 列出所有可用的数据来源和对话
// 扫描 parsed 和 data 目录,返回所有可用的对话 ID
func ListSourcesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 支持的数据来源列表
	sources := []string{"gpt", "claude", "claude_code", "codex"}
	result := make(map[string]interface{})

	// 遍历每个数据来源,查找对话文件
	for _, source := range sources {
		// 优先检查 parsed/{source}/conversation/ 目录
		parsedSourceDir := filepath.Join(projectRoot, ParsedDir, source, "conversation")
		conversations := []string{}

		// 读取目录中的所有 .json 文件
		if files, err := os.ReadDir(parsedSourceDir); err == nil {
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
					// 提取对话 ID(去除 .json 后缀)
					convID := strings.TrimSuffix(file.Name(), ".json")
					conversations = append(conversations, convID)
				}
			}
		}

		// 如果 parsed 目录为空,检查 data 目录
		if len(conversations) == 0 {
			dataSourceDir := filepath.Join(projectRoot, "data", source)
			if files, err := os.ReadDir(dataSourceDir); err == nil {
				for _, file := range files {
					if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
						convID := strings.TrimSuffix(file.Name(), ".json")
						conversations = append(conversations, convID)
					}
				}
			}
		}

		result[source] = conversations
	}

	// 返回 JSON 格式的结果
	json.NewEncoder(w).Encode(result)
}

// LoggingMiddleware 日志中间件
// 记录所有 HTTP 请求的信息
func LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("收到请求: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next(w, r)
	}
}

func main() {
	// === 注册路由 ===
	http.HandleFunc("/health", LoggingMiddleware(HealthHandler))       // 健康检查端点
	http.HandleFunc("/list", LoggingMiddleware(ListSourcesHandler))    // 列出所有对话
	http.HandleFunc("/", LoggingMiddleware(ConversationHandler))       // 获取对话数据(默认路由)

	// === 启动 HTTP 服务器 ===
	addr := ":" + Port
	log.Printf("启动服务器在端口 %s", Port)
	log.Printf("API 端点:")
	log.Printf("  - GET /health - 健康检查")
	log.Printf("  - GET /list - 列出所有可用的对话")
	log.Printf("  - GET /{source}/{conversation_id} - 获取对话数据")
	log.Printf("    有效的 source: gpt, claude, claude_code, codex")
	log.Printf("    示例: http://localhost:%s/gpt/d4d4ddf6-5452-4dbb-9c1c-8a59ebfdb8fa", Port)

	// 启动服务器(阻塞调用)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("启动服务器失败:", err)
	}
}
