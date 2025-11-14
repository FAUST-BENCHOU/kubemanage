package kube

import (
	"context"
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
