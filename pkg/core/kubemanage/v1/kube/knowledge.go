package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/noovertime7/kubemanage/dto/kubeDto"
)

var Knowledge knowledge

type knowledge struct{}

// DeployKnowledge 部署知识库到指定节点
func (k *knowledge) DeployKnowledge(data *kubeDto.KnowledgeDeployInput) error {
	// 设置默认标签
	labels := map[string]string{
		"app":     "knowledge",
		"name":    data.Name,
		"managed": "kubemanage",
	}
	// 合并用户提供的标签
	for k, v := range data.Labels {
		labels[k] = v
	}

	// 创建 PVC（如果需要存储）
	if data.StorageSize != "" {
		if err := k.createPVC(data); err != nil {
			return fmt.Errorf("创建PVC失败: %v", err)
		}
	}

	// 创建 Service
	if err := k.createService(data, labels); err != nil {
		return fmt.Errorf("创建Service失败: %v", err)
	}

	// 根据部署类型创建 Deployment 或 DaemonSet
	if data.DeployType == "daemonset" {
		return k.createDaemonSet(data, labels)
	}
	return k.createDeployment(data, labels)
}

// createDeployment 创建 Deployment
func (k *knowledge) createDeployment(data *kubeDto.KnowledgeDeployInput, labels map[string]string) error {
	replicas := int32(1)
	deployment := &appsV1.Deployment{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      data.Name,
			Namespace: data.NameSpace,
			Labels:    labels,
		},
		Spec: appsV1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metaV1.LabelSelector{
				MatchLabels: labels,
			},
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: labels,
				},
				Spec: coreV1.PodSpec{
					Containers: []coreV1.Container{
						{
							Name:            "knowledge",
							Image:           data.Image,
							ImagePullPolicy: coreV1.PullIfNotPresent,
							Ports: []coreV1.ContainerPort{
								{
									Name:          "http",
									Protocol:      coreV1.ProtocolTCP,
									ContainerPort: data.Port,
								},
							},
						},
					},
				},
			},
		},
	}

	// 设置节点选择器
	if len(data.NodeSelector) > 0 {
		deployment.Spec.Template.Spec.NodeSelector = data.NodeSelector
	}

	// 设置资源限制（可选，如果不指定则不设置资源限制，让 Kubernetes 自动分配）
	if data.Cpu != "" || data.Memory != "" {
		deployment.Spec.Template.Spec.Containers[0].Resources = coreV1.ResourceRequirements{}
		if data.Cpu != "" {
			deployment.Spec.Template.Spec.Containers[0].Resources.Limits = coreV1.ResourceList{
				coreV1.ResourceCPU: resource.MustParse(data.Cpu),
			}
			deployment.Spec.Template.Spec.Containers[0].Resources.Requests = coreV1.ResourceList{
				coreV1.ResourceCPU: resource.MustParse(data.Cpu),
			}
		}
		if data.Memory != "" {
			if deployment.Spec.Template.Spec.Containers[0].Resources.Limits == nil {
				deployment.Spec.Template.Spec.Containers[0].Resources.Limits = coreV1.ResourceList{}
			}
			if deployment.Spec.Template.Spec.Containers[0].Resources.Requests == nil {
				deployment.Spec.Template.Spec.Containers[0].Resources.Requests = coreV1.ResourceList{}
			}
			deployment.Spec.Template.Spec.Containers[0].Resources.Limits[coreV1.ResourceMemory] = resource.MustParse(data.Memory)
			deployment.Spec.Template.Spec.Containers[0].Resources.Requests[coreV1.ResourceMemory] = resource.MustParse(data.Memory)
		}
	}
	// 如果没有指定资源，不设置资源限制，让 Kubernetes 使用默认值（通常更宽松）

	// 设置存储卷
	if data.StorageSize != "" {
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = []coreV1.VolumeMount{
			{
				Name:      "knowledge-data",
				MountPath: "/data",
			},
		}
		deployment.Spec.Template.Spec.Volumes = []coreV1.Volume{
			{
				Name: "knowledge-data",
				VolumeSource: coreV1.VolumeSource{
					PersistentVolumeClaim: &coreV1.PersistentVolumeClaimVolumeSource{
						ClaimName: fmt.Sprintf("%s-pvc", data.Name),
					},
				},
			},
		}
	}

	// 添加环境变量（如果绑定了 Ollama）
	if data.OllamaPodName != "" {
		ollamaNamespace := data.NameSpace
		if data.OllamaNamespace != "" {
			ollamaNamespace = data.OllamaNamespace
		}
		// 设置 Ollama 服务地址环境变量
		deployment.Spec.Template.Spec.Containers[0].Env = append(
			deployment.Spec.Template.Spec.Containers[0].Env,
			coreV1.EnvVar{
				Name:  "OLLAMA_POD_NAME",
				Value: data.OllamaPodName,
			},
			coreV1.EnvVar{
				Name:  "OLLAMA_NAMESPACE",
				Value: ollamaNamespace,
			},
		)
		if data.OllamaModel != "" {
			deployment.Spec.Template.Spec.Containers[0].Env = append(
				deployment.Spec.Template.Spec.Containers[0].Env,
				coreV1.EnvVar{
					Name:  "OLLAMA_MODEL",
					Value: data.OllamaModel,
				},
			)
		}
	}

	// 不添加健康检查，让 Pod 可以正常启动
	// 如果知识库服务需要健康检查，可以在部署后手动配置

	_, err := K8s.ClientSet.AppsV1().Deployments(data.NameSpace).Create(context.TODO(), deployment, metaV1.CreateOptions{})
	return err
}

// createDaemonSet 创建 DaemonSet
func (k *knowledge) createDaemonSet(data *kubeDto.KnowledgeDeployInput, labels map[string]string) error {
	daemonSet := &appsV1.DaemonSet{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      data.Name,
			Namespace: data.NameSpace,
			Labels:    labels,
		},
		Spec: appsV1.DaemonSetSpec{
			Selector: &metaV1.LabelSelector{
				MatchLabels: labels,
			},
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: labels,
				},
				Spec: coreV1.PodSpec{
					Containers: []coreV1.Container{
						{
							Name:            "knowledge",
							Image:           data.Image,
							ImagePullPolicy: coreV1.PullIfNotPresent,
							Ports: []coreV1.ContainerPort{
								{
									Name:          "http",
									Protocol:      coreV1.ProtocolTCP,
									ContainerPort: data.Port,
								},
							},
						},
					},
				},
			},
		},
	}

	// 设置节点选择器
	if len(data.NodeSelector) > 0 {
		daemonSet.Spec.Template.Spec.NodeSelector = data.NodeSelector
	}

	// 设置资源限制
	if data.Cpu != "" || data.Memory != "" {
		daemonSet.Spec.Template.Spec.Containers[0].Resources = coreV1.ResourceRequirements{}
		if data.Cpu != "" {
			daemonSet.Spec.Template.Spec.Containers[0].Resources.Limits = coreV1.ResourceList{
				coreV1.ResourceCPU: resource.MustParse(data.Cpu),
			}
			daemonSet.Spec.Template.Spec.Containers[0].Resources.Requests = coreV1.ResourceList{
				coreV1.ResourceCPU: resource.MustParse(data.Cpu),
			}
		}
		if data.Memory != "" {
			if daemonSet.Spec.Template.Spec.Containers[0].Resources.Limits == nil {
				daemonSet.Spec.Template.Spec.Containers[0].Resources.Limits = coreV1.ResourceList{}
			}
			if daemonSet.Spec.Template.Spec.Containers[0].Resources.Requests == nil {
				daemonSet.Spec.Template.Spec.Containers[0].Resources.Requests = coreV1.ResourceList{}
			}
			daemonSet.Spec.Template.Spec.Containers[0].Resources.Limits[coreV1.ResourceMemory] = resource.MustParse(data.Memory)
			daemonSet.Spec.Template.Spec.Containers[0].Resources.Requests[coreV1.ResourceMemory] = resource.MustParse(data.Memory)
		}
	}

	// 设置存储卷
	if data.StorageSize != "" {
		daemonSet.Spec.Template.Spec.Containers[0].VolumeMounts = []coreV1.VolumeMount{
			{
				Name:      "knowledge-data",
				MountPath: "/data",
			},
		}
		daemonSet.Spec.Template.Spec.Volumes = []coreV1.Volume{
			{
				Name: "knowledge-data",
				VolumeSource: coreV1.VolumeSource{
					PersistentVolumeClaim: &coreV1.PersistentVolumeClaimVolumeSource{
						ClaimName: fmt.Sprintf("%s-pvc", data.Name),
					},
				},
			},
		}
	}

	// 添加环境变量（如果绑定了 Ollama）
	if data.OllamaPodName != "" {
		ollamaNamespace := data.NameSpace
		if data.OllamaNamespace != "" {
			ollamaNamespace = data.OllamaNamespace
		}
		daemonSet.Spec.Template.Spec.Containers[0].Env = append(
			daemonSet.Spec.Template.Spec.Containers[0].Env,
			coreV1.EnvVar{
				Name:  "OLLAMA_POD_NAME",
				Value: data.OllamaPodName,
			},
			coreV1.EnvVar{
				Name:  "OLLAMA_NAMESPACE",
				Value: ollamaNamespace,
			},
		)
		if data.OllamaModel != "" {
			daemonSet.Spec.Template.Spec.Containers[0].Env = append(
				daemonSet.Spec.Template.Spec.Containers[0].Env,
				coreV1.EnvVar{
					Name:  "OLLAMA_MODEL",
					Value: data.OllamaModel,
				},
			)
		}
	}

	_, err := K8s.ClientSet.AppsV1().DaemonSets(data.NameSpace).Create(context.TODO(), daemonSet, metaV1.CreateOptions{})
	return err
}

