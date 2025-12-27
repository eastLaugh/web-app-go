package cnllm

import (
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
	)

}
