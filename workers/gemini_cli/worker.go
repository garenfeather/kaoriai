package gemini_cli

import (
	"fmt"
	"log"
	"sync"
	"time"

	"cli-session-history/workers/common"
)

// Worker Gemini CLI数据同步Worker
type Worker struct {
	config     *common.Config
	apiClient  *common.APIClient
	status     *common.WorkerStatus
	statusLock sync.RWMutex
	stopChan   chan struct{}
	wg         sync.WaitGroup

	// Gemini CLI特定配置
	dataDir string
	lastMD5 map[string]string
}

// NewWorker 创建Gemini CLI Worker
func NewWorker(config *common.Config, dataDir string) *Worker {
	return &Worker{
		config:    config,
		apiClient: common.NewAPIClient(config.Server.BaseURL, config.Security.WorkerToken, config.Server.Timeout),
		dataDir:   dataDir,
		lastMD5:   make(map[string]string),
		stopChan:  make(chan struct{}),
		status: &common.WorkerStatus{
			Name:       "gemini-cli-sync-worker",
			SourceType: "gemini_cli",
			Running:    false,
		},
	}
}

// Name 返回Worker名称
func (w *Worker) Name() string {
	return w.status.Name
}

// SourceType 返回数据源类型
func (w *Worker) SourceType() string {
	return w.status.SourceType
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

	log.Printf("[%s] Starting worker (TODO: implement)", w.Name())
	return nil
}

// Stop 停止Worker
func (w *Worker) Stop() error {
	log.Printf("[%s] Stopping worker (TODO: implement)", w.Name())
	return nil
}

// Check 执行一次数据检查和同步
func (w *Worker) Check() error {
	w.statusLock.Lock()
	w.status.LastCheckTime = time.Now().Format(time.RFC3339)
	w.statusLock.Unlock()

	log.Printf("[%s] Check not implemented yet", w.Name())
	return nil
}

// GetStatus 获取Worker状态
func (w *Worker) GetStatus() *common.WorkerStatus {
	w.statusLock.RLock()
	defer w.statusLock.RUnlock()

	status := *w.status
	return &status
}