// createService 创建 Service
func (k *knowledge) createService(data *kubeDto.KnowledgeDeployInput, labels map[string]string) error {
	service := &coreV1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fmt.Sprintf("%s-svc", data.Name),
			Namespace: data.NameSpace,
			Labels:    labels,
		},
		Spec: coreV1.ServiceSpec{
			Selector: labels,
			Ports: []coreV1.ServicePort{
				{
					Name:       "http",
					Port:       data.Port,
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: data.Port},
					Protocol:   coreV1.ProtocolTCP,
				},
			},
			Type: coreV1.ServiceTypeClusterIP,
		},
	}

	_, err := K8s.ClientSet.CoreV1().Services(data.NameSpace).Create(context.TODO(), service, metaV1.CreateOptions{})
	return err
}

// createPVC 创建 PVC
func (k *knowledge) createPVC(data *kubeDto.KnowledgeDeployInput) error {
	storageClass := data.StorageClass
	if storageClass == "" {
		storageClass = "" // 使用默认存储类
	}

	pvc := &coreV1.PersistentVolumeClaim{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fmt.Sprintf("%s-pvc", data.Name),
			Namespace: data.NameSpace,
			Labels: map[string]string{
				"app":     "knowledge",
				"name":    data.Name,
				"managed": "kubemanage",
			},
		},
		Spec: coreV1.PersistentVolumeClaimSpec{
			AccessModes: []coreV1.PersistentVolumeAccessMode{coreV1.ReadWriteOnce},
			Resources: coreV1.ResourceRequirements{
				Requests: coreV1.ResourceList{
					coreV1.ResourceStorage: resource.MustParse(data.StorageSize),
				},
			},
		},
	}

	if storageClass != "" {
		pvc.Spec.StorageClassName = &storageClass
	}

	_, err := K8s.ClientSet.CoreV1().PersistentVolumeClaims(data.NameSpace).Create(context.TODO(), pvc, metaV1.CreateOptions{})
	return err
}

// UploadDocument 上传文档到知识库（支持 ChromaDB、Milvus、Weaviate）
func (k *knowledge) UploadDocument(podName, namespace, knowledgeType string, fileContent []byte, fileName, collectionName string, chunkSize int) (interface{}, error) {
	// 根据知识库类型调用不同的上传方法
	knowledgeType = strings.ToLower(knowledgeType)
	switch knowledgeType {
	case "chromadb", "chroma":
		return k.uploadToChroma(podName, namespace, fileContent, fileName, collectionName, chunkSize)
	case "milvus":
		return k.uploadToMilvus(podName, namespace, fileContent, fileName, collectionName, chunkSize)
	case "weaviate":
		return k.uploadToWeaviate(podName, namespace, fileContent, fileName, collectionName, chunkSize)
	default:
		return nil, fmt.Errorf("不支持的知识库类型: %s，支持的类型: chromadb, milvus, weaviate", knowledgeType)
	}
}

// ========== 辅助函数 ==========

// getPodInfo 获取 Pod 信息和端口
func (k *knowledge) getPodInfo(podName, namespace string, defaultPort int32) (*coreV1.Pod, int32, error) {
	pod, err := Pod.GetPodDetail(podName, namespace)
	if err != nil {
		return nil, 0, fmt.Errorf("获取Pod信息失败: %v", err)
	}

	if pod.Status.Phase != coreV1.PodRunning {
		return nil, 0, fmt.Errorf("pod %s 状态为 %s，请等待Pod启动完成", podName, pod.Status.Phase)
	}

	port := defaultPort
	if len(pod.Spec.Containers) > 0 {
		for _, containerPort := range pod.Spec.Containers[0].Ports {
			if containerPort.Name == "http" || containerPort.ContainerPort == defaultPort {
				port = containerPort.ContainerPort
				break
			}
		}
	}

	return pod, port, nil
}

// getOllamaInfo 从 Pod 环境变量获取 Ollama 信息
func (k *knowledge) getOllamaInfo(pod *coreV1.Pod, namespace string) (podName, ollamaNamespace, model string) {
	if len(pod.Spec.Containers) == 0 {
		return "", "", ""
	}

	for _, env := range pod.Spec.Containers[0].Env {
		switch env.Name {
		case "OLLAMA_POD_NAME":
			podName = env.Value
		case "OLLAMA_NAMESPACE":
			ollamaNamespace = env.Value
		case "OLLAMA_MODEL":
			model = env.Value
		}
	}

	if ollamaNamespace == "" {
		ollamaNamespace = namespace
	}

	return podName, ollamaNamespace, model
}

// splitText 将文本分块
func (k *knowledge) splitText(text string, chunkSize int) []string {
	if chunkSize <= 0 {
		chunkSize = 1000
	}

	if len(text) <= chunkSize {
		return []string{text}
	}

	var chunks []string
	for i := 0; i < len(text); i += chunkSize {
		end := i + chunkSize
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[i:end])
	}
	return chunks
}

// generateEmbeddings 使用 Ollama 生成向量嵌入
func (k *knowledge) generateEmbeddings(ollamaPodName, ollamaNamespace, ollamaModel string, texts []string) ([][]float64, error) {
	if ollamaPodName == "" || ollamaModel == "" {
		return nil, nil // 没有绑定 Ollama，返回 nil
	}

	embeddings := make([][]float64, 0, len(texts))
	for _, text := range texts {
		result, err := Ollama.Embeddings(ollamaPodName, ollamaNamespace, ollamaModel, text)
		if err != nil {
			return nil, fmt.Errorf("生成向量失败: %v", err)
		}

		resultMap, ok := result.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("ollama 响应格式错误")
		}

		embeddingInterface, ok := resultMap["embedding"]
		if !ok {
			return nil, fmt.Errorf("ollama 响应中缺少 embedding 字段")
		}

		embeddingSlice, ok := embeddingInterface.([]interface{})
		if !ok {
			return nil, fmt.Errorf("embedding 格式错误")
		}

		embedding := make([]float64, len(embeddingSlice))
		for i, v := range embeddingSlice {
			if f, ok := v.(float64); ok {
				embedding[i] = f
			} else {
				return nil, fmt.Errorf("embedding 元素类型错误")
			}
		}

		embeddings = append(embeddings, embedding)
	}
	return embeddings, nil
}

// sanitizeCollectionName 清理集合名称
func (k *knowledge) sanitizeCollectionName(name string) string {
	var result []rune
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result = append(result, r)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}

