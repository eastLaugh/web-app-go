package adapters

import (
	"context"
	"fmt"
	"math"

	"github.com/tmc/langchaingo/embeddings"  // Langchaingo 层：文本转向量
	"github.com/tmc/langchaingo/llms/openai" // Langchaingo 层：LLM 客户端
	"github.com/tmc/langchaingo/schema"      // Langchaingo 层：文档结构
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// VecAdapter 向量存储适配器接口（你的业务层接口）
type VecAdapter interface {
	Search(ctx context.Context, query string, topK int) ([]schema.Document, error)
	AddDocuments(ctx context.Context, docs []schema.Document) error
	ClearAll(ctx context.Context) error
	GetIndexedFiles(ctx context.Context) ([]string, error)
}

// MangoVec MongoDB 向量存储适配器实现（不使用 Atlas，自己实现向量搜索）
type MangoVec struct {
	coll     *mongo.Collection        // MongoDB Client 层：用于数据库操作
	embedder *embeddings.EmbedderImpl // Langchaingo 层：用于文本转向量
}

// NewMangoVec 创建 MongoDB 向量存储适配器
func NewMangoVec(coll *mongo.Collection, llm *openai.LLM) *MangoVec {
	// Langchaingo 层：创建 Embedder（将 LLM 包装成 Embedder 接口）
	embedder, _ := embeddings.NewEmbedder(llm)

	return &MangoVec{
		coll:     coll,
		embedder: embedder,
	}
}

// Search 搜索相似文档（自己实现向量搜索，不依赖 Atlas）
func (m *MangoVec) Search(ctx context.Context, query string, topK int) ([]schema.Document, error) {
	// 1. 将查询文本转为向量（使用 langchaingo embedder）
	queryVector, err := m.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("向量化查询失败: %w", err)
	}

	// 2. 从 MongoDB 获取所有文档（包含向量）
	cursor, err := m.coll.Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{
		"pageContent": 1,
		"embedding":   1,
		"metadata":    1,
	}))
	if err != nil {
		return nil, fmt.Errorf("查询文档失败: %w", err)
	}
	defer cursor.Close(ctx)

	type docWithVector struct {
		PageContent string                 `bson:"pageContent"`
		Embedding   []float32              `bson:"embedding"`
		Metadata    map[string]interface{} `bson:"metadata"`
	}

	var allDocs []docWithVector
	if err := cursor.All(ctx, &allDocs); err != nil {
		return nil, fmt.Errorf("读取文档失败: %w", err)
	}

	// 3. 计算相似度并排序
	type scoredDoc struct {
		doc   schema.Document
		score float64
	}
	scoredDocs := make([]scoredDoc, 0, len(allDocs))
	for _, d := range allDocs {
		if len(d.Embedding) == 0 {
			continue
		}
		score := cosineSimilarity(queryVector, d.Embedding)
		scoredDocs = append(scoredDocs, scoredDoc{
			doc: schema.Document{
				PageContent: d.PageContent,
				Metadata:    d.Metadata,
			},
			score: score,
		})
	}

	// 简单排序（冒泡排序，数据量不大时够用）
	for i := 0; i < len(scoredDocs)-1; i++ {
		for j := i + 1; j < len(scoredDocs); j++ {
			if scoredDocs[i].score < scoredDocs[j].score {
				scoredDocs[i], scoredDocs[j] = scoredDocs[j], scoredDocs[i]
			}
		}
	}

	// 4. 返回 Top-K
	result := make([]schema.Document, 0, topK)
	for i := 0; i < topK && i < len(scoredDocs); i++ {
		result = append(result, scoredDocs[i].doc)
	}

	return result, nil
}

// AddDocuments 添加文档（使用 langchaingo embedder 生成向量）
// 分批处理，每批最多 10 个文档（API 限制）
func (m *MangoVec) AddDocuments(ctx context.Context, docs []schema.Document) error {
	if len(docs) == 0 {
		return nil
	}

	const batchSize = 10 // API 限制：每批最多 10 个

	// 分批处理
	for i := 0; i < len(docs); i += batchSize {
		end := i + batchSize
		if end > len(docs) {
			end = len(docs)
		}

		batch := docs[i:end]
		texts := make([]string, len(batch))
		for j, doc := range batch {
			texts[j] = doc.PageContent
		}

		// 批量向量化（使用 langchaingo embedder）
		embeddings, err := m.embedder.EmbedDocuments(ctx, texts)
		if err != nil {
			return fmt.Errorf("向量化文档失败 (batch %d-%d): %w", i, end, err)
		}

		// 构建 MongoDB 文档格式并批量插入
		docsInterface := make([]interface{}, len(batch))
		for j, doc := range batch {
			docsInterface[j] = bson.M{
				"pageContent": doc.PageContent,
				"embedding":   embeddings[j],
				"metadata":    doc.Metadata,
			}
		}

		_, err = m.coll.InsertMany(ctx, docsInterface)
		if err != nil {
			return fmt.Errorf("插入文档失败 (batch %d-%d): %w", i, end, err)
		}
	}

	return nil
}

// ClearAll 清空所有向量文档
func (m *MangoVec) ClearAll(ctx context.Context) error {
	_, err := m.coll.DeleteMany(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("清空文档失败: %w", err)
	}
	return nil
}

// GetIndexedFiles 获取已索引的文件列表
func (m *MangoVec) GetIndexedFiles(ctx context.Context) ([]string, error) {
	var files []string
	if err := m.coll.Distinct(ctx, "metadata.file", bson.M{}).Decode(&files); err != nil {
		return nil, err
	}
	return files, nil
}

// cosineSimilarity 计算余弦相似度
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
