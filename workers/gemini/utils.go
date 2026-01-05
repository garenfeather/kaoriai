package gemini

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// extractImageFilenameFromURL 从Google URL中提取图片文件名
// URL示例: https://lh3.googleusercontent.com/gg/AIJ2gl-AKA8hgRzvEUWzrWu8Z3gVph40Vgd5NnkmrdG9NFh0BNiUeEBTIr2GIEQneJY4nw9ikwtUhqXiulrQtJWqR1dfva6GDwFNeoeVYmwhd7N9CYuDSFd9RufZSMYJztlhqCnZO7Itl34VRpVmo0HU6DDwJgwP3byEXK-aKpNxAckmv9jUvJMspd_5SAa1H4wGbBNMjjwrpUe9TU6tSjHVSXxxbnjozU-HrbhG1Xg4rx74RiGd4mya6rU3Dslr9OdfyOwTvvEa3NE18r6nOfKo6T02d9f5HsDu--U?authuser=3
// 提取: /gg/ 之后、? 之前的部分
func extractImageFilenameFromURL(url string) string {
	// 查找 /gg/ 的位置
	ggIndex := strings.Index(url, "/gg/")
	if ggIndex == -1 {
		// 如果没有 /gg/，尝试使用最后一个路径段
		parts := strings.Split(url, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i] != "" && !strings.Contains(parts[i], "?") {
				// 移除查询参数
				filename := strings.Split(parts[i], "?")[0]
				if filename != "" {
					return filename + ".jpg" // 默认添加.jpg扩展名
				}
			}
		}
		return "unknown.jpg"
	}

	// 提取 /gg/ 之后的部分
	afterGG := url[ggIndex+4:]

	// 查找 ? 的位置
	queryIndex := strings.Index(afterGG, "?")
	if queryIndex != -1 {
		afterGG = afterGG[:queryIndex]
	}

	// 确保文件名有扩展名
	if !strings.Contains(afterGG, ".") {
		afterGG += ".jpg"
	}

	return afterGG
}

// downloadImage 下载图片到指定目录
func downloadImage(url, destDir string) (string, error) {
	// 提取文件名
	filename := extractImageFilenameFromURL(url)
	destPath := filepath.Join(destDir, filename)

	// 如果文件已存在，直接返回
	if _, err := os.Stat(destPath); err == nil {
		return filename, nil
	}

	// 创建HTTP请求
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http status: %d", resp.StatusCode)
	}

	// 创建目标文件
	out, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	// 复制数据
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("copy data: %w", err)
	}

	return filename, nil
}

// copyVideoFile 复制视频文件到目标目录
func copyVideoFile(srcPath, destDir, conversationID, originalFilename string) (string, error) {
	// 目标文件名格式：{conversation_id}-{original_filename}
	destFilename := fmt.Sprintf("%s-%s", conversationID, originalFilename)
	destPath := filepath.Join(destDir, destFilename)

	// 如果文件已存在，直接返回
	if _, err := os.Stat(destPath); err == nil {
		return destFilename, nil
	}

	// 打开源文件
	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("open source: %w", err)
	}
	defer src.Close()

	// 创建目标文件
	dest, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create dest: %w", err)
	}
	defer dest.Close()

	// 复制数据
	_, err = io.Copy(dest, src)
	if err != nil {
		return "", fmt.Errorf("copy data: %w", err)
	}

	return destFilename, nil
}

// markAsProcessed 标记JSON文件为已处理
func markAsProcessed(jsonPath string) error {
	newPath := jsonPath + ".processed"
	return os.Rename(jsonPath, newPath)
}

// markAsFailed 标记JSON文件为失败，并记录错误信息
func markAsFailed(jsonPath string, errMsg string) error {
	// 重命名为.failed
	failedPath := jsonPath + ".failed"
	if err := os.Rename(jsonPath, failedPath); err != nil {
		return fmt.Errorf("rename to failed: %w", err)
	}

	// 创建.error文件记录错误
	errorPath := jsonPath + ".error"
	if err := os.WriteFile(errorPath, []byte(errMsg), 0644); err != nil {
		return fmt.Errorf("write error file: %w", err)
	}

	return nil
}

// copyFile 复制文件（通用函数）
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

	_, err = io.Copy(destFile, sourceFile)
	return err
}
