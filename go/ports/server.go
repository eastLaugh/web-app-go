package ports

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/eastLaugh/web-app-go/go/internal/api"
	"github.com/eastLaugh/web-app-go/go/internal/repo"
	"github.com/eastLaugh/web-app-go/go/pkg/adapters"
	"github.com/eastLaugh/web-app-go/go/pkg/tokens"
	"github.com/eastLaugh/web-app-go/go/pkg/tools"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var _ api.ServerInterface = &Server{}

type Server struct {
	postRepo   repo.PostRepo
	userRepo   *repo.MangoUserRepo
	mg         *mongo.Client
	client     *openai.Client
	chatModel  string
	vecAdapter adapters.VecAdapter
	Tools      *tools.Registry
}

func NewServer(mg *mongo.Client) *Server {
	client := openai.NewClient(
		option.WithBaseURL(os.Getenv("OPENAI_BASE_URL")),
		option.WithMiddleware(func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
			// tracer
			return next(req)
		}),
	)
	chatModel := os.Getenv("OPENAI_MODEL")

	embModel := os.Getenv("OPENAI_EMBEDDING_MODEL")

	vecAdapter := adapters.NewMangoVec(mg.Database("webapp").Collection("vector_docs"), &client, embModel)

	s := &Server{
		postRepo:   repo.NewMangoPostRepo(mg.Database("webapp").Collection("posts")),
		userRepo:   repo.NewMangoUserRepo(mg.Database("webapp").Collection("users")),
		mg:         mg,
		client:     &client,
		chatModel:  chatModel,
		vecAdapter: vecAdapter,
	}
	s.Tools = registerChatTools(s)
	return s
}

var codeStore sync.Map // email -> codeEntry

type codeEntry struct {
	Code   string
	Expire time.Time
}

func (s *Server) PostAuthCode(w http.ResponseWriter, r *http.Request) {
	var req api.PostAuthCodeJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	code := fmt.Sprintf("%06d", n.Int64())

	codeStore.Store(string(req.Email), codeEntry{Code: code, Expire: time.Now().Add(5 * time.Minute)})

	if err := tools.SendMail("验证码", "你的验证码是: "+code+"，5 分钟内有效。", string(req.Email)); err != nil {
		slog.Error("发送验证码失败", "email", req.Email, "err", err)
		http.Error(w, "发送验证码失败", http.StatusInternalServerError)
		return
	}

	slog.Info("验证码已发送", "email", req.Email)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) PostAuth(w http.ResponseWriter, r *http.Request) {
	var request api.PostAuthJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	v, ok := codeStore.LoadAndDelete(string(request.Email))
	if !ok {
		http.Error(w, "请先发送验证码", http.StatusBadRequest)
		return
	}
	entry := v.(codeEntry)
	if time.Now().After(entry.Expire) {
		http.Error(w, "验证码已过期", http.StatusBadRequest)
		return
	}
	if entry.Code != request.Code {
		codeStore.Store(string(request.Email), entry)
		http.Error(w, "验证码错误", http.StatusBadRequest)
		return
	}

	payload := tokens.Payload{
		Email:  string(request.Email),
		Expire: time.Now().Add(24 * time.Hour).Unix(),
	}
	token, err := payload.Export()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (s *Server) GetPosts(w http.ResponseWriter, r *http.Request, params api.GetPostsParams) {
	posts, err := s.postRepo.GetPostsByFile(r.Context(), params.File)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(posts)
}

func (s *Server) PostPosts(w http.ResponseWriter, r *http.Request) {
	email := tokens.RequireAuth(r.Context())
	req := new(api.PostPostsJSONRequestBody)
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.postRepo.InsertPost(r.Context(), email, req.Content, req.File); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("新评论", "email", email, "file", req.File, "content", req.Content)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) BuildRAGIndex(ctx context.Context, fsys fs.FS) error {
	log.Printf("开始增量构建 RAG 索引...")

	indexed, err := s.vecAdapter.GetIndexedFiles(ctx)
	if err != nil {
		log.Printf("警告: 获取已索引文件失败: %v", err)
	}
	log.Printf("已索引文件: %d 个", len(indexed))
	indexedSet := make(map[string]bool, len(indexed))
	for _, f := range indexed {
		indexedSet[f] = true
	}

	var docs []adapters.Document
	var newFiles []string
	err = fs.WalkDir(fsys, "blogs", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		if indexedSet[path] {
			return nil
		}

		newFiles = append(newFiles, path)
		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}

		chunks := processMarkdownCoarse(string(content), path)
		for _, chunk := range chunks {
			docs = append(docs, adapters.Document{
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

	if len(newFiles) == 0 {
		log.Printf("无新文件需要索引")
		return nil
	}

	log.Printf("新增文件: %v", newFiles)
	log.Printf("共 %d 个文档块，开始添加到向量存储...", len(docs))
	if err := s.vecAdapter.AddDocuments(ctx, docs); err != nil {
		return fmt.Errorf("添加文档失败: %w", err)
	}

	log.Printf("RAG 索引构建完成")
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
		if after, ok := strings.CutPrefix(line, "# "); ok {
			title = after
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

func formatDocs(docs []adapters.Document) string {
	var sb strings.Builder
	for i, doc := range docs {
		title, _ := doc.Metadata["title"].(string)
		file, _ := doc.Metadata["file"].(string)
		fmt.Fprintf(&sb, "[%d]", i+1)
		if title != "" {
			fmt.Fprintf(&sb, " %s", title)
		}
		if file != "" {
			fmt.Fprintf(&sb, " (%s)", file)
		}
		fmt.Fprintf(&sb, "\n")
		fmt.Fprintf(&sb, "%s", doc.PageContent)
		fmt.Fprintf(&sb, "\n\n")
	}
	return sb.String()
}
