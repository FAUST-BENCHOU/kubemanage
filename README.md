简体中文 | [English](./README_en.md)

# Kubernetes AI 应用管理中心

> 本项目致力于在 Kubernetes 集群中统一部署、运行与治理多种 AI 应用（大模型、知识库、MCP 工具等），让 AIOps 团队可以在同一个控制台完成模型接入、知识库构建、推理服务暴露以及多模型协同调度。项目基于 gin + gorm + client-go 构建后端，前端使用 Vue3 技术栈，适合作为企业级 AI 应用管理平台的脚手架。

## 核心能力

- **Ollama 模型编排**：一键部署 / 列表管理 / 模型拉取 / 会话接口 / Embedding，原生适配集群内的节点调度与资源限制。
- **知识库工作台**：支持多种向量数据库（ChromaDB、Milvus、Weaviate）部署、文档上传切分、向量检索与问答。
- **MCP 生态衔接**：可注册多种 Model Context Protocol Server，为智能体提供工具集。
- **AI 场景编排**：封装 `/api/ai/chat_with_kb` 接口，联动知识库与大模型，构建企业级检索增强生成（RAG）服务。
- **平台治理**：RBAC、操作审计、资产管理、CMDB、工单等传统能力仍然保留，可与 AI 场景结合。

## 架构概览

- **后端**：Gin + GORM，负责 API、鉴权、Kubernetes client-go 资源编排、Casbin RBAC。
- **前端**：Vue3 + Vite + Pinia，提供 AI 应用控制台（项目仓库：[kubemanage-web](https://github.com/noovertime7/kubemanage-web)）。
- **Kubernetes 适配**：使用 kubeconfig 或 InCluster 配置连接集群，所有 AI 服务均以 Deployment/Service 形式管理。
- **存储**：MySQL（权限、CMDB、审计等业务数据）。

## 快速开始

### 1. 准备环境
- Go 1.20+
- Node.js 16+
- 可访问的 Kubernetes 集群（本地 kind/minikube 或云上集群）
- MySQL（默认数据库名 `kubemanage`）

### 2. 初始化数据库
```sql
CREATE DATABASE kubemanage CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
```

### 3. 启动后端
```bash
git clone https://github.com/FAUST-BENCHOU/kubemanage.git
cd kubemanage

go mod tidy

# 指定配置文件：环境变量 KUBEMANAGE-CONFIG 或命令行 --configFile
go run cmd/main.go --configFile=config.yaml
```
> 默认账号密码：`admin / kubemanage`

### 4. 启动前端
```bash
git clone https://github.com/FAUST-BENCHOU/kubemanage-web.git
cd kubemanage-web
npm install
npm run dev
```

## AI 模块说明

| 模块 | 接口 | 主要输入 | 典型返回 |
| ---- | ---- | -------- | -------- |
| Ollama 部署 | `POST /api/k8s/ollama/deploy` | `kubeDto.OllamaDeployInput`（名称、命名空间、镜像、端口…） | `{"code":200,"data":"部署成功"}` |
| Ollama 列表 | `GET /api/k8s/ollama/list` | `filter_name / namespace / node_name / page / limit` | `{"data":{"total":n,"items":[...]}}` |
| 模型拉取 | `POST /api/k8s/ollama/model/pull` | `pod_name / namespace / model_name` | `{"data":"拉取成功"}` |
| 模型列表/详情/删除 | `/model/list` `/model/detail` `/model/del` | Pod 信息 + 模型名 | 返回模型集合或“删除成功” |
| 聊天 / Embedding | `/ollama/chat` `/ollama/embeddings` | Pod + 模型 + 对话/文本 | 返回 LLM 答复或向量 |
| 知识库部署 | `POST /api/k8s/knowledge/deploy` | `kubeDto.KnowledgeDeployInput`（镜像、端口、绑定 Ollama 信息等） | `{"data":"部署成功"}` |
| 知识库列表/详情 | `GET /api/k8s/knowledge/list|detail` | 过滤条件或 name/namespace | 返回部署清单/详情 |
| 文档上传 | `POST /api/k8s/knowledge/document/upload` | form-data（Pod、知识库类型、文件…） | 返回入库结果 |
| 知识库查询 | `POST /api/k8s/knowledge/query` | Pod、collection、query_text、top_k | 返回相关文档列表 |
| AI Chat with KB | `POST /api/ai/chat_with_kb` | `ChatWithKBInput`（知识库参数 + Ollama 模型 + question） | 返回模型回答或流式内容 |

> 更完整的接口注释可见 `dao/model/init.go` 注册清单；Swagger 文档可通过 `swag init --pd -d ./cmd,docs` 生成并访问 `http://127.0.0.1:6180/swagger/index.html`。

## 配置说明

`config.yaml` 关键段落：

```yaml
default:
  listenAddr: ":6180"
  kubernetesConfigFile: "/path/to/kubeconfig"

mcp:
  enable: true
  implementationName: "kubemanage-mcp-client"
  # ...
mysql:
  host: "127.0.0.1"
  user: "root"
  password: "123456"
```

- 可通过环境变量 `KUBEMANAGE-CONFIG` 覆盖配置文件路径。
- 支持 InCluster 模式读取 ServiceAccount。
- MCP 段可配置默认工具 server，方便为 LLM/Agent 提供外部工具。

## 开发与贡献

### Issue
- 仅用于提交 Bug / Feature / 设计讨论，提问请先搜索是否已有。

### Pull Request
- fork 后新建分支，commit message 使用 `feat(module): desc` 或 `fix(module): desc`。
- 提交前运行 `go test ./...` 与 `swag init`。
- 至少两名维护者 review 通过后合并。

## Roadmap

- ✅ Ollama 模型管理
- ✅ 知识库部署、上传与查询
- 🕑 自动扩缩容策略
- 🕑 Agent 工作流编排

---

如需联系或集成更多 AI 场景，欢迎提 Issue / PR，一起建设面向企业的 Kubernetes AI 应用管理中心。
