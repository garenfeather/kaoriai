package gemini

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Worker Gemini数据同步Worker
type Worker struct {
	config   *Config
	db       *Database
	status   *WorkerStatus
	statusLock sync.RWMutex
	stopChan chan struct{}
	wg       sync.WaitGroup

	// 已处理的JSON文件缓存，避免重复处理
	processedJSONs map[string]bool
}

// Config Worker配置
type Config struct {
	// 数据库配置
	DBPath string

	// 目录配置
	OriginalDataDir string // 原始数据目录: data/original/gemini
	ImagesDir       string // 图片保存目录: data/images
	VideosDir       string // 视频保存目录: data/videos

	// Worker配置
	CheckInterval time.Duration // 检查间隔
}

// WorkerStatus Worker状态
type WorkerStatus struct {
	Name          string
	SourceType    string
	Running       bool
	LastCheckTime time.Time
	LastSyncCount int
	TotalSynced   int
	ErrorCount    int
	LastError     string
}

// NewWorker 创建Gemini Worker
func NewWorker(config *Config) (*Worker, error) {
	// 初始化数据库
	db, err := NewDatabase(config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	// 确保目录存在
	dirs := []string{
		config.OriginalDataDir,
		config.ImagesDir,
		config.VideosDir,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			db.Close()
			return nil, fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	return &Worker{
		config:         config,
		db:             db,
		stopChan:       make(chan struct{}),
		processedJSONs: make(map[string]bool),
		status: &WorkerStatus{
			Name:          "gemini-sync-worker",
			SourceType:    "gemini",
			Running:       false,
			LastCheckTime: time.Now(),
		},
	}, nil
}

// Start 启动Worker
func (w *Worker) Start() error {
	w.statusLock.Lock()
	if w.status.Running {
		w.statusLock.Unlock()
		return fmt.Errorf("worker already running")
	}
	w.status.Running = true
	w.statusLock.Unlock()

	log.Printf("[%s] Starting worker, check interval: %v", w.status.Name, w.config.CheckInterval)

	w.wg.Add(1)
	go w.runLoop()

	return nil
}

// Stop 停止Worker
func (w *Worker) Stop() error {
	w.statusLock.Lock()
	if !w.status.Running {
		w.statusLock.Unlock()
		return fmt.Errorf("worker not running")
	}
	w.statusLock.Unlock()

	log.Printf("[%s] Stopping worker...", w.status.Name)
	close(w.stopChan)
	w.wg.Wait()

	w.statusLock.Lock()
	w.status.Running = false
	w.statusLock.Unlock()

	// 关闭数据库
	if err := w.db.Close(); err != nil {
		log.Printf("[%s] Error closing database: %v", w.status.Name, err)
	}

	log.Printf("[%s] Worker stopped", w.status.Name)
	return nil
}

// runLoop Worker主循环
func (w *Worker) runLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.CheckInterval)
	defer ticker.Stop()

	// 启动时立即执行一次检查
	if err := w.Check(); err != nil {
		log.Printf("[%s] Initial check failed: %v", w.status.Name, err)
	}

	for {
		select {
		case <-ticker.C:
			if err := w.Check(); err != nil {
				log.Printf("[%s] Check failed: %v", w.status.Name, err)
			}
		case <-w.stopChan:
			return
		}
	}
}

// Check 执行一次数据检查和同步
func (w *Worker) Check() error {
	w.statusLock.Lock()
	w.status.LastCheckTime = time.Now()
	w.statusLock.Unlock()

	log.Printf("[%s] Starting check...", w.status.Name)

	// 扫描目录下的所有 JSON 文件
	jsonFiles, err := w.findJSONFiles()
	if err != nil {
		w.updateError(err)
		return fmt.Errorf("find json files: %w", err)
	}

	if len(jsonFiles) == 0 {
		log.Printf("[%s] No json files found", w.status.Name)
		return nil
	}

	// 处理每个 JSON 文件
	totalSyncCount := 0
	for _, jsonPath := range jsonFiles {
		// 检查是否已处理过
		if w.processedJSONs[jsonPath] {
			continue
		}

		// 处理 JSON 文件
		syncCount, err := w.processJSON(jsonPath)
		if err != nil {
			log.Printf("[%s] Error processing %s: %v", w.status.Name, jsonPath, err)
			w.updateError(err)

			// 标记为失败
			if markErr := markAsFailed(jsonPath, err.Error()); markErr != nil {
				log.Printf("[%s] Failed to mark as failed: %v", w.status.Name, markErr)
			} else {
				log.Printf("[%s] Marked as failed: %s", w.status.Name, jsonPath)
			}
			continue
		}

		// 标记为已处理
		if err := markAsProcessed(jsonPath); err != nil {
			log.Printf("[%s] Failed to mark as processed: %v", w.status.Name, err)
		} else {
			log.Printf("[%s] Marked as processed: %s", w.status.Name, jsonPath)
		}

		w.processedJSONs[jsonPath] = true
		totalSyncCount += syncCount
	}

	if totalSyncCount > 0 {
		w.statusLock.Lock()
		w.status.LastSyncCount = totalSyncCount
		w.status.TotalSynced += totalSyncCount
		w.statusLock.Unlock()

		log.Printf("[%s] Successfully synced %d conversations", w.status.Name, totalSyncCount)
	}

	return nil
}

// findJSONFiles 查找所有待处理的 JSON 文件
func (w *Worker) findJSONFiles() ([]string, error) {
	var jsonFiles []string

	entries, err := os.ReadDir(w.config.OriginalDataDir)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()

		// 跳过已处理和失败的文件
		if strings.HasSuffix(fileName, ".processed") ||
			strings.HasSuffix(fileName, ".failed") ||
			strings.HasSuffix(fileName, ".error") {
			continue
		}

		// 只处理 .json 文件
		if filepath.Ext(fileName) == ".json" {
			jsonFiles = append(jsonFiles, filepath.Join(w.config.OriginalDataDir, fileName))
		}
	}

	return jsonFiles, nil
}

