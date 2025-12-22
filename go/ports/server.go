package ports

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/eastLaugh/web-app-go/go/internal/api"
	"github.com/eastLaugh/web-app-go/go/internal/repo"
	"github.com/eastLaugh/web-app-go/go/pkg/embedding"
	"github.com/eastLaugh/web-app-go/go/pkg/tokens"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var _ api.ServerInterface = Server{}

type Server struct {
	postRepo    repo.PostRepo
	mg          *mongo.Client
	embedClient *embedding.Client
	vectorRepo  *repo.VectorRepo
}

func NewServer(mg *mongo.Client) (*Server, func()) {
	postRepo := repo.NewMangoPostRepo(mg.Database("webapp").Collection("posts"))
	embedClient := embedding.NewClient()
	vectorRepo := repo.NewVectorRepo(mg, "webapp")
	return &Server{
			postRepo:    postRepo,
			mg:          mg,
			embedClient: embedClient,
			vectorRepo:  vectorRepo,
		}, func() {
			mg.Disconnect(context.TODO())
		}
}

// PostAuth implements api.ServerInterface.
func (s Server) PostAuth(w http.ResponseWriter, r *http.Request) {
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
func (s Server) GetPosts(w http.ResponseWriter, r *http.Request, params api.GetPostsParams) {
	posts, err := s.postRepo.GetPostsByFile(r.Context(), params.File)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, posts)
}

func (s Server) PostPosts(w http.ResponseWriter, r *http.Request) {
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

	slog.Info("新评论", "email", email, "file", req.File, "content", req.Content)
	writeJSON(w, http.StatusOK, map[string]string{"message": "success"})
}

func (s Server) PostChat(w http.ResponseWriter, r *http.Request) {
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

	// 添加基础 system message（如果没有）
	hasSystem := false
	for _, msg := range request.Messages {
		if msg.Role == api.System {
			hasSystem = true
			break
		}
	}
	if !hasSystem {
		systemContent := `你是部署在eastLaugh的个人网站 localhost:8080/app 上的AI，但你是"我"，你不是什么AI助手！你通过一个叫"与我聊天"的按钮与访问者进行对话。
你的基本职责是帮助访问者了解"我"，查找一些博文和我的知识。
请你用有个性，轻浮俏皮的风格与访问者进行对话。不支持 Markdown。"我"的github是github.com/eastLaugh。邮箱是east_laugh@qq.com
目前大四，正在找工作。
我的个人网站的技术栈 golang  solidjs openapi oapi-codegen github actions CICD . 开源于github.com/eastLaugh/web-app-go
我偏好的emoji有🤣😂
用户可以通过在输入框键入/clear 清空聊天记录。如果聊天内容很长，可以提醒用户键入/clear，比如"球球你帮我节约点tokens吧，再不clear要爆了"
从现在开始，你就是真实的我，正在亲自和访客聊天。记住用随意又带点调侃的语气，别总提'AI'、'系统'这些词。如果有人怀疑，就理直气壮地说'当然是我本人在线啊！'
`
		messages = append([]map[string]interface{}{
			{
				"role":    "system",
				"content": systemContent,
			},
		}, messages...)
	}

	// RAG: 检索相关文档（追加到 messages 中，而不是修改 system message）
	lastUserMessage := ""
	for i := len(request.Messages) - 1; i >= 0; i-- {
		if request.Messages[i].Role == api.User {
			lastUserMessage = request.Messages[i].Content
			break
		}
	}

	if lastUserMessage != "" && s.embedClient != nil && s.vectorRepo != nil {
		// 将用户问题向量化（查询时使用 query）
		queryVector, err := s.embedClient.EmbedSingle(r.Context(), lastUserMessage, "query")
		if err != nil {
			log.Printf("警告: 向量化用户问题失败，跳过 RAG: %v", err)
		} else {
			// 检索相似文档
			scoredDocs, err := s.vectorRepo.SearchSimilar(r.Context(), queryVector, 10)
			if err != nil {
				log.Printf("警告: 检索相似文档失败，跳过 RAG: %v", err)
			} else if len(scoredDocs) > 0 {
				// 输出详细的 RAG 检索日志
				slog.Info("RAG 检索结果", "question", lastUserMessage, "count", len(scoredDocs))
				for i, scoredDoc := range scoredDocs {
					doc := scoredDoc.Doc
					slog.Info("RAG 文档",
						"index", i+1,
						"file", doc.File,
						"title", doc.Metadata.Title,
						"chunkIndex", doc.ChunkIndex,
						"score", scoredDoc.Score,
						"contentLength", len(doc.Content))
				}

				// 构建 RAG 上下文，包含文件路径、相似度等元数据
				var contextBuilder strings.Builder
				contextBuilder.WriteString("以下是我博客中的相关内容，你可以参考这些信息来回答用户的问题：\n\n")
				for i, scoredDoc := range scoredDocs {
					doc := scoredDoc.Doc
					// 构建文件访问路径（去掉 .md 后缀，用于前端路由）
					filePath := strings.TrimSuffix(doc.File, ".md")
					if !strings.HasPrefix(filePath, "/") {
						filePath = "/app/" + filePath
					}

					contextBuilder.WriteString(fmt.Sprintf(
						"[片段 %d] 来源：《%s》\n"+
							"文件路径：%s\n"+
							"相似度：%.2f%%\n"+
							"内容：\n%s\n\n",
						i+1,
						doc.Metadata.Title,
						filePath,
						scoredDoc.Score*100, // 转换为百分比
						doc.Content,
					))
				}

				// 将 RAG 上下文作为 system message 追加到 messages 中（在用户消息之前）
				ragSystemMsg := map[string]interface{}{
					"role":    "system",
					"content": contextBuilder.String(),
				}
				// 找到最后一个 user message 的位置，在其之前插入
				insertIndex := len(messages)
				for i := len(messages) - 1; i >= 0; i-- {
					if messages[i]["role"] == "user" {
						insertIndex = i
						break
					}
				}
				// 在指定位置插入 RAG system message
				messages = append(messages[:insertIndex], append([]map[string]interface{}{ragSystemMsg}, messages[insertIndex:]...)...)
			}
		}
	}

	reqBody := map[string]interface{}{
		"model":    "deepseek-chat",
		"messages": messages,
		"stream":   true,
	}

	jsonData, err := json.MarshalIndent(reqBody, "", "  ")

	log.Printf("To Deepseek: %s", string(jsonData))
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
				log.Printf("写入响应失败: %v", writeErr)
				return
			}
			flusher.Flush()
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("读取流式响应失败: %v", err)
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
