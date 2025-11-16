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

	// 添加健康检查（使用默认路径）
	deployment.Spec.Template.Spec.Containers[0].LivenessProbe = &coreV1.Probe{
		ProbeHandler: coreV1.ProbeHandler{
			HTTPGet: &coreV1.HTTPGetAction{
				Path: "/health",
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
				Path: "/health",
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
