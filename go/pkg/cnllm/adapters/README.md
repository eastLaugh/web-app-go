# MongoDB 向量存储架构说明

## 架构层次

```
┌─────────────────────────────────────────────────────────┐
│                    Langchaingo 层                        │
│  (提供统一的向量存储接口，隐藏底层实现)                    │
│  - schema.Document (文档结构)                            │
│  - vectorstores.Store (向量存储接口)                      │
│  - mongovector.Store (MongoDB 实现)                      │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│                    Store 层                              │
│  (mongovector.Store - langchaingo 的 MongoDB 适配器)     │
│  - AddDocuments() 添加文档                               │
│  - SimilaritySearch() 相似度搜索                          │
│  - 内部使用 MongoDB Client 操作数据库                      │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│                    MongoDB Client 层                     │
│  (go.mongodb.org/mongo-driver - 官方 MongoDB 驱动)        │
│  - mongo.Client 连接数据库                                │
│  - mongo.Collection 操作集合                              │
│  - SearchIndexes() 管理搜索索引                           │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│                    MongoDB Atlas 层                      │
│  (MongoDB 云数据库服务)                                  │
│  - 存储文档数据                                           │
│  - 执行向量搜索                                           │
│  - 使用向量索引加速搜索                                    │
└─────────────────────────────────────────────────────────┘
```

## 关键概念

### 1. MongoDB 向量索引（Vector Search Index）

**什么是索引？**
- 索引是数据库的"目录"，用于加速查询
- 普通索引：加速文本/数字查询
- 向量索引：加速向量相似度搜索

**向量索引的作用：**
- 告诉 MongoDB：哪个字段存储向量、向量维度、使用什么相似度算法
- MongoDB 根据索引定义创建数据结构，加速向量搜索

**索引定义示例：**
```go
fields := []vectorField{
    {
        Type:          "vector",        // 这是向量字段
        Path:          "plot_embedding", // 向量存储在文档的这个字段
        NumDimensions: 1024,            // 向量是 1024 维
        Similarity:    "dotProduct",    // 用点积计算相似度
    },
}
```

### 2. 代码分层说明

#### Langchaingo 层（最上层）
```go
// 这是 langchaingo 提供的统一接口
store := mongovector.New(coll, embedder, mongovector.WithIndex("index_name"))
docs, err := store.SimilaritySearch(ctx, "query", 5)
```
- **作用**：提供统一的向量存储 API，隐藏 MongoDB 细节
- **你只需要**：调用 `AddDocuments` 和 `SimilaritySearch`
- **不需要关心**：MongoDB 如何存储、如何搜索

#### Store 层（中间层）
```go
// mongovector.Store 是 langchaingo 的 MongoDB 实现
type Store struct {
    collection *mongo.Collection
    embedder   *embeddings.EmbedderImpl
    indexName  string
}
```
- **作用**：将 langchaingo 的接口转换为 MongoDB 操作
- **内部做**：
  - `AddDocuments`: 将文档转为 MongoDB 文档格式，调用 `collection.InsertMany()`
  - `SimilaritySearch`: 构建 MongoDB 向量搜索查询，调用 `collection.Aggregate()`

#### MongoDB Client 层（底层）
```go
// 这是 MongoDB 官方驱动
client, _ := mongo.Connect(options.Client().ApplyURI(uri))
coll := client.Database("db").Collection("collection")

// 管理索引
view := coll.SearchIndexes()
view.CreateOne(ctx, mongo.SearchIndexModel{...})
```
- **作用**：直接操作 MongoDB 数据库
- **功能**：
  - 连接数据库
  - 创建/查询/删除索引
  - 插入/查询文档

#### Index 层（配置层）
```go
// 索引定义（告诉 MongoDB 如何创建索引）
fields := []vectorField{
    {Type: "vector", Path: "plot_embedding", ...}
}

// 创建索引（通过 MongoDB Client）
view.CreateOne(ctx, mongo.SearchIndexModel{
    Definition: def,
    Options: options.SearchIndexes().SetName("index_name"),
})
```
- **作用**：定义和创建向量索引
- **时机**：在开始使用向量搜索之前创建一次
- **位置**：通过 MongoDB Client 的 `SearchIndexes()` API 创建

## 数据流示例

### 添加文档流程：
```
1. 你调用: store.AddDocuments(ctx, docs)
   ↓
2. Store 层: 
   - 使用 embedder 将文档内容转为向量
   - 构建 MongoDB 文档格式: {plot_embedding: [向量], pageContent: "文本", metadata: {...}}
   - 调用: collection.InsertMany(docs)
   ↓
3. MongoDB Client:
   - 将文档插入到 MongoDB 数据库
   - MongoDB 根据索引定义，将向量存储到索引结构中
```

### 搜索流程：
```
1. 你调用: store.SimilaritySearch(ctx, "query", 5)
   ↓
2. Store 层:
   - 使用 embedder 将查询文本转为向量
   - 构建 MongoDB 向量搜索聚合管道
   - 调用: collection.Aggregate(pipeline)
   ↓
3. MongoDB Client:
   - 发送查询到 MongoDB
   ↓
4. MongoDB Atlas:
   - 使用向量索引快速找到最相似的 5 个文档
   - 返回结果
```

## 为什么需要索引？

**没有索引：**
- MongoDB 需要扫描所有文档
- 对每个文档计算向量相似度
- 非常慢（O(n) 复杂度）

**有索引：**
- MongoDB 使用索引结构（如 HNSW）
- 快速找到相似向量
- 非常快（O(log n) 复杂度）

## 在你的代码中

```go
// MangoVec 是你的适配器层
type MangoVec struct {
    store mongovector.Store  // langchaingo 的 Store
    coll  *mongo.Collection  // MongoDB Client 的 Collection
}

// ensureIndex: 确保索引存在（MongoDB Client 层操作）
func (m *MangoVec) ensureIndex(ctx context.Context) error {
    // 1. 检查索引是否存在（MongoDB Client API）
    ok, _ := m.searchIndexExists(ctx)
    
    // 2. 如果不存在，创建索引（MongoDB Client API）
    if !ok {
        return m.createVectorSearchIndex(ctx)
    }
}

// Search: 使用 Store 层搜索（Langchaingo API）
func (m *MangoVec) Search(ctx context.Context, query string, topK int) {
    // 确保索引存在
    m.ensureIndex(ctx)  // MongoDB Client 层
    
    // 使用 Store 搜索
    return m.store.SimilaritySearch(ctx, query, topK)  // Langchaingo 层
}
```

## 总结

- **Langchaingo**: 提供统一接口，你主要和它交互
- **Store (mongovector)**: langchaingo 的 MongoDB 实现，处理数据转换
- **MongoDB Client**: 官方驱动，用于连接数据库和管理索引
- **Index**: 配置信息，告诉 MongoDB 如何创建向量索引

**你只需要关心：**
1. 创建索引（一次，通过 MongoDB Client）
2. 使用 Store 添加和搜索文档（通过 Langchaingo API）

