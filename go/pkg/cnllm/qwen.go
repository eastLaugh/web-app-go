package cnllm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/tmc/langchaingo/llms/openai"
)

const (
	QWEN3_MAX = "qwen3-max"
	QWEN_PLUS = "qwen-plus"
)

func Qwen() (*openai.LLM, error) {
	return openai.New(
		openai.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
		openai.WithToken(os.Getenv("QWEN_API_KEY")),
		openai.WithModel(QWEN_PLUS),

		//https://bailian.console.aliyun.com/?tab=model#/model-market/detail/text-embedding-v4
		openai.WithEmbeddingModel("text-embedding-v4"),

		openai.WithHTTPClient(&http.Client{
			Transport: &myRoundTripper{
				RoundTripper: http.DefaultTransport,
			},
		}),
	)

}

// 可观测性
type myRoundTripper struct {
	http.RoundTripper
}

func (t *myRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}

	// m["enable_thinking"] = true

	newBody, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}

	req.Body = io.NopCloser(bytes.NewReader(newBody))
	req.ContentLength = int64(len(newBody))
	slog.Info("request", "body", string(newBody))
	fmt.Println(string(newBody))
	return t.RoundTripper.RoundTrip(req)
}