// ========== ChromaDB 上传 ==========

// uploadToChroma 上传文档到 ChromaDB
func (k *knowledge) uploadToChroma(podName, namespace string, fileContent []byte, fileName, collectionName string, chunkSize int) (interface{}, error) {
	pod, port, err := k.getPodInfo(podName, namespace, 8000)
	if err != nil {
		return nil, err
	}

	textContent := string(fileContent)
	if textContent == "" {
		return nil, fmt.Errorf("文件内容为空")
	}

	chunks := k.splitText(textContent, chunkSize)
	if len(chunks) == 0 {
		return nil, fmt.Errorf("文件分块后为空")
	}

	ollamaPodName, ollamaNamespace, ollamaModel := k.getOllamaInfo(pod, namespace)
	embeddings, err := k.generateEmbeddings(ollamaPodName, ollamaNamespace, ollamaModel, chunks)
	if err != nil {
		return nil, fmt.Errorf("生成向量嵌入失败: %v", err)
	}

	if collectionName == "" {
		collectionName = fileName
	}
	collectionName = k.sanitizeCollectionName(collectionName)

	// 确保集合存在（现在返回 UUID，但这里不需要，因为 addToChroma 内部会处理）
	_, err = k.ensureChromaCollection(podName, namespace, port, collectionName)
	if err != nil {
		return nil, fmt.Errorf("创建集合失败: %v", err)
	}

	// 准备数据
	ids := make([]string, len(chunks))
	metadatas := make([]map[string]interface{}, len(chunks))
	for i := range chunks {
		ids[i] = fmt.Sprintf("%s_chunk_%d", collectionName, i)
		metadatas[i] = map[string]interface{}{
			"source":   fileName,
			"chunk_id": i,
		}
	}

	// 添加文档到 Chroma（使用和 Ollama 相同的方式）
	result, err := k.addToChroma(podName, namespace, port, collectionName, chunks, embeddings, ids, metadatas)
	if err != nil {
		return nil, fmt.Errorf("添加文档到 Chroma 失败: %v", err)
	}

	return map[string]interface{}{
		"status":          "success",
		"message":         "文档上传成功",
		"knowledge_type":  "chromadb",
		"collection_name": collectionName,
		"chunks_count":    len(chunks),
		"result":          result,
	}, nil
}

