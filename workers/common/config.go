package common

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 通用配置结构
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Worker   WorkerConfig   `yaml:"worker"`
	Security SecurityConfig `yaml:"security"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// ServerConfig API服务器配置
type ServerConfig struct {
	BaseURL string `yaml:"base_url"` // API服务器地址
	Timeout int    `yaml:"timeout"`  // 请求超时(秒)
}

// WorkerConfig Worker通用配置
type WorkerConfig struct {
	CheckInterval int    `yaml:"check_interval"` // 检查间隔(秒)
	BatchSize     int    `yaml:"batch_size"`     // 批量上传大小
	DataDir       string `yaml:"data_dir"`       // 数据目录
	TempDir       string `yaml:"temp_dir"`       // 临时目录
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	WorkerToken string `yaml:"worker_token"` // Worker访问API的Token
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `yaml:"level"`  // 日志级别: debug, info, warn, error
	Output string `yaml:"output"` // 输出: stdout, file
	File   string `yaml:"file"`   // 日志文件路径(当output=file时)
}

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	// 设置默认值
	if config.Server.Timeout == 0 {
		config.Server.Timeout = 30
	}
	if config.Worker.CheckInterval == 0 {
		config.Worker.CheckInterval = 60
	}
	if config.Worker.BatchSize == 0 {
		config.Worker.BatchSize = 100
	}

	return &config, nil
}

// GetCheckInterval 获取检查间隔时间
func (c *Config) GetCheckInterval() time.Duration {
	return time.Duration(c.Worker.CheckInterval) * time.Second
}
