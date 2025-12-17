# 移除了 Gin，拥抱标准库

最近把项目从 Gin 框架迁移到了 Go 标准库，感觉整个人都清爽了 🤣

## 为什么移除 Gin？

其实也没啥特别的原因，就是觉得：
- 标准库够用了，不需要额外的依赖
- 想写更 "Go" 的代码
- 减少依赖，降低复杂度
- 学习标准库的最佳实践

## 主要改动

### 1. 路由注册

**之前（Gin）：**
```go
v1 := gin.New()
api.RegisterHandlersWithOptions(v1, server, api.GinServerOptions{
    BaseURL:     "/api/v1/",
    Middlewares: []api.MiddlewareFunc{tokens.Middleware},
})
```

**现在（标准库）：**
```go
mux := http.NewServeMux()
apiHandler := api.HandlerWithOptions(server, api.StdHTTPServerOptions{
    BaseURL:     "/api/v1",
    Middlewares: []api.MiddlewareFunc{tokens.Middleware},
})
mux.Handle("/api/v1/", http.StripPrefix("/api/v1", apiHandler))
```

### 2. 中间件风格

**之前（Gin）：**
```go
func Middleware(ctx *gin.Context) {
    // ...
    ctx.Next()
}
```

**现在（标准库）：**
```go
func Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // ...
        next.ServeHTTP(w, r)
    })
}
```

这才是 Go 中间件的标准写法！

### 3. Handler 方法签名

**之前（Gin）：**
```go
func (s server) PostAuth(c *gin.Context) {
    var request api.PostAuthJSONRequestBody
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    // ...
    c.JSON(http.StatusOK, gin.H{"token": token})
}
```

**现在（标准库）：**
```go
func (s server) PostAuth(w http.ResponseWriter, r *http.Request) {
    var request api.PostAuthJSONRequestBody
    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        writeJSONError(w, http.StatusBadRequest, err.Error())
        return
    }
    // ...
    writeJSON(w, http.StatusOK, map[string]string{"token": token})
}
```

### 4. 参数获取

**之前（Gin）：**
```go
email := c.GetString("email")
```

**现在（标准库）：**
```go
email, ok := r.Context().Value("email").(string)
if !ok {
    writeJSONError(w, http.StatusUnauthorized, "未授权")
    return
}
```

使用 context 传递值，更符合 Go 的惯用法。

### 5. 日志和恢复中间件

自己实现了一个简单的日志和恢复中间件：

```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        logrus.Infof("%s %s %v", r.Method, r.URL.Path, time.Since(start))
    })
}

func recoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                logrus.Errorf("Panic recovered: %v", err)
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

## 使用标准库的好处

1. **零依赖**：不需要维护额外的框架依赖
2. **更轻量**：编译后的二进制更小
3. **更灵活**：完全控制 HTTP 处理流程
4. **更 Go**：符合 Go 社区的最佳实践
5. **更容易理解**：标准库的代码更直观

## 迁移过程

整个迁移过程其实挺顺利的：

1. 修改 OpenAPI 代码生成配置，从 `gin-server: true` 改为 `std-http-server: true`
2. 修改所有 handler 方法签名
3. 手动解析 JSON 请求体（之前 Gin 自动处理）
4. 使用 context 传递值（之前用 Gin 的 `Set`/`Get`）
5. 实现辅助函数处理 JSON 响应
6. 运行 `go mod tidy` 清理依赖

## 总结

移除 Gin 后，代码更简洁、更符合 Go 的惯用法。虽然需要手动处理一些之前框架自动做的事情（比如 JSON 解析），但换来的是更清晰的控制和更少的依赖。

如果你也在考虑是否要移除某个框架，我的建议是：**如果标准库够用，就用标准库**。毕竟，Go 的标准库已经很强大了 😂

---

**技术栈更新**：Go + 标准库 + SolidJS + OpenAPI + oapi-codegen

