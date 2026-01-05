package gpt

import (
	"archive/zip"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ProtonMail/go-proton-api"
	"github.com/schollz/progressbar/v3"
)

const (
	targetSender  = "noreply@tm.openai.com"
	appVersion    = "web-mail@5.0.311.1"
)

var (
	downloadLinkRegex = regexp.MustCompile(`https://chatgpt\.com/backend-api/estuary/content\?[^\s"<>]+`)
)

// EmailMonitor 邮件监控器
type EmailMonitor struct {
	username     string
	password     string
	sessionToken string
	dataDir      string // 原始数据保存目录
}

// NewEmailMonitor 创建邮件监控器
func NewEmailMonitor(username, password, sessionToken, dataDir string) *EmailMonitor {
	return &EmailMonitor{
		username:     username,
		password:     password,
		sessionToken: sessionToken,
		dataDir:      dataDir,
	}
}

// CheckNewEmail 检查并下载新邮件
// 返回: (下载的ZIP文件路径, 是否有新邮件, error)
func (em *EmailMonitor) CheckNewEmail(ctx context.Context, lastCheckTime time.Time) (string, bool, error) {
	m := proton.New(proton.WithAppVersion(appVersion))
	defer m.Close()

	log.Printf("正在登录邮箱: %s", em.username)
	c, auth, err := m.NewClientWithLogin(ctx, em.username, []byte(em.password))
	if err != nil {
		return "", false, fmt.Errorf("login failed: %w", err)
	}
	defer c.Close()

	if auth.TwoFA.Enabled != 0 {
		return "", false, fmt.Errorf("2FA enabled, not supported")
	}

	user, err := c.GetUser(ctx)
	if err != nil {
		return "", false, fmt.Errorf("get user: %w", err)
	}

	addresses, err := c.GetAddresses(ctx)
	if err != nil {
		return "", false, fmt.Errorf("get addresses: %w", err)
	}

	salts, err := c.GetSalts(ctx)
	if err != nil {
		return "", false, fmt.Errorf("get salts: %w", err)
	}

	saltedKeyPass, err := salts.SaltForKey([]byte(em.password), user.Keys.Primary().ID)
	if err != nil {
		return "", false, fmt.Errorf("salt key: %w", err)
	}

	_, addrKRs, err := proton.Unlock(user, addresses, saltedKeyPass, nil)
	if err != nil {
		return "", false, fmt.Errorf("unlock keys: %w", err)
	}

	log.Printf("开始检查新邮件...")

	// 获取邮件列表
	filter := proton.MessageFilter{
		LabelID: proton.InboxLabel,
	}

	messages, err := c.GetMessageMetadataPage(ctx, 0, 10, filter)
	if err != nil {
		return "", false, fmt.Errorf("get messages: %w", err)
	}

	if len(messages) == 0 {
		log.Println("收件箱没有邮件")
		return "", false, nil
	}

	// 查找最新的符合条件的邮件
	var newestMsg *proton.MessageMetadata
	var newestTime time.Time

	for _, msg := range messages {
		if msg.Sender.Address != targetSender {
			continue
		}

		msgTime := time.Unix(msg.Time, 0)
		if !msgTime.After(lastCheckTime) {
			continue
		}

		if newestMsg == nil || msgTime.After(newestTime) {
			newestMsg = &msg
			newestTime = msgTime
		}
	}

	if newestMsg == nil {
		log.Printf("没有新邮件 (上次检查: %s)", lastCheckTime.Format("2006-01-02 15:04:05"))
		return "", false, nil
	}

	log.Printf("发现新邮件: %s (时间: %s)", newestMsg.Subject, newestTime.Format("2006-01-02 15:04:05"))

	// 获取邮件详情
	fullMessage, err := c.GetMessage(ctx, newestMsg.ID)
	if err != nil {
		return "", false, fmt.Errorf("get message details: %w", err)
	}

	// 解密邮件
	decrypted, err := fullMessage.Decrypt(addrKRs[fullMessage.AddressID])
	if err != nil {
		log.Printf("解密失败，使用未加密正文: %v", err)
		decrypted = []byte(fullMessage.Body)
	}

	// 提取下载链接
	body := string(decrypted)
	links := downloadLinkRegex.FindAllString(body, -1)
	if len(links) == 0 {
		log.Println("邮件中未找到下载链接")
		return "", false, nil
	}

	log.Printf("找到 %d 个下载链接", len(links))

	// 下载第一个链接
	for i, link := range links {
		log.Printf("处理链接 %d/%d", i+1, len(links))

		zipPath, err := em.downloadFile(link)
		if err != nil {
			log.Printf("下载失败: %v", err)
			continue
		}

		log.Printf("下载成功: %s", zipPath)
		return zipPath, true, nil
	}

	return "", false, fmt.Errorf("all downloads failed")
}

// downloadFile 下载文件
func (em *EmailMonitor) downloadFile(url string) (string, error) {
	// 确保数据目录存在
	if err := os.MkdirAll(em.dataDir, 0755); err != nil {
		return "", fmt.Errorf("create data dir: %w", err)
	}

	// 提取文件名
	filename := extractFilenameFromURL(url)
	if filename == "" {
		filename = fmt.Sprintf("download-%d.zip", time.Now().Unix())
	}

	savePath := filepath.Join(em.dataDir, filename)

	// 创建HTTP请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	// 添加必要的请求头
	req.Header.Set("User-Agent", "Mozilla/5.0")
	if em.sessionToken != "" {
		req.Header.Set("Cookie", fmt.Sprintf("__Secure-next-auth.session-token=%s", em.sessionToken))
	}

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	// 创建文件
	out, err := os.Create(savePath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	// 创建进度条
	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"下载中",
	)

	// 下载文件
	_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
	if err != nil {
		os.Remove(savePath)
		return "", fmt.Errorf("download: %w", err)
	}

	return savePath, nil
}

// extractFilenameFromURL 从URL提取文件名
func extractFilenameFromURL(url string) string {
	parts := strings.Split(url, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.Contains(parts[i], ".zip") {
			return parts[i]
		}
	}

	// 尝试从查询参数提取
	if idx := strings.Index(url, "file_id="); idx != -1 {
		fileID := url[idx+8:]
		if sepIdx := strings.IndexAny(fileID, "&?"); sepIdx != -1 {
			fileID = fileID[:sepIdx]
		}
		return fileID + ".zip"
	}

	return ""
}

// CalculateFileMD5 计算文件MD5
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

// UnzipFile 解压ZIP文件
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
