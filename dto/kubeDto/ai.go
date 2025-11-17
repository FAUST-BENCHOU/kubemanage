package kubeDto

import (
	"github.com/gin-gonic/gin"
	"github.com/noovertime7/kubemanage/pkg"
)

// ChatWithKBInput 结合知识库的聊天输入参数
type ChatWithKBInput struct {
	// 知识库相关参数
	KnowledgePodName   string `json:"knowledge_pod_name" form:"knowledge_pod_name" comment:"知识库Pod名称" validate:"required"`
	KnowledgeNamespace string `json:"knowledge_namespace" form:"knowledge_namespace" comment:"知识库命名空间" validate:"required"`
	KnowledgeType      string `json:"knowledge_type" form:"knowledge_type" comment:"知识库类型: chromadb, milvus, weaviate" validate:"required"`
	CollectionName     string `json:"collection_name" form:"collection_name" comment:"集合名称" validate:"required"`

	// Ollama 相关参数
	OllamaPodName   string `json:"ollama_pod_name" form:"ollama_pod_name" comment:"Ollama Pod名称" validate:"required"`
	OllamaNamespace string `json:"ollama_namespace" form:"ollama_namespace" comment:"Ollama命名空间" validate:"required"`
	OllamaModel     string `json:"ollama_model" form:"ollama_model" comment:"Ollama模型名称" validate:"required"`

	// 聊天相关参数
	Question string `json:"question" form:"question" comment:"用户问题" validate:"required"`
	TopK     int    `json:"top_k" form:"top_k" comment:"从知识库返回的相关文档数量（默认5）"`
	Stream   bool   `json:"stream" form:"stream" comment:"是否流式返回"`

	// 可选参数
	SystemPrompt string `json:"system_prompt" form:"system_prompt" comment:"自定义系统提示词（可选）"`
}

func (params *ChatWithKBInput) BindingValidParams(c *gin.Context) error {
	return pkg.DefaultGetValidParams(c, params)
}
