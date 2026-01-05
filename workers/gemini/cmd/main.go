package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cli-session-history/workers/gemini"
)

func main() {
	// 命令行参数
	dbPath := flag.String("db", "data/conversations.db", "数据库文件路径")
	dataDir := flag.String("data", "data/original/gemini", "原始数据目录")
	imagesDir := flag.String("images-dir", "data/images", "图片保存目录")
	videosDir := flag.String("videos-dir", "data/videos", "视频保存目录")
	checkInterval := flag.Duration("interval", 30*time.Second, "检查间隔")
	flag.Parse()

	log.Println("======================================")
	log.Println("Gemini Sync Worker")
	log.Println("======================================")

	// 创建Worker配置
	config := &gemini.Config{
		DBPath:          *dbPath,
		OriginalDataDir: *dataDir,
		ImagesDir:       *imagesDir,
		VideosDir:       *videosDir,
		CheckInterval:   *checkInterval,
	}

	// 创建Worker
	worker, err := gemini.NewWorker(config)
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	// 启动Worker
	if err := worker.Start(); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}

	// 输出启动信息
	log.Printf("Gemini Sync Worker started successfully")
	log.Printf("Database: %s", *dbPath)
	log.Printf("Data directory: %s", *dataDir)
	log.Printf("Images directory: %s", *imagesDir)
	log.Printf("Videos directory: %s", *videosDir)
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

	log.Println("Gemini Sync Worker stopped")
}
