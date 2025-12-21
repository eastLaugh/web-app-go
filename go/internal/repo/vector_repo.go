package repo

import (
	"context"
	"log"
	"math"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// VectorDoc 向量文档
type VectorDoc struct {
	ID         interface{} `bson:"_id,omitempty"`
	File       string      `bson:"file"`
	ChunkIndex int         `bson:"chunk_index"`
	Content    string      `bson:"content"`
	Embedding  []float32   `bson:"embedding"`
	Metadata   struct {
		Title     string `bson:"title,omitempty"`
		CreatedAt string `bson:"created_at,omitempty"`
	} `bson:"metadata,omitempty"`
}

// VectorRepo 向量存储仓库
type VectorRepo struct {
	collection *mongo.Collection
}

// NewVectorRepo 创建向量存储仓库
func NewVectorRepo(mg *mongo.Client, dbName string) *VectorRepo {
	collection := mg.Database(dbName).Collection("vector_docs")
	return &VectorRepo{collection: collection}
}

// InsertVector 插入向量文档
func (r *VectorRepo) InsertVector(ctx context.Context, doc *VectorDoc) error {
	_, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		log.Printf("插入向量文档失败: %v", err)
		return err
	}
	return nil
}

// InsertVectors 批量插入向量文档
func (r *VectorRepo) InsertVectors(ctx context.Context, docs []*VectorDoc) error {
	if len(docs) == 0 {
		return nil
	}

	// 转换为 interface{} 切片
	docsInterface := make([]interface{}, len(docs))
	for i, doc := range docs {
		docsInterface[i] = doc
	}

	_, err := r.collection.InsertMany(ctx, docsInterface)
	if err != nil {
		log.Printf("批量插入向量文档失败: %v", err)
		return err
	}
	return nil
}

// ScoredDoc 带相似度分数的文档
type ScoredDoc struct {
	Doc   *VectorDoc
	Score float64
}

// SearchSimilar 搜索相似向量（Top-K），返回带相似度分数的结果
func (r *VectorRepo) SearchSimilar(ctx context.Context, queryVector []float32, topK int) ([]*ScoredDoc, error) {
	if topK <= 0 {
		topK = 5 // 默认返回 5 个
	}

	// 获取所有文档（如果数据量很大，这里需要优化）
	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{
		"file":        1,
		"chunk_index": 1,
		"content":     1,
		"embedding":   1,
		"metadata":    1,
	}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var allDocs []*VectorDoc
	if err := cursor.All(ctx, &allDocs); err != nil {
		return nil, err
	}

	// 计算相似度并排序
	scoredDocs := make([]*ScoredDoc, 0, len(allDocs))
	for _, doc := range allDocs {
		score := cosineSimilarity(queryVector, doc.Embedding)
		scoredDocs = append(scoredDocs, &ScoredDoc{
			Doc:   doc,
			Score: score,
		})
	}

	// 按分数降序排序
	for i := 0; i < len(scoredDocs)-1; i++ {
		for j := i + 1; j < len(scoredDocs); j++ {
			if scoredDocs[i].Score < scoredDocs[j].Score {
				scoredDocs[i], scoredDocs[j] = scoredDocs[j], scoredDocs[i]
			}
		}
	}

	// 返回 Top-K
	result := make([]*ScoredDoc, 0, topK)
	for i := 0; i < topK && i < len(scoredDocs); i++ {
		result = append(result, scoredDocs[i])
	}

	return result, nil
}

// DeleteByFile 根据文件删除所有相关向量
func (r *VectorRepo) DeleteByFile(ctx context.Context, file string) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"file": file})
	if err != nil {
		log.Printf("删除向量文档失败: %v", err)
		return err
	}
	return nil
}

// ClearAll 清空所有向量文档
func (r *VectorRepo) ClearAll(ctx context.Context) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{})
	if err != nil {
		log.Printf("清空向量文档失败: %v", err)
		return err
	}
	return nil
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
