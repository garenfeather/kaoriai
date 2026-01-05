package codex

import (
	"archive/zip"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CalculateFileMD5 计算文件 MD5
func CalculateFileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// UnzipFile 解压 ZIP 文件
func UnzipFile(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	// 确保目标目录存在
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}

	for _, f := range r.File {
		if err := extractZipFile(f, destDir); err != nil {
			return fmt.Errorf("extract %s: %w", f.Name, err)
		}
	}

	return nil
}

// extractZipFile 提取单个文件
func extractZipFile(f *zip.File, destDir string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	path := filepath.Join(destDir, f.Name)

	// 检查路径安全性
	if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return fmt.Errorf("illegal file path: %s", path)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(path, f.Mode())
	}

	// 创建文件所在目录
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// 创建文件
	outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}

// MarkAsAbandoned 标记文件为已放弃
func MarkAsAbandoned(zipPath, reason string) error {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	abandonedPath := zipPath + ".abandoned_" + timestamp

	// 创建原因文件
	reasonFile := abandonedPath + ".reason.txt"
	if err := os.WriteFile(reasonFile, []byte(reason), 0644); err != nil {
		return fmt.Errorf("write reason file: %w", err)
	}

	// 重命名 ZIP 文件
	if err := os.Rename(zipPath, abandonedPath); err != nil {
		return fmt.Errorf("rename zip: %w", err)
	}

	return nil
}

// ValidateCodexStructure 验证解压后的目录结构
// 返回 sessions 目录的路径
func ValidateCodexStructure(extractDir string) (string, error) {
	// 检查根目录下的 sessions
	sessionsDir := filepath.Join(extractDir, "sessions")
	if info, err := os.Stat(sessionsDir); err == nil && info.IsDir() {
		return sessionsDir, nil
	}

	// 检查是否有嵌套的顶层目录（例如 codex/sessions）
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", fmt.Errorf("read extract dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		nestedSessionsDir := filepath.Join(extractDir, entry.Name(), "sessions")
		if info, err := os.Stat(nestedSessionsDir); err == nil && info.IsDir() {
			return nestedSessionsDir, nil
		}
	}

	return "", fmt.Errorf("sessions directory not found")
}

// FindJSONLFiles 递归查找 sessions 目录下的所有 rollout-*.jsonl 文件
func FindJSONLFiles(sessionsDir string) ([]string, error) {
	var jsonlFiles []string

	err := filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// 只处理 rollout-*.jsonl 文件
		baseName := filepath.Base(path)
		if strings.HasPrefix(baseName, "rollout-") && strings.HasSuffix(baseName, ".jsonl") {
			jsonlFiles = append(jsonlFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk sessions dir: %w", err)
	}

	return jsonlFiles, nil
}
