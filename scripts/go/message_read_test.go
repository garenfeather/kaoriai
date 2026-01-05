package proton_test

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/ProtonMail/gluon/async"
	"github.com/ProtonMail/go-proton-api"
	"github.com/stretchr/testify/require"
)

// TestReadLastMessage 测试读取用户最后一封邮件
// 需要设置环境变量:
// - GO_PROTON_API_TEST_USERNAME: Proton 账户用户名
// - GO_PROTON_API_TEST_PASSWORD: Proton 账户密码
func TestReadLastMessage(t *testing.T) {
	// 从环境变量读取用户名和密码
	username := os.Getenv("GO_PROTON_API_TEST_USERNAME")
	password := os.Getenv("GO_PROTON_API_TEST_PASSWORD")

	// 如果环境变量未设置,跳过测试
	if username == "" || password == "" {
		t.Skip("Skipping test: GO_PROTON_API_TEST_USERNAME or GO_PROTON_API_TEST_PASSWORD environment variable not set")
		return
	}

	// 创建 Manager - 使用默认的 Proton Mail API URL
	// 需要设置有效的 AppVersion,否则真实 API 会拒绝请求
	m := proton.New(
		proton.WithAppVersion("web-mail@5.0.311.1"),
	)
	defer m.Close()

	ctx := context.Background()

	// 使用用户名和密码登录
	t.Logf("Logging in as user: %s", username)
	c, auth, err := m.NewClientWithLogin(ctx, username, []byte(password))
	require.NoError(t, err, "Failed to login")
	defer c.Close()

	t.Logf("Login successful, UserID: %s", auth.UserID)

	// 检查是否需要 2FA
	if auth.TwoFA.Enabled != 0 {
		t.Skip("2FA is enabled for this account, please disable it or implement 2FA handling")
		return
	}

	// 获取收件箱中的消息元数据,按时间降序排列,只获取第一页
	// Desc=true 表示降序(最新的在前)
	filter := proton.MessageFilter{
		LabelID: proton.InboxLabel, // 收件箱
		Desc:    proton.Bool(true),  // 降序排列
	}

	t.Log("Fetching messages from inbox...")
	messages, err := c.GetMessageMetadataPage(ctx, 0, 1, filter)
	require.NoError(t, err, "Failed to get messages")

	if len(messages) == 0 {
		t.Log("No messages found in inbox")
		return
	}

	// 获取最后一封邮件的详细信息
	lastMessage := messages[0]
	t.Logf("Last message ID: %s", lastMessage.ID)
	t.Logf("Subject: %s", lastMessage.Subject)
	t.Logf("From: %s", lastMessage.Sender.String())
	t.Logf("Time: %d", lastMessage.Time)

	// 获取完整的消息内容
	fullMessage, err := c.GetMessage(ctx, lastMessage.ID)
	require.NoError(t, err, "Failed to get full message")

	// 获取邮件正文
	body := fullMessage.Body

	// 如果正文为空,尝试使用 Header
	if body == "" {
		t.Log("Message body is empty, might be encrypted or no body content")
		body = fullMessage.Header
	}

	// 输出正文前50个字符
	bodyPreview := body
	if len(body) > 50 {
		// 处理 UTF-8 字符,避免截断多字节字符
		runes := []rune(body)
		if len(runes) > 50 {
			bodyPreview = string(runes[:50])
		} else {
			bodyPreview = string(runes)
		}
	}

	// 去除首尾空白
	bodyPreview = strings.TrimSpace(bodyPreview)

	t.Logf("Message body preview (first 50 chars):")
	t.Logf("%s", bodyPreview)
	t.Logf("Message MIME type: %s", fullMessage.MIMEType)
	t.Logf("Message has %d attachments", len(fullMessage.Attachments))

	// 确保我们至少获取到了一些内容
	require.NotEmpty(t, fullMessage.Subject, "Message should have a subject")
}

