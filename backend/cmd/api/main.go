// Package main 是 API 服务器的入口点
// 这是一个基于 Gin 框架的 RESTful API 服务器,用于 AI 对话数据管理系统
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"gpt-tools/backend/internal/db"
	"gpt-tools/backend/internal/server"
)

func main() {
	// 解析命令行参数
	dbPath := flag.String("db", "", "数据库文件路径 (默认: ./data/conversation.db)")
	port := flag.String("port", "8080", "HTTP 服务器端口")
	flag.Parse()

	// 设置默认数据库路径
	if *dbPath == "" {
		// 获取当前工作目录
		wd, err := os.Getwd()
		if err != nil {
			log.Fatalf("获取工作目录失败: %v", err)
		}
		// 如果是 backend/cmd/api,则数据库在项目根目录
		if filepath.Base(filepath.Dir(wd)) == "cmd" {
			*dbPath = filepath.Join(filepath.Dir(filepath.Dir(wd)), "data", "conversation.db")
		} else {
			*dbPath = filepath.Join(wd, "data", "conversation.db")
		}
	}

	// 确保数据目录存在
	dataDir := filepath.Dir(*dbPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("创建数据目录失败: %v", err)
	}

	log.Printf("使用数据库: %s", *dbPath)

	// 初始化数据库
	database, err := db.InitDB(*dbPath)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer database.Close()

	// 设置 Gin 为发布模式(生产环境),减少日志输出
	gin.SetMode(gin.ReleaseMode)

	// 创建并配置路由器,注入数据库连接
	r := server.NewRouter(database)

	// 启动 HTTP 服务器
	addr := ":" + *port
	log.Printf("启动 API 服务器在端口 %s", *port)
	log.Printf("API 文档: docs/architecture.md")

	if err := r.Run(addr); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}