// ensureChromaCollection 确保 Chroma 集合存在（使用 v2 API），返回集合的 UUID
func (k *knowledge) ensureChromaCollection(podName, namespace string, port int32, collectionName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	// Chroma v2 API 需要 tenant 和 database 参数
	tenant := "default_tenant"
	database := "default_database"

	// 先列出所有集合，查找匹配名称的集合
	listReq := K8s.ClientSet.CoreV1().RESTClient().Get().
		Namespace(namespace).
		Resource("pods").
		Name(fmt.Sprintf("%s:%d", podName, port)).
		SubResource("proxy").
		Suffix(fmt.Sprintf("/api/v2/tenants/%s/databases/%s/collections", tenant, database))

	listResult := listReq.Do(ctx)
	if listResult.Error() == nil {
		body, err := listResult.Raw()
		if err == nil && len(body) > 0 {
			var collections []map[string]interface{}
			if json.Unmarshal(body, &collections) == nil {
				// 查找匹配名称的集合
				for _, collection := range collections {
					if name, ok := collection["name"].(string); ok && name == collectionName {
						if id, ok := collection["id"].(string); ok {
							return id, nil // 返回集合的 UUID
						}
					}
				}
			}
		}
	}

	// 集合不存在，创建集合
	createBody := map[string]interface{}{
		"name": collectionName,
	}
	jsonData, err := json.Marshal(createBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求体失败: %v", err)
	}

	createReq := K8s.ClientSet.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		Name(fmt.Sprintf("%s:%d", podName, port)).
		SubResource("proxy").
		Suffix(fmt.Sprintf("/api/v2/tenants/%s/databases/%s/collections", tenant, database)).
		Body(jsonData).
		SetHeader("Content-Type", "application/json")

	createResult := createReq.Do(ctx)
	if createResult.Error() != nil {
		// 检查是否是集合已存在的错误
		body, _ := createResult.Raw()
		if len(body) > 0 {
			var errorResponse map[string]interface{}
			if json.Unmarshal(body, &errorResponse) == nil {
				if errMsg, ok := errorResponse["error"].(string); ok {
					// 如果错误是集合已存在，尝试再次查找
					if strings.Contains(strings.ToLower(errMsg), "already exists") || strings.Contains(strings.ToLower(errMsg), "duplicate") {
						// 重新查找集合
						listResult := listReq.Do(ctx)
						if listResult.Error() == nil {
							body, err := listResult.Raw()
							if err == nil && len(body) > 0 {
								var collections []map[string]interface{}
								if json.Unmarshal(body, &collections) == nil {
									for _, collection := range collections {
										if name, ok := collection["name"].(string); ok && name == collectionName {
											if id, ok := collection["id"].(string); ok {
												return id, nil
											}
										}
									}
								}
							}
						}
						return "", fmt.Errorf("集合已存在但无法获取ID: %s", errMsg)
					}
					return "", fmt.Errorf("创建集合失败: %s", errMsg)
				}
			}
		}
		return "", fmt.Errorf("创建集合失败: %v", createResult.Error())
	}

	// 获取创建的集合的 UUID
	body, err := createResult.Raw()
	if err == nil && len(body) > 0 {
		var response map[string]interface{}
		if json.Unmarshal(body, &response) == nil {
			if id, ok := response["id"].(string); ok {
				return id, nil
			}
		}
	}

	// 如果无法从响应中获取 ID，重新查找
	listResult = listReq.Do(ctx)
	if listResult.Error() == nil {
		body, err := listResult.Raw()
		if err == nil && len(body) > 0 {
			var collections []map[string]interface{}
			if json.Unmarshal(body, &collections) == nil {
				for _, collection := range collections {
					if name, ok := collection["name"].(string); ok && name == collectionName {
						if id, ok := collection["id"].(string); ok {
							return id, nil
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("创建集合成功但无法获取ID")
}

// addToChroma 添加文档到 Chroma（使用 v2 API）
func (k *knowledge) addToChroma(podName, namespace string, port int32, collectionName string, documents []string, embeddings [][]float64, ids []string, metadatas []map[string]interface{}) (interface{}, error) {
	// Chroma v2 API 需要 tenant 和 database 参数
	tenant := "default_tenant"
	database := "default_database"

	// 确保集合存在并获取集合的 UUID
	collectionUUID, err := k.ensureChromaCollection(podName, namespace, port, collectionName)
	if err != nil {
		return nil, fmt.Errorf("确保集合存在失败: %v", err)
	}

	// Chroma v2 API 要求 embeddings 字段是必需的
	// embeddings 必须是二维数组（每个文档一个向量数组）或字符串数组（base64 编码）
	// 如果没有提供 embeddings，返回错误（应该由调用方生成 embeddings）
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("embeddings 字段是必需的，请先生成向量嵌入")
	}

	// 确保 embeddings 数量与 documents 数量一致
	if len(embeddings) != len(documents) {
		return nil, fmt.Errorf("embeddings 数量 (%d) 与 documents 数量 (%d) 不一致", len(embeddings), len(documents))
	}

	requestBody := map[string]interface{}{
		"ids":        ids,
		"documents":  documents,
		"metadatas":  metadatas,
		"embeddings": embeddings, // 必需字段，必须是二维数组
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %v", err)
	}

	// 使用 Kubernetes API Server 代理访问 Pod（和 Ollama 相同的方式）
	// Chroma v2 API 路径: /api/v2/tenants/{tenant}/databases/{database}/collections/{collection_id}/add
	// 注意：collection_id 必须是 UUID，不是名称
	req := K8s.ClientSet.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		Name(fmt.Sprintf("%s:%d", podName, port)).
		SubResource("proxy").
		Suffix(fmt.Sprintf("/api/v2/tenants/%s/databases/%s/collections/%s/add", tenant, database, collectionUUID)).
		Body(jsonData).
		SetHeader("Content-Type", "application/json")

	// 发送请求（使用 context.TODO()，和 Ollama 保持一致）
	result := req.Do(context.TODO())
	if result.Error() != nil {
		return nil, fmt.Errorf("请求 Chroma API 失败: %v", result.Error())
	}

	// 读取响应
	body, err := result.Raw()
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析 JSON 响应
	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, body: %s", err, string(body))
	}

	// 检查是否有错误
	if errMsg, ok := responseData["error"].(string); ok && errMsg != "" {
		return nil, fmt.Errorf("chroma API 返回错误: %s", errMsg)
	}

	return responseData, nil
}

// ========== Milvus 上传 ==========

// uploadToMilvus 上传文档到 Milvus
func (k *knowledge) uploadToMilvus(podName, namespace string, fileContent []byte, fileName, collectionName string, chunkSize int) (interface{}, error) {
	pod, port, err := k.getPodInfo(podName, namespace, 19530)
	if err != nil {
		return nil, err
	}

	textContent := string(fileContent)
	if textContent == "" {
		return nil, fmt.Errorf("文件内容为空")
	}

	chunks := k.splitText(textContent, chunkSize)
	if len(chunks) == 0 {
		return nil, fmt.Errorf("文件分块后为空")
	}

	ollamaPodName, ollamaNamespace, ollamaModel := k.getOllamaInfo(pod, namespace)
	embeddings, err := k.generateEmbeddings(ollamaPodName, ollamaNamespace, ollamaModel, chunks)
	if err != nil {
		return nil, fmt.Errorf("生成向量嵌入失败: %v", err)
	}

	if embeddings == nil {
		return nil, fmt.Errorf("milvus 需要向量嵌入，请确保绑定了 Ollama")
	}

	if collectionName == "" {
		collectionName = fileName
	}
	collectionName = k.sanitizeCollectionName(collectionName)

	// 确保集合存在
	if err := k.ensureMilvusCollection(podName, namespace, port, collectionName, len(embeddings[0])); err != nil {
		return nil, fmt.Errorf("创建集合失败: %v", err)
	}

	// 准备数据
	ids := make([]int64, len(chunks))
	data := make([][]interface{}, len(chunks))
	for i := range chunks {
		ids[i] = int64(i)
		// Milvus 数据格式: [id, text, vector]
		data[i] = []interface{}{
			int64(i),
			chunks[i],
			embeddings[i],
		}
	}

	// 插入数据到 Milvus
	result, err := k.insertToMilvus(podName, namespace, port, collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("插入数据到 Milvus 失败: %v", err)
	}

	return map[string]interface{}{
		"status":          "success",
		"message":         "文档上传成功",
		"knowledge_type":  "milvus",
		"collection_name": collectionName,
		"chunks_count":    len(chunks),
		"result":          result,
	}, nil
}

// ensureMilvusCollection 确保 Milvus 集合存在
func (k *knowledge) ensureMilvusCollection(podName, namespace string, port int32, collectionName string, vectorDim int) error {
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	// 检查集合是否存在
	req := K8s.ClientSet.CoreV1().RESTClient().Get().
		Namespace(namespace).
		Resource("pods").
		Name(fmt.Sprintf("%s:%d", podName, port)).
		SubResource("proxy").
		Suffix(fmt.Sprintf("/v1/collections/%s", collectionName))

	result := req.Do(ctx)
	if result.Error() == nil {
		return nil // 集合已存在
	}

	// 创建集合
	createBody := map[string]interface{}{
		"collection_name": collectionName,
		"dimension":       vectorDim,
		"metric_type":     "L2",
	}
	jsonData, _ := json.Marshal(createBody)

	createReq := K8s.ClientSet.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		Name(fmt.Sprintf("%s:%d", podName, port)).
		SubResource("proxy").
		Suffix("/v1/collections").
		Body(jsonData).
		SetHeader("Content-Type", "application/json")

	createReq.Do(ctx)
	return nil
}

// insertToMilvus 插入数据到 Milvus
func (k *knowledge) insertToMilvus(podName, namespace string, port int32, collectionName string, data [][]interface{}) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Minute)
	defer cancel()

	requestBody := map[string]interface{}{
		"collection_name": collectionName,
		"data":            data,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %v", err)
	}

	req := K8s.ClientSet.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		Name(fmt.Sprintf("%s:%d", podName, port)).
		SubResource("proxy").
		Suffix(fmt.Sprintf("/v1/collections/%s/insert", collectionName)).
		Body(jsonData).
		SetHeader("Content-Type", "application/json")

	result := req.Do(ctx)
	if result.Error() != nil {
		return nil, fmt.Errorf("请求 Milvus API 失败: %v", result.Error())
	}

	body, err := result.Raw()
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var responseData map[string]interface{}
	json.Unmarshal(body, &responseData)
	return responseData, nil
}

// ========== Weaviate 上传 ==========

// uploadToWeaviate 上传文档到 Weaviate
func (k *knowledge) uploadToWeaviate(podName, namespace string, fileContent []byte, fileName, collectionName string, chunkSize int) (interface{}, error) {
	pod, port, err := k.getPodInfo(podName, namespace, 8080)
	if err != nil {
		return nil, err
	}

	textContent := string(fileContent)
	if textContent == "" {
		return nil, fmt.Errorf("文件内容为空")
	}

	chunks := k.splitText(textContent, chunkSize)
	if len(chunks) == 0 {
		return nil, fmt.Errorf("文件分块后为空")
	}

	ollamaPodName, ollamaNamespace, ollamaModel := k.getOllamaInfo(pod, namespace)
	embeddings, err := k.generateEmbeddings(ollamaPodName, ollamaNamespace, ollamaModel, chunks)
	if err != nil {
		return nil, fmt.Errorf("生成向量嵌入失败: %v", err)
	}

	if collectionName == "" {
		collectionName = fileName
	}
	collectionName = k.sanitizeCollectionName(collectionName)

	// 确保类存在
	if err := k.ensureWeaviateClass(podName, namespace, port, collectionName); err != nil {
		return nil, fmt.Errorf("创建类失败: %v", err)
	}

	// 批量添加对象
	objects := make([]map[string]interface{}, len(chunks))
	for i, chunk := range chunks {
		obj := map[string]interface{}{
			"text":   chunk,
			"source": fileName,
			"chunk":  i,
		}
		if embeddings != nil && i < len(embeddings) {
			obj["vector"] = embeddings[i]
		}
		objects[i] = obj
	}

	result, err := k.batchAddToWeaviate(podName, namespace, port, collectionName, objects)
	if err != nil {
		return nil, fmt.Errorf("添加对象到 Weaviate 失败: %v", err)
	}

	return map[string]interface{}{
		"status":          "success",
		"message":         "文档上传成功",
		"knowledge_type":  "weaviate",
		"collection_name": collectionName,
		"chunks_count":    len(chunks),
		"result":          result,
	}, nil
}

// ensureWeaviateClass 确保 Weaviate 类存在
func (k *knowledge) ensureWeaviateClass(podName, namespace string, port int32, className string) error {
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	// 检查类是否存在
	req := K8s.ClientSet.CoreV1().RESTClient().Get().
		Namespace(namespace).
		Resource("pods").
		Name(fmt.Sprintf("%s:%d", podName, port)).
		SubResource("proxy").
		Suffix(fmt.Sprintf("/v1/schema/%s", className))

	result := req.Do(ctx)
	if result.Error() == nil {
		return nil // 类已存在
	}

	// 创建类
	createBody := map[string]interface{}{
		"class": className,
		"properties": []map[string]interface{}{
			{"name": "text", "dataType": []string{"text"}},
			{"name": "source", "dataType": []string{"string"}},
			{"name": "chunk", "dataType": []string{"int"}},
		},
	}
	jsonData, _ := json.Marshal(createBody)

	createReq := K8s.ClientSet.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		Name(fmt.Sprintf("%s:%d", podName, port)).
		SubResource("proxy").
		Suffix("/v1/schema").
		Body(jsonData).
		SetHeader("Content-Type", "application/json")

	createReq.Do(ctx)
	return nil
}

// batchAddToWeaviate 批量添加对象到 Weaviate
func (k *knowledge) batchAddToWeaviate(podName, namespace string, port int32, className string, objects []map[string]interface{}) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Minute)
	defer cancel()

	// Weaviate 批量添加
	things := make([]map[string]interface{}, len(objects))
	for i, obj := range objects {
		things[i] = map[string]interface{}{
			"class":      className,
			"properties": obj,
		}
		if vector, ok := obj["vector"].([]float64); ok {
			things[i]["vector"] = vector
			delete(things[i]["properties"].(map[string]interface{}), "vector")
		}
	}

	requestBody := map[string]interface{}{
		"objects": things,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %v", err)
	}

	req := K8s.ClientSet.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		Name(fmt.Sprintf("%s:%d", podName, port)).
		SubResource("proxy").
		Suffix("/v1/batch/objects").
		Body(jsonData).
		SetHeader("Content-Type", "application/json")

	result := req.Do(ctx)
	if result.Error() != nil {
		return nil, fmt.Errorf("请求 Weaviate API 失败: %v", result.Error())
	}

	body, err := result.Raw()
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var responseData map[string]interface{}
	json.Unmarshal(body, &responseData)
	return responseData, nil
}

