package kubeController

import (
	"github.com/gin-gonic/gin"
	"github.com/noovertime7/kubemanage/dto/kubeDto"
	"github.com/noovertime7/kubemanage/middleware"
	v1 "github.com/noovertime7/kubemanage/pkg/core/kubemanage/v1"
	"github.com/noovertime7/kubemanage/pkg/core/kubemanage/v1/kube"
	"github.com/noovertime7/kubemanage/pkg/globalError"
)

var AI ai

type ai struct{}

// ChatWithKB 结合知识库进行聊天
// @Summary      结合知识库进行聊天
// @Description  查询知识库获取相关文档，然后使用模型基于文档内容回答问题
// @Tags         ai
// @ID           /api/ai/chat_with_kb
// @Accept       json
// @Produce      json
// @Param        body  body  kubeDto.ChatWithKBInput  true  "聊天参数"
// @Success      200   {object}  middleware.Response"{"code": 200, msg="","data": object}"
// @Router       /api/ai/chat_with_kb [post]
func (a *ai) ChatWithKB(ctx *gin.Context) {
	params := &kubeDto.ChatWithKBInput{}
	if err := params.BindingValidParams(ctx); err != nil {
		v1.Log.ErrorWithCode(globalError.ParamBindError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.ParamBindError, err))
		return
	}

	data, err := kube.Knowledge.ChatWithKnowledgeBase(params)
	if err != nil {
		v1.Log.ErrorWithCode(globalError.GetError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.GetError, err))
		return
	}
	middleware.ResponseSuccess(ctx, data)
}
