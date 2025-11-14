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
