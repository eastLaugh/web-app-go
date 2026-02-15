# web-app-go

个人网站：博客 + 评论 + AI 聊天（RAG）。前后端分离。

## 技术栈

- **前端**：SolidJS，静态构建后由 Go 托管
- **后端**：Go 标准库 HTTP + OpenAPI (oapi-codegen)
- **AI**：openai-go，兼容 OpenAI / 千问等（对话 + 向量）
- **数据**：MongoDB

## 本地运行

```bash
# 构建前端 + 后端单文件
make

# 运行（会读项目根目录 .env）
./server_binary
```

需事先配置 `.env`（参考 `.env.example`），并保证 MongoDB 可用。

## Docker 运行

```bash
docker compose up -d --build
```

访问 `http://localhost`。环境变量来自 `.env`，改 env 后需 `docker compose up -d --force-recreate` 才会生效。

## 目录结构

| 目录 | 说明 |
|------|------|
| `solid-project/` | 前端源码，构建输出到 `go/dist` |
| `go/` | 后端 Go 代码、API 定义、嵌入的前端静态资源 |
| `go/internal/api/` | OpenAPI YAML 与生成代码 |

## 部署

Makefile 构建出单文件 `server_binary`，可 SCP 上传到服务器后手动启动；或使用 `docker compose` 部署。

---

协作与 AI 参与时的约定见 [AGENTS.md](./AGENTS.md)。
