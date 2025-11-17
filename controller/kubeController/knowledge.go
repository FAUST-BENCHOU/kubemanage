package kubeController

import (
	"io"

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

// UploadDocument 上传文档到知识库
// @Summary      上传文档到知识库
// @Description  向指定的知识库 Pod 上传文档文件，支持 ChromaDB、Milvus、Weaviate
// @Tags         knowledge
// @ID           /api/k8s/knowledge/document/upload
// @Accept       multipart/form-data
// @Produce      json
// @Param        pod_name        formData  string  true   "知识库Pod名称"
// @Param        namespace       formData  string  true   "命名空间"
// @Param        knowledge_type  formData  string  true   "知识库类型: chromadb, milvus, weaviate"
// @Param        file            formData  file    true   "文档文件"
// @Param        collection_name formData  string  false  "集合名称（可选）"
// @Param        chunk_size      formData  int     false  "分块大小（可选，默认1000）"
// @Success      200             {object}  middleware.Response"{"code": 200, msg="","data": object}"
// @Router       /api/k8s/knowledge/document/upload [post]
func (k *knowledge) UploadDocument(ctx *gin.Context) {
	params := &kubeDto.KnowledgeUploadDocumentInput{}
	if err := params.BindingValidParams(ctx); err != nil {
		v1.Log.ErrorWithCode(globalError.ParamBindError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.ParamBindError, err))
		return
	}

	file, err := ctx.FormFile("file")
	if err != nil {
		v1.Log.ErrorWithCode(globalError.ParamBindError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.ParamBindError, err))
		return
	}

	src, err := file.Open()
	if err != nil {
		v1.Log.ErrorWithCode(globalError.ParamBindError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.ParamBindError, err))
		return
	}
	defer src.Close()

	fileContent, err := io.ReadAll(src)
	if err != nil {
		v1.Log.ErrorWithCode(globalError.ParamBindError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.ParamBindError, err))
		return
	}

	collectionName := params.CollectionName
	if collectionName == "" {
		collectionName = file.Filename
	}

	data, err := kube.Knowledge.UploadDocument(
		params.PodName,
		params.NameSpace,
		params.KnowledgeType,
		fileContent,
		file.Filename,
		collectionName,
		params.ChunkSize,
	)
	if err != nil {
		v1.Log.ErrorWithCode(globalError.CreateError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.CreateError, err))
		return
	}
	middleware.ResponseSuccess(ctx, data)
}

// QueryDocument 查询知识库
// @Summary      查询知识库
// @Description  在指定的知识库中查询相似文档，支持 ChromaDB、Milvus、Weaviate
// @Tags         knowledge
// @ID           /api/k8s/knowledge/query
// @Accept       json
// @Produce      json
// @Param        body  body  kubeDto.KnowledgeQueryInput  true  "查询参数"
// @Success      200   {object}  middleware.Response"{"code": 200, msg="","data": object}"
// @Router       /api/k8s/knowledge/query [post]
func (k *knowledge) QueryDocument(ctx *gin.Context) {
	params := &kubeDto.KnowledgeQueryInput{}
	if err := params.BindingValidParams(ctx); err != nil {
		v1.Log.ErrorWithCode(globalError.ParamBindError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.ParamBindError, err))
		return
	}

	topK := params.TopK
	if topK <= 0 {
		topK = 5
	}

	data, err := kube.Knowledge.QueryKnowledge(
		params.PodName,
		params.NameSpace,
		params.KnowledgeType,
		params.CollectionName,
		params.QueryText,
		topK,
	)
	if err != nil {
		v1.Log.ErrorWithCode(globalError.CreateError, err)
		middleware.ResponseError(ctx, globalError.NewGlobalError(globalError.CreateError, err))
		return
	}
	middleware.ResponseSuccess(ctx, data)
}
