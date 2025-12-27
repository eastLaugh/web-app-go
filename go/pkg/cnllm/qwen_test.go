package cnllm_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/eastLaugh/web-app-go/go/pkg/cnllm"
	"github.com/tmc/langchaingo/llms"
)

func TestQwen(t *testing.T) {
	llm := Must(cnllm.Qwen())
	content := Must(llm.GenerateContent(context.Background(), []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "Hello, world!"),
	}))
	_ = content
	runtime.Breakpoint()
}

func Must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}
