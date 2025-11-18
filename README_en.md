English | [简体中文](./README.md)

# Kubernetes AI Application Control Center

This project turns a Kubernetes cluster into an AI application hub. It provides a unified control plane to deploy Ollama models, run knowledge bases, register MCP servers, and compose RAG-style services. Backend is powered by gin + gorm + client-go; the UI (repo: [kubemanage-web](https://github.com/noovertime7/kubemanage-web)) is built with Vue3/Vite.

## Highlights

- **Ollama lifecycle** – deploy models, pull weights, inspect pods, run chat & embedding APIs.
- **Knowledge base toolkit** – spin up ChromaDB / Milvus / Weaviate instances, upload documents, run semantic queries.
- **MCP integration** – register Model Context Protocol servers so agents can invoke external tools.
- **AI scenario orchestration** – `/api/ai/chat_with_kb` combines knowledge retrieval with LLM answers for enterprise RAG.
- **Platform features** – RBAC, CMDB, auditing, workflow, etc. remain available for ops teams.

## Architecture at a Glance

| Layer | Description |
| ----- | ----------- |
| API Gateway | Gin services, Casbin-based RBAC, swagger doc generation |
| Cluster adapter | client-go to create Deployment/Service/PVC for AI workloads |
| Data plane | MySQL for accounts / CMDB / audit; Kubernetes for AI pods |
| Front-end | Vue3 + Pinia + Vite (separate repo) |

## Quick Start

### Prerequisites
- Go 1.20+, Node.js 16+
- Accessible Kubernetes cluster (minikube/kind/production)
- MySQL (`kubemanage` database)
- kubeconfig path or InCluster permissions

### Database
```sql
CREATE DATABASE kubemanage CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
```

### Backend
```bash
git clone https://github.com/noovertime7/kubemanage.git
cd kubemanage
go mod tidy

# provide config via env KUBEMANAGE-CONFIG or --configFile
go run cmd/main.go --configFile=config.yaml
```
Default credentials: `admin / kubemanage`

### Frontend
```bash
git clone https://github.com/noovertime7/kubemanage-web.git
cd kubemanage-web
npm install
npm run dev
```

## AI Module Reference

| Module | Endpoint | Inputs | Successful response |
| ------ | -------- | ------ | ------------------- |
| Ollama deploy | `POST /api/k8s/ollama/deploy` | `kubeDto.OllamaDeployInput` (name, namespace, image, port, nodeSelector, etc.) | `{"code":200,"data":"部署成功"}` |
| Ollama list | `GET /api/k8s/ollama/list` | `filter_name/namespace/node_name/page/limit` | `{"data":{"total":n,"items":[...]}}` |
| Pull model | `POST /api/k8s/ollama/model/pull` | `pod_name/namespace/model_name` | `{"data":"拉取成功"}` |
| Model list/detail/delete | `/model/list|detail|del` | Pod & model identifiers | object payload or `"删除成功"` |
| Chat / Embedding | `/ollama/chat`, `/ollama/embeddings` | Pod + model + chat/ prompt payload | answer text or vector |
| Knowledge deploy | `POST /api/k8s/knowledge/deploy` | `KnowledgeDeployInput` (Ollama binding optional) | `{"data":"部署成功"}` |
| Knowledge list/detail | `GET /api/k8s/knowledge/list|detail` | filters or name/namespace | deployments or detailed spec |
| Document upload | `POST /api/k8s/knowledge/document/upload` | multipart form data + file | ingestion result |
| Knowledge query | `POST /api/k8s/knowledge/query` | `KnowledgeQueryInput` (collection, text, top_k) | relevant document array |
| AI chat with KB | `POST /api/ai/chat_with_kb` | `ChatWithKBInput` (knowledge & Ollama params + question) | RAG answer or streaming chunks |

Swagger:
```bash
swag init --pd -d ./cmd,docs
# visit http://127.0.0.1:6180/swagger/index.html
```

## Configuration Notes

Key sections in `config.yaml`:

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

- Configuration priority: default < env `KUBEMANAGE-CONFIG` < CLI `--configFile`.
- Support InCluster usage when running inside Kubernetes.
- MCP section lets you pre-register tool servers for agents.

## Contribution Guide

- **Issues** – focus on bugs, features, or design proposals.
- **Pull Requests** – fork, create branch, use `feat(module): desc` commit messages, include tests/description.
- **Review** – at least two maintainers review & approve.

## Roadmap

- ✅ Ollama lifecycle management
- ✅ Knowledge base deployment + RAG APIs
- ✅ MCP tool registration
- 🕑 Multi-cluster federation
- 🕑 Auto scaling policies for AI pods
- 🕑 Agent workflow templates

---

欢迎提交 issues / PRs to push the Kubernetes AI Application Control Center forward together.