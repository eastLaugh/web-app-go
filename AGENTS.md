# 协作与 AI 约定

## 原则

- **最小修改**：只做用户明确要求的事，不顺手多做。
- **少动后端**：能改前端解决的，优先改前端；非必要不碰后端。
- **不隐式行为**：不写隐藏分支、不写“超时则 fallback”等隐式逻辑。

## 码风

- **最小设计**：不过度设计、不过度封装、不过度抽象、不为“以后可能”预留接口或导出 API。
- **能省就省**：参数能少就少（如能从反射/类型推出来的就不让调用方传）；不单独加“方便调试”的导出函数。
- **简单直接**：逻辑能内联就内联，不拆多余辅助函数；取名等用语言/库给的直接结果（如 `runtime.FuncForPC(...).Name()` 全名），不额外截取、格式化。
- **一步到位**：能一次调用完成的就不拆成“构造 + 多次 Register”（如 `New(fn, desc, ...)` 而非 `NewRegistry()` + 多次 `Register`）。
- **能一行就一行**：像 schema 那样，能一句 map 字面量表意就一句；字段说明等用 tag 直接取（如 `f.Tag.Get("description")`），空就空串，不写「有值才往里塞」的分支。
- **无效就 panic**：不合法（如未导出字段参与 schema）直接 panic，不静默 skip。

## 新增或修改 API 的流程

1. **改 OpenAPI**  
   在 `go/internal/api/` 下编辑或新增 YAML，保证符合 OpenAPI 3 规范。

2. **生成代码**  
   用 oapi-codegen 生成服务端类型与接口，入口见 `go/internal/api/gen.go`。

3. **按需补充**  
   如需调试，可少量添加或更新 `go/*.http` 请求示例。

4. **同步配置**  
   若新 API 依赖环境变量，在 `.env.example` 中补充说明。

## Cursor Cloud specific instructions

### Architecture

Single Go binary (`go/`) serves the SolidJS frontend (`solid-project/`) via `//go:embed dist/*`. The frontend must be built before the backend compiles. MongoDB is the only required external service.

### Running locally

1. **MongoDB**: `sudo docker start mongo` (container already exists) or `sudo docker run -d --name mongo -p 27017:27017 mongo:latest`
2. **Frontend build**: `cd solid-project && npm run build` (outputs to `go/dist/`)
3. **Backend build + run**: `cd go && go build -o ../server_binary . && cd .. && ./server_binary`
4. App at `http://localhost:8080/app/`, console at `:2333`

The `.env` at repo root is read by `godotenv`. Minimum required vars: `EASTLAUGH_MONGODB_URI`, `EASTLAUGH_ADDR`, `EASTLAUGH_TOKEN_PWD`. See `.env.example`.

### Lint / Test / Build

| What | Command |
|------|---------|
| Go vet | `cd go && go vet ./...` |
| TypeScript check | `cd solid-project && npx tsc --noEmit` |
| Go tests | `cd go && go test ./...` |
| Build all | `make` (or manually: npm build then go build) |

### Gotchas

- Go 1.26.0 is required (`go.mod` specifies it). The default system Go may be older; ensure `/usr/local/go/bin` is on `PATH`.
- The Go binary embeds `go/dist/` at compile time. If `go/dist/` doesn't exist (frontend not built), the Go build fails.
- AI chat features require `OPENAI_API_KEY` and related env vars; without them the blog and comments still work but chat returns errors.
- Docker daemon in the Cloud VM needs `fuse-overlayfs` storage driver and `iptables-legacy`; these are configured during initial setup.
