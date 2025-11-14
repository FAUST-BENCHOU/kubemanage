package kubeController

import (
	"github.com/gin-gonic/gin"
	"github.com/noovertime7/kubemanage/dto/kubeDto"
	"github.com/noovertime7/kubemanage/middleware"
	v1 "github.com/noovertime7/kubemanage/pkg/core/kubemanage/v1"
	"github.com/noovertime7/kubemanage/pkg/core/kubemanage/v1/kube"
	"github.com/noovertime7/kubemanage/pkg/globalError"
)

var Ollama ollama

type ollama struct{}

// DeployOllama 部署Ollama
// ListPage godoc
// @Summary      部署Ollama到指定节点
// @Description  在K8s集群的指定节点上部署Ollama服务
// @Tags         ollama
// @ID           /api/k8s/ollama/deploy
// @Accept       json
// @Produce      json
// @Param        body  body  kubeDto.OllamaDeployInput  true  "部署参数"
// @Success       200  {object}  middleware.Response"{"code": 200, msg="","data": "部署成功}"
// @Router       /api/k8s/ollama/deploy [post]
func (o *ollama) DeployOllama(ctx *gin.Context) {
	params := &kubeDto.OllamaDeployInput{}
	if err := params.BindingValidParams(ctx); err != nil {
		v1.Log.ErrorWithCode(globalError.ParamBindError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.ParamBindError, err))
		return
	}
	if err := kube.Ollama.DeployOllama(params); err != nil {
		v1.Log.ErrorWithCode(globalError.CreateError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.CreateError, err))
		return
	}
	middleware.ResponseSuccess(ctx, "部署成功")
}

// GetOllamaList 获取Ollama部署列表
// ListPage godoc
// @Summary      获取Ollama部署列表
// @Description  获取Ollama部署列表，支持分页和过滤
// @Tags         ollama
// @ID           /api/k8s/ollama/list
// @Accept       json
// @Produce      json
// @Param        filter_name  query  string  false  "过滤名"
// @Param        namespace    query  string  false  "命名空间"
// @Param        node_name    query  string  false  "节点名称"
// @Param        page         query  int     false  "页码"
// @Param        limit        query  int     false  "分页限制"
// @Success      200 {object}  middleware.Response"{"code": 200, msg="","data": []}"
// @Router       /api/k8s/ollama/list [get]
func (o *ollama) GetOllamaList(ctx *gin.Context) {
	params := &kubeDto.OllamaListInput{}
	if err := params.BindingValidParams(ctx); err != nil {
		v1.Log.ErrorWithCode(globalError.ParamBindError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.ParamBindError, err))
		return
	}
	data, err := kube.Ollama.GetOllamaList(params.FilterName, params.NameSpace, params.NodeName, params.Limit, params.Page)
	if err != nil {
		v1.Log.ErrorWithCode(globalError.GetError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.GetError, err))
		return
	}
	middleware.ResponseSuccess(ctx, data)
}