// TestReadLastMessageWithDecryption 测试读取并解密用户最后一封邮件
// 这个测试展示了如何处理加密的邮件内容
func TestReadLastMessageWithDecryption(t *testing.T) {
	// 从环境变量读取用户名和密码
	username := os.Getenv("GO_PROTON_API_TEST_USERNAME")
	password := os.Getenv("GO_PROTON_API_TEST_PASSWORD")

	// 如果环境变量未设置,跳过测试
	if username == "" || password == "" {
		t.Skip("Skipping test: GO_PROTON_API_TEST_USERNAME or GO_PROTON_API_TEST_PASSWORD environment variable not set")
		return
	}

	// 创建 Manager - 使用默认的 Proton Mail API URL
	// 需要设置有效的 AppVersion,否则真实 API 会拒绝请求
	m := proton.New(
		proton.WithAppVersion("web-mail@5.0.311.1"),
	)
	defer m.Close()

	ctx := context.Background()

	// 使用用户名和密码登录
	t.Logf("Logging in as user: %s", username)
	c, auth, err := m.NewClientWithLogin(ctx, username, []byte(password))
	require.NoError(t, err, "Failed to login")
	defer c.Close()

	t.Logf("Login successful, UserID: %s", auth.UserID)

	// 检查是否需要 2FA
	if auth.TwoFA.Enabled != 0 {
		t.Skip("2FA is enabled for this account, please disable it or implement 2FA handling")
		return
	}

	// 获取用户信息和地址
	user, err := c.GetUser(ctx)
	require.NoError(t, err, "Failed to get user info")
	t.Logf("User: %s, Email: %s", user.Name, user.Email)

	addresses, err := c.GetAddresses(ctx)
	require.NoError(t, err, "Failed to get addresses")
	t.Logf("Found %d addresses", len(addresses))

	// 获取 salts 并生成 salted key password
	salts, err := c.GetSalts(ctx)
	require.NoError(t, err, "Failed to get salts")

	saltedKeyPass, err := salts.SaltForKey([]byte(password), user.Keys.Primary().ID)
	require.NoError(t, err, "Failed to salt key password")

	// 解锁用户密钥环以便解密邮件
	_, addrKRs, err := proton.Unlock(user, addresses, saltedKeyPass, async.NoopPanicHandler{})
	require.NoError(t, err, "Failed to unlock keys")
	t.Log("Successfully unlocked user keys")

	// 获取收件箱中的消息元数据
	filter := proton.MessageFilter{
		LabelID: proton.InboxLabel,
		Desc:    proton.Bool(true),
	}

	t.Log("Fetching messages from inbox...")
	messages, err := c.GetMessageMetadataPage(ctx, 0, 1, filter)
	require.NoError(t, err, "Failed to get messages")

	if len(messages) == 0 {
		t.Log("No messages found in inbox")
		return
	}

	// 获取最后一封邮件的详细信息
	lastMessage := messages[0]
	t.Logf("Processing message: %s", lastMessage.Subject)

	// 获取完整的消息内容
	fullMessage, err := c.GetMessage(ctx, lastMessage.ID)
	require.NoError(t, err, "Failed to get full message")

	// 输出邮件信息
	t.Logf("Message ID: %s", fullMessage.ID)
	t.Logf("Subject: %s", fullMessage.Subject)
	t.Logf("From: %s", fullMessage.Sender.String())
	t.Logf("MIME Type: %s", fullMessage.MIMEType)
	t.Logf("Flags: %d", fullMessage.Flags)
	t.Logf("Body length (encrypted): %d bytes", len(fullMessage.Body))

	// 解密邮件正文
	decrypted, err := fullMessage.Decrypt(addrKRs[fullMessage.AddressID])
	if err != nil {
		t.Logf("Failed to decrypt message body: %v", err)
		t.Log("Showing encrypted body preview instead:")
		bodyPreview := fullMessage.Body
		if len(fullMessage.Body) > 50 {
			runes := []rune(fullMessage.Body)
			if len(runes) > 50 {
				bodyPreview = string(runes[:50])
			}
		}
		t.Logf("Body preview (encrypted): %s", strings.TrimSpace(bodyPreview))
	} else {
		t.Logf("Successfully decrypted message, length: %d bytes", len(decrypted))

		// 提取纯文本内容(去除 HTML 标签)
		decryptedStr := string(decrypted)
		plainText := extractPlainText(decryptedStr)

		// 输出前300个字符的可读文本
		textPreview := plainText
		runes := []rune(plainText)
		if len(runes) > 300 {
			textPreview = string(runes[:300])
		}

		t.Logf("Plain text content (first 300 chars):")
		t.Logf("%s", strings.TrimSpace(textPreview))
		t.Logf("\nTotal plain text length: %d characters", len(runes))
	}
}

// extractPlainText 从 HTML 中提取纯文本内容
func extractPlainText(html string) string {
	// 移除 script 和 style 标签及其内容
	reScript := regexp.MustCompile(`(?i)<script[^>]*>[\s\S]*?</script>`)
	html = reScript.ReplaceAllString(html, "")
	reStyle := regexp.MustCompile(`(?i)<style[^>]*>[\s\S]*?</style>`)
	html = reStyle.ReplaceAllString(html, "")

	// 移除所有 HTML 标签
	reTag := regexp.MustCompile(`<[^>]*>`)
	text := reTag.ReplaceAllString(html, "")

	// 解码常见的 HTML 实体
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	// 移除多余的空白字符
	reWhitespace := regexp.MustCompile(`\s+`)
	text = reWhitespace.ReplaceAllString(text, " ")

	// 移除首尾空白
	text = strings.TrimSpace(text)

	return text
}
