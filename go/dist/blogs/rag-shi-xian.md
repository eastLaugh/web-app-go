# 为个人网站添加 RAG 功能

最近给网站添加了 RAG（检索增强生成）功能，让 AI 能够基于我的博客内容回答问题。整个过程还挺有意思的，记录一下实现细节 🤣

## 背景

之前网站的聊天功能只是直接调用 DeepSeek API，AI 对我的博客内容一无所知。用户问"你写过哪些关于 Go 的文章？"时，AI 只能瞎猜。

所以决定引入 RAG，让 AI 能够：
- 检索相关的博客内容
- 基于检索到的内容回答问题
- 提供文章链接，方便用户查看原文

## 技术选型

### Embedding 服务

选择了**阿里云 text-embedding-v4**：
- 支持中文，效果不错
- 提供 OpenAI 兼容接口，集成简单
- 新用户有免费额度（90 天内 100 万 Token）
- 成本低（每千 Token 0.0005 元）

### 向量存储

使用 **MongoDB** 存储向量：
- 项目已有 MongoDB，无需额外依赖
- 支持存储高维向量数组
- 简单实现余弦相似度搜索（数据量不大时够用）

### 实现方式

坚持"最小代码"原则，使用**纯 Go 标准库**：
- 不使用 langchaingo 等框架（避免过度设计）
- 直接用 `net/http` 调用 API
- 代码简洁，易于维护

## 架构设计

```
用户提问 → 向量化查询 → MongoDB 检索 → 获取相关文档 → 注入上下文 → DeepSeek 生成回答
```

### 核心组件

1. **Embedding 服务** (`pkg/embedding/`)
   - 调用阿里云 API 生成向量
   - 支持批量向量化

2. **向量存储** (`internal/repo/vector_repo.go`)
   - MongoDB 存储文档 chunk 和向量
   - 实现余弦相似度搜索

3. **文档处理** (`pkg/document/`)
   - 读取 Markdown 文件
   - 按段落切分（chunking）

4. **RAG 集成** (`cmd/server/ports/server.go`)
   - 用户提问时检索相关文档
   - 将检索结果注入到 LLM 上下文

## 具体实现

### 1. Embedding 服务

使用 OpenAI 兼容接口调用阿里云：

```go
// pkg/embedding/embedding.go
func (c *Client) Embed(ctx context.Context, texts []string, textType string) ([][]float32, error) {
    // input 可以是字符串（单个）或字符串数组（批量）
    var input interface{}
    if len(texts) == 1 {
        input = texts[0]
    } else {
        input = texts
    }

    reqBody := map[string]interface{}{
        "model": "text-embedding-v4",
        "input": input,
    }

    req, _ := http.NewRequestWithContext(ctx, "POST", 
        "https://dashscope.aliyuncs.com/compatible-mode/v1/embeddings", 
        bytes.NewBuffer(jsonData))
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
    // ...
}
```

**关键点**：
- 使用 `compatible-mode/v1/embeddings` 接口（OpenAI 兼容）
- `input` 字段支持字符串或字符串数组
- 响应格式遵循 OpenAI 标准

### 2. 文档切分策略

按段落切分，保留语义完整性：

```go
// pkg/document/document.go
func ProcessMarkdown(content string, file string) []Chunk {
    // 按双换行符切分段落
    paragraphs := strings.Split(content, "\n\n")
    
    for i, para := range paragraphs {
        // 如果段落太长（超过 1000 字符），进一步按句子切分
        if len(para) > 1000 {
            sentences := splitBySentences(para)
            // 合并句子直到接近 1000 字符
        }
    }
}
```

**切分原则**：
- 优先按段落切分（保留语义）
- 长段落（>1000 字符）按句子进一步切分
- 提取文章标题作为元数据

### 3. 向量存储结构

MongoDB 文档结构：

```go
type VectorDoc struct {
    ID         interface{} `bson:"_id,omitempty"`  // ObjectID
    File       string      `bson:"file"`           // blogs/xxx.md
    ChunkIndex int         `bson:"chunk_index"`    // chunk 索引
    Content    string      `bson:"content"`        // 原始文本
    Embedding  []float32   `bson:"embedding"`      // 1024 维向量
    Metadata   struct {
        Title     string `bson:"title,omitempty"`
        CreatedAt string `bson:"created_at,omitempty"`
    } `bson:"metadata,omitempty"`
}
```

### 4. 相似度搜索

实现简单的余弦相似度搜索：

