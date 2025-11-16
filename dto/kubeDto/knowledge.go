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

func (params *KnowledgeDeployInput) BindingValidParams(c *gin.Context) error {
	return pkg.DefaultGetValidParams(c, params)
}
