package ports

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/coder/websocket"
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

	if request.ConversationId == "" {
		http.Error(w, "conversation_id required", http.StatusBadRequest)
		return
	}
	s.convMu.RLock()
	hist := s.convStore[request.ConversationId]
	s.convMu.RUnlock()

	var messages []openai.ChatCompletionMessageParamUnion
	if len(hist) == 0 {
		messages = []openai.ChatCompletionMessageParamUnion{openai.SystemMessage(api.GetSystemPrompt()), openai.UserMessage(request.Content)}
	} else {
		messages = append(hist, openai.UserMessage(request.Content))
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher, ok := w.(http.Flusher)
	if !ok {
		panic("not support SSE")
	}

	ctx := context.WithValue(r.Context(), reflect.TypeFor[*Server](), s)
	for range maxToolRounds {
		params := openai.ChatCompletionNewParams{
			Messages: messages,
			Model:    s.chatModel,
			Tools:    s.Tools.ToParams(),
		}

		stream := s.client.Chat.Completions.NewStreaming(ctx, params)
		var acc openai.ChatCompletionAccumulator
		for stream.Next() {
			chunk := stream.Current()
			if !acc.AddChunk(chunk) {
				break
			}
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				escaped, _ := json.Marshal(chunk.Choices[0].Delta.Content)
				fmt.Fprintf(w, "data: %s\n\n", escaped)
				flusher.Flush()
			}
		}
		if err := stream.Err(); err != nil {
			fmt.Fprintf(w, "data: [ERROR]%s\n\n", err.Error())
			flusher.Flush()
			return
		}
		if len(acc.Choices) == 0 {
			fmt.Fprintf(w, "data: [ERROR]empty choices\n\n")
			flusher.Flush()
			return
		}

		msg := acc.Choices[0].Message
		messages = append(messages, msg.ToParam())

		// no tool call, save and return
		if len(msg.ToolCalls) == 0 {
			s.convMu.Lock()
			s.convStore[request.ConversationId] = messages
			s.convMu.Unlock()
			flusher.Flush()
			return
		}

		var toolNames []string
		for _, tc := range msg.ToolCalls {
			if tc.Type == "function" {
				toolNames = append(toolNames, tc.Function.Name)
			}
		}
		if len(toolNames) > 0 {
			event, _ := json.Marshal(map[string]any{"event": "tool_call", "tools": toolNames})
			fmt.Fprintf(w, "data: %s\n\n", event)
			flusher.Flush()
		}

		for _, tc := range msg.ToolCalls {
			if tc.Type == "function" {
				result, err := s.Tools.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
				if err != nil {
					result = err.Error()
				}
				messages = append(messages, openai.ToolMessage(result, tc.ID))
			} else {
				messages = append(messages, openai.ToolMessage("不支持的 tool call", tc.ID))
			}
		}
	}

	s.convMu.Lock()
	s.convStore[request.ConversationId] = messages
	s.convMu.Unlock()
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

func (s *Server) PostChatWebsocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "websocket accepted")

}
