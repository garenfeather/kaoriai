package gpt

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// WorkerMode Worker工作模式
type WorkerMode string

const (
	ModeEmail WorkerMode = "email" // 邮件监测模式
	ModeFile  WorkerMode = "file"  // 文件监测模式
)

// Worker GPT数据同步Worker
type Worker struct {
	config       *Config
	db           *Database
	emailMonitor *EmailMonitor
	status       *WorkerStatus
	statusLock   sync.RWMutex
	stopChan     chan struct{}
	wg           sync.WaitGroup

	// 已处理的ZIP文件记录，用于避免重复处理
	processedZips map[string]bool
}

// Config Worker配置
type Config struct {
	// 数据库配置
	DBPath string

	// 工作模式
	Mode WorkerMode // email 或 file

	// 邮件配置 (仅在 email 模式下需要)
	EmailUsername    string
	EmailPassword    string
	SessionToken     string

	// 目录配置
	OriginalDataDir  string // 原始ZIP保存目录: data/original/gpt
	ImagesDir        string // 图片保存目录: data/images
	TempDir          string // 临时解压目录

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

// NewWorker 创建GPT Worker
func NewWorker(config *Config) (*Worker, error) {
	// 验证模式
	if config.Mode == "" {
		config.Mode = ModeFile // 默认使用 file 模式
	}

	if config.Mode != ModeEmail && config.Mode != ModeFile {
		return nil, fmt.Errorf("invalid mode: %s (must be 'email' or 'file')", config.Mode)
	}

	// 初始化数据库
	db, err := NewDatabase(config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	worker := &Worker{
		config:        config,
		db:            db,
		stopChan:      make(chan struct{}),
		processedZips: make(map[string]bool),
		status: &WorkerStatus{
			Name:          "gpt-sync-worker",
			SourceType:    "gpt",
			Running:       false,
			LastCheckTime: time.Now(),
		},
	}

	// 如果是 email 模式，创建邮件监控器
	if config.Mode == ModeEmail {
		worker.emailMonitor = NewEmailMonitor(
			config.EmailUsername,
			config.EmailPassword,
			config.SessionToken,
			config.OriginalDataDir,
		)
		log.Printf("GPT Worker mode: email")
	} else {
		log.Printf("GPT Worker mode: file")
	}

	// 确保目录存在
	dirs := []string{
		config.OriginalDataDir,
		config.ImagesDir,
		config.TempDir,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			db.Close()
			return nil, fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	return worker, nil
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

	// 根据模式获取ZIP文件
	switch w.config.Mode {
	case ModeEmail:
		return w.checkEmailMode()
	case ModeFile:
		return w.checkFileMode()
	default:
		return fmt.Errorf("unknown mode: %s", w.config.Mode)
	}
}

// checkEmailMode Email 模式检查
func (w *Worker) checkEmailMode() error {
	// 检查新邮件
	ctx := context.Background()
	zipPath, hasNew, err := w.emailMonitor.CheckNewEmail(ctx, w.status.LastCheckTime.Add(-1*time.Hour))
	if err != nil {
		w.updateError(err)
		return fmt.Errorf("check email: %w", err)
	}

	if !hasNew {
		log.Printf("[%s] No new emails", w.status.Name)
		return nil
	}

	// 检查是否已处理过
	if w.processedZips[zipPath] {
		log.Printf("[%s] ZIP already processed, skipping: %s", w.status.Name, zipPath)
		return nil
	}

	// 处理ZIP文件
	syncCount, err := w.processZip(zipPath)
	if err != nil {
		w.updateError(err)
		return fmt.Errorf("process zip: %w", err)
	}

	// 标记为已处理
	w.processedZips[zipPath] = true

	w.statusLock.Lock()
	w.status.LastSyncCount = syncCount
	w.status.TotalSynced += syncCount
	w.statusLock.Unlock()

	log.Printf("[%s] Successfully synced %d conversations", w.status.Name, syncCount)
	return nil
}

// checkFileMode File 模式检查（类似 claude/codex worker）
func (w *Worker) checkFileMode() error {
	// 扫描目录下的所有 ZIP 文件
	zipFiles, err := w.findZipFiles()
	if err != nil {
		w.updateError(err)
		return fmt.Errorf("find zip files: %w", err)
	}

	if len(zipFiles) == 0 {
		log.Printf("[%s] No zip files found", w.status.Name)
		return nil
	}

	// 处理每个 ZIP 文件
	totalSyncCount := 0
	for _, zipPath := range zipFiles {
		// 检查是否已处理过
		if w.processedZips[zipPath] {
			continue
		}

		// 处理 ZIP 文件
		syncCount, err := w.processZip(zipPath)
		if err != nil {
			log.Printf("[%s] Error processing %s: %v", w.status.Name, zipPath, err)
			w.updateError(err)
			continue
		}

		// 标记为已处理
		w.processedZips[zipPath] = true
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

// findZipFiles 查找所有待处理的 ZIP 文件
func (w *Worker) findZipFiles() ([]string, error) {
	var zipFiles []string

	entries, err := os.ReadDir(w.config.OriginalDataDir)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()

		// 只处理 .zip 文件
		if strings.HasSuffix(strings.ToLower(fileName), ".zip") {
			zipFiles = append(zipFiles, filepath.Join(w.config.OriginalDataDir, fileName))
		}
	}

	return zipFiles, nil
}

// processZip 处理ZIP文件
func (w *Worker) processZip(zipPath string) (int, error) {
	log.Printf("[%s] Processing ZIP: %s", w.status.Name, zipPath)

	// 创建临时解压目录
	tempDir := filepath.Join(w.config.TempDir, fmt.Sprintf("gpt-%d", time.Now().Unix()))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return 0, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir) // 清理临时目录

	// 解压ZIP
	log.Printf("[%s] Extracting ZIP...", w.status.Name)
	if err := UnzipFile(zipPath, tempDir); err != nil {
		return 0, fmt.Errorf("unzip: %w", err)
	}

	// 复制图片文件到data/images
	log.Printf("[%s] Copying images...", w.status.Name)
	imageCount, err := w.copyImages(tempDir)
	if err != nil {
		log.Printf("[%s] Warning: copy images failed: %v", w.status.Name, err)
		// 图片复制失败不影响主流程
	} else {
		log.Printf("[%s] Copied %d images", w.status.Name, imageCount)
	}

	// 解析conversations.json
	conversationsFile := filepath.Join(tempDir, "conversations.json")
	if _, err := os.Stat(conversationsFile); os.IsNotExist(err) {
		return 0, fmt.Errorf("conversations.json not found in ZIP")
	}

	log.Printf("[%s] Parsing conversations.json...", w.status.Name)
	conversations, err := ParseConversationsFile(conversationsFile)
	if err != nil {
		return 0, fmt.Errorf("parse conversations: %w", err)
	}

	log.Printf("[%s] Parsed %d conversations", w.status.Name, len(conversations))

	// 写入数据库
	log.Printf("[%s] Writing to database...", w.status.Name)
	syncCount, err := w.syncToDatabase(conversations)
	if err != nil {
		return 0, fmt.Errorf("sync to database: %w", err)
	}

	return syncCount, nil
}

// copyImages 复制图片文件
func (w *Worker) copyImages(sourceDir string) (int, error) {
	count := 0

	// 遍历源目录中的所有文件
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// 只处理图片文件（简单判断扩展名）
		ext := filepath.Ext(path)
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
			return nil
		}

		// 复制到images目录
		destPath := filepath.Join(w.config.ImagesDir, info.Name())
		if err := copyFile(path, destPath); err != nil {
			log.Printf("[%s] Warning: copy image %s failed: %v", w.status.Name, info.Name(), err)
			return nil // 继续处理其他文件
		}

		count++
		return nil
	})

	return count, err
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}

