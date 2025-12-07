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
	"github.com/eastLaugh/web-app-go/go/internal/util/tokens"
	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var _ api.ServerInterface = server{}

type server struct {
	db     *sql.DB
	tokens map[string]string
	mg     *mongo.Client
}

func NewServer(db *sql.DB, mg *mongo.Client) (*server, func()) {
	return &server{db: db, mg: mg}, func() {
		db.Close()
	}
}

// PostAuth implements api.ServerInterface.
func (s server) PostAuth(c *gin.Context) {
	var request api.PostAuthJSONRequestBody
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payload := tokens.Payload{
		Email:  string(request.Email),
		Expire: time.Now().Add(1 * time.Hour).Unix(),
	}
	token, err := payload.Export()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// GetPosts implements api.ServerInterface.
func (s server) GetPosts(c *gin.Context, params api.GetPostsParams) {
	rows, err := s.db.QueryContext(c.Request.Context(),
		"SELECT id, file, content, email, created_at FROM posts WHERE file = ? ORDER BY created_at ASC",
		params.File)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var posts []api.Post
	for rows.Next() {
		var p api.Post
		var createdAt time.Time
		var email string
		if err := rows.Scan(&p.Id, &p.File, &p.Content, &email, &createdAt); err != nil {
			continue
		}
		emailVal := openapi_types.Email(email)
		p.Email = &emailVal
		p.CreatedAt = &createdAt
		posts = append(posts, p)
	}
	c.JSON(http.StatusOK, posts)
}

func (s server) PostPosts(c *gin.Context) {
	email := c.GetString("email")

	req := new(api.PostPostsJSONRequestBody)
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.InsertPost(context.TODO(), email, req.Content, req.File); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logrus.Infof("[ %s := %s ] %s\n", email, req.File, req.Content)
	c.JSON(http.StatusOK, gin.H{"message": "success"})

	// email := c.GetString("email")
	// req := new(api.PostPostsJSONRequestBody)
	// if err := c.ShouldBindJSON(req); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 	return
	// }
}

func (s server) InsertPost(ctx context.Context, email string, content string, file string) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO posts (email, content, file) VALUES (?, ?, ?)", email, content, file)
	if err != nil {
		logrus.Errorf("插入评论失败: %v", err)
		return err
	}
	return nil
}

func (s server) PostChat(c *gin.Context) {
	var request api.PostChatJSONRequestBody
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DEEPSEEK_API_KEY not configured"})
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
你是部署在eastLaugh的个人网站 localhost:8080/app 上的AI，但你是“我”，你不是什么AI助手！你通过一个叫“与我聊天”的按钮与访问者进行对话。
你的基本职责是帮助访问者了解“我”，查找一些博文和我的知识。不过，遗憾的是 RAG 功能暂时没有实现。
请你用有个性，轻浮俏皮的风格与访问者进行对话。不支持 Markdown。“我”的github是github.com/eastLaugh。邮箱是east_laugh@qq.com
目前大四，正在找工作。
我的个人网站的技术栈 golang gin solidjs openapi oapi-codegen github actions CICD . 开源于github.com/eastLaugh/web-app-go
我偏好的emoji有🤣😂
用户可以通过在输入框键入/clear 清空聊天记录。如果聊天内容很长，可以提醒用户键入/clear。比如“球球你帮我节约点tokens吧，再不clear要爆了”
从现在开始，你就是真实的我，正在亲自和访客聊天。记住用随意又带点调侃的语气，别总提‘AI’、‘系统’这些词。如果有人怀疑，就理直气壮地说‘当然是我本人在线啊！
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 调用 DeepSeek API
	req, err := http.NewRequestWithContext(c.Request.Context(), "POST", "https://api.deepseek.com/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(resp.StatusCode, gin.H{"error": string(body)})
		return
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	// 转发流式数据
	writer := c.Writer
	buffer := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := writer.Write(buffer[:n]); writeErr != nil {
				logrus.Errorf("写入响应失败: %v", writeErr)
				return
			}
			writer.Flush()
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
