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

const (
	Port       = "8080"
	ParsedDir  = "parsed"
	ProjectDir = ".."
)

// 全局变量存储项目根目录
var projectRoot string

func init() {
	// 获取当前执行文件的目录
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal("无法获取执行路径:", err)
	}
	execDir := filepath.Dir(execPath)

	// 假设 backend 在项目根目录下
	projectRoot = filepath.Join(execDir, ProjectDir)

	// 如果是开发模式（go run），使用当前工作目录的上级
	if strings.Contains(execPath, "go-build") {
		wd, err := os.Getwd()
		if err == nil {
			projectRoot = filepath.Dir(wd)
		}
	}

	log.Printf("项目根目录: %s", projectRoot)
}

// ConversationHandler 处理对话请求
func ConversationHandler(w http.ResponseWriter, r *http.Request) {
	// 设置 CORS 头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "仅支持 GET 请求", http.StatusMethodNotAllowed)
		return
	}

	// 解析路径: /{source}/{conversation_id}
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	if len(pathParts) < 2 {
		http.Error(w, "无效的路径格式。应为: /{source}/{conversation_id}", http.StatusBadRequest)
		log.Printf("错误: 无效的路径格式: %s", r.URL.Path)
		return
	}

	source := pathParts[0]
	conversationID := pathParts[1]

	// 验证 source
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

	// 构建文件路径
	var filePath string

	// 根据不同的来源，文件可能在不同的子目录
	// 优先查找 parsed 目录，如果不存在则查找 data 目录
	parsedPath := filepath.Join(projectRoot, ParsedDir, source, "conversation", conversationID+".json")
	dataPath := filepath.Join(projectRoot, "data", source, conversationID+".json")

	// 检查 parsed 目录
	if _, err := os.Stat(parsedPath); err == nil {
		filePath = parsedPath
	} else if _, err := os.Stat(dataPath); err == nil {
		filePath = dataPath
	} else {
		// 尝试其他可能的路径
		altPath := filepath.Join(projectRoot, ParsedDir, source, conversationID+".json")
		if _, err := os.Stat(altPath); err == nil {
			filePath = altPath
		} else {
			http.Error(w, fmt.Sprintf("未找到对话文件: %s", conversationID), http.StatusNotFound)
			log.Printf("错误: 未找到对话文件。尝试的路径: %s, %s, %s", parsedPath, dataPath, altPath)
			return
		}
	}

	log.Printf("请求: source=%s, id=%s, 文件路径=%s", source, conversationID, filePath)

	// 读取文件
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("读取文件失败: %v", err), http.StatusInternalServerError)
		log.Printf("错误: 读取文件失败 %s: %v", filePath, err)
		return
	}

	// 验证 JSON 格式
	var jsonData interface{}
	if err := json.Unmarshal(fileData, &jsonData); err != nil {
		http.Error(w, fmt.Sprintf("JSON 格式错误: %v", err), http.StatusInternalServerError)
		log.Printf("错误: JSON 格式错误 %s: %v", filePath, err)
		return
	}

	// 返回 JSON 数据
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(fileData)

	log.Printf("成功: 返回对话数据 %s/%s", source, conversationID)
}

// HealthHandler 健康检查端点
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, `{"status":"ok","service":"conversation-api"}`)
}

// ListSourcesHandler 列出所有可用的来源和对话
func ListSourcesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	sources := []string{"gpt", "claude", "claude_code", "codex"}
	result := make(map[string]interface{})

	for _, source := range sources {
		// 检查 parsed 目录
		parsedSourceDir := filepath.Join(projectRoot, ParsedDir, source, "conversation")
		conversations := []string{}

		if files, err := os.ReadDir(parsedSourceDir); err == nil {
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
					convID := strings.TrimSuffix(file.Name(), ".json")
					conversations = append(conversations, convID)
				}
			}
		}

		// 如果 parsed 目录为空，检查 data 目录
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

	json.NewEncoder(w).Encode(result)
}

// LoggingMiddleware 日志中间件
func LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("收到请求: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next(w, r)
	}
}

func main() {
	// 设置路由
	http.HandleFunc("/health", LoggingMiddleware(HealthHandler))
	http.HandleFunc("/list", LoggingMiddleware(ListSourcesHandler))
	http.HandleFunc("/", LoggingMiddleware(ConversationHandler))

	// 启动服务器
	addr := ":" + Port
	log.Printf("启动服务器在端口 %s", Port)
	log.Printf("API 端点:")
	log.Printf("  - GET /health - 健康检查")
	log.Printf("  - GET /list - 列出所有可用的对话")
	log.Printf("  - GET /{source}/{conversation_id} - 获取对话数据")
	log.Printf("    有效的 source: gpt, claude, claude_code, codex")
	log.Printf("    示例: http://localhost:%s/gpt/d4d4ddf6-5452-4dbb-9c1c-8a59ebfdb8fa", Port)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("启动服务器失败:", err)
	}
}
