package kube

import (
	"context"
	"encoding/json"
	"fmt"

	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/noovertime7/kubemanage/dto/kubeDto"
)

var Ollama ollama

type ollama struct{}

type OllamaResp struct {
	Total int                    `json:"total"`
	Items []OllamaDeploymentInfo `json:"items"`
}

type OllamaDeploymentInfo struct {
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace"`
	Type         string            `json:"type"` // deployment 或 daemonset
	Image        string            `json:"image"`
	Port         int32             `json:"port"`
	NodeSelector map[string]string `json:"node_selector"`
	Status       string            `json:"status"`
	Pods         int32             `json:"pods"`
	ReadyPods    int32             `json:"ready_pods"`
}

// DeployOllama 部署Ollama到指定节点
func (o *ollama) DeployOllama(data *kubeDto.OllamaDeployInput) error {
	// 设置默认标签
	labels := map[string]string{
		"app":     "ollama",
		"name":    data.Name,
		"managed": "kubemanage",
	}
	// 合并用户提供的标签
	for k, v := range data.Labels {
		labels[k] = v
	}

	// 创建 PVC（如果需要存储）
	if data.StorageSize != "" {
		if err := o.createPVC(data); err != nil {
			return fmt.Errorf("创建PVC失败: %v", err)
		}
	}

	// 创建 Service
	if err := o.createService(data, labels); err != nil {
		return fmt.Errorf("创建Service失败: %v", err)
	}

	// 根据部署类型创建 Deployment 或 DaemonSet
	if data.DeployType == "daemonset" {
		return o.createDaemonSet(data, labels)
	}
	return o.createDeployment(data, labels)
}

// createDeployment 创建 Deployment
func (o *ollama) createDeployment(data *kubeDto.OllamaDeployInput, labels map[string]string) error {
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
							Name:  "ollama",
							Image: data.Image,
							Ports: []coreV1.ContainerPort{
								{
									Name:          "http",
									Protocol:      coreV1.ProtocolTCP,
									ContainerPort: data.Port,
								},
							},
							Env: []coreV1.EnvVar{
								{
									Name:  "OLLAMA_HOST",
									Value: fmt.Sprintf("0.0.0.0:%d", data.Port),
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

	// 设置资源限制
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

	// 设置存储卷
	if data.StorageSize != "" {
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = []coreV1.VolumeMount{
			{
				Name:      "ollama-data",
				MountPath: "/root/.ollama",
			},
		}
		deployment.Spec.Template.Spec.Volumes = []coreV1.Volume{
			{
				Name: "ollama-data",
				VolumeSource: coreV1.VolumeSource{
					PersistentVolumeClaim: &coreV1.PersistentVolumeClaimVolumeSource{
						ClaimName: fmt.Sprintf("%s-pvc", data.Name),
					},
				},
			},
		}
	}

	// 添加健康检查
	deployment.Spec.Template.Spec.Containers[0].LivenessProbe = &coreV1.Probe{
		ProbeHandler: coreV1.ProbeHandler{
			HTTPGet: &coreV1.HTTPGetAction{
				Path: "/api/tags",
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: data.Port,
				},
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       10,
		TimeoutSeconds:      5,
	}

	deployment.Spec.Template.Spec.Containers[0].ReadinessProbe = &coreV1.Probe{
		ProbeHandler: coreV1.ProbeHandler{
			HTTPGet: &coreV1.HTTPGetAction{
				Path: "/api/tags",
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: data.Port,
				},
			},
		},
		InitialDelaySeconds: 10,
		PeriodSeconds:       5,
		TimeoutSeconds:      3,
	}

	_, err := K8s.ClientSet.AppsV1().Deployments(data.NameSpace).Create(context.TODO(), deployment, metaV1.CreateOptions{})
	return err
}

// createDaemonSet 创建 DaemonSet
func (o *ollama) createDaemonSet(data *kubeDto.OllamaDeployInput, labels map[string]string) error {
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
							Name:  "ollama",
							Image: data.Image,
							Ports: []coreV1.ContainerPort{
								{
									Name:          "http",
									Protocol:      coreV1.ProtocolTCP,
									ContainerPort: data.Port,
								},
							},
							Env: []coreV1.EnvVar{
								{
									Name:  "OLLAMA_HOST",
									Value: fmt.Sprintf("0.0.0.0:%d", data.Port),
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
				Name:      "ollama-data",
				MountPath: "/root/.ollama",
			},
		}
		daemonSet.Spec.Template.Spec.Volumes = []coreV1.Volume{
			{
				Name: "ollama-data",
				VolumeSource: coreV1.VolumeSource{
					PersistentVolumeClaim: &coreV1.PersistentVolumeClaimVolumeSource{
						ClaimName: fmt.Sprintf("%s-pvc", data.Name),
					},
				},
			},
		}
	}

	// 添加健康检查
	daemonSet.Spec.Template.Spec.Containers[0].LivenessProbe = &coreV1.Probe{
		ProbeHandler: coreV1.ProbeHandler{
			HTTPGet: &coreV1.HTTPGetAction{
				Path: "/api/tags",
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: data.Port,
				},
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       10,
		TimeoutSeconds:      5,
	}

	daemonSet.Spec.Template.Spec.Containers[0].ReadinessProbe = &coreV1.Probe{
		ProbeHandler: coreV1.ProbeHandler{
			HTTPGet: &coreV1.HTTPGetAction{
				Path: "/api/tags",
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: data.Port,
				},
			},
		},
		InitialDelaySeconds: 10,
		PeriodSeconds:       5,
		TimeoutSeconds:      3,
	}

	_, err := K8s.ClientSet.AppsV1().DaemonSets(data.NameSpace).Create(context.TODO(), daemonSet, metaV1.CreateOptions{})
	return err
}

// createService 创建 Service
func (o *ollama) createService(data *kubeDto.OllamaDeployInput, labels map[string]string) error {
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
					Protocol:   coreV1.ProtocolTCP,
					Port:       data.Port,
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: data.Port},
				},
			},
			Type: coreV1.ServiceTypeClusterIP,
		},
	}

	_, err := K8s.ClientSet.CoreV1().Services(data.NameSpace).Create(context.TODO(), service, metaV1.CreateOptions{})
	return err
}

