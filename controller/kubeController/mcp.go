package kubeController

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/noovertime7/kubemanage/dto/kubeDto"
	"github.com/noovertime7/kubemanage/middleware"
	v1 "github.com/noovertime7/kubemanage/pkg/core/kubemanage/v1"
	"github.com/noovertime7/kubemanage/pkg/globalError"
	"github.com/noovertime7/kubemanage/pkg/mcpclient"
)

var MCPServer mcpController

type mcpController struct{}

// ListServers 列出可用的 MCP Server 模板
func (m *mcpController) ListServers(ctx *gin.Context) {
	list := mcpclient.ListServers()
	if list == nil {
		list = []mcpclient.ServerMeta{}
	}
	middleware.ResponseSuccess(ctx, list)
}

// ListServerTools 列出指定 MCP Server 暴露的工具
func (m *mcpController) ListServerTools(ctx *gin.Context) {
	params := &kubeDto.MCPListToolsInput{}
	if err := params.BindingValidParams(ctx); err != nil {
		v1.Log.ErrorWithCode(globalError.ParamBindError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.ParamBindError, err))
		return
	}

	client, err := mcpclient.ClientByName(params.ServerName)
	if err != nil {
		v1.Log.ErrorWithErr("获取 MCP 客户端失败", err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.GetError, err))
		return
	}

	result, err := client.ListTools(ctx.Request.Context())
	if err != nil {
		v1.Log.ErrorWithErr("获取 MCP 工具列表失败", err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.GetError, err))
		return
	}
	middleware.ResponseSuccess(ctx, result.Tools)
}

// CreateServer 动态注册一个新的 MCP server
func (m *mcpController) CreateServer(ctx *gin.Context) {
	params := &kubeDto.MCPCreateServerInput{}
	if err := params.BindingValidParams(ctx); err != nil {
		v1.Log.ErrorWithCode(globalError.ParamBindError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.ParamBindError, err))
		return
	}

	manager := mcpclient.DefaultManager()
	if manager == nil {
		err := fmt.Errorf("MCP 功能未启用")
		v1.Log.ErrorWithErr("创建 MCP server 失败", err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.GetError, err))
		return
	}

	opts := &mcpclient.CreateServerOptions{
		Name:        params.Name,
		DisplayName: params.DisplayName,
		Description: params.Description,
		Image:       params.Image,
		Command:     params.Command,
		Args:        append([]string(nil), params.Args...),
		Env:         params.Env,
		DefaultTool: params.DefaultTool,
		Homepage:    params.Homepage,
		Tags:        append([]string(nil), params.Tags...),
		SetDefault:  params.SetDefault,
	}

	meta, err := manager.AddServer(opts)
	if err != nil {
		v1.Log.ErrorWithErr("创建 MCP server 失败", err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.CreateError, err))
		return
	}

	middleware.ResponseSuccess(ctx, meta)
}
