package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// Client embedding 客户端
type Client struct {
	apiKey     string
	baseURL    string
	dimension  int
	httpClient *http.Client
}

// NewClient 创建 embedding 客户端
func NewClient() *Client {
	apiKey := os.Getenv("ALIBABA_EMBEDDING_API_KEY")
	if apiKey == "" {
		log.Printf("警告: ALIBABA_EMBEDDING_API_KEY 未设置，embedding 功能将不可用")
	}

	dimension := 1024 // 默认 1024 维
	if dimStr := os.Getenv("ALIBABA_EMBEDDING_DIMENSION"); dimStr != "" {
		if _, err := fmt.Sscanf(dimStr, "%d", &dimension); err != nil {
			log.Printf("警告: ALIBABA_EMBEDDING_DIMENSION 格式错误，使用默认值 1024: %v", err)
			dimension = 1024
		}
	}

	return &Client{
		apiKey:     apiKey,
		baseURL:    "https://dashscope.aliyuncs.com/compatible-mode/v1/embeddings",
		dimension:  dimension,
		httpClient: &http.Client{},
	}
}

// EmbedRequest embedding 请求
type EmbedRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Parameters struct {
		TextType string `json:"text_type,omitempty"`
	} `json:"parameters,omitempty"`
}

// EmbedResponse embedding 响应（OpenAI 兼容格式）
type EmbedResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// Embed 生成文本的向量表示
// textType: "document" 用于构建索引, "query" 用于查询
func (c *Client) Embed(ctx context.Context, texts []string, textType string) ([][]float32, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("ALIBABA_EMBEDDING_API_KEY 未设置")
	}

	if len(texts) == 0 {
		return nil, fmt.Errorf("文本列表为空")
	}

	if textType == "" {
		textType = "document" // 默认值
	}

	// 构建请求体，使用 OpenAI 兼容接口格式
	// input 可以是字符串（单个）或字符串数组（批量）
	var input interface{}
	if len(texts) == 1 {
		input = texts[0] // 单个字符串
	} else {
		input = texts // 字符串数组
	}

	reqBody := map[string]interface{}{
		"model": "text-embedding-v4",
		"input": input,
		"parameters": map[string]interface{}{
			"text_type": textType,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("API 请求体: %s", string(jsonData))
		return nil, fmt.Errorf("API 返回错误: %d, %s", resp.StatusCode, string(body))
	}

	var embedResp EmbedResponse
	if err := json.Unmarshal(body, &embedResp); err != nil {
		log.Printf("API 响应体: %s", string(body))
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if len(embedResp.Data) != len(texts) {
		return nil, fmt.Errorf("返回的向量数量不匹配: 期望 %d, 实际 %d", len(texts), len(embedResp.Data))
	}

	// 按 Index 排序（确保顺序正确）
	result := make([][]float32, len(embedResp.Data))
	for _, item := range embedResp.Data {
		if item.Index >= 0 && item.Index < len(result) {
			result[item.Index] = item.Embedding
		}
	}

	return result, nil
}

// EmbedSingle 生成单个文本的向量表示（便捷方法）
// textType: "document" 用于构建索引, "query" 用于查询
func (c *Client) EmbedSingle(ctx context.Context, text string, textType string) ([]float32, error) {
	embeddings, err := c.Embed(ctx, []string{text}, textType)
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("未返回向量")
	}
	return embeddings[0], nil
}