// syncToDatabase 同步到数据库
func (w *Worker) syncToDatabase(conversations []ParsedConversation) (int, error) {
	syncCount := 0

	for _, conv := range conversations {
		// 使用事务
		tx, err := w.db.BeginTx()
		if err != nil {
			log.Printf("[%s] Error begin tx for conversation %s: %v", w.status.Name, conv.UUID, err)
			continue
		}

		// 插入对话
		err = w.db.InsertConversationTx(tx, conv.UUID, "gpt", conv.Title, conv.Metadata, conv.CreatedAt, conv.UpdatedAt)
		if err != nil {
			tx.Rollback()
			log.Printf("[%s] Error insert conversation %s: %v", w.status.Name, conv.UUID, err)
			continue
		}

		// 插入消息
		messageCount := 0
		for _, msg := range conv.Messages {
			// 构建content JSON
			contentJSON, err := BuildContentJSON(msg.ContentText, msg.ContentImages)
			if err != nil {
				log.Printf("[%s] Error build content json for message %s: %v", w.status.Name, msg.UUID, err)
				continue
			}

			err = w.db.InsertMessageTx(tx, msg.UUID, conv.UUID, msg.ParentUUID,
				msg.RoundIndex, msg.Role, msg.ContentType, contentJSON, msg.CreatedAt)
			if err != nil {
				log.Printf("[%s] Error insert message %s: %v", w.status.Name, msg.UUID, err)
				continue
			}

			messageCount++
		}

		// 提交事务
		if err := tx.Commit(); err != nil {
			log.Printf("[%s] Error commit tx for conversation %s: %v", w.status.Name, conv.UUID, err)
			continue
		}

		log.Printf("[%s] Synced conversation %s (%d messages)", w.status.Name, conv.UUID, messageCount)
		syncCount++
	}

	return syncCount, nil
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
