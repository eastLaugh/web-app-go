package ports

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/eastLaugh/web-app-go/go/internal/api"
	"github.com/openai/openai-go/v3"
)

const maxToolRounds = 10

func (s *Server) PostChat(w http.ResponseWriter, r *http.Request) {
	var request api.PostChatJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	messages := api.ChatRequestToOpenAIMessages(request)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher, ok := w.(http.Flusher)
	if !ok {
		panic("不支持 SSE")
	}

	ctx := context.WithValue(r.Context(), Server{}, s)
	for range maxToolRounds {
		params := openai.ChatCompletionNewParams{
			Messages: messages,
			Model:    openai.ChatModel(s.chatModel),
			Tools:    s.Tools.ToParams(),
		}

		completion, err := s.client.Chat.Completions.New(ctx, params)
		if err != nil {
			fmt.Fprintf(w, "data: [ERROR]%s\n\n", err.Error())
			flusher.Flush()
			return
		}

		if len(completion.Choices) == 0 {
			fmt.Fprintf(w, "data: [ERROR]empty choices\n\n")
			flusher.Flush()
			return
		}

		msg := completion.Choices[0].Message
		messages = append(messages, msg.ToParam())

		if len(msg.ToolCalls) == 0 {
			// 无 tool call，为最终回复，以 SSE 写出
			content := msg.Content
			if content != "" {
				escaped, _ := json.Marshal(content)
				fmt.Fprintf(w, "data: %s\n\n", escaped)
			}
			flusher.Flush()
			return
		}

		// 执行 tool calls，追加 tool 结果到 messages
		for _, tc := range msg.ToolCalls {
			var result string
			switch v := tc.AsAny().(type) {
			case openai.ChatCompletionMessageFunctionToolCall:
				var err error
				result, err = s.Tools.Execute(ctx, v.Function.Name, v.Function.Arguments)
				if err != nil {
					result = err.Error()
				}
			default:
				result = "不支持的 tool call"
			}
			messages = append(messages, openai.ToolMessage(result, tc.ID))
		}
	}

	fmt.Fprintf(w, "data: [ERROR]超过最大 tool 轮数\n\n")
	flusher.Flush()
}

func (s *Server) runVectorSearch(ctx context.Context, query string) (string, error) {
	docs, err := s.vecAdapter.Search(ctx, query, 5)
	if err != nil {
		return "", err
	}
	if len(docs) == 0 {
		return "未找到相关文档", nil
	}
	return fmt.Sprintf("找到 %d 个相关文档：\n\n%s", len(docs), formatDocs(docs)), nil
}
