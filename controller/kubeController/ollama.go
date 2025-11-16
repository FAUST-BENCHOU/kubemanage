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

// PullModel 拉取模型到指定的 Pod
// @Summary      拉取模型到指定的 Pod
// @Description  在指定的 Pod 中拉取模型
// @Tags         ollama
// @ID           /api/k8s/ollama/model/pull
// @Accept       json
// @Produce      json
// @Param        body  body  kubeDto.OllamaPullModelInput  true  "拉取模型参数"
// @Success      200   {object}  middleware.Response"{"code": 200, msg="","data": "拉取成功}"
// @Router       /api/k8s/ollama/model/pull [post]
func (o *ollama) PullModel(ctx *gin.Context) {
	params := &kubeDto.OllamaPullModelInput{}
	if err := params.BindingValidParams(ctx); err != nil {
		v1.Log.ErrorWithCode(globalError.ParamBindError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.ParamBindError, err))
		return
	}
	if err := kube.Ollama.PullModel(params.PodName, params.NameSpace, params.ModelName); err != nil {
		v1.Log.ErrorWithCode(globalError.CreateError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.CreateError, err))
		return
	}
	middleware.ResponseSuccess(ctx, "拉取成功")
}

// GetModelList 获取指定 Pod 的模型列表
// @Summary      获取指定 Pod 的模型列表
// @Description  获取指定 Pod 中已安装的模型列表
// @Tags         ollama
// @ID           /api/k8s/ollama/model/list
// @Accept       json
// @Produce      json
// @Param        pod_name   query  string  true  "Pod名称"
// @Param        namespace  query  string  true  "命名空间"
// @Success      200        {object}  middleware.Response"{"code": 200, msg="","data": object}"
// @Router       /api/k8s/ollama/model/list [get]
func (o *ollama) GetModelList(ctx *gin.Context) {
	params := &kubeDto.OllamaModelListInput{}
	if err := params.BindingValidParams(ctx); err != nil {
		v1.Log.ErrorWithCode(globalError.ParamBindError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.ParamBindError, err))
		return
	}
	data, err := kube.Ollama.GetModelList(params.PodName, params.NameSpace)
	if err != nil {
		v1.Log.ErrorWithCode(globalError.GetError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.GetError, err))
		return
	}
	middleware.ResponseSuccess(ctx, data)
}

// DeleteModel 删除指定 Pod 中的模型
// @Summary      删除指定 Pod 中的模型
// @Description  删除指定 Pod 中的 Ollama 模型
// @Tags         ollama
// @ID           /api/k8s/ollama/model/del
// @Accept       json
// @Produce      json
// @Param        pod_name   query  string  true  "Pod名称"
// @Param        namespace  query  string  true  "命名空间"
// @Param        model_name query  string  true  "模型名称"
// @Success      200        {object}  middleware.Response"{"code": 200, msg="","data": "删除成功}"
// @Router       /api/k8s/ollama/model/del [delete]
func (o *ollama) DeleteModel(ctx *gin.Context) {
	params := &kubeDto.OllamaDeleteModelInput{}
	if err := params.BindingValidParams(ctx); err != nil {
		v1.Log.ErrorWithCode(globalError.ParamBindError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.ParamBindError, err))
		return
	}
	if err := kube.Ollama.DeleteModel(params.PodName, params.NameSpace, params.ModelName); err != nil {
		v1.Log.ErrorWithCode(globalError.DeleteError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.DeleteError, err))
		return
	}
	middleware.ResponseSuccess(ctx, "删除成功")
}

// GetModelDetail 获取指定 Pod 中模型的详情
// @Summary      获取指定 Pod 中模型的详情
// @Description  获取指定 Pod 中 Ollama 模型的详细信息
// @Tags         ollama
// @ID           /api/k8s/ollama/model/detail
// @Accept       json
// @Produce      json
// @Param        pod_name   query  string  true  "Pod名称"
// @Param        namespace  query  string  true  "命名空间"
// @Param        model_name query  string  true  "模型名称"
// @Success      200        {object}  middleware.Response"{"code": 200, msg="","data": object}"
// @Router       /api/k8s/ollama/model/detail [get]
func (o *ollama) GetModelDetail(ctx *gin.Context) {
	params := &kubeDto.OllamaModelDetailInput{}
	if err := params.BindingValidParams(ctx); err != nil {
		v1.Log.ErrorWithCode(globalError.ParamBindError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.ParamBindError, err))
		return
	}
	data, err := kube.Ollama.GetModelDetail(params.PodName, params.NameSpace, params.ModelName)
	if err != nil {
		v1.Log.ErrorWithCode(globalError.GetError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.GetError, err))
		return
	}
	middleware.ResponseSuccess(ctx, data)
}

// Chat 调用指定 Pod 上的模型进行聊天
// @Summary      调用指定 Pod 上的模型进行聊天
// @Description  调用指定 Pod 上的 Ollama 模型进行对话
// @Tags         ollama
// @ID           /api/k8s/ollama/chat
// @Accept       json
// @Produce      json
// @Param        body  body  kubeDto.OllamaChatInput  true  "聊天参数"
// @Success      200   {object}  middleware.Response"{"code": 200, msg="","data": object}"
// @Router       /api/k8s/ollama/chat [post]
func (o *ollama) Chat(ctx *gin.Context) {
	params := &kubeDto.OllamaChatInput{}
	if err := params.BindingValidParams(ctx); err != nil {
		v1.Log.ErrorWithCode(globalError.ParamBindError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.ParamBindError, err))
		return
	}
	data, err := kube.Ollama.Chat(params.PodName, params.NameSpace, params.Model, params.Messages, params.Stream)
	if err != nil {
		v1.Log.ErrorWithCode(globalError.GetError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.GetError, err))
		return
	}
	middleware.ResponseSuccess(ctx, data)
}

// Embeddings 调用指定 Pod 上的模型生成文本向量嵌入
// @Summary      调用指定 Pod 上的模型生成文本向量嵌入
// @Description  调用指定 Pod 上的 Ollama 模型生成文本的向量嵌入
// @Tags         ollama
// @ID           /api/k8s/ollama/embeddings
// @Accept       json
// @Produce      json
// @Param        body  body  kubeDto.OllamaEmbeddingsInput  true  "向量嵌入参数"
// @Success      200   {object}  middleware.Response"{"code": 200, msg="","data": object}"
// @Router       /api/k8s/ollama/embeddings [post]
func (o *ollama) Embeddings(ctx *gin.Context) {
	params := &kubeDto.OllamaEmbeddingsInput{}
	if err := params.BindingValidParams(ctx); err != nil {
		v1.Log.ErrorWithCode(globalError.ParamBindError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.ParamBindError, err))
		return
	}
	data, err := kube.Ollama.Embeddings(params.PodName, params.NameSpace, params.Model, params.Prompt)
	if err != nil {
		v1.Log.ErrorWithCode(globalError.GetError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.GetError, err))
		return
	}
	middleware.ResponseSuccess(ctx, data)
}
