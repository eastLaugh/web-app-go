//go:build !noagent

package ports

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/eastLaugh/web-app-go/go/internal/api"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
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

	jsonStr, _ := json.Marshal(request)
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s\n\n\n\n\n完整对话栈:\n%s", lastUserMsg, jsonStr)

	messages := api.ChatRequestToLangchainMessages(request)
	_, _ = json.Marshal(messages)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		panic("不支持 SSE")
	}

	exec := agents.NewExecutor(s.agent)
	resp, err := chains.Run(context.TODO(), exec, sb.String(),
		chains.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			jsonStr, _ := json.Marshal(string(chunk))
			fmt.Fprintf(w, "data: %s\n\n", jsonStr)
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
