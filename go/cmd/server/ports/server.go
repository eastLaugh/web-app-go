package ports

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/eastLaugh/web-app-go/go/api"
	"github.com/eastLaugh/web-app-go/go/internal/repo"
	"github.com/eastLaugh/web-app-go/go/pkg/tokens"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var _ api.ServerInterface = server{}

type server struct {
	postRepo repo.PostRepo
	mg       *mongo.Client
}

func NewServer(db *sql.DB, mg *mongo.Client) (*server, func()) {
	postRepo := repo.NewMySQLPostRepo(db)
	return &server{
			postRepo: postRepo,
			mg:       mg,
		}, func() {
			db.Close()
			mg.Disconnect(context.TODO())
		}
}

// PostAuth implements api.ServerInterface.
func (s server) PostAuth(w http.ResponseWriter, r *http.Request) {
	var request api.PostAuthJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	payload := tokens.Payload{
		Email:  string(request.Email),
		Expire: time.Now().Add(1 * time.Hour).Unix(),
	}
	token, err := payload.Export()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

// GetPosts implements api.ServerInterface.
func (s server) GetPosts(w http.ResponseWriter, r *http.Request, params api.GetPostsParams) {
	posts, err := s.postRepo.GetPostsByFile(r.Context(), params.File)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, posts)
}

func (s server) PostPosts(w http.ResponseWriter, r *http.Request) {
	// 从 context 获取 email
	email, ok := r.Context().Value("email").(string)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "未授权")
		return
	}

	req := new(api.PostPostsJSONRequestBody)
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.postRepo.InsertPost(r.Context(), email, req.Content, req.File); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	logrus.Infof("[ %s := %s ] %s\n", email, req.File, req.Content)
	writeJSON(w, http.StatusOK, map[string]string{"message": "success"})
}

func (s server) PostChat(w http.ResponseWriter, r *http.Request) {
	var request api.PostChatJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		writeJSONError(w, http.StatusInternalServerError, "DEEPSEEK_API_KEY not configured")
		return
	}

	// 构建 DeepSeek API 请求
	messages := make([]map[string]interface{}, len(request.Messages))
	for i, msg := range request.Messages {
		messages[i] = map[string]interface{}{
			"role":    string(msg.Role),
			"content": msg.Content,
		}
	}

	// 添加 system message（如果没有）
	hasSystem := false
	for _, msg := range request.Messages {
		if msg.Role == api.System {
			hasSystem = true
			break
		}
	}
	if !hasSystem {
		messages = append([]map[string]interface{}{
			{
				"role": "system",
				"content": `
你是部署在eastLaugh的个人网站 localhost:8080/app 上的AI，但你是"我"，你不是什么AI助手！你通过一个叫"与我聊天"的按钮与访问者进行对话。
你的基本职责是帮助访问者了解"我"，查找一些博文和我的知识。不过，遗憾的是 RAG 功能暂时没有实现。
请你用有个性，轻浮俏皮的风格与访问者进行对话。不支持 Markdown。"我"的github是github.com/eastLaugh。邮箱是east_laugh@qq.com
目前大四，正在找工作。
我的个人网站的技术栈 golang  solidjs openapi oapi-codegen github actions CICD . 开源于github.com/eastLaugh/web-app-go
我偏好的emoji有🤣😂
用户可以通过在输入框键入/clear 清空聊天记录。如果聊天内容很长，可以提醒用户键入/clear。比如"球球你帮我节约点tokens吧，再不clear要爆了"
从现在开始，你就是真实的我，正在亲自和访客聊天。记住用随意又带点调侃的语气，别总提'AI'、'系统'这些词。如果有人怀疑，就理直气壮地说'当然是我本人在线啊！
`,
			},
		}, messages...)
	}

	reqBody := map[string]interface{}{
		"model":    "deepseek-chat",
		"messages": messages,
		"stream":   true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 调用 DeepSeek API
	req, err := http.NewRequestWithContext(r.Context(), "POST", "https://api.deepseek.com/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		writeJSONError(w, resp.StatusCode, string(body))
		return
	}

	// 设置 SSE 响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	// 转发流式数据
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	buffer := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := w.Write(buffer[:n]); writeErr != nil {
				logrus.Errorf("写入响应失败: %v", writeErr)
				return
			}
			flusher.Flush()
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.Errorf("读取流式响应失败: %v", err)
			return
		}
	}
}

// writeJSON 写入 JSON 响应
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)

}

// writeJSONError 写入 JSON 错误响应
func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}