// KnowledgeResp 知识库列表响应
type KnowledgeResp struct {
	Total int                   `json:"total"`
	Items []KnowledgeDeployInfo `json:"items"`
}

// KnowledgeDeployInfo 知识库部署信息
type KnowledgeDeployInfo struct {
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	Type            string            `json:"type"` // deployment 或 daemonset
	Image           string            `json:"image"`
	Port            int32             `json:"port"`
	NodeSelector    map[string]string `json:"node_selector"`
	Status          string            `json:"status"`
	Pods            int32             `json:"pods"`
	ReadyPods       int32             `json:"ready_pods"`
	OllamaPodName   string            `json:"ollama_pod_name,omitempty"`
	OllamaModel     string            `json:"ollama_model,omitempty"`
	OllamaNamespace string            `json:"ollama_namespace,omitempty"`
	StorageSize     string            `json:"storage_size,omitempty"`
}

// ListKnowledge 获取知识库部署列表
func (k *knowledge) ListKnowledge(filterName, namespace, nodeName string, limit, page int) (*KnowledgeResp, error) {
	var items []KnowledgeDeployInfo

	// 获取所有命名空间的 Deployment 和 DaemonSet
	namespaces := []string{namespace}
	if namespace == "" {
		nsList, err := K8s.ClientSet.CoreV1().Namespaces().List(context.TODO(), metaV1.ListOptions{})
		if err != nil {
			return nil, err
		}
		namespaces = make([]string, len(nsList.Items))
		for i, ns := range nsList.Items {
			namespaces[i] = ns.Name
		}
	}

	// 收集所有带有 knowledge 标签的资源
	for _, ns := range namespaces {
		// 获取 Deployment
		deployList, err := K8s.ClientSet.AppsV1().Deployments(ns).List(context.TODO(), metaV1.ListOptions{
			LabelSelector: "app=knowledge,managed=kubemanage",
		})
		if err == nil {
			for _, deploy := range deployList.Items {
				if filterName == "" || deploy.Name == filterName {
					info := k.convertDeploymentToInfo(&deploy, nodeName)
					if info != nil {
						items = append(items, *info)
					}
				}
			}
		}

		// 获取 DaemonSet
		dsList, err := K8s.ClientSet.AppsV1().DaemonSets(ns).List(context.TODO(), metaV1.ListOptions{
			LabelSelector: "app=knowledge,managed=kubemanage",
		})
		if err == nil {
			for _, ds := range dsList.Items {
				if filterName == "" || ds.Name == filterName {
					info := k.convertDaemonSetToInfo(&ds, nodeName)
					if info != nil {
						items = append(items, *info)
					}
				}
			}
		}
	}

	// 分页处理
	total := len(items)
	start := (page - 1) * limit
	end := start + limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	if start < 0 {
		start = 0
	}

	var pagedItems []KnowledgeDeployInfo
	if start < end {
		pagedItems = items[start:end]
	}

	return &KnowledgeResp{
		Total: total,
		Items: pagedItems,
	}, nil
}

// convertDeploymentToInfo 转换 Deployment 为 KnowledgeDeployInfo
func (k *knowledge) convertDeploymentToInfo(deploy *appsV1.Deployment, nodeName string) *KnowledgeDeployInfo {
	// 如果指定了节点名称，检查节点选择器
	if nodeName != "" {
		if deploy.Spec.Template.Spec.NodeSelector != nil {
			if hostname, ok := deploy.Spec.Template.Spec.NodeSelector["kubernetes.io/hostname"]; ok {
				if hostname != nodeName {
					return nil
				}
			}
		} else {
			return nil
		}
	}

	info := &KnowledgeDeployInfo{
		Name:         deploy.Name,
		Namespace:    deploy.Namespace,
		Type:         "deployment",
		NodeSelector: deploy.Spec.Template.Spec.NodeSelector,
		Pods:         *deploy.Spec.Replicas,
		ReadyPods:    deploy.Status.ReadyReplicas,
	}

	// 获取容器信息
	if len(deploy.Spec.Template.Spec.Containers) > 0 {
		container := deploy.Spec.Template.Spec.Containers[0]
		info.Image = container.Image
		if len(container.Ports) > 0 {
			info.Port = container.Ports[0].ContainerPort
		}

		// 获取环境变量（Ollama 绑定信息）
		for _, env := range container.Env {
			switch env.Name {
			case "OLLAMA_POD_NAME":
				info.OllamaPodName = env.Value
			case "OLLAMA_NAMESPACE":
				info.OllamaNamespace = env.Value
			case "OLLAMA_MODEL":
				info.OllamaModel = env.Value
			}
		}
	}

	// 获取存储信息（从 PVC）
	if len(deploy.Spec.Template.Spec.Volumes) > 0 {
		for _, volume := range deploy.Spec.Template.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil {
				pvcName := volume.PersistentVolumeClaim.ClaimName
				pvc, err := K8s.ClientSet.CoreV1().PersistentVolumeClaims(deploy.Namespace).Get(context.TODO(), pvcName, metaV1.GetOptions{})
				if err == nil {
					if storage, ok := pvc.Spec.Resources.Requests[coreV1.ResourceStorage]; ok {
						info.StorageSize = storage.String()
					}
				}
			}
		}
	}

	// 设置状态
	if deploy.Status.ReadyReplicas == *deploy.Spec.Replicas && *deploy.Spec.Replicas > 0 {
		info.Status = "Ready"
	} else if deploy.Status.ReadyReplicas > 0 {
		info.Status = "NotReady"
	} else {
		info.Status = "Pending"
	}

	return info
}

// convertDaemonSetToInfo 转换 DaemonSet 为 KnowledgeDeployInfo
func (k *knowledge) convertDaemonSetToInfo(ds *appsV1.DaemonSet, nodeName string) *KnowledgeDeployInfo {
	// 如果指定了节点名称，检查节点选择器
	if nodeName != "" {
		if ds.Spec.Template.Spec.NodeSelector != nil {
			if hostname, ok := ds.Spec.Template.Spec.NodeSelector["kubernetes.io/hostname"]; ok {
				if hostname != nodeName {
					return nil
				}
			}
		} else {
			return nil
		}
	}

	info := &KnowledgeDeployInfo{
		Name:         ds.Name,
		Namespace:    ds.Namespace,
		Type:         "daemonset",
		NodeSelector: ds.Spec.Template.Spec.NodeSelector,
		Pods:         ds.Status.DesiredNumberScheduled,
		ReadyPods:    ds.Status.NumberReady,
	}

	// 获取容器信息
	if len(ds.Spec.Template.Spec.Containers) > 0 {
		container := ds.Spec.Template.Spec.Containers[0]
		info.Image = container.Image
		if len(container.Ports) > 0 {
			info.Port = container.Ports[0].ContainerPort
		}

		// 获取环境变量（Ollama 绑定信息）
		for _, env := range container.Env {
			switch env.Name {
			case "OLLAMA_POD_NAME":
				info.OllamaPodName = env.Value
			case "OLLAMA_NAMESPACE":
				info.OllamaNamespace = env.Value
			case "OLLAMA_MODEL":
				info.OllamaModel = env.Value
			}
		}
	}

	// 获取存储信息（从 PVC）
	if len(ds.Spec.Template.Spec.Volumes) > 0 {
		for _, volume := range ds.Spec.Template.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil {
				pvcName := volume.PersistentVolumeClaim.ClaimName
				pvc, err := K8s.ClientSet.CoreV1().PersistentVolumeClaims(ds.Namespace).Get(context.TODO(), pvcName, metaV1.GetOptions{})
				if err == nil {
					if storage, ok := pvc.Spec.Resources.Requests[coreV1.ResourceStorage]; ok {
						info.StorageSize = storage.String()
					}
				}
			}
		}
	}

	// 设置状态
	if ds.Status.NumberReady == ds.Status.DesiredNumberScheduled && ds.Status.DesiredNumberScheduled > 0 {
		info.Status = "Ready"
	} else if ds.Status.NumberReady > 0 {
		info.Status = "NotReady"
	} else {
		info.Status = "Pending"
	}

	return info
}

