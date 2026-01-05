package claude

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Worker Claude数据同步Worker
type Worker struct {
	config   *Config
	db       *Database
	status   *WorkerStatus
	statusLock sync.RWMutex
	stopChan chan struct{}
	wg       sync.WaitGroup

	// MD5缓存，用于避免重复处理同一个ZIP
	processedZips map[string]bool
}

// Config Worker配置
type Config struct {
	// 数据库配置
	DBPath string

	// 目录配置
	OriginalDataDir string // 原始ZIP保存目录: data/original/claude
	TempDir         string // 临时解压目录

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

// NewWorker 创建Claude Worker
func NewWorker(config *Config) (*Worker, error) {
	// 初始化数据库
	db, err := NewDatabase(config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	// 确保目录存在
	dirs := []string{
		config.OriginalDataDir,
		config.TempDir,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			db.Close()
			return nil, fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	return &Worker{
		config:        config,
		db:            db,
		stopChan:      make(chan struct{}),
		processedZips: make(map[string]bool),
		status: &WorkerStatus{
			Name:          "claude-sync-worker",
			SourceType:    "claude",
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

		// 跳过废弃的压缩包
		if isAbandonedZip(fileName) {
			continue
		}

		// 只处理 .zip 文件
		if filepath.Ext(fileName) == ".zip" {
			zipFiles = append(zipFiles, filepath.Join(w.config.OriginalDataDir, fileName))
		}
	}

	return zipFiles, nil
}

// isAbandonedZip 检查是否为废弃的 ZIP 文件
func isAbandonedZip(filename string) bool {
	return strings.Contains(filename, ".abandoned_")
}

// processZip 处理 ZIP 文件
func (w *Worker) processZip(zipPath string) (int, error) {
	log.Printf("[%s] Processing ZIP: %s", w.status.Name, zipPath)

	// 创建临时解压目录
	tempDir := filepath.Join(w.config.TempDir, fmt.Sprintf("claude-%d", time.Now().Unix()))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return 0, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir) // 清理临时目录

	// 解压 ZIP
	log.Printf("[%s] Extracting ZIP...", w.status.Name)
	if err := UnzipFile(zipPath, tempDir); err != nil {
		// 解压失败，标记为废弃
		reason := fmt.Sprintf("Failed to unzip: %v", err)
		if markErr := MarkAsAbandoned(zipPath, reason); markErr != nil {
			log.Printf("[%s] Failed to mark as abandoned: %v", w.status.Name, markErr)
		} else {
			log.Printf("[%s] Marked as abandoned: %s (reason: %s)", w.status.Name, zipPath, reason)
		}
		return 0, fmt.Errorf("unzip: %w", err)
	}

	// 验证目录结构并获取 conversations.json 文件路径
	conversationsPath, err := ValidateClaudeStructure(tempDir)
	if err != nil {
		// 结构不符合，标记为废弃
		reason := fmt.Sprintf("Invalid structure: %v", err)
		if markErr := MarkAsAbandoned(zipPath, reason); markErr != nil {
			log.Printf("[%s] Failed to mark as abandoned: %v", w.status.Name, markErr)
		} else {
			log.Printf("[%s] Marked as abandoned: %s (reason: %s)", w.status.Name, zipPath, reason)
		}
		return 0, fmt.Errorf("validate structure: %w", err)
	}

	// 解析 conversations.json
	log.Printf("[%s] Parsing conversations.json...", w.status.Name)
	conversations, err := ParseConversationsJSON(conversationsPath)
	if err != nil {
		return 0, fmt.Errorf("parse conversations json: %w", err)
	}

	log.Printf("[%s] Found %d conversations", w.status.Name, len(conversations))

	// 同步到数据库
	syncCount, err := w.syncConversations(conversations)
	if err != nil {
		return 0, fmt.Errorf("sync conversations: %w", err)
	}

	// 处理成功，删除 ZIP 文件
	log.Printf("[%s] Deleting processed ZIP: %s", w.status.Name, zipPath)
	if err := os.Remove(zipPath); err != nil {
		log.Printf("[%s] Warning: failed to delete ZIP: %v", w.status.Name, err)
		// 删除失败不影响主流程
	}

	return syncCount, nil
}

// syncConversations 同步多个对话到数据库
func (w *Worker) syncConversations(conversations []*ParsedConversation) (int, error) {
	syncCount := 0

	for _, conv := range conversations {
		// 写入数据库
		if err := w.syncToDatabase(conv); err != nil {
			log.Printf("[%s] Error syncing conversation %s: %v", w.status.Name, conv.UUID, err)
			continue
		}

		log.Printf("[%s] Synced conversation %s (%d messages)", w.status.Name, conv.UUID, len(conv.Messages))
		syncCount++
	}

	return syncCount, nil
}

// syncToDatabase 同步单个对话到数据库
func (w *Worker) syncToDatabase(conv *ParsedConversation) error {
	// 使用事务
	tx, err := w.db.BeginTx()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	// 插入对话
	err = w.db.InsertConversationTx(tx, conv.UUID, "claude", conv.Title, "", conv.CreatedAt, conv.UpdatedAt)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("insert conversation: %w", err)
	}

	// 插入消息
	for _, msg := range conv.Messages {
		// 构建 content JSON
		contentJSON, err := BuildContentJSON(msg.Content)
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
