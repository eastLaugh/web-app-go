package ports

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/eastLaugh/web-app-go/go/api"
	"github.com/eastLaugh/web-app-go/go/internal/util/tokens"
	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

var _ api.ServerInterface = server{}

type server struct {
	db     *sql.DB
	tokens map[string]string
}

func NewServer(db *sql.DB) *server {
	return &server{db: db}
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

// PostPosts implements api.ServerInterface.
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

}

func (s server) InsertPost(ctx context.Context, email string, content string, file string) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO posts (email, content, file) VALUES (?, ?, ?)", email, content, file)
	if err != nil {
		logrus.Errorf("插入评论失败: %v", err)
		return err
	}
	return nil
}

func (s *server) Close() {
}