// GetKnowledgeDetail 获取知识库详情
func (k *knowledge) GetKnowledgeDetail(name, namespace string) (interface{}, error) {
	// 先尝试获取 Deployment
	deploy, err := K8s.ClientSet.AppsV1().Deployments(namespace).Get(context.TODO(), name, metaV1.GetOptions{})
	if err == nil {
		// 检查是否是知识库（通过标签判断）
		if deploy.Labels["app"] == "knowledge" && deploy.Labels["managed"] == "kubemanage" {
			// 转换为详细信息
			detail := k.convertDeploymentToDetail(deploy)
			return detail, nil
		}
	}

	// 如果不是 Deployment 或不是知识库，尝试获取 DaemonSet
	ds, err := K8s.ClientSet.AppsV1().DaemonSets(namespace).Get(context.TODO(), name, metaV1.GetOptions{})
	if err == nil {
		// 检查是否是知识库（通过标签判断）
		if ds.Labels["app"] == "knowledge" && ds.Labels["managed"] == "kubemanage" {
			// 转换为详细信息
			detail := k.convertDaemonSetToDetail(ds)
			return detail, nil
		}
	}

	return nil, fmt.Errorf("未找到知识库 %s/%s", namespace, name)
}

// KnowledgeDetail 知识库详情
type KnowledgeDetail struct {
	Name             string                 `json:"name"`
	Namespace        string                 `json:"namespace"`
	Type             string                 `json:"type"` // deployment 或 daemonset
	Image            string                 `json:"image"`
	Port             int32                  `json:"port"`
	NodeSelector     map[string]string      `json:"node_selector"`
	Status           string                 `json:"status"`
	Pods             int32                  `json:"pods"`
	ReadyPods        int32                  `json:"ready_pods"`
	OllamaPodName    string                 `json:"ollama_pod_name,omitempty"`
	OllamaModel      string                 `json:"ollama_model,omitempty"`
	OllamaNamespace  string                 `json:"ollama_namespace,omitempty"`
	StorageSize      string                 `json:"storage_size,omitempty"`
	Labels           map[string]string      `json:"labels"`
	Annotations      map[string]string      `json:"annotations"`
	CreatedAt        string                 `json:"created_at"`
	ResourceLimits   map[string]interface{} `json:"resource_limits,omitempty"`
	ResourceRequests map[string]interface{} `json:"resource_requests,omitempty"`
	ServiceName      string                 `json:"service_name,omitempty"`
	PVCName          string                 `json:"pvc_name,omitempty"`
}

// convertDeploymentToDetail 转换 Deployment 为详细信息
func (k *knowledge) convertDeploymentToDetail(deploy *appsV1.Deployment) *KnowledgeDetail {
	detail := &KnowledgeDetail{
		Name:         deploy.Name,
		Namespace:    deploy.Namespace,
		Type:         "deployment",
		NodeSelector: deploy.Spec.Template.Spec.NodeSelector,
		Pods:         *deploy.Spec.Replicas,
		ReadyPods:    deploy.Status.ReadyReplicas,
		Labels:       deploy.Labels,
		Annotations:  deploy.Annotations,
	}

	// 设置创建时间
	if !deploy.CreationTimestamp.IsZero() {
		detail.CreatedAt = deploy.CreationTimestamp.Format(time.RFC3339)
	}

	// 获取容器信息
	if len(deploy.Spec.Template.Spec.Containers) > 0 {
		container := deploy.Spec.Template.Spec.Containers[0]
		detail.Image = container.Image
		if len(container.Ports) > 0 {
			detail.Port = container.Ports[0].ContainerPort
		}

		// 获取环境变量（Ollama 绑定信息）
		for _, env := range container.Env {
			switch env.Name {
			case "OLLAMA_POD_NAME":
				detail.OllamaPodName = env.Value
			case "OLLAMA_NAMESPACE":
				detail.OllamaNamespace = env.Value
			case "OLLAMA_MODEL":
				detail.OllamaModel = env.Value
			}
		}

		// 获取资源限制和请求
		if container.Resources.Limits != nil {
			detail.ResourceLimits = make(map[string]interface{})
			if cpu, ok := container.Resources.Limits[coreV1.ResourceCPU]; ok {
				detail.ResourceLimits["cpu"] = cpu.String()
			}
			if memory, ok := container.Resources.Limits[coreV1.ResourceMemory]; ok {
				detail.ResourceLimits["memory"] = memory.String()
			}
		}
		if container.Resources.Requests != nil {
			detail.ResourceRequests = make(map[string]interface{})
			if cpu, ok := container.Resources.Requests[coreV1.ResourceCPU]; ok {
				detail.ResourceRequests["cpu"] = cpu.String()
			}
			if memory, ok := container.Resources.Requests[coreV1.ResourceMemory]; ok {
				detail.ResourceRequests["memory"] = memory.String()
			}
		}
	}

	// 获取存储信息（从 PVC）
	if len(deploy.Spec.Template.Spec.Volumes) > 0 {
		for _, volume := range deploy.Spec.Template.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil {
				detail.PVCName = volume.PersistentVolumeClaim.ClaimName
				pvc, err := K8s.ClientSet.CoreV1().PersistentVolumeClaims(deploy.Namespace).Get(context.TODO(), detail.PVCName, metaV1.GetOptions{})
				if err == nil {
					if storage, ok := pvc.Spec.Resources.Requests[coreV1.ResourceStorage]; ok {
						detail.StorageSize = storage.String()
					}
				}
			}
		}
	}

	// 获取 Service 名称
	serviceName := fmt.Sprintf("%s-svc", deploy.Name)
	_, err := K8s.ClientSet.CoreV1().Services(deploy.Namespace).Get(context.TODO(), serviceName, metaV1.GetOptions{})
	if err == nil {
		detail.ServiceName = serviceName
	}

	// 设置状态
	if deploy.Status.ReadyReplicas == *deploy.Spec.Replicas && *deploy.Spec.Replicas > 0 {
		detail.Status = "Ready"
	} else if deploy.Status.ReadyReplicas > 0 {
		detail.Status = "NotReady"
	} else {
		detail.Status = "Pending"
	}

	return detail
}

