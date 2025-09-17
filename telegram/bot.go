package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/net/proxy"
	"log"
	"net/http"
	"net/url"
	"time"

	"hyper-notify-bot/config"
)

// TelegramBot 结构体封装了Telegram Bot的功能
type TelegramBot struct {
	Token  string
	ChatID string
	Client *http.Client
}

// NewTelegramBot 创建一个新的TelegramBot实例
func NewTelegramBot(token, chatID string, TelegramProxy string) *TelegramBot {
	// 创建Transport
	transport := &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  true,
		MaxConnsPerHost:     5,
		MaxIdleConnsPerHost: 2,
	}

	//proxyURL := "http://127.0.0.1:7777"
	// 如果提供了代理URL，则设置代理
	if TelegramProxy != "" {
		// 解析代理URL
		proxyUrl, err := url.Parse(TelegramProxy)
		if err != nil {
			return nil
		}

		// 根据代理的协议进行处理
		switch proxyUrl.Scheme {
		case "http", "https":
			transport.Proxy = http.ProxyURL(proxyUrl)
		case "socks5":
			// 创建一个socks5的dialer
			dialer, err := proxy.FromURL(proxyUrl, proxy.Direct)
			if err != nil {
				return nil
			}
			// 将dialer转换为ContextDialer，因为新版本要求
			dialerContext, ok := dialer.(proxy.ContextDialer)
			if !ok {
				return nil
			}
			// 设置Transport的DialContext
			transport.DialContext = dialerContext.DialContext
		default:
			return nil
		}
	}
	return &TelegramBot{
		Token:  token,
		ChatID: chatID,
		Client: &http.Client{
			Timeout:   30 * time.Second, // 增加超时时间
			Transport: transport,
		},
	}
}

// sendMessageRequest 定义Telegram发送消息的请求结构
type sendMessageRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// sendMessageResponse 定义Telegram发送消息的响应结构
type sendMessageResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
	ErrorCode   int    `json:"error_code"`
}

// SendMessage 发送消息到Telegram
func (b *TelegramBot) SendMessage(ctx context.Context, message string, parseMode string) error {
	// 创建API URL
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.Token)

	// 构建请求体
	reqBody := sendMessageRequest{
		ChatID:    b.ChatID,
		Text:      message,
		ParseMode: parseMode,
	}

	// 序列化请求体
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := b.Client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API返回非200状态码: %d", resp.StatusCode)
	}

	// 解析响应
	var respBody sendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查Telegram API返回的状态
	if !respBody.OK {
		return fmt.Errorf("Telegram API错误[%d]: %s", respBody.ErrorCode, respBody.Description)
	}

	log.Printf("消息发送成功")
	return nil
}

// SendWithRetry 带重试机制的消息发送
func (b *TelegramBot) SendWithRetry(ctx context.Context, message string, parseMode string, cfg *config.Config) error {
	var lastErr error
	attempts := 0

	for attempts < cfg.RetryCount {
		attempts++
		log.Printf("尝试发送消息 (尝试 %d/%d)", attempts, cfg.RetryCount)

		// 创建带超时的上下文
		reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		// 尝试发送消息
		err := b.SendMessage(reqCtx, message, parseMode)
		if err == nil {
			return nil // 发送成功
		}

		lastErr = err
		log.Printf("发送失败 (尝试 %d/%d): %v", attempts, cfg.RetryCount, err)

		// 如果错误是永久性的，不进行重试
		if isPermanentError(err) {
			log.Printf("遇到永久性错误，停止重试: %v", err)
			break
		}

		// 等待重试延迟
		select {
		case <-time.After(cfg.RetryDelay):
			// 继续下一次尝试
		case <-ctx.Done():
			return fmt.Errorf("发送被取消: %v", ctx.Err())
		}
	}

	return fmt.Errorf("发送失败，已达最大重试次数: %v", lastErr)
}

// isPermanentError 判断是否为永久性错误（不需要重试）
func isPermanentError(err error) bool {
	if err == nil {
		return false
	}

	// 检查是否是特定类型的错误
	if apiErr, ok := err.(interface{ ErrorCode() int }); ok {
		errorCode := apiErr.ErrorCode()

		// Telegram API 永久性错误码
		permanentErrors := map[int]bool{
			400: true, // Bad Request
			401: true, // Unauthorized
			403: true, // Forbidden
			404: true, // Not Found
			420: true, // Too Many Requests (但有时可重试)
		}

		if _, exists := permanentErrors[errorCode]; exists {
			return true
		}
	}

	// 根据错误信息判断
	errorMsg := err.Error()
	permanentErrorMsgs := []string{
		"chat not found",
		"bot was blocked by the user",
		"message is too long",
		"invalid chat_id",
	}

	for _, msg := range permanentErrorMsgs {
		if contains(errorMsg, msg) {
			return true
		}
	}

	return false
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

// GetBotInfo 获取Telegram Bot的基本信息
func (b *TelegramBot) GetBotInfo(ctx context.Context) (map[string]interface{}, error) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", b.Token)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	resp, err := b.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回非200状态码: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if ok, exists := result["ok"]; !exists || !ok.(bool) {
		return nil, fmt.Errorf("Telegram API错误: %v", result["description"])
	}

	return result, nil
}

// SendLargeMessage 发送长消息（自动分割）
func (b *TelegramBot) SendLargeMessage(ctx context.Context, message string, parseMode string) error {
	// Telegram 消息长度限制为4096个字符
	const maxLength = 4000 // 留出一些空间

	if len(message) <= maxLength {
		return b.SendWithRetry(ctx, message, parseMode, &config.Config{
			RetryCount: 3,
			RetryDelay: 5 * time.Second,
		})
	}

	// 分割长消息
	messages := splitMessage(message, maxLength)

	for i, msg := range messages {
		// 添加消息序号
		fullMsg := fmt.Sprintf("(%d/%d)\n%s", i+1, len(messages), msg)

		if err := b.SendWithRetry(ctx, fullMsg, parseMode, &config.Config{
			RetryCount: 3,
			RetryDelay: 5 * time.Second,
		}); err != nil {
			return fmt.Errorf("发送消息部分 %d/%d 失败: %v", i+1, len(messages), err)
		}

		// 在消息之间添加短暂延迟
		if i < len(messages)-1 {
			select {
			case <-time.After(500 * time.Millisecond):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return nil
}

// splitMessage 将长消息分割为多个小消息
func splitMessage(message string, maxLength int) []string {
	var parts []string
	runes := []rune(message) // 处理多字节字符

	for len(runes) > 0 {
		// 如果剩余部分小于最大长度，直接添加
		if len(runes) <= maxLength {
			parts = append(parts, string(runes))
			break
		}

		// 在最大长度处寻找合适的分割点
		splitIndex := maxLength

		// 尝试在句子结束处分割
		for i := maxLength - 1; i > maxLength-100 && i > 0; i-- {
			if runes[i] == '\n' || runes[i] == '.' || runes[i] == ';' {
				splitIndex = i + 1
				break
			}
		}

		// 添加分割部分
		parts = append(parts, string(runes[:splitIndex]))
		runes = runes[splitIndex:]
	}

	return parts
}
