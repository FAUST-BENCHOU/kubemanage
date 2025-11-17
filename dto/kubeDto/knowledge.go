package kubeDto

import (
	"github.com/gin-gonic/gin"
	"github.com/noovertime7/kubemanage/pkg"
)

// KnowledgeDeployInput 知识库部署输入参数
type KnowledgeDeployInput struct {
	Name            string            `json:"name" form:"name" comment:"部署名称" validate:"required"`
	NameSpace       string            `json:"namespace" form:"namespace" comment:"命名空间" validate:"required"`
	Image           string            `json:"image" form:"image" comment:"知识库镜像" validate:"required"`
	Port            int32             `json:"port" form:"port" comment:"服务端口" validate:"required"`
	NodeSelector    map[string]string `json:"node_selector" form:"node_selector" comment:"节点选择器"`
	Labels          map[string]string `json:"labels" form:"labels" comment:"标签"`
	Cpu             string            `json:"cpu" form:"cpu" comment:"CPU限制"`
	Memory          string            `json:"memory" form:"memory" comment:"内存限制"`
	StorageSize     string            `json:"storage_size" form:"storage_size" comment:"存储大小"`
	StorageClass    string            `json:"storage_class" form:"storage_class" comment:"存储类"`
	OllamaPodName   string            `json:"ollama_pod_name" form:"ollama_pod_name" comment:"绑定的Ollama Pod名称"`
	OllamaModel     string            `json:"ollama_model" form:"ollama_model" comment:"绑定的模型名称"`
	OllamaNamespace string            `json:"ollama_namespace" form:"ollama_namespace" comment:"Ollama Pod所在命名空间"`
	DeployType      string            `json:"deploy_type" form:"deploy_type" comment:"部署类型: deployment 或 daemonset" validate:"required"`
}

// KnowledgeUploadDocumentInput 知识库上传文档输入参数
type KnowledgeUploadDocumentInput struct {
	PodName        string `form:"pod_name" comment:"知识库Pod名称" validate:"required"`
	NameSpace      string `form:"namespace" comment:"命名空间" validate:"required"`
	KnowledgeType  string `form:"knowledge_type" comment:"知识库类型: chromadb, milvus, weaviate" validate:"required"`
	CollectionName string `form:"collection_name" comment:"集合名称（可选，默认使用文件名）"`
	ChunkSize      int    `form:"chunk_size" comment:"分块大小（可选，默认1000）"`
}

// KnowledgeQueryInput 知识库查询输入参数
type KnowledgeQueryInput struct {
	PodName        string `json:"pod_name" form:"pod_name" comment:"知识库Pod名称" validate:"required"`
	NameSpace      string `json:"namespace" form:"namespace" comment:"命名空间" validate:"required"`
	KnowledgeType  string `json:"knowledge_type" form:"knowledge_type" comment:"知识库类型: chromadb, milvus, weaviate" validate:"required"`
	CollectionName string `json:"collection_name" form:"collection_name" comment:"集合名称" validate:"required"`
	QueryText      string `json:"query_text" form:"query_text" comment:"查询文本" validate:"required"`
	TopK           int    `json:"top_k" form:"top_k" comment:"返回结果数量（可选，默认5）"`
}

func (params *KnowledgeDeployInput) BindingValidParams(c *gin.Context) error {
	return pkg.DefaultGetValidParams(c, params)
}

func (params *KnowledgeUploadDocumentInput) BindingValidParams(c *gin.Context) error {
	return pkg.DefaultGetValidParams(c, params)
}

func (params *KnowledgeQueryInput) BindingValidParams(c *gin.Context) error {
	return pkg.DefaultGetValidParams(c, params)
}
