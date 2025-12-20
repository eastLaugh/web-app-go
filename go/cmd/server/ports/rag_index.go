package ports

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/eastLaugh/web-app-go/go/internal/repo"
	"github.com/eastLaugh/web-app-go/go/pkg/document"
	"github.com/sirupsen/logrus"
)

// BuildRAGIndex 构建 RAG 索引
func (s *Server) BuildRAGIndex(ctx context.Context, fsys fs.FS) error {
	if s.embedClient == nil || s.vectorRepo == nil {
		return fmt.Errorf("embedding 客户端或向量仓库未初始化")
	}

	logrus.Info("开始构建 RAG 索引...")

	// 清空现有索引
	if err := s.vectorRepo.ClearAll(ctx); err != nil {
		logrus.Warnf("清空现有索引失败: %v", err)
	}

	// 加载所有博客文件
	blogs, err := document.LoadBlogsFromFS(fsys, "blogs")
	if err != nil {
		return fmt.Errorf("加载博客文件失败: %w", err)
	}

	logrus.Infof("找到 %d 个博客文件", len(blogs))

	// 处理每个博客文件
	totalChunks := 0
	for file, content := range blogs {
		chunks := document.ProcessMarkdown(content, file)
		logrus.Infof("文件 %s 切分为 %d 个 chunk", file, len(chunks))

		// 批量向量化（每次最多 10 个）
		batchSize := 10
		for i := 0; i < len(chunks); i += batchSize {
			end := i + batchSize
			if end > len(chunks) {
				end = len(chunks)
			}

			batch := chunks[i:end]
			texts := make([]string, len(batch))
			for j, chunk := range batch {
				texts[j] = chunk.Content
			}

			// 向量化（构建索引时使用 document）
			embeddings, err := s.embedClient.Embed(ctx, texts, "document")
			if err != nil {
				logrus.Errorf("向量化失败 (文件: %s, batch: %d-%d): %v", file, i, end, err)
				continue
			}

			// 存储到 MongoDB
			vectorDocs := make([]*repo.VectorDoc, len(batch))
			for j, chunk := range batch {
				doc := &repo.VectorDoc{
					File:       chunk.File,
					ChunkIndex: chunk.ChunkIndex,
					Content:    chunk.Content,
					Embedding:  embeddings[j],
				}
				doc.Metadata.Title = chunk.Title
				vectorDocs[j] = doc
			}

			if err := s.vectorRepo.InsertVectors(ctx, vectorDocs); err != nil {
				logrus.Errorf("存储向量失败 (文件: %s, batch: %d-%d): %v", file, i, end, err)
				continue
			}

			totalChunks += len(batch)
			logrus.Debugf("已处理 %d 个 chunk", totalChunks)
		}
	}

	logrus.Infof("RAG 索引构建完成，共处理 %d 个 chunk", totalChunks)
	return nil
}