// convertDaemonSetToDetail 转换 DaemonSet 为详细信息
func (k *knowledge) convertDaemonSetToDetail(ds *appsV1.DaemonSet) *KnowledgeDetail {
	detail := &KnowledgeDetail{
		Name:         ds.Name,
		Namespace:    ds.Namespace,
		Type:         "daemonset",
		NodeSelector: ds.Spec.Template.Spec.NodeSelector,
		Pods:         ds.Status.DesiredNumberScheduled,
		ReadyPods:    ds.Status.NumberReady,
		Labels:       ds.Labels,
		Annotations:  ds.Annotations,
	}

	// 设置创建时间
	if !ds.CreationTimestamp.IsZero() {
		detail.CreatedAt = ds.CreationTimestamp.Format(time.RFC3339)
	}

	// 获取容器信息
	if len(ds.Spec.Template.Spec.Containers) > 0 {
		container := ds.Spec.Template.Spec.Containers[0]
		detail.Image = container.Image
		if len(container.Ports) > 0 {
			detail.Port = container.Ports[0].ContainerPort
		}

		// 获取环境变量（Ollama 绑定信息）
		for _, env := range container.Env {
			switch env.Name {
			case "OLLAMA_POD_NAME":
				detail.OllamaPodName = env.Value
			case "OLLAMA_NAMESPACE":
				detail.OllamaNamespace = env.Value
			case "OLLAMA_MODEL":
				detail.OllamaModel = env.Value
			}
		}

		// 获取资源限制和请求
		if container.Resources.Limits != nil {
			detail.ResourceLimits = make(map[string]interface{})
			if cpu, ok := container.Resources.Limits[coreV1.ResourceCPU]; ok {
				detail.ResourceLimits["cpu"] = cpu.String()
			}
			if memory, ok := container.Resources.Limits[coreV1.ResourceMemory]; ok {
				detail.ResourceLimits["memory"] = memory.String()
			}
		}
		if container.Resources.Requests != nil {
			detail.ResourceRequests = make(map[string]interface{})
			if cpu, ok := container.Resources.Requests[coreV1.ResourceCPU]; ok {
				detail.ResourceRequests["cpu"] = cpu.String()
			}
			if memory, ok := container.Resources.Requests[coreV1.ResourceMemory]; ok {
				detail.ResourceRequests["memory"] = memory.String()
			}
		}
	}

	// 获取存储信息（从 PVC）
	if len(ds.Spec.Template.Spec.Volumes) > 0 {
		for _, volume := range ds.Spec.Template.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil {
				detail.PVCName = volume.PersistentVolumeClaim.ClaimName
				pvc, err := K8s.ClientSet.CoreV1().PersistentVolumeClaims(ds.Namespace).Get(context.TODO(), detail.PVCName, metaV1.GetOptions{})
				if err == nil {
					if storage, ok := pvc.Spec.Resources.Requests[coreV1.ResourceStorage]; ok {
						detail.StorageSize = storage.String()
					}
				}
			}
		}
	}

	// 获取 Service 名称
	serviceName := fmt.Sprintf("%s-svc", ds.Name)
	_, err := K8s.ClientSet.CoreV1().Services(ds.Namespace).Get(context.TODO(), serviceName, metaV1.GetOptions{})
	if err == nil {
		detail.ServiceName = serviceName
	}

	// 设置状态
	if ds.Status.NumberReady == ds.Status.DesiredNumberScheduled && ds.Status.DesiredNumberScheduled > 0 {
		detail.Status = "Ready"
	} else if ds.Status.NumberReady > 0 {
		detail.Status = "NotReady"
	} else {
		detail.Status = "Pending"
	}

	return detail
}

// QueryKnowledge 查询知识库（支持 ChromaDB、Milvus、Weaviate）
func (k *knowledge) QueryKnowledge(podName, namespace, knowledgeType, collectionName, queryText string, topK int) (interface{}, error) {
	// 根据知识库类型调用不同的查询方法
	knowledgeType = strings.ToLower(knowledgeType)
	switch knowledgeType {
	case "chromadb", "chroma":
		return k.queryChroma(podName, namespace, collectionName, queryText, topK)
	case "milvus":
		return k.queryMilvus(podName, namespace, collectionName, queryText, topK)
	case "weaviate":
		return k.queryWeaviate(podName, namespace, collectionName, queryText, topK)
	default:
		return nil, fmt.Errorf("不支持的知识库类型: %s，支持的类型: chromadb, milvus, weaviate", knowledgeType)
	}
}

// ========== ChromaDB 查询 ==========

// queryChroma 查询 ChromaDB
func (k *knowledge) queryChroma(podName, namespace, collectionName, queryText string, topK int) (interface{}, error) {
	pod, port, err := k.getPodInfo(podName, namespace, 8000)
	if err != nil {
		return nil, err
	}

	if topK <= 0 {
		topK = 5
	}

	// 获取 Ollama 信息并生成查询向量
	ollamaPodName, ollamaNamespace, ollamaModel := k.getOllamaInfo(pod, namespace)
	if ollamaPodName == "" || ollamaModel == "" {
		return nil, fmt.Errorf("查询需要向量嵌入，请确保知识库绑定了 Ollama")
	}

	embeddings, err := k.generateEmbeddings(ollamaPodName, ollamaNamespace, ollamaModel, []string{queryText})
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %v", err)
	}
	if len(embeddings) == 0 || len(embeddings[0]) == 0 {
		return nil, fmt.Errorf("生成的查询向量为空")
	}

	collectionName = k.sanitizeCollectionName(collectionName)

	// 获取集合的 UUID
	collectionUUID, err := k.ensureChromaCollection(podName, namespace, port, collectionName)
	if err != nil {
		return nil, fmt.Errorf("获取集合信息失败: %v", err)
	}

	// Chroma v2 API 查询接口
	tenant := "default_tenant"
	database := "default_database"

	requestBody := map[string]interface{}{
		"query_embeddings": [][]float64{embeddings[0]},
		"n_results":        topK,
		"include":          []string{"documents", "metadatas", "distances"},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %v", err)
	}

	req := K8s.ClientSet.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		Name(fmt.Sprintf("%s:%d", podName, port)).
		SubResource("proxy").
		Suffix(fmt.Sprintf("/api/v2/tenants/%s/databases/%s/collections/%s/query", tenant, database, collectionUUID)).
		Body(jsonData).
		SetHeader("Content-Type", "application/json")

	result := req.Do(context.TODO())
	if result.Error() != nil {
		return nil, fmt.Errorf("请求 Chroma API 失败: %v", result.Error())
	}

	body, err := result.Raw()
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, body: %s", err, string(body))
	}

	// 检查是否有错误
	if errMsg, ok := responseData["error"].(string); ok && errMsg != "" {
		return nil, fmt.Errorf("chroma API 返回错误: %s", errMsg)
	}

	// 检查结果是否为空，如果为空可能是集合中没有数据
	if ids, ok := responseData["ids"].([]interface{}); ok && len(ids) > 0 {
		if idList, ok := ids[0].([]interface{}); ok && len(idList) == 0 {
			// 结果为空，可能是集合中没有数据或查询向量不匹配
			// 尝试获取集合信息来确认是否有数据
			return map[string]interface{}{
				"status":          "success",
				"knowledge_type":  "chromadb",
				"collection_name": collectionName,
				"query_text":      queryText,
				"top_k":           topK,
				"results":         responseData,
				"warning":         "查询结果为空，请确认：1. 集合中是否有数据；2. 查询向量和存储向量是否使用相同的模型",
			}, nil
		}
	}

	return map[string]interface{}{
		"status":          "success",
		"knowledge_type":  "chromadb",
		"collection_name": collectionName,
		"query_text":      queryText,
		"top_k":           topK,
		"results":         responseData,
	}, nil
}

// ========== Milvus 查询 ==========

