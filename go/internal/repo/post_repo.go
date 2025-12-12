package repo

import (
	"context"

	"github.com/eastLaugh/web-app-go/go/api"
)

// PostRepo 定义评论数据访问接口
type PostRepo interface {
	// GetPostsByFile 根据文件名获取评论列表
	GetPostsByFile(ctx context.Context, file string) ([]api.Post, error)
	// InsertPost 插入新评论
	InsertPost(ctx context.Context, email string, content string, file string) error
}


