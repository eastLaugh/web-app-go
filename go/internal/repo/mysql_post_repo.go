package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/eastLaugh/web-app-go/go/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

// MySQLPostRepo MySQL 实现的评论仓库
type MySQLPostRepo struct {
	db *sql.DB
}

// NewMySQLPostRepo 创建 MySQL 评论仓库
func NewMySQLPostRepo(db *sql.DB) *MySQLPostRepo {
	return &MySQLPostRepo{db: db}
}

// GetPostsByFile 根据文件名获取评论列表
func (r *MySQLPostRepo) GetPostsByFile(ctx context.Context, file string) ([]api.Post, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, file, content, email, created_at FROM posts WHERE file = ? ORDER BY created_at ASC",
		file)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []api.Post
	for rows.Next() {
		var p api.Post
		var id int
		var file string
		var content string
		var createdAt time.Time
		var email string
		if err := rows.Scan(&id, &file, &content, &email, &createdAt); err != nil {
			logrus.Errorf("扫描评论数据失败: %v", err)
			continue
		}
		p.Id = &id
		p.File = &file
		p.Content = &content
		emailVal := openapi_types.Email(email)
		p.Email = &emailVal
		p.CreatedAt = &createdAt
		posts = append(posts, p)
	}

	return posts, rows.Err()
}

// InsertPost 插入新评论
func (r *MySQLPostRepo) InsertPost(ctx context.Context, email string, content string, file string) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO posts (email, content, file) VALUES (?, ?, ?)", email, content, file)
	if err != nil {
		logrus.Errorf("插入评论失败: %v", err)
		return err
	}
	return nil
}
