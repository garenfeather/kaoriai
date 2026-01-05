package claude_code

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

// UnzipFile 解压 ZIP 文件到指定目录
func UnzipFile(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		// 构建目标路径
		destPath := filepath.Join(destDir, file.Name)

		// 防止 Zip Slip 漏洞
		if !strings.HasPrefix(destPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			// 创建目录
			if err := os.MkdirAll(destPath, file.Mode()); err != nil {
				return fmt.Errorf("create directory: %w", err)
			}
		} else {
			// 创建文件的父目录
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("create parent directory: %w", err)
			}

			// 解压文件
			if err := extractFile(file, destPath); err != nil {
				return fmt.Errorf("extract file %s: %w", file.Name, err)
			}
		}
	}

	return nil
}

// extractFile 解压单个文件
func extractFile(file *zip.File, destPath string) error {
	srcFile, err := file.Open()
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}

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

// MarkAsAbandoned 标记压缩包为废弃
func MarkAsAbandoned(zipPath string, reason string) error {
	// 生成新文件名：原文件名.abandoned_timestamp.zip
	dir := filepath.Dir(zipPath)
	base := filepath.Base(zipPath)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	timestamp := time.Now().Format("20060102_150405")
	newName := fmt.Sprintf("%s.abandoned_%s%s", nameWithoutExt, timestamp, ext)
	newPath := filepath.Join(dir, newName)

	// 重命名文件
	if err := os.Rename(zipPath, newPath); err != nil {
		return fmt.Errorf("rename file: %w", err)
	}

	// 写入废弃原因到同名的 .reason 文件
	reasonFile := newPath + ".reason"
	if err := os.WriteFile(reasonFile, []byte(reason), 0644); err != nil {
		// 忽略写入原因文件失败的错误
		return nil
	}

	return nil
}

// IsAbandonedZip 检查文件是否是废弃的压缩包
func IsAbandonedZip(fileName string) bool {
	return strings.Contains(fileName, ".abandoned_")
}

// ValidateClaudeCodeStructure 验证解压后的目录结构是否符合 Claude Code 格式
// 返回 projects 目录的完整路径
func ValidateClaudeCodeStructure(extractDir string) (string, error) {
	// 首先检查根目录下是否直接有 projects 目录
	projectsDir := filepath.Join(extractDir, "projects")
	if info, err := os.Stat(projectsDir); err == nil && info.IsDir() {
		// 检查 projects 目录下是否有内容
		entries, err := os.ReadDir(projectsDir)
		if err != nil {
			return "", fmt.Errorf("read projects directory: %w", err)
		}
		if len(entries) > 0 {
			return projectsDir, nil
		}
	}

	// 如果根目录下没有 projects，检查是否有单个子目录（可能是 claude_code）
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", fmt.Errorf("read extract directory: %w", err)
	}

	// 找到第一个目录
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			subDir := filepath.Join(extractDir, entry.Name())
			projectsDir := filepath.Join(subDir, "projects")
			if info, err := os.Stat(projectsDir); err == nil && info.IsDir() {
				// 检查 projects 目录下是否有内容
				entries, err := os.ReadDir(projectsDir)
				if err != nil {
					return "", fmt.Errorf("read projects directory: %w", err)
				}
				if len(entries) > 0 {
					return projectsDir, nil
				}
			}
		}
	}

	return "", fmt.Errorf("projects directory not found")
}

// FindJSONLFiles 在 projects 目录下查找所有 .jsonl 文件
func FindJSONLFiles(projectsDir string) ([]string, error) {
	var jsonlFiles []string

	err := filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理 .jsonl 文件
		if !info.IsDir() && filepath.Ext(path) == ".jsonl" {
			jsonlFiles = append(jsonlFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk directory: %w", err)
	}

	return jsonlFiles, nil
}
