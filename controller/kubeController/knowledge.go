package kubeController

import (
	"github.com/gin-gonic/gin"
	"github.com/noovertime7/kubemanage/dto/kubeDto"
	"github.com/noovertime7/kubemanage/middleware"
	v1 "github.com/noovertime7/kubemanage/pkg/core/kubemanage/v1"
	"github.com/noovertime7/kubemanage/pkg/core/kubemanage/v1/kube"
	"github.com/noovertime7/kubemanage/pkg/globalError"
)

var Knowledge knowledge

type knowledge struct{}

// DeployKnowledge 部署知识库
// @Summary      部署知识库到指定节点
// @Description  在K8s集群的指定节点上部署知识库服务，支持绑定Ollama模型
// @Tags         knowledge
// @ID           /api/k8s/knowledge/deploy
// @Accept       json
// @Produce      json
// @Param        body  body  kubeDto.KnowledgeDeployInput  true  "部署参数"
// @Success      200   {object}  middleware.Response"{"code": 200, msg="","data": "部署成功}"
// @Router       /api/k8s/knowledge/deploy [post]
func (k *knowledge) DeployKnowledge(ctx *gin.Context) {
	params := &kubeDto.KnowledgeDeployInput{}
	if err := params.BindingValidParams(ctx); err != nil {
		v1.Log.ErrorWithCode(globalError.ParamBindError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.ParamBindError, err))
		return
	}
	if err := kube.Knowledge.DeployKnowledge(params); err != nil {
		v1.Log.ErrorWithCode(globalError.CreateError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.CreateError, err))
		return
	}
	middleware.ResponseSuccess(ctx, "部署成功")
}
