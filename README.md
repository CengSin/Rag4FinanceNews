## Rag4FinanceNews

面向财经资讯的 RAG 与对话系统。采用 Temporal 协调文章处理与会话，Echo 提供 HTTP 接口，Qdrant 存储向量，Redis 持久化会话，MCP 工具扩展检索能力。

### V0.1.1 Agent 能力
- 会话路由：基于关键词判定结构化/语义查询，过滤工具列表，无法判定时要求用户澄清。
- 工具调用：通过 MCP 暴露的查询/检索工具，遵循 `systemPrompt.md` 的规则。
- CDC 控制：HTTP 接口 `/cdc/start`、`/cdc/stop` 可启动/停止多配置的 CDC 监听。
- 强类型 CDC：使用 `CdcEnvelope` + 业务 struct（如 `ContentMessageEvent`）传输 CDC 事件，避免类型漂移/解码失败。

### 功能与流程
- 文章处理：HTTP 接收原始文章，Temporal Workflow 负责摘要、向量化和写入 Qdrant。
- 会话与工具调用：基于会话的聊天，支持工具调用（结构化检索、语义检索），历史消息存 Redis。
- CDC 同步：监听 MySQL Binlog，将增量数据通过 Sync Temporal Workflow 同步到向量库。

### 技术栈
- 语言：Go 1.25+
- Web/API：Echo v4
- 协调：Temporal（默认 namespace `default`，CDC 同步 namespace `CDC_SYNC_NAMESPACE`）
- 数据：Qdrant（向量库）、MySQL（业务库）、Redis（会话）
- AI：OpenRouter/OpenAI（go-openai）、MCP 工具（mcp-go）
- 配置：cleanenv 读取 `config/config.yaml`

### 目录结构
- `main.go` / `worker.go`：启动 HTTP 服务与 Temporal Worker。
- `api/`：Echo Handler（文章处理、QA、会话管理）。
- `workflow/`：Temporal Workflows（文章处理、聊天、会话、CDC）。
- `activity/`：Workflow Activities（LLM 调用、向量化、Qdrant Upsert、MCP 调用、会话存储）。
- `handle/`：CDC 监听与处理（go-mysql canal）。
- `client/`：外部客户端初始化（Temporal、Qdrant、OpenAI、Redis、MCP）。
- `dao/`：会话消息持久化。
- `config/`：配置结构与示例 `config.yaml`。
- `cdc-listener/`：独立的 CDC/Temporal 示例代码。

### 配置
编辑 `config/config.yaml`，关键字段：
- `openAI.baseURL`、`openAI.apiKey`（可用环境变量 `OPENROUTER_API_KEY` 覆盖）
- `temporal.hostPort` / `namespace`
- `syncTemporal.hostPort` / `namespace`
- `qdrant.host` / `port`（默认 6334）
- `redis.addr` / `DB`
- `mcpServer`（如 `http://localhost:8085/mcp`）
- `cdc`：MySQL 连接信息与监听表

### 本地依赖示例
Qdrant
```bash
docker run -p 6333:6333 -p 6334:6334 \
  -v $(pwd)/qdrant_storage:/qdrant/storage \
  qdrant/qdrant
```

Temporal
```bash
temporal server start-dev --namespace default
temporal operator namespace create --namespace CDC_SYNC_NAMESPACE
```

Redis
```bash
docker run -p 6379:6379 redis:7
```

MySQL（需开启 Binlog，row 模式），并保证 `config.cdc` 中的用户具备 REPLICATION 权限。

MCP 服务器（工具提供方）需对外暴露 HTTP，默认示例端口 `8085`。

### 启动
```bash
go run ./main.go   # 启动 HTTP 服务与两个 Worker（默认端口 8081）
```

### 主要接口
- `POST /article/process`：提交文章，触发处理 Workflow。
- `POST /ai/query`：直接问答。
- `POST /ai/temporal`：通过 Workflow 的 RAG 对话。
- `POST /ai/new/session`：开启/继续会话。
- `GET /session/history`、`GET /session/list`、`DELETE /session/:session_id`：会话管理。

### Temporal 队列
- 业务队列：`financial-news-queue`（文章处理、聊天）
- CDC 同步队列：`CDC_SYNC_QUEUE`

### 备注
- 工具调用遵循 `systemPrompt.md` 中的路由规则；无法判断工具时会要求用户澄清。
- 生产环境建议为外部调用配置合理超时与重试策略，并在 Temporal 中调整 Activity/Workflow 的超时配置以匹配外部服务 SLA。
