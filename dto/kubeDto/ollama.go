package kubeDto

import (
	"github.com/gin-gonic/gin"
	"github.com/noovertime7/kubemanage/pkg"
)

// OllamaDeployInput Ollama 部署输入参数
type OllamaDeployInput struct {
	Name         string            `json:"name" form:"name" comment:"部署名称" validate:"required"`
	NameSpace    string            `json:"namespace" form:"namespace" comment:"命名空间" validate:"required"`
	Image        string            `json:"image" form:"image" comment:"Ollama镜像" validate:"required"`
	Port         int32             `json:"port" form:"port" comment:"服务端口" validate:"required"`
	NodeSelector map[string]string `json:"node_selector" form:"node_selector" comment:"节点选择器"`
	Labels       map[string]string `json:"labels" form:"labels" comment:"标签"`
	Cpu          string            `json:"cpu" form:"cpu" comment:"CPU限制"`
	Memory       string            `json:"memory" form:"memory" comment:"内存限制"`
	StorageSize  string            `json:"storage_size" form:"storage_size" comment:"存储大小"`
	StorageClass string            `json:"storage_class" form:"storage_class" comment:"存储类"`
	DeployType   string            `json:"deploy_type" form:"deploy_type" comment:"部署类型: deployment 或 daemonset" validate:"required"`
}

// OllamaListInput Ollama 列表查询参数
type OllamaListInput struct {
	FilterName string `json:"filter_name" form:"filter_name" validate:"" comment:"过滤名"`
	NameSpace  string `json:"namespace" form:"namespace" validate:"" comment:"命名空间"`
	NodeName   string `json:"node_name" form:"node_name" validate:"" comment:"节点名称"`
	Limit      int    `json:"limit" form:"limit" validate:"" comment:"分页限制"`
	Page       int    `json:"page" form:"page" validate:"" comment:"页码"`
}

// OllamaNameNS Ollama 名称和命名空间
type OllamaNameNS struct {
	Name      string `json:"name" form:"name" comment:"Ollama部署名称" validate:"required"`
	NameSpace string `json:"namespace" form:"namespace" comment:"命名空间" validate:"required"`
}

// OllamaPullModelInput Ollama 拉取模型输入参数
type OllamaPullModelInput struct {
	PodName   string `json:"pod_name" form:"pod_name" comment:"Pod名称" validate:"required"`
	NameSpace string `json:"namespace" form:"namespace" comment:"命名空间" validate:"required"`
	ModelName string `json:"model_name" form:"model_name" comment:"模型名称" validate:"required"`
}

// OllamaModelListInput Ollama 模型列表查询参数
type OllamaModelListInput struct {
	PodName   string `json:"pod_name" form:"pod_name" comment:"Pod名称" validate:"required"`
	NameSpace string `json:"namespace" form:"namespace" comment:"命名空间" validate:"required"`
}

// OllamaDeleteModelInput Ollama 删除模型输入参数
type OllamaDeleteModelInput struct {
	PodName   string `json:"pod_name" form:"pod_name" comment:"Pod名称" validate:"required"`
	NameSpace string `json:"namespace" form:"namespace" comment:"命名空间" validate:"required"`
	ModelName string `json:"model_name" form:"model_name" comment:"模型名称" validate:"required"`
}

// OllamaModelDetailInput Ollama 模型详情查询参数
type OllamaModelDetailInput struct {
	PodName   string `json:"pod_name" form:"pod_name" comment:"Pod名称" validate:"required"`
	NameSpace string `json:"namespace" form:"namespace" comment:"命名空间" validate:"required"`
	ModelName string `json:"model_name" form:"model_name" comment:"模型名称" validate:"required"`
}

// OllamaChatMessage Ollama 聊天消息
type OllamaChatMessage struct {
	Role    string `json:"role" comment:"角色: user, assistant, system" validate:"required"`
	Content string `json:"content" comment:"消息内容" validate:"required"`
}

// OllamaChatInput Ollama 聊天输入参数
type OllamaChatInput struct {
	PodName   string              `json:"pod_name" form:"pod_name" comment:"Pod名称" validate:"required"`
	NameSpace string              `json:"namespace" form:"namespace" comment:"命名空间" validate:"required"`
	Model     string              `json:"model" form:"model" comment:"模型名称" validate:"required"`
	Messages  []OllamaChatMessage `json:"messages" form:"messages" comment:"消息列表" validate:"required,min=1"`
	Stream    bool                `json:"stream" form:"stream" comment:"是否流式返回"`
}

// OllamaEmbeddingsInput Ollama 向量嵌入输入参数
type OllamaEmbeddingsInput struct {
	PodName   string `json:"pod_name" form:"pod_name" comment:"Pod名称" validate:"required"`
	NameSpace string `json:"namespace" form:"namespace" comment:"命名空间" validate:"required"`
	Model     string `json:"model" form:"model" comment:"模型名称" validate:"required"`
	Prompt    string `json:"prompt" form:"prompt" comment:"要嵌入的文本" validate:"required"`
}

func (params *OllamaDeployInput) BindingValidParams(c *gin.Context) error {
	return pkg.DefaultGetValidParams(c, params)
}

func (params *OllamaListInput) BindingValidParams(c *gin.Context) error {
	return pkg.DefaultGetValidParams(c, params)
}

func (params *OllamaNameNS) BindingValidParams(c *gin.Context) error {
	return pkg.DefaultGetValidParams(c, params)
}

func (params *OllamaPullModelInput) BindingValidParams(c *gin.Context) error {
	return pkg.DefaultGetValidParams(c, params)
}

func (params *OllamaModelListInput) BindingValidParams(c *gin.Context) error {
	return pkg.DefaultGetValidParams(c, params)
}

func (params *OllamaDeleteModelInput) BindingValidParams(c *gin.Context) error {
	return pkg.DefaultGetValidParams(c, params)
}

func (params *OllamaModelDetailInput) BindingValidParams(c *gin.Context) error {
	return pkg.DefaultGetValidParams(c, params)
}

func (params *OllamaChatInput) BindingValidParams(c *gin.Context) error {
	return pkg.DefaultGetValidParams(c, params)
}

func (params *OllamaEmbeddingsInput) BindingValidParams(c *gin.Context) error {
	return pkg.DefaultGetValidParams(c, params)
}
