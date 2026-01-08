//go:build noagent

package ports

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"

	"github.com/eastLaugh/web-app-go/go/internal/api"
	"github.com/tmc/langchaingo/llms"
)

func (s *Server) PostChat(w http.ResponseWriter, r *http.Request) {
	var request api.PostChatJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	lastUserMsg := ""
	for i := len(request.Messages) - 1; i >= 0; i-- {
		if request.Messages[i].Role == api.User {
			lastUserMsg = request.Messages[i].Content
			break
		}
	}

	if lastUserMsg != "" {
		docs, err := s.vecAdapter.Search(r.Context(), lastUserMsg, 5)
		if err != nil {
			log.Printf("RAG 检索失败: %v", err)
			runtime.Breakpoint()
		} else if len(docs) > 0 {
			ragCtx := "以下是我博客中的相关内容，你可以参考这些信息来回答用户的问题：\n\n" + formatDocs(docs)
			request.Messages = append(request.Messages, struct {
				Content string                      `json:"content"`
				Role    api.ChatRequestMessagesRole `json:"role"`
			}{
				Role: api.System, Content: ragCtx,
			})
		}
	}

	messages := api.ChatRequestToLangchainMessages(request)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		panic("不支持 SSE")
	}
	resp, err := s.llm.GenerateContent(r.Context(), messages, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		fmt.Fprintf(w, "data: %s\n\n", chunk)
		flusher.Flush()
		return nil
	}))

	if err != nil {
		fmt.Fprintf(w, "data: [ERROR]%s\n\n", err.Error())
		flusher.Flush()
		return
	}

	_ = resp
}