```go
// internal/repo/vector_repo.go
func (r *VectorRepo) SearchSimilar(ctx context.Context, queryVector []float32, topK int) ([]*ScoredDoc, error) {
    // 获取所有文档（数据量不大时可行）
    cursor, _ := r.collection.Find(ctx, bson.M{})
    
    // 计算余弦相似度
    scoredDocs := make([]*ScoredDoc, 0)
    for _, doc := range allDocs {
        score := cosineSimilarity(queryVector, doc.Embedding)
        scoredDocs = append(scoredDocs, &ScoredDoc{
            Doc:   doc,
            Score: score,
        })
    }
    
    // 按分数降序排序，返回 Top-K
    // ...
}

func cosineSimilarity(a, b []float32) float64 {
    var dotProduct, normA, normB float64
    for i := 0; i < len(a); i++ {
        dotProduct += float64(a[i] * b[i])
        normA += float64(a[i] * a[i])
        normB += float64(b[i] * b[i])
    }
    return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
```

**注意**：如果数据量很大，这种方式效率低，应该使用 MongoDB 的 `$vectorSearch` 或专门的向量数据库。

### 5. RAG 上下文注入

将检索结果作为 system message 追加到对话中：

```go
// cmd/server/ports/server.go
if len(scoredDocs) > 0 {
    // 构建 RAG 上下文
    var contextBuilder strings.Builder
    contextBuilder.WriteString("以下是我博客中的相关内容：\n\n")
    
    for i, scoredDoc := range scoredDocs {
        doc := scoredDoc.Doc
        filePath := "/app/" + strings.TrimSuffix(doc.File, ".md")
        
        contextBuilder.WriteString(fmt.Sprintf(
            "[片段 %d] 来源：《%s》\n"+
                "文件路径：%s\n"+
                "相似度：%.2f%%\n"+
                "内容：\n%s\n\n",
            i+1, doc.Metadata.Title, filePath,
            scoredDoc.Score*100, doc.Content,
        ))
    }
    
    // 作为 system message 追加到 messages 中
    ragSystemMsg := map[string]interface{}{
        "role":    "system",
        "content": contextBuilder.String(),
    }
    // 在用户消息之前插入
    messages = append(messages[:insertIndex], 
        append([]map[string]interface{}{ragSystemMsg}, 
            messages[insertIndex:]...)...)
}
```

**设计考虑**：
- 不修改基础 system message（保持角色设定不变）
- 每次对话动态追加 RAG 结果
- 包含文件路径、相似度等元数据，方便 AI 引用

## 遇到的问题

### 1. MongoDB ObjectID 解码错误

**问题**：`error decoding key _id: decoding an object ID into a string is not supported`

**解决**：将 `ID` 字段类型改为 `interface{}`，可以接受 MongoDB 的 ObjectID 类型。

```go
type VectorDoc struct {
    ID interface{} `bson:"_id,omitempty"`  // 而不是 string
}
```

### 2. 阿里云 API 请求格式错误

**问题**：API 返回 `"The input parameter requires json format"`

**解决**：
- 使用 OpenAI 兼容接口：`compatible-mode/v1/embeddings`
- `input` 字段直接使用字符串或字符串数组，不需要嵌套对象
- 移除 `parameters` 字段（兼容接口不支持）

### 3. MongoDB 连接提前关闭

**问题**：`client is disconnected`

**解决**：移除 `initMongo()` 函数中的 `defer client.Disconnect()`，让连接保持打开。

## 索引构建

提供手动触发索引重建的接口：

```bash
curl -X POST http://localhost:8080/api/v1/rag/rebuild-index
```

构建流程：
1. 清空现有索引
2. 读取所有博客 Markdown 文件
3. 按段落切分
4. 批量向量化（每次 10 个）
5. 存储到 MongoDB

## 效果

现在用户问"你写过哪些关于 Go 的文章？"时，AI 能够：
- 检索到相关的博客内容
- 基于实际内容回答
- 提供文章链接（如 `/app/blogs/yi-chu-gin`）
- 显示相似度，让用户知道相关性

## 总结

这次实现 RAG 功能，坚持了"最小代码"原则：
- ✅ 零额外依赖（只用标准库 + 现有 mongo-driver）
- ✅ 代码简洁，易于维护
- ✅ 与现有代码风格一致

虽然实现比较简单（比如向量搜索是 O(n) 的），但对于个人博客这种数据量不大的场景完全够用。如果后续数据量增长，再考虑优化或迁移到专业向量数据库。

**技术栈**：Go 标准库 + MongoDB + 阿里云 text-embedding-v4 + DeepSeek

---

**相关链接**：
- 项目地址：https://github.com/eastLaugh/web-app-go
- 阿里云百炼：https://bailian.console.aliyun.com/