// processJSON 处理 JSON 文件
func (w *Worker) processJSON(jsonPath string) (int, error) {
	log.Printf("[%s] Processing JSON: %s", w.status.Name, jsonPath)

	// 从文件名提取conversation ID
	fileName := filepath.Base(jsonPath)
	conversationID := strings.TrimSuffix(fileName, ".json")

	// 解析 JSON 文件
	conv, err := ParseJSONFile(jsonPath, conversationID)
	if err != nil {
		return 0, fmt.Errorf("parse json: %w", err)
	}

	// 处理图片和视频
	if err := w.processAssets(conv); err != nil {
		return 0, fmt.Errorf("process assets: %w", err)
	}

	// 写入数据库
	if err := w.syncToDatabase(conv); err != nil {
		return 0, fmt.Errorf("sync to database: %w", err)
	}

	// 清理已复制的视频文件
	if err := w.cleanupVideos(conversationID); err != nil {
		log.Printf("[%s] Warning: cleanup videos failed: %v", w.status.Name, err)
		// 清理失败不影响主流程
	}

	log.Printf("[%s] Synced conversation %s (%d messages)", w.status.Name, conv.UUID, len(conv.Messages))
	return 1, nil
}

// processAssets 处理图片和视频素材
func (w *Worker) processAssets(conv *ParsedConversation) error {
	for i := range conv.Messages {
		msg := &conv.Messages[i]

		// 下载图片
		if len(msg.ContentImages) > 0 {
			downloadedImages := []string{}
			for _, imageURL := range msg.ContentImages {
				filename, err := downloadImage(imageURL, w.config.ImagesDir)
				if err != nil {
					log.Printf("[%s] Warning: download image failed: %v", w.status.Name, err)
					continue
				}
				downloadedImages = append(downloadedImages, filename)
			}
			msg.ContentImages = downloadedImages
		}

		// 复制视频（视频文件已经在original目录中）
		if len(msg.ContentVideos) > 0 {
			copiedVideos := []string{}
			for _, videoFilename := range msg.ContentVideos {
				// 视频源文件路径
				srcPath := filepath.Join(w.config.OriginalDataDir, videoFilename)

				// 检查源文件是否存在
				if _, err := os.Stat(srcPath); os.IsNotExist(err) {
					log.Printf("[%s] Warning: video file not found: %s", w.status.Name, srcPath)
					continue
				}

				// 目标文件名与源文件名相同
				destPath := filepath.Join(w.config.VideosDir, videoFilename)

				// 如果目标文件不存在，则复制
				if _, err := os.Stat(destPath); os.IsNotExist(err) {
					if err := copyFile(srcPath, destPath); err != nil {
						log.Printf("[%s] Warning: copy video failed: %v", w.status.Name, err)
						continue
					}
				}

				copiedVideos = append(copiedVideos, videoFilename)
			}
			msg.ContentVideos = copiedVideos
		}
	}

	return nil
}

// cleanupVideos 清理original目录中的视频文件
func (w *Worker) cleanupVideos(conversationID string) error {
	// 查找所有以conversationID开头的视频文件
	entries, err := os.ReadDir(w.config.OriginalDataDir)
	if err != nil {
		return fmt.Errorf("read directory: %w", err)
	}

	prefix := conversationID + "-"
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		// 检查是否是该对话的视频文件
		if strings.HasPrefix(fileName, prefix) {
			// 检查扩展名是否为视频格式
			ext := strings.ToLower(filepath.Ext(fileName))
			if ext == ".mp4" || ext == ".mov" || ext == ".avi" || ext == ".mkv" {
				filePath := filepath.Join(w.config.OriginalDataDir, fileName)
				if err := os.Remove(filePath); err != nil {
					log.Printf("[%s] Warning: remove video file failed: %s: %v", w.status.Name, filePath, err)
				} else {
					log.Printf("[%s] Removed video file: %s", w.status.Name, filePath)
				}
			}
		}
	}

	return nil
}

// syncToDatabase 同步单个对话到数据库
func (w *Worker) syncToDatabase(conv *ParsedConversation) error {
	// 使用事务
	tx, err := w.db.BeginTx()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	// 插入对话
	err = w.db.InsertConversationTx(tx, conv.UUID, "gemini", conv.Title, conv.Metadata, conv.CreatedAt, conv.UpdatedAt)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("insert conversation: %w", err)
	}

	// 插入消息
	for _, msg := range conv.Messages {
		// 构建 content JSON
		contentJSON, err := BuildContentJSON(msg.ContentText, msg.ContentImages, msg.ContentVideos)
		if err != nil {
			log.Printf("[%s] Error build content json for message %s: %v", w.status.Name, msg.UUID, err)
			continue
		}

		err = w.db.InsertMessageTx(tx, msg.UUID, conv.UUID, msg.ParentUUID,
			msg.RoundIndex, msg.Role, msg.ContentType, contentJSON, msg.Thinking, msg.Model, msg.CreatedAt)
		if err != nil {
			log.Printf("[%s] Error insert message %s: %v", w.status.Name, msg.UUID, err)
			continue
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

// updateError 更新错误信息
func (w *Worker) updateError(err error) {
	w.statusLock.Lock()
	defer w.statusLock.Unlock()

	w.status.ErrorCount++
	w.status.LastError = err.Error()
}

// GetStatus 获取Worker状态
func (w *Worker) GetStatus() *WorkerStatus {
	w.statusLock.RLock()
	defer w.statusLock.RUnlock()

	status := *w.status
	return &status
}
