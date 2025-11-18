package kubeDto

import (
	"github.com/gin-gonic/gin"

	"github.com/noovertime7/kubemanage/pkg"
)

type MCPListToolsInput struct {
	ServerName string `json:"server_name" form:"server_name" comment:"MCP server 名称" validate:"required"`
}

func (params *MCPListToolsInput) BindingValidParams(c *gin.Context) error {
	return pkg.DefaultGetValidParams(c, params)
}

type MCPCreateServerInput struct {
	Name        string            `json:"name" comment:"服务器唯一名称" validate:"required"`
	DisplayName string            `json:"display_name" comment:"展示名称"`
	Description string            `json:"description" comment:"描述"`
	Image       string            `json:"image" comment:"容器镜像"`
	Command     string            `json:"command" comment:"启动命令"`
	Args        []string          `json:"args" comment:"启动参数"`
	Env         map[string]string `json:"env" comment:"环境变量"`
	DefaultTool string            `json:"default_tool" comment:"默认工具名"`
	Homepage    string            `json:"homepage" comment:"文档地址"`
	Tags        []string          `json:"tags" comment:"标签"`
	SetDefault  bool              `json:"set_default" comment:"是否设为默认 server"`
}

func (params *MCPCreateServerInput) BindingValidParams(c *gin.Context) error {
	return pkg.DefaultGetValidParams(c, params)
}