// createPVC 创建 PersistentVolumeClaim
func (o *ollama) createPVC(data *kubeDto.OllamaDeployInput) error {
	storageClass := data.StorageClass
	if storageClass == "" {
		storageClass = "standard"
	}

	pvc := &coreV1.PersistentVolumeClaim{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fmt.Sprintf("%s-pvc", data.Name),
			Namespace: data.NameSpace,
			Labels: map[string]string{
				"app":  "ollama",
				"name": data.Name,
			},
		},
		Spec: coreV1.PersistentVolumeClaimSpec{
			AccessModes: []coreV1.PersistentVolumeAccessMode{
				coreV1.ReadWriteOnce,
			},
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

// GetOllamaList 获取Ollama部署列表
func (o *ollama) GetOllamaList(filterName, namespace, nodeName string, limit, page int) (*OllamaResp, error) {
	var items []OllamaDeploymentInfo

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

	// 收集所有带有 ollama 标签的资源
	for _, ns := range namespaces {
		// 获取 Deployment
		deployList, err := K8s.ClientSet.AppsV1().Deployments(ns).List(context.TODO(), metaV1.ListOptions{
			LabelSelector: "app=ollama",
		})
		if err == nil {
			for _, deploy := range deployList.Items {
				if filterName == "" || deploy.Name == filterName {
					info := o.convertDeploymentToInfo(&deploy, nodeName)
					if info != nil {
						items = append(items, *info)
					}
				}
			}
		}

		// 获取 DaemonSet
		dsList, err := K8s.ClientSet.AppsV1().DaemonSets(ns).List(context.TODO(), metaV1.ListOptions{
			LabelSelector: "app=ollama",
		})
		if err == nil {
			for _, ds := range dsList.Items {
				if filterName == "" || ds.Name == filterName {
					info := o.convertDaemonSetToInfo(&ds, nodeName)
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

	var pagedItems []OllamaDeploymentInfo
	if start < end {
		pagedItems = items[start:end]
	}

	return &OllamaResp{
		Total: total,
		Items: pagedItems,
	}, nil
}

// convertDeploymentToInfo 转换 Deployment 为 OllamaDeploymentInfo
func (o *ollama) convertDeploymentToInfo(deploy *appsV1.Deployment, nodeName string) *OllamaDeploymentInfo {
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

	info := &OllamaDeploymentInfo{
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

// convertDaemonSetToInfo 转换 DaemonSet 为 OllamaDeploymentInfo
func (o *ollama) convertDaemonSetToInfo(ds *appsV1.DaemonSet, nodeName string) *OllamaDeploymentInfo {
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

	info := &OllamaDeploymentInfo{
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

// PullModel 拉取模型到指定的 Pod
func (o *ollama) PullModel(podName, namespace, modelName string) error {
	// 获取 Pod 信息以确定端口
	pod, err := Pod.GetPodDetail(podName, namespace)
	if err != nil {
		return fmt.Errorf("获取Pod信息失败: %v", err)
	}

	// 检查 Pod 是否就绪
	if pod.Status.Phase != coreV1.PodRunning {
		return fmt.Errorf("pod %s 状态为 %s，请等待Pod启动完成", podName, pod.Status.Phase)
	}

	// 获取端口
	var port int32 = 11434
	if len(pod.Spec.Containers) > 0 {
		for _, containerPort := range pod.Spec.Containers[0].Ports {
			if containerPort.Name == "http" || containerPort.ContainerPort == 11434 {
				port = containerPort.ContainerPort
				break
			}
		}
	}

	// 准备请求体
	requestBody := map[string]string{
		"name": modelName,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %v", err)
	}

	// 使用 Kubernetes API Server 代理访问 Pod
	// 路径格式: /api/v1/namespaces/{namespace}/pods/{name}:{port}/proxy/{path}
	req := K8s.ClientSet.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		Name(fmt.Sprintf("%s:%d", podName, port)).
		SubResource("proxy").
		Suffix("/api/pull").
		Body(jsonData).
		SetHeader("Content-Type", "application/json")

	// 发送请求（设置较长的超时时间，因为模型下载可能需要较长时间）
	result := req.Do(context.TODO())
	if result.Error() != nil {
		return fmt.Errorf("请求Ollama API失败: %v", result.Error())
	}

	// 读取响应
	body, err := result.Raw()
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查状态码（通过检查响应内容来判断是否成功）
	// Ollama pull API 返回流式响应，即使成功也可能不是 200
	// 检查响应是否包含错误信息
	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err == nil {
		// 如果能解析为 JSON，检查是否有错误字段
		if errMsg, ok := responseData["error"].(string); ok && errMsg != "" {
			return fmt.Errorf("ollama API返回错误: %s", errMsg)
		}
	}

	return nil
}

// GetModelList 获取指定 Pod 的模型列表
func (o *ollama) GetModelList(podName, namespace string) (interface{}, error) {
	// 获取 Pod 信息以确定端口
	pod, err := Pod.GetPodDetail(podName, namespace)
	if err != nil {
		return nil, fmt.Errorf("获取Pod信息失败: %v", err)
	}

	// 检查 Pod 是否就绪
	if pod.Status.Phase != coreV1.PodRunning {
		return nil, fmt.Errorf("pod %s 状态为 %s，请等待Pod启动完成", podName, pod.Status.Phase)
	}

	// 获取端口
	var port int32 = 11434
	if len(pod.Spec.Containers) > 0 {
		for _, containerPort := range pod.Spec.Containers[0].Ports {
			if containerPort.Name == "http" || containerPort.ContainerPort == 11434 {
				port = containerPort.ContainerPort
				break
			}
		}
	}

	// 使用 Kubernetes API Server 代理访问 Pod
	// 路径格式: /api/v1/namespaces/{namespace}/pods/{name}:{port}/proxy/{path}
	req := K8s.ClientSet.CoreV1().RESTClient().Get().
		Namespace(namespace).
		Resource("pods").
		Name(fmt.Sprintf("%s:%d", podName, port)).
		SubResource("proxy").
		Suffix("/api/tags")

	// 发送请求
	result := req.Do(context.TODO())
	if result.Error() != nil {
		return nil, fmt.Errorf("请求Ollama API失败: %v", result.Error())
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

	return responseData, nil
}

// DeleteModel 删除指定 Pod 中的模型
func (o *ollama) DeleteModel(podName, namespace, modelName string) error {
	// 获取 Pod 信息以确定端口
	pod, err := Pod.GetPodDetail(podName, namespace)
	if err != nil {
		return fmt.Errorf("获取Pod信息失败: %v", err)
	}

	// 检查 Pod 是否就绪
	if pod.Status.Phase != coreV1.PodRunning {
		return fmt.Errorf("pod %s 状态为 %s，请等待Pod启动完成", podName, pod.Status.Phase)
	}

	// 获取端口
	var port int32 = 11434
	if len(pod.Spec.Containers) > 0 {
		for _, containerPort := range pod.Spec.Containers[0].Ports {
			if containerPort.Name == "http" || containerPort.ContainerPort == 11434 {
				port = containerPort.ContainerPort
				break
			}
		}
	}

	// 准备请求体
	requestBody := map[string]string{
		"name": modelName,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %v", err)
	}

	// 使用 Kubernetes API Server 代理访问 Pod
	// 路径格式: /api/v1/namespaces/{namespace}/pods/{name}:{port}/proxy/{path}
	// Ollama delete API 使用 POST 方法
	req := K8s.ClientSet.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		Name(fmt.Sprintf("%s:%d", podName, port)).
		SubResource("proxy").
		Suffix("/api/delete").
		Body(jsonData).
		SetHeader("Content-Type", "application/json")

	// 发送请求
	result := req.Do(context.TODO())
	if result.Error() != nil {
		return fmt.Errorf("请求Ollama API失败: %v", result.Error())
	}

	// 读取响应
	body, err := result.Raw()
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查响应是否包含错误信息
	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err == nil {
		if errMsg, ok := responseData["error"].(string); ok && errMsg != "" {
			return fmt.Errorf("ollama API返回错误: %s", errMsg)
		}
	}

	return nil
}

// GetModelDetail 获取指定 Pod 中模型的详情
func (o *ollama) GetModelDetail(podName, namespace, modelName string) (interface{}, error) {
	// 获取 Pod 信息以确定端口
	pod, err := Pod.GetPodDetail(podName, namespace)
	if err != nil {
		return nil, fmt.Errorf("获取Pod信息失败: %v", err)
	}

	// 检查 Pod 是否就绪
	if pod.Status.Phase != coreV1.PodRunning {
		return nil, fmt.Errorf("pod %s 状态为 %s，请等待Pod启动完成", podName, pod.Status.Phase)
	}

	// 获取端口
	var port int32 = 11434
	if len(pod.Spec.Containers) > 0 {
		for _, containerPort := range pod.Spec.Containers[0].Ports {
			if containerPort.Name == "http" || containerPort.ContainerPort == 11434 {
				port = containerPort.ContainerPort
				break
			}
		}
	}

	// 准备请求体
	requestBody := map[string]string{
		"name": modelName,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %v", err)
	}

	// 使用 Kubernetes API Server 代理访问 Pod
	// 路径格式: /api/v1/namespaces/{namespace}/pods/{name}:{port}/proxy/{path}
	// Ollama show API 使用 POST 方法，需要传递模型名称作为请求体
	req := K8s.ClientSet.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		Name(fmt.Sprintf("%s:%d", podName, port)).
		SubResource("proxy").
		Suffix("/api/show").
		Body(jsonData).
		SetHeader("Content-Type", "application/json")

	// 发送请求
	result := req.Do(context.TODO())
	if result.Error() != nil {
		return nil, fmt.Errorf("请求Ollama API失败: %v", result.Error())
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

	return responseData, nil
}
