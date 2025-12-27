package ports

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/eastLaugh/web-app-go/go/internal/api"
	"github.com/eastLaugh/web-app-go/go/internal/repo"
	util "github.com/eastLaugh/web-app-go/go/pkg"
	"github.com/eastLaugh/web-app-go/go/pkg/cnllm"
	"github.com/eastLaugh/web-app-go/go/pkg/cnllm/adapters"
	"github.com/eastLaugh/web-app-go/go/pkg/tokens"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var _ api.ServerInterface = &Server{}

type Server struct {
	postRepo   repo.PostRepo
	mg         *mongo.Client
	llm        *openai.LLM
	vecAdapter adapters.VecAdapter
}

func NewServer(mg *mongo.Client) *Server {
	llm := util.Must(cnllm.Qwen())

	return &Server{
		postRepo:   repo.NewMangoPostRepo(mg.Database("webapp").Collection("posts")),
		mg:         mg,
		llm:        llm,
		vecAdapter: adapters.NewMangoVec(mg.Database("webapp").Collection("vector_docs"), llm),
	}
}

func (s *Server) PostAuth(w http.ResponseWriter, r *http.Request) {
	var request api.PostAuthJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	payload := tokens.Payload{
		Email:  string(request.Email),
		Expire: time.Now().Add(24 * time.Hour).Unix(),
	}
	token, err := payload.Export()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (s *Server) GetPosts(w http.ResponseWriter, r *http.Request, params api.GetPostsParams) {
	posts, err := s.postRepo.GetPostsByFile(r.Context(), params.File)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, posts)
}

func (s *Server) PostPosts(w http.ResponseWriter, r *http.Request) {
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
		} else if len(docs) > 0 {
			var ragCtx strings.Builder
			ragCtx.WriteString("以下是我博客中的相关内容，你可以参考这些信息来回答用户的问题：\n\n")
			for i, doc := range docs {
				title := ""
				if t, ok := doc.Metadata["title"].(string); ok {
					title = t
				}
				file := ""
				if f, ok := doc.Metadata["file"].(string); ok {
					file = f
				}
				ragCtx.WriteString(fmt.Sprintf("[片段 %d]", i+1))
				if title != "" {
					ragCtx.WriteString(fmt.Sprintf(" 来源：《%s》", title))
				}
				if file != "" {
					ragCtx.WriteString(fmt.Sprintf(" 文件：%s", file))
				}
				ragCtx.WriteString("\n")
				ragCtx.WriteString(doc.PageContent)
				ragCtx.WriteString("\n\n")
			}
			request.Messages = append([]struct {
				Content string                      `json:"content"`
				Role    api.ChatRequestMessagesRole `json:"role"`
			}{
				{Role: api.System, Content: ragCtx.String()},
			}, request.Messages...)
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

func (s *Server) BuildRAGIndex(ctx context.Context, fsys fs.FS) error {
	log.Printf("开始构建 RAG 索引...")

	if err := s.vecAdapter.ClearAll(ctx); err != nil {
		log.Printf("警告: 清空现有索引失败: %v", err)
	}

	var docs []schema.Document
	err := fs.WalkDir(fsys, "blogs", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}

		chunks := processMarkdownCoarse(string(content), path)
		for _, chunk := range chunks {
			docs = append(docs, schema.Document{
				PageContent: chunk.Content,
				Metadata: map[string]any{
					"file":  chunk.File,
					"title": chunk.Title,
				},
			})
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("加载博客文件失败: %w", err)
	}

	log.Printf("找到 %d 个文档块，开始添加到向量存储...", len(docs))
	if err := s.vecAdapter.AddDocuments(ctx, docs); err != nil {
		return fmt.Errorf("添加文档失败: %w", err)
	}

	log.Printf("RAG 索引构建完成，共处理 %d 个文档块", len(docs))
	return nil
}

type markdownChunk struct {
	File    string
	Content string
	Title   string
}

func processMarkdownCoarse(content string, file string) []markdownChunk {
	lines := strings.Split(content, "\n")
	title := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			title = strings.TrimPrefix(line, "# ")
			break
		}
	}

	paragraphs := strings.Split(content, "\n\n")
	chunks := make([]markdownChunk, 0)

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" || strings.HasPrefix(para, "#") {
			continue
		}

		if len(para) > 2000 {
			lines := strings.Split(para, "\n")
			currentChunk := ""
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if len(currentChunk)+len(line) > 2000 && currentChunk != "" {
					chunks = append(chunks, markdownChunk{File: file, Content: currentChunk, Title: title})
					currentChunk = line
				} else {
					if currentChunk != "" {
						currentChunk += "\n"
					}
					currentChunk += line
				}
			}
			if currentChunk != "" {
				chunks = append(chunks, markdownChunk{File: file, Content: currentChunk, Title: title})
			}
		} else {
			chunks = append(chunks, markdownChunk{File: file, Content: para, Title: title})
		}
	}

	return chunks
}

type ToolVectorStore Server

func (t *ToolVectorStore) Call(ctx context.Context, input string) (string, error) {
	docs, err := t.vecAdapter.Search(ctx, input, 5)
	if err != nil {
		return "", fmt.Errorf("向量搜索失败: %w", err)
	}
	if len(docs) == 0 {
		return "未找到相关文档", nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("找到 %d 个相关文档：\n\n", len(docs)))
	for i, doc := range docs {
		title := ""
		if t, ok := doc.Metadata["title"].(string); ok {
			title = t
		}
		file := ""
		if f, ok := doc.Metadata["file"].(string); ok {
			file = f
		}
		result.WriteString(fmt.Sprintf("[%d] %s\n", i+1, title))
		if file != "" {
			result.WriteString(fmt.Sprintf("文件: %s\n", file))
		}
		result.WriteString(fmt.Sprintf("内容: %s\n\n", doc.PageContent))
	}
	return result.String(), nil
}

func (t *ToolVectorStore) Description() string {
	return "在博客文档中搜索相关内容，返回最相似的文档片段"
}

func (t *ToolVectorStore) Name() string {
	return "vector_search"
}

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}
