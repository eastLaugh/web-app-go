package adapters

import (
	"context"
	"fmt"
	"math"

	"github.com/openai/openai-go/v3"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// VecAdapter 向量存储适配器
type VecAdapter interface {
	Search(ctx context.Context, query string, topK int) ([]Document, error)
	AddDocuments(ctx context.Context, docs []Document) error
	ClearAll(ctx context.Context) error
	GetIndexedFiles(ctx context.Context) ([]string, error)
}

// MangoVec MongoDB 向量存储，使用 OpenAI 兼容 API 做 embedding
type MangoVec struct {
	coll     *mongo.Collection
	client   *openai.Client
	embModel string
}

// NewMangoVec 创建向量存储，embModel 为 embedding 模型名（如 text-embedding-3-small）
func NewMangoVec(coll *mongo.Collection, client *openai.Client, embModel string) *MangoVec {
	return &MangoVec{coll: coll, client: client, embModel: embModel}
}

func (m *MangoVec) embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	input := openai.EmbeddingNewParamsInputUnion{}
	input.OfArrayOfStrings = texts
	resp, err := m.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: input,
		Model: openai.EmbeddingModel(m.embModel),
	})
	if err != nil {
		return nil, err
	}
	out := make([][]float32, len(resp.Data))
	for i := range resp.Data {
		vec := resp.Data[i].Embedding
		out[i] = make([]float32, len(vec))
		for j, v := range vec {
			out[i][j] = float32(v)
		}
	}
	return out, nil
}

func (m *MangoVec) Search(ctx context.Context, query string, topK int) ([]Document, error) {
	vecs, err := m.embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("向量化查询失败: %w", err)
	}
	queryVector := vecs[0]

	type docWithVector struct {
		PageContent string         `bson:"pageContent"`
		Embedding   []float32      `bson:"embedding"`
		Metadata    map[string]any `bson:"metadata"`
	}

	cursor, err := m.coll.Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{
		"pageContent": 1, "embedding": 1, "metadata": 1,
	}))
	if err != nil {
		return nil, fmt.Errorf("查询文档失败: %w", err)
	}
	defer cursor.Close(ctx)

	var allDocs []docWithVector
	if err := cursor.All(ctx, &allDocs); err != nil {
		return nil, fmt.Errorf("读取文档失败: %w", err)
	}

	type scoredDoc struct {
		doc   Document
		score float64
	}
	scoredDocs := make([]scoredDoc, 0, len(allDocs))
	for _, d := range allDocs {
		if len(d.Embedding) == 0 {
			continue
		}
		score := cosineSimilarity(queryVector, d.Embedding)
		scoredDocs = append(scoredDocs, scoredDoc{
			doc:   Document{PageContent: d.PageContent, Metadata: d.Metadata},
			score: score,
		})
	}
	for i := 0; i < len(scoredDocs)-1; i++ {
		for j := i + 1; j < len(scoredDocs); j++ {
			if scoredDocs[i].score < scoredDocs[j].score {
				scoredDocs[i], scoredDocs[j] = scoredDocs[j], scoredDocs[i]
			}
		}
	}
	result := make([]Document, 0, topK)
	for i := 0; i < topK && i < len(scoredDocs); i++ {
		result = append(result, scoredDocs[i].doc)
	}
	return result, nil
}

func (m *MangoVec) AddDocuments(ctx context.Context, docs []Document) error {
	if len(docs) == 0 {
		return nil
	}
	const batchSize = 10
	for i := 0; i < len(docs); i += batchSize {
		end := min(i+batchSize, len(docs))
		batch := docs[i:end]
		texts := make([]string, len(batch))
		for j, doc := range batch {
			texts[j] = doc.PageContent
		}
		vecs, err := m.embed(ctx, texts)
		if err != nil {
			return fmt.Errorf("向量化文档失败 (batch %d-%d): %w", i, end, err)
		}
		docsInterface := make([]any, len(batch))
		for j, doc := range batch {
			docsInterface[j] = bson.M{
				"pageContent": doc.PageContent,
				"embedding":   vecs[j],
				"metadata":    doc.Metadata,
			}
		}
		if _, err := m.coll.InsertMany(ctx, docsInterface); err != nil {
			return fmt.Errorf("插入文档失败 (batch %d-%d): %w", i, end, err)
		}
	}
	return nil
}

func (m *MangoVec) ClearAll(ctx context.Context) error {
	_, err := m.coll.DeleteMany(ctx, bson.M{})
	return err
}

func (m *MangoVec) GetIndexedFiles(ctx context.Context) ([]string, error) {
	var files []string
	err := m.coll.Distinct(ctx, "metadata.file", bson.M{}).Decode(&files)
	return files, err
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
