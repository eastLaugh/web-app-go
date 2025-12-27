# 从 MySQL 迁移到 MongoDB：告别关系型数据库 🤣

最近把项目的评论系统从 MySQL 迁移到了 MongoDB，感觉整个人都轻松了！记录一下这次迁移的过程和感受。

## 为什么迁移？

其实最开始用 MySQL 也没啥问题，就是觉得：
- 项目已经有 MongoDB 了（用于 RAG 向量存储），不想维护两套数据库
- 评论数据本身就很适合文档型数据库（一条评论就是一个文档）
- 减少依赖，简化部署（docker-compose 里少一个服务）
- 想体验一下 MongoDB 的简洁

## 迁移过程

### 1. 数据结构对比

**之前（MySQL）：**
```sql
CREATE TABLE posts (
    id INT AUTO_INCREMENT PRIMARY KEY,
    file VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_file (file)
);
```

**现在（MongoDB）：**
```go
type PostDoc struct {
    Id        interface{} `bson:"_id,omitempty"`
    File      string      `bson:"file"`
    Content   string      `bson:"content"`
    Email     string      `bson:"email"`
    CreatedAt time.Time   `bson:"created_at"`
}
```

不需要定义表结构，直接定义 Go 结构体就行，MongoDB 会自动处理！

### 2. 代码对比

**之前（MySQL）：**
```go
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
            log.Printf("扫描评论数据失败: %v", err)
            continue
        }
        // ... 转换逻辑
    }
    return posts, rows.Err()
}
```

**现在（MongoDB）：**
```go
func (coll *MangoPostRepo) GetPostsByFile(ctx context.Context, file string) ([]api.Post, error) {
    cursor, err := coll.Find(ctx, bson.M{"file": file})
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)
    
    var posts []api.Post
    for cursor.Next(ctx) {
        var post PostDoc
        if err := cursor.Decode(&post); err != nil {
            return nil, err
        }
        // ... 转换逻辑
    }
    return posts, nil
}
```

代码简洁多了！不需要写 SQL，不需要手动 Scan，直接 Decode 到结构体就行。

### 3. 插入数据对比

**之前（MySQL）：**
```go
func (r *MySQLPostRepo) InsertPost(ctx context.Context, email string, content string, file string) error {
    _, err := r.db.ExecContext(ctx, 
        "INSERT INTO posts (email, content, file) VALUES (?, ?, ?)", 
        email, content, file)
    return err
}
```

**现在（MongoDB）：**
```go
func (coll *MangoPostRepo) InsertPost(ctx context.Context, email string, content string, file string) error {
    _, err := coll.InsertOne(ctx, PostDoc{
        File:      file,
        Content:   content,
        Email:     email,
        CreatedAt: time.Now(),
    })
    return err
}
```

直接插入结构体，不需要写 SQL，不需要关心字段顺序！

## 遇到的问题

### MongoDB ObjectID 解码错误

迁移过程中遇到了一个经典问题：`error decoding key _id: cannot decode objectID into an array`

**原因**：MongoDB 的 `_id` 字段可能是 ObjectID 类型，但代码中定义的是 `primitive.ObjectID`，在某些情况下会有类型不匹配的问题。

**解决**：将 `_id` 字段类型改为 `interface{}`，可以接受任何类型的 `_id`：

```go
type PostDoc struct {
    Id        interface{} `bson:"_id,omitempty"`  // 改为 interface{}
    File      string      `bson:"file"`
    Content   string      `bson:"content"`
    Email     string      `bson:"email"`
    CreatedAt time.Time   `bson:"created_at"`
}
```

然后在解码时处理不同类型的 `_id`：

```go
var idStr *string
if post.Id != nil {
    if oid, ok := post.Id.(primitive.ObjectID); ok {
        hex := oid.Hex()
        idStr = &hex
    } else if str, ok := post.Id.(string); ok {
        idStr = &str
    }
}
```

这样无论 MongoDB 中存储的是什么格式的 `_id`，都能正确解码。

## 清理工作

迁移完成后，做了以下清理：

1. **删除 MySQL 相关代码**：
   - 删除了 `mysql_post_repo.go`
   - 删除了 `schema.sql`
   - 移除了 `database/sql` 导入
   - 移除了 MySQL 驱动依赖

2. **简化配置**：
   - 从 `docker-compose.yml` 中移除了 MySQL 服务
   - 从 `.env.example` 中移除了 MySQL 相关环境变量
   - 从 `go.mod` 中移除了 MySQL 驱动

3. **简化代码**：
   - `NewServer()` 函数不再需要 `db *sql.DB` 参数
   - 移除了 `initDB()` 函数
   - 移除了 `db.Close()` 调用

## 迁移后的好处

1. **代码更简洁**：不需要写 SQL，直接操作结构体
2. **部署更简单**：docker-compose 里少一个服务
3. **维护成本更低**：只需要维护一套数据库
4. **开发体验更好**：类型安全，IDE 提示更友好
5. **扩展性更好**：如果以后需要存储更复杂的评论数据（比如嵌套回复），MongoDB 更灵活

## 总结

从 MySQL 迁移到 MongoDB 后，代码更简洁、部署更简单、维护成本更低。虽然遇到了一些小问题（比如 ObjectID 解码），但整体迁移过程很顺利。

如果你也在考虑是否要迁移到 MongoDB，我的建议是：**如果你的数据结构比较灵活，或者已经有 MongoDB 了，那就用 MongoDB 吧**。毕竟，少维护一套数据库总是好的 😂

---

**技术栈更新**：Go + MongoDB + SolidJS + OpenAPI + oapi-codegen









