// Package deepseek 提供 OpenAI 兼容的 DeepSeek 聊天客户端。
// 所有 HTTP 调用集中于此，service 层通过本包导出的方法与常量消费，便于替换其他 AI 服务。
package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultBaseURL DeepSeek API 默认地址。
	DefaultBaseURL = "https://api.deepseek.com"
	// DefaultModel 默认聊天模型。
	DefaultModel = "deepseek-chat"

	// requestTimeout 单次请求超时（AI 生成较慢，留足 60s）。
	requestTimeout = 60 * time.Second
	// errBodyLimit 错误响应体在错误信息中的截断长度。
	errBodyLimit = 200
)

// Client DeepSeek 聊天客户端（OpenAI 兼容接口）。
type Client struct {
	HTTPClient *http.Client
	BaseURL    string
	Model      string
	apiKey     string // 私有字段，禁止写入日志。
}

// New 创建 DeepSeek 聊天客户端，model 为空时使用默认模型 deepseek-chat。
func New(apiKey, model string) *Client {
	if model == "" {
		model = DefaultModel
	}
	return &Client{
		HTTPClient: &http.Client{},
		BaseURL:    strings.TrimRight(DefaultBaseURL, "/"),
		Model:      model,
		apiKey:     apiKey,
	}
}

// chatRequest OpenAI 兼容的聊天请求体。
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

// chatMessage 单条消息（role: system/user/assistant）。
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse OpenAI 兼容的聊天响应体。
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Chat 发送系统提示与用户提示，调用 POST /chat/completions 并返回模型回复文本。
// 在调用方 ctx 之上叠加 60s 超时（AI 生成较慢），任一先到即终止。
func (c *Client) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	payload, err := json.Marshal(chatRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.7,
		MaxTokens:   2000,
	})
	if err != nil {
		return "", fmt.Errorf("deepseek: marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("deepseek: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("deepseek: sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("deepseek: reading response: %w", err)
	}

	// 非 2xx 返回状态码与响应体片段（截断 200 字符），便于排查且不泄露 apiKey。
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		preview := string(respBody)
		if len(preview) > errBodyLimit {
			preview = preview[:errBodyLimit]
		}
		return "", fmt.Errorf("deepseek: HTTP %d: %s", resp.StatusCode, preview)
	}

	var result chatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("deepseek: decoding response: %w", err)
	}
	if len(result.Choices) == 0 || result.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("deepseek: response contains no content")
	}
	return result.Choices[0].Message.Content, nil
}
