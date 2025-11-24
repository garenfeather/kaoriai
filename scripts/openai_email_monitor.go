package main

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
	"strconv"
	"strings"
	"time"

	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/joho/godotenv"
	"github.com/schollz/progressbar/v3"
)

// SimplePanicHandler 简单的panic处理器
type SimplePanicHandler struct{}

func (SimplePanicHandler) HandlePanic(v interface{}) {
	log.Printf("Recovered from panic: %v", v)
}

const (
	targetSender        = "noreply@tm.openai.com"
	checkInterval       = 30 * time.Second
	dataDir             = "../data/gpt"
	appVersion          = "web-mail@5.0.311.1"
	envFile             = ".env"
	lastCheckTimeEnvKey = "LAST_EMAIL_CHECK_TIME"
	sessionTokenEnvKey  = "SECURE_NEXT_AUTH_SESSION_TOKEN"
	lastFileMD5EnvKey   = "LAST_FILE_MD5"
)

var (
	downloadLinkRegex = regexp.MustCompile(`https://chatgpt\.com/backend-api/estuary/content\?[^\s"<>]+`)
	sessionToken      string
)

func main() {
	// 加载 .env 文件（如果存在）
	_ = godotenv.Load(envFile)

	username := os.Getenv("GO_PROTON_API_TEST_USERNAME")
	password := os.Getenv("GO_PROTON_API_TEST_PASSWORD")
	sessionToken = os.Getenv(sessionTokenEnvKey)

	if username == "" || password == "" {
		log.Fatal("错误: 必须设置环境变量 GO_PROTON_API_TEST_USERNAME 和 GO_PROTON_API_TEST_PASSWORD")
	}

	if sessionToken == "" {
		log.Fatalf("错误: 必须设置环境变量 %s", sessionTokenEnvKey)
	}

	// 确保父目录存在（./data）
	parentDir := filepath.Dir(dataDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		log.Fatalf("创建父目录失败: %v", err)
	}

	// 获取上次检查时间戳
	lastCheckTime := getLastCheckTime()

	m := proton.New(proton.WithAppVersion(appVersion))
	defer m.Close()

	ctx := context.Background()

	log.Printf("正在登录邮箱: %s", username)
	c, auth, err := m.NewClientWithLogin(ctx, username, []byte(password))
	if err != nil {
		log.Fatalf("登录失败: %v", err)
	}
	defer c.Close()

	log.Printf("登录成功, UserID: %s", auth.UserID)

	if auth.TwoFA.Enabled != 0 {
		log.Fatal("此账户启用了 2FA，请禁用或实现 2FA 处理逻辑")
	}

	user, err := c.GetUser(ctx)
	if err != nil {
		log.Fatalf("获取用户信息失败: %v", err)
	}

	addresses, err := c.GetAddresses(ctx)
	if err != nil {
		log.Fatalf("获取地址失败: %v", err)
	}

	salts, err := c.GetSalts(ctx)
	if err != nil {
		log.Fatalf("获取 salts 失败: %v", err)
	}

	saltedKeyPass, err := salts.SaltForKey([]byte(password), user.Keys.Primary().ID)
	if err != nil {
		log.Fatalf("生成 salted key password 失败: %v", err)
	}

	_, addrKRs, err := proton.Unlock(user, addresses, saltedKeyPass, SimplePanicHandler{})
	if err != nil {
		log.Fatalf("解锁密钥失败: %v", err)
	}

	log.Printf("密钥解锁成功，开始监控邮件... (上次检查时间: %s)", lastCheckTime.Format("2006-01-02 15:04:05"))

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// 首次检查
	if shouldExit := checkNewEmails(ctx, c, addrKRs, &lastCheckTime); shouldExit {
		log.Println("检测到新邮件，程序结束")
		return
	}

	// 定时检查
	for range ticker.C {
		if shouldExit := checkNewEmails(ctx, c, addrKRs, &lastCheckTime); shouldExit {
			log.Println("检测到新邮件，程序结束")
			return
		}
	}
}

// getLastCheckTime 从环境变量获取上次检查时间，如果不存在则返回当前时间
func getLastCheckTime() time.Time {
	timeStr := os.Getenv(lastCheckTimeEnvKey)
	if timeStr == "" {
		now := time.Now()
		log.Printf("未找到环境变量 %s，使用当前时间: %s", lastCheckTimeEnvKey, now.Format("2006-01-02 15:04:05"))
		return now
	}

	timestamp, err := strconv.ParseInt(timeStr, 10, 64)
	if err != nil {
		now := time.Now()
		log.Printf("解析时间戳失败: %v，使用当前时间: %s", err, now.Format("2006-01-02 15:04:05"))
		return now
	}

	return time.Unix(timestamp, 0)
}

