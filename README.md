# 🍮 AI Gateway

一个轻量级的 AI API 网关，支持多供应商聚合、负载均衡、多租户管理和额度控制。

## ✨ 特性

- **多供应商聚合**：开箱即支持 12 家 AI 供应商、50+ 最新模型
- **负载均衡**：同一模型可映射到多个供应商，自动按权重轮询分配
- **双协议兼容**：对外提供 OpenAI 兼容和 Anthropic 兼容两种 API 格式
- **多租户管理**：Admin 可以管理租户、分配额度、查看所有用量
- **租户自助面板**：租户登录后可查看额度、创建/吊销/轮换 API Key
- **灵活额度**：支持按 Token 和按调用次数两种额度管理方式
- **不限额 Key**：Admin 可以为特定租户分配不限额度的 Key
- **用量日志**：详细的调用日志，包含 Token 用量、延迟、状态码
- **SQLite**：轻量级数据库，无需额外依赖
- **Docker 部署**：一键启动

## 🚀 快速开始

### Docker 部署

```bash
# 克隆项目
git clone https://github.com/your-org/aigateway.git
cd aigateway

# 修改环境变量
cp .env.example .env
# 编辑 .env，设置 ADMIN_PASSWORD 等

# 启动
DOCKER_BUILDKIT=1 docker compose up -d --build

# 查看日志
docker compose logs -f
```

### Docker 构建优化

当前 Docker 构建已包含以下优化：

- **多阶段构建**：仅将最终二进制和运行所需静态资源打进运行镜像
- **BuildKit 缓存**：缓存 `go mod download` 和 `go build`，加快重复构建
- **更小体积**：使用 `-trimpath -ldflags="-s -w"` 减少二进制大小
- **非 root 运行**：容器内默认以 `app` 用户启动
- **运行资源完整**：自动包含 `templates/` 和 `static/`

### 多架构构建（amd64 / arm64）

先初始化 buildx：

```bash
docker buildx create --name multiarch --use 2>/dev/null || true
docker buildx inspect --bootstrap
```

构建本地镜像：

```bash
make docker-buildx
# 或
IMAGE=aigateway TAG=latest docker buildx bake local
```

发布多架构镜像：

```bash
IMAGE=your-dockerhub/aigateway TAG=latest make release
# 或
IMAGE=your-dockerhub/aigateway TAG=latest docker buildx bake --push release
```

### GitHub Actions 自动发版

已内置工作流：`.github/workflows/docker.yml`

触发规则：
- push 到 `main`，自动构建多架构镜像并推送到 `ghcr.io`
- push tag（如 `v1.0.0`），自动发布对应版本 tag
- pull request，仅做构建校验，不推送

镜像地址格式：

```text
ghcr.io/<你的 GitHub 用户名或组织名>/<仓库名>
```

例如仓库是 `cuckoo/aigateway`，则镜像会发布到：

```text
ghcr.io/cuckoo/aigateway
```

默认会生成这些 tag：
- `latest`（默认分支）
- `main`
- `v1.0.0` 这类版本 tag
- Git SHA 短标签

如果你还想同时推送到 Docker Hub，请在 GitHub 仓库里配置：

- `Repository Variable`: `DOCKERHUB_IMAGE`，例如 `cuckoo/aigateway`
- `Repository Secret`: `DOCKERHUB_USERNAME`
- `Repository Secret`: `DOCKERHUB_TOKEN`

另外，当你 push `v*` tag 时，工作流还会自动创建 GitHub Release，并附上镜像地址、多架构说明，以及镜像 digest。

供应链相关产物也会自动生成：
- **SBOM**（随镜像作为 OCI attestation 发布）
- **Provenance** / 构建来源证明（随镜像作为 OCI attestation 发布）
- `image-digest` artifact（可在 GitHub Actions 构建产物中下载）

### 本地开发

```bash
# 下载依赖
go mod tidy

# 运行（会自动初始化数据库和默认数据）
go run ./cmd/server
```

## 📋 支持的供应商（预设）

| 供应商 | 类型 | 通用 API Base URL | Coding Plan Base URL | 预设模型 |
|--------|------|-------------------|---------------------|----------|
| OpenAI | openai | `https://api.openai.com` | - | gpt-5.5, gpt-5.5-pro, gpt-5.5-mini, gpt-5.4, o3, o4-mini, gpt-oss |
| Anthropic | anthropic | `https://api.anthropic.com` | - | claude-sonnet-4-6, claude-opus-4-6, claude-opus-4-7, claude-haiku-4-5 |
| Google Gemini | openai | `https://generativelanguage.googleapis.com/v1beta/openai` | - | gemini-3.1-pro, gemini-3.1-flash, gemini-3.1-flash-lite, gemini-3-flash, gemini-2.5-pro |
| DeepSeek | openai | `https://api.deepseek.com` | - | deepseek-v4-pro, deepseek-v4-flash, deepseek-v3.2 |
| 阿里云百炼 | openai/anthropic | `https://dashscope.aliyuncs.com/compatible-mode/v1` | `https://coding.dashscope.aliyuncs.com/v1` | qwen3.6-max-preview, qwen3.6-plus, qwen3.5-plus, qwen3-max, qwen3.6-flash, qwen3-coder |
| Moonshot/Kimi | openai | `https://api.moonshot.cn/v1` | `https://api.kimi.com/coding/v1` | kimi-k2, kimi-k2-thinking, kimi-k2-6, kimi-for-coding |
| 智谱 AI | openai/anthropic | `https://open.bigmodel.cn/api/paas/v4` | `https://open.bigmodel.cn/api/coding/paas/v4` | glm-5, glm-5-flash, glm-4-plus, glm-4v |
| Minimax | openai | `https://api.minimax.chat/v1` | `https://api.minimax.chat/v1` (Token Plan) | minimax-m2.7, minimax-m2.5, minimax-m2, minimax-m2-her |
| 小米 AI | openai | `https://api.xiaomi.com/v1` | - | mimo, mimo-v2 |
| Groq | openai | `https://api.groq.com/openai/v1` | - | - |
| SiliconFlow | openai | `https://api.siliconflow.cn/v1` | - | - |
| Together AI | openai | `https://api.together.xyz/v1` | - | - |

