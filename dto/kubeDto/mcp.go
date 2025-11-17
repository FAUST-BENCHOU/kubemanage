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