// updateLastCheckTime 将时间戳写入环境变量（通过更新 .env 文件）
func updateLastCheckTime(t time.Time) error {
	return updateEnvVar(lastCheckTimeEnvKey, strconv.FormatInt(t.Unix(), 10))
}

// updateLastFileMD5 将文件 MD5 写入环境变量
func updateLastFileMD5(md5Hash string) error {
	return updateEnvVar(lastFileMD5EnvKey, md5Hash)
}

// updateEnvVar 更新单个环境变量到 .env 文件
func updateEnvVar(key, value string) error {
	// 读取现有 .env 文件内容
	envMap, err := godotenv.Read(envFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("读取 .env 文件失败: %w", err)
	}
	if envMap == nil {
		envMap = make(map[string]string)
	}

	// 更新键值
	envMap[key] = value

	// 写回 .env 文件
	if err := godotenv.Write(envMap, envFile); err != nil {
		return fmt.Errorf("写入 .env 文件失败: %w", err)
	}

	// 更新当前进程的环境变量
	os.Setenv(key, value)

	return nil
}

// calculateFileMD5 计算文件的 MD5 哈希值
func calculateFileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("计算 MD5 失败: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// getLastFileMD5 从环境变量获取上次的文件 MD5
func getLastFileMD5() string {
	return os.Getenv(lastFileMD5EnvKey)
}

func checkNewEmails(ctx context.Context, c *proton.Client, addrKRs map[string]*crypto.KeyRing, lastCheckTime *time.Time) bool {
	filter := proton.MessageFilter{
		LabelID: proton.InboxLabel,
		Desc:    proton.Bool(true),
	}

	messages, err := c.GetMessageMetadataPage(ctx, 0, 10, filter)
	if err != nil {
		log.Printf("获取邮件失败: %v", err)
		return false
	}

	if len(messages) == 0 {
		log.Println("收件箱没有邮件")
		return false
	}

	// 第一步：找到所有符合条件的新邮件，并选出最新的一封
	var newestMsg *proton.MessageMetadata
	var newestTime time.Time

	for _, msg := range messages {
		// 跳过非目标发件人
		if msg.Sender.Address != targetSender {
			continue
		}

		// 邮件时间（Unix时间戳秒）
		msgTime := time.Unix(msg.Time, 0)

		// 只处理比上次检查时间更新的邮件
		if !msgTime.After(*lastCheckTime) {
			continue
		}

		// 记录最新的邮件
		if newestMsg == nil || msgTime.After(newestTime) {
			newestMsg = &msg
			newestTime = msgTime
		}
	}

	// 如果没有找到符合条件的新邮件
	if newestMsg == nil {
		log.Printf("[%s] 没有新邮件", time.Now().Format("2006-01-02 15:04:05"))
		return false
	}

	// 第二步：处理最新的那封邮件
	log.Printf("发现来自 %s 的新邮件 (时间: %s): %s",
		targetSender,
		newestTime.Format("2006-01-02 15:04:05"),
		newestMsg.Subject)

	fullMessage, err := c.GetMessage(ctx, newestMsg.ID)
	if err != nil {
		log.Printf("获取邮件详情失败: %v", err)
		// 获取详情失败，不更新时间戳，不退出
		return false
	}

	decrypted, err := fullMessage.Decrypt(addrKRs[fullMessage.AddressID])
	if err != nil {
		log.Printf("解密邮件失败: %v，尝试使用未加密正文", err)
		decrypted = []byte(fullMessage.Body)
	}

	body := string(decrypted)
	links := downloadLinkRegex.FindAllString(body, -1)
	if len(links) == 0 {
		log.Println("邮件中未找到下载链接")
		// 没有下载链接，不更新时间戳，不退出
		return false
	}

	log.Printf("找到 %d 个下载链接", len(links))

	var downloadSuccess bool
	for i, link := range links {
		log.Printf("处理链接 %d/%d: %s", i+1, len(links), link)

		zipPath, err := downloadFile(link)
		if err != nil {
			log.Printf("下载失败: %v", err)
			continue
		}

		log.Printf("下载成功: %s", zipPath)

		// 计算下载文件的 MD5
		log.Println("正在计算文件 MD5...")
		currentMD5, err := calculateFileMD5(zipPath)
		if err != nil {
			log.Printf("计算 MD5 失败: %v", err)
			os.Remove(zipPath)
			continue
		}

		log.Printf("文件 MD5: %s", currentMD5)

		// 获取上次的 MD5
		lastMD5 := getLastFileMD5()
		if lastMD5 != "" {
			log.Printf("上次文件 MD5: %s", lastMD5)
			if currentMD5 == lastMD5 {
				log.Println("文件 MD5 与上次一致，跳过解压，删除压缩包")
				if err := os.Remove(zipPath); err != nil {
					log.Printf("删除压缩包失败: %v", err)
				} else {
					log.Printf("压缩包已删除: %s", zipPath)
				}
				// MD5 一致也算成功，因为文件没有变化
				downloadSuccess = true
				continue
			}
			log.Println("文件 MD5 不同，继续解压")
		} else {
			log.Println("未找到上次 MD5 记录，继续解压")
		}

		// 解压前清理目标目录
		if err := cleanAndCreateDir(dataDir); err != nil {
			log.Printf("清理目标目录失败: %v", err)
			os.Remove(zipPath)
			continue
		}

		if err := unzipFile(zipPath, dataDir); err != nil {
			log.Printf("解压失败: %v", err)
			os.Remove(zipPath)
			continue
		}

		log.Printf("解压成功，正在删除压缩包...")

		if err := os.Remove(zipPath); err != nil {
			log.Printf("删除压缩包失败: %v", err)
		} else {
			log.Printf("压缩包已删除: %s", zipPath)
		}

		// 更新 MD5 到环境变量
		if err := updateLastFileMD5(currentMD5); err != nil {
			log.Printf("更新 MD5 失败: %v", err)
		} else {
			log.Printf("已更新文件 MD5: %s", currentMD5)
		}

		downloadSuccess = true
	}

	// 只有下载成功才更新时间戳
	if downloadSuccess {
		*lastCheckTime = newestTime
		if err := updateLastCheckTime(newestTime); err != nil {
			log.Printf("更新时间戳失败: %v", err)
		} else {
			log.Printf("已更新检查时间戳: %s", newestTime.Format("2006-01-02 15:04:05"))
		}
		log.Println("下载处理完成，程序退出")
		return true
	} else {
		log.Println("下载失败，不更新时间戳，程序继续监控")
		return false
	}
}

// cleanAndCreateDir 清理并重新创建目录
func cleanAndCreateDir(dir string) error {
	// 检查目录是否存在
	if _, err := os.Stat(dir); err == nil {
		// 目录存在，删除它
		log.Printf("删除现有目录: %s", dir)
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("删除目录失败: %w", err)
		}
	}

	// 创建新目录
	log.Printf("创建目录: %s", dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	return nil
}

func downloadFile(url string) (string, error) {
	filename := extractFilenameFromURL(url)
	if filename == "" {
		filename = fmt.Sprintf("download_%d.zip", time.Now().Unix())
	}

	filepath := filepath.Join(dataDir, filename)

	out, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	// 创建 HTTP 请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	// 添加 Headers
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,ko;q=0.7")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Priority", "u=0, i")
	req.Header.Set("Sec-CH-UA", `"Google Chrome";v="141", "Not?A_Brand";v="8", "Chromium";v="141"`)
	req.Header.Set("Sec-CH-UA-Arch", "arm")
	req.Header.Set("Sec-CH-UA-Bitness", "64")
	req.Header.Set("Sec-CH-UA-Full-Version", "141.0.7390.66")
	req.Header.Set("Sec-CH-UA-Full-Version-List", `"Google Chrome";v="141.0.7390.66", "Not?A_Brand";v="8.0.0.0", "Chromium";v="141.0.7390.66"`)
	req.Header.Set("Sec-CH-UA-Mobile", "?0")
	req.Header.Set("Sec-CH-UA-Model", "")
	req.Header.Set("Sec-CH-UA-Platform", `"macOS"`)
	req.Header.Set("Sec-CH-UA-Platform-Version", `"14.4.1"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36")

	// 添加 Cookie
	req.AddCookie(&http.Cookie{
		Name:  "__Secure-next-auth.session-token",
		Value: sessionToken,
	})

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP 状态码错误: %d", resp.StatusCode)
	}

	// 创建进度条
	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		fmt.Sprintf("下载 %s", filename),
	)

	// 使用进度条包装的 Writer 进行下载
	_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
	if err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	// 确保进度条完成
	bar.Finish()
	fmt.Println() // 添加换行使输出更清晰

	return filepath, nil
}

func extractFilenameFromURL(url string) string {
	re := regexp.MustCompile(`id=([^&]+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		id := matches[1]
		if len(id) > 16 {
			id = id[:16]
		}
		return fmt.Sprintf("openai_%s.zip", id)
	}
	return ""
}

func unzipFile(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("打开 zip 文件失败: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		err := extractFile(f, destDir)
		if err != nil {
			return fmt.Errorf("解压文件 %s 失败: %w", f.Name, err)
		}
	}

	return nil
}

func extractFile(f *zip.File, destDir string) error {
	path := filepath.Join(destDir, f.Name)

	if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return fmt.Errorf("非法文件路径: %s", f.Name)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(path, f.Mode())
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	_, err = io.Copy(outFile, rc)
	return err
}
