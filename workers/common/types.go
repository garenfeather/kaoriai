package common

// Worker 定义Worker接口
type Worker interface {
	// Name 返回Worker名称
	Name() string

	// SourceType 返回数据源类型
	SourceType() string

	// Start 启动Worker
	Start() error

	// Stop 停止Worker
	Stop() error

	// Check 执行一次数据检查和同步
	Check() error
}

// WorkerStatus Worker状态
type WorkerStatus struct {
	Name          string `json:"name"`
	SourceType    string `json:"source_type"`
	Running       bool   `json:"running"`
	LastCheckTime string `json:"last_check_time"`
	LastSyncCount int    `json:"last_sync_count"`
	TotalSynced   int    `json:"total_synced"`
	ErrorCount    int    `json:"error_count"`
	LastError     string `json:"last_error"`
}
