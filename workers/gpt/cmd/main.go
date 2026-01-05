package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cli-session-history/workers/gpt"
)

func main() {
	// 命令行参数
	dbPath := flag.String("db", "data/conversation.db", "数据库文件路径")
	originalDir := flag.String("original-dir", "data/original/gpt", "原始ZIP保存目录")
	imagesDir := flag.String("images-dir", "data/images", "图片保存目录")
	tempDir := flag.String("temp-dir", "data/temp", "临时目录")
	checkInterval := flag.Duration("interval", 30*time.Second, "检查间隔")
	flag.Parse()

	// 从环境变量读取工作模式
	modeStr := os.Getenv("GPT_WORKER_MODE")
	if modeStr == "" {
		modeStr = "file" // 默认使用 file 模式
	}

	mode := gpt.WorkerMode(modeStr)
	if mode != gpt.ModeEmail && mode != gpt.ModeFile {
		log.Fatalf("错误: 无效的工作模式 '%s' (必须是 'email' 或 'file')", modeStr)
	}

	// 创建Worker配置
	config := &gpt.Config{
		DBPath:          *dbPath,
		Mode:            mode,
		OriginalDataDir: *originalDir,
		ImagesDir:       *imagesDir,
		TempDir:         *tempDir,
		CheckInterval:   *checkInterval,
	}

	// 如果是 email 模式，需要读取邮件配置
	if mode == gpt.ModeEmail {
		emailUsername := os.Getenv("GO_PROTON_API_TEST_USERNAME")
		emailPassword := os.Getenv("GO_PROTON_API_TEST_PASSWORD")
		sessionToken := os.Getenv("SECURE_NEXT_AUTH_SESSION_TOKEN")

		if emailUsername == "" || emailPassword == "" {
			log.Fatal("错误: email 模式需要设置环境变量 GO_PROTON_API_TEST_USERNAME 和 GO_PROTON_API_TEST_PASSWORD")
		}

		if sessionToken == "" {
			log.Fatal("错误: email 模式需要设置环境变量 SECURE_NEXT_AUTH_SESSION_TOKEN")
		}

		config.EmailUsername = emailUsername
		config.EmailPassword = emailPassword
		config.SessionToken = sessionToken
	}

	// 创建Worker
	worker, err := gpt.NewWorker(config)
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	// 启动Worker
	if err := worker.Start(); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}

	log.Printf("GPT Sync Worker started successfully")
	log.Printf("Database: %s", *dbPath)
	log.Printf("Original data dir: %s", *originalDir)
	log.Printf("Images dir: %s", *imagesDir)
	log.Printf("Check interval: %v", *checkInterval)

	// 等待终止信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Received shutdown signal")

	// 停止Worker
	if err := worker.Stop(); err != nil {
		log.Printf("Error stopping worker: %v", err)
	}

	log.Println("GPT Sync Worker shutdown complete")
}