// queryMilvus 查询 Milvus
func (k *knowledge) queryMilvus(podName, namespace, collectionName, queryText string, topK int) (interface{}, error) {
	pod, port, err := k.getPodInfo(podName, namespace, 19530)
	if err != nil {
		return nil, err
	}

	if topK <= 0 {
		topK = 5
	}

	// 获取 Ollama 信息并生成查询向量
	ollamaPodName, ollamaNamespace, ollamaModel := k.getOllamaInfo(pod, namespace)
	if ollamaPodName == "" || ollamaModel == "" {
		return nil, fmt.Errorf("查询需要向量嵌入，请确保知识库绑定了 Ollama")
	}

	embeddings, err := k.generateEmbeddings(ollamaPodName, ollamaNamespace, ollamaModel, []string{queryText})
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %v", err)
	}
	if len(embeddings) == 0 || len(embeddings[0]) == 0 {
		return nil, fmt.Errorf("生成的查询向量为空")
	}

	collectionName = k.sanitizeCollectionName(collectionName)

	// Milvus 使用 HTTP API 进行查询
	// 注意：Milvus 的 HTTP API 可能因版本而异，这里使用通用的搜索接口
	requestBody := map[string]interface{}{
		"collection_name": collectionName,
		"vector":          embeddings[0],
		"top_k":           topK,
		"output_fields":   []string{"id", "text"},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %v", err)
	}

	req := K8s.ClientSet.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		Name(fmt.Sprintf("%s:%d", podName, port)).
		SubResource("proxy").
		Suffix("/v1/vector/search").
		Body(jsonData).
		SetHeader("Content-Type", "application/json")

	result := req.Do(context.TODO())
	if result.Error() != nil {
		return nil, fmt.Errorf("请求 Milvus API 失败: %v", result.Error())
	}

	body, err := result.Raw()
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, body: %s", err, string(body))
	}

	return map[string]interface{}{
		"status":          "success",
		"knowledge_type":  "milvus",
		"collection_name": collectionName,
		"query_text":      queryText,
		"top_k":           topK,
		"results":         responseData,
	}, nil
}

// ========== Weaviate 查询 ==========

// queryWeaviate 查询 Weaviate
func (k *knowledge) queryWeaviate(podName, namespace, collectionName, queryText string, topK int) (interface{}, error) {
	pod, port, err := k.getPodInfo(podName, namespace, 8080)
	if err != nil {
		return nil, err
	}

	if topK <= 0 {
		topK = 5
	}

	// 获取 Ollama 信息并生成查询向量
	ollamaPodName, ollamaNamespace, ollamaModel := k.getOllamaInfo(pod, namespace)
	if ollamaPodName == "" || ollamaModel == "" {
		return nil, fmt.Errorf("查询需要向量嵌入，请确保知识库绑定了 Ollama")
	}

	embeddings, err := k.generateEmbeddings(ollamaPodName, ollamaNamespace, ollamaModel, []string{queryText})
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %v", err)
	}
	if len(embeddings) == 0 || len(embeddings[0]) == 0 {
		return nil, fmt.Errorf("生成的查询向量为空")
	}

	collectionName = k.sanitizeCollectionName(collectionName)

	// Weaviate 使用 GraphQL 进行查询
	// 使用向量搜索
	graphQLQuery := fmt.Sprintf(`{
		Get {
			%s(nearVector: {
				vector: %s
			}, limit: %d) {
				text
				source
				chunk
				_additional {
					distance
				}
			}
		}
	}`, collectionName, k.formatVectorForGraphQL(embeddings[0]), topK)

	jsonData, err := json.Marshal(map[string]interface{}{
		"query": graphQLQuery,
	})
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %v", err)
	}

	req := K8s.ClientSet.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		Name(fmt.Sprintf("%s:%d", podName, port)).
		SubResource("proxy").
		Suffix("/v1/graphql").
		Body(jsonData).
		SetHeader("Content-Type", "application/json")

	result := req.Do(context.TODO())
	if result.Error() != nil {
		return nil, fmt.Errorf("请求 Weaviate API 失败: %v", result.Error())
	}

	body, err := result.Raw()
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, body: %s", err, string(body))
	}

	return map[string]interface{}{
		"status":          "success",
		"knowledge_type":  "weaviate",
		"collection_name": collectionName,
		"query_text":      queryText,
		"top_k":           topK,
		"results":         responseData,
	}, nil
}

// formatVectorForGraphQL 将向量格式化为 GraphQL 格式
func (k *knowledge) formatVectorForGraphQL(vector []float64) string {
	var parts []string
	for _, v := range vector {
		parts = append(parts, fmt.Sprintf("%.6f", v))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// ChatWithKnowledgeBase 结合知识库进行聊天
func (k *knowledge) ChatWithKnowledgeBase(params *kubeDto.ChatWithKBInput) (interface{}, error) {
	// 设置默认值
	topK := params.TopK
	if topK <= 0 {
		topK = 5
	}

	// 1. 查询知识库获取相关文档
	queryResult, err := k.QueryKnowledge(
		params.KnowledgePodName,
		params.KnowledgeNamespace,
		params.KnowledgeType,
		params.CollectionName,
		params.Question,
		topK,
	)
	if err != nil {
		return nil, fmt.Errorf("查询知识库失败: %v", err)
	}

	// 2. 从查询结果中提取文档内容
	documents := k.extractDocumentsFromQueryResult(queryResult, params.KnowledgeType)
	if len(documents) == 0 {
		return nil, fmt.Errorf("知识库中未找到相关文档，请确认集合中是否有数据")
	}

	// 3. 构建包含上下文的系统提示词
	systemPrompt := k.buildSystemPromptWithContext(params.SystemPrompt, documents)

	// 4. 构建消息列表
	messages := []kubeDto.OllamaChatMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: params.Question,
		},
	}

	// 5. 调用 Ollama Chat API
	chatResult, err := Ollama.Chat(
		params.OllamaPodName,
		params.OllamaNamespace,
		params.OllamaModel,
		messages,
		params.Stream,
	)
	if err != nil {
		return nil, fmt.Errorf("调用模型失败: %v", err)
	}

	// 6. 返回结果（包含查询到的文档和模型回答）
	return map[string]interface{}{
		"answer":            chatResult,
		"related_documents": documents,
		"question":          params.Question,
		"top_k":             topK,
	}, nil
}

// extractDocumentsFromQueryResult 从查询结果中提取文档内容
func (k *knowledge) extractDocumentsFromQueryResult(queryResult interface{}, knowledgeType string) []string {
	var documents []string

	queryResultMap, ok := queryResult.(map[string]interface{})
	if !ok {
		return documents
	}

	results, ok := queryResultMap["results"].(map[string]interface{})
	if !ok {
		return documents
	}

	knowledgeType = strings.ToLower(knowledgeType)
	switch knowledgeType {
	case "chromadb", "chroma":
		// ChromaDB 返回格式: {"ids": [[...]], "documents": [[...]], "metadatas": [[...]], "distances": [[...]]}
		if docs, ok := results["documents"].([]interface{}); ok && len(docs) > 0 {
			if docList, ok := docs[0].([]interface{}); ok {
				for _, doc := range docList {
					if docStr, ok := doc.(string); ok && docStr != "" {
						documents = append(documents, docStr)
					}
				}
			}
		}
	case "milvus":
		// Milvus 返回格式可能不同，需要根据实际 API 调整
		if data, ok := results["data"].([]interface{}); ok {
			for _, item := range data {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if text, ok := itemMap["text"].(string); ok && text != "" {
						documents = append(documents, text)
					}
				}
			}
		}
	case "weaviate":
		// Weaviate 返回格式: {"data": {"Get": {"ClassName": [...]}}}
		if data, ok := results["data"].(map[string]interface{}); ok {
			if get, ok := data["Get"].(map[string]interface{}); ok {
				for _, classData := range get {
					if items, ok := classData.([]interface{}); ok {
						for _, item := range items {
							if itemMap, ok := item.(map[string]interface{}); ok {
								if props, ok := itemMap["properties"].(map[string]interface{}); ok {
									if text, ok := props["text"].(string); ok && text != "" {
										documents = append(documents, text)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return documents
}

// buildSystemPromptWithContext 构建包含上下文的系统提示词
func (k *knowledge) buildSystemPromptWithContext(customPrompt string, documents []string) string {
	var prompt strings.Builder

	if customPrompt != "" {
		prompt.WriteString(customPrompt)
		prompt.WriteString("\n\n")
	} else {
		prompt.WriteString("你是一个智能助手，能够基于提供的知识库内容回答问题。")
		prompt.WriteString("请仔细阅读以下相关文档，然后回答用户的问题。")
		prompt.WriteString("如果文档中没有相关信息，请如实告知用户。\n\n")
	}

	prompt.WriteString("相关文档内容：\n")
	for i, doc := range documents {
		prompt.WriteString(fmt.Sprintf("文档 %d:\n%s\n\n", i+1, doc))
	}

	prompt.WriteString("请基于以上文档内容回答用户的问题。")

	return prompt.String()
}
