package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"cli-session-history/workers/claude_code"
)

func main() {
	// 解析命令行参数
	dbPath := flag.String("db", "data/conversations.db", "Database file path")
	dataDir := flag.String("data", "data/original/claude_code", "Original data directory")
	tempDir := flag.String("temp", "data/temp", "Temporary directory")
	checkInterval := flag.Duration("interval", 30*time.Second, "Check interval")
	flag.Parse()

	// 转换为绝对路径
	absDBPath, err := filepath.Abs(*dbPath)
	if err != nil {
		log.Fatalf("Failed to get absolute path for db: %v", err)
	}

	absDataDir, err := filepath.Abs(*dataDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path for data dir: %v", err)
	}

	absTempDir, err := filepath.Abs(*tempDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path for temp dir: %v", err)
	}

	// 创建Worker配置
	config := &claude_code.Config{
		DBPath:          absDBPath,
		OriginalDataDir: absDataDir,
		TempDir:         absTempDir,
		CheckInterval:   *checkInterval,
	}

	log.Printf("Starting Claude Code Worker...")
	log.Printf("Database: %s", config.DBPath)
	log.Printf("Data directory: %s", config.OriginalDataDir)
	log.Printf("Temp directory: %s", config.TempDir)
	log.Printf("Check interval: %v", config.CheckInterval)

	// 创建Worker
	worker, err := claude_code.NewWorker(config)
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	// 启动Worker
	if err := worker.Start(); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")

	// 停止Worker
	if err := worker.Stop(); err != nil {
		log.Printf("Error stopping worker: %v", err)
	}

	log.Println("Worker stopped successfully")
}