## ⚖️ 负载均衡

当同一模型名映射到多个供应商时，系统会自动启用负载均衡：

- **策略**：平滑加权轮询（Nginx-style Smooth Weighted Round-Robin）
- **权重**：每个映射可配置权重，权重越高分配比例越大
- **故障转移**：当某个供应商不可用时，下次请求自动切换到其他供应商

### 示例：为 deepseek-chat 配置多供应商

```bash
# 在管理后台中：
# 1. 添加供应商：deepseek-official（已有）
# 2. 添加供应商：siliconflow（已有）
# 3. 添加模型映射：
#    - deepseek-chat → deepseek-official（权重: 2）
#    - deepseek-chat → siliconflow（权重: 1）
# 
# 结果：2/3 的请求走 DeepSeek 官方，1/3 走 SiliconFlow
```

## 🔑 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PORT` | `8080` | 服务端口 |
| `DB_PATH` | `aigateway.db` | 数据库路径 |
| `ADMIN_PASSWORD` | `admin123` | 默认管理员密码 |
| `PROXY_TIMEOUT` | `60s` | 代理超时时间 |

## 📡 API 使用

### OpenAI 兼容

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### Anthropic 兼容

```bash
curl http://localhost:8080/v1/messages \
  -H "Authorization: Bearer sk-your-key" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "messages": [{"role": "user", "content": "Hello!"}],
    "max_tokens": 1024
  }'
```

## 🔧 配置流程

1. **启动服务** → 自动创建 admin 账号和默认供应商/模型数据
2. **填写 API Key** → 登录管理后台，编辑各供应商的 API Key
3. **创建租户** → 创建用户并分配额度
4. **生成 API Key** → 为租户生成 API Key（支持不限额度）
5. **租户使用** → 租户用 API Key 调用 `/v1/chat/completions` 或 `/v1/messages`
6. **配置负载均衡** → 同一模型映射到多个供应商，设置权重

## 🔑 Coding Plan 支持

国内多家 AI 平台推出了专属编程套餐（Coding Plan），使用独立的 API Key 和 Base URL：

| 平台 | Coding Plan Base URL | Key 特征 |
|------|---------------------|----------|
| 阿里云百炼 | `https://coding.dashscope.aliyuncs.com/v1` | `sk-sp-` 开头 |
| 阿里云百炼 (Anthropic) | `https://coding.dashscope.aliyuncs.com/apps/anthropic/v1` | `sk-sp-` 开头 |
| 智谱 AI | `https://open.bigmodel.cn/api/coding/paas/v4` | 专属 Coding Key |
| 智谱 AI (Anthropic) | `https://open.bigmodel.cn/api/anthropic` | 专属 Coding Key |
| Kimi | `https://api.kimi.com/coding/v1` | 专属 Coding Key |

**在管理后台中添加 Coding Plan 供应商：**
1. 进入「供应商配置」，点击「+ 添加供应商」
2. 从预设下拉菜单选择对应的 Coding Plan 配置
3. 填入专属 API Key，保存即可
4. 将模型映射到该 Coding Plan 供应商，租户即可使用

## 🏗️ 技术栈

- **后端**：Go + Gin + GORM
- **前端**：HTML + Tailwind CSS + Vanilla JS
- **数据库**：SQLite
- **部署**：Docker

## 📂 项目结构

```
aigateway/
├── cmd/server/main.go          # 入口
├── internal/
│   ├── db/
│   │   ├── init.go             # 数据库初始化
│   │   ├── models.go           # 数据模型
│   │   └── seed.go             # 默认数据（供应商+模型）
│   ├── middleware/
│   │   └── auth.go             # API Key 认证中间件
│   ├── proxy/
│   │   └── proxy.go            # 代理转发 + 负载均衡
│   └── handlers/
│       ├── openai.go           # OpenAI 兼容接口
│       ├── anthropic.go        # Anthropic 兼容接口
│       ├── admin.go            # 管理后台 API
│       └── keys.go             # 租户 Key 管理 API
├── templates/
│   ├── login.html              # 登录页
│   ├── admin.html              # 管理后台
│   └── dashboard.html          # 租户面板
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## 📄 License

MIT
